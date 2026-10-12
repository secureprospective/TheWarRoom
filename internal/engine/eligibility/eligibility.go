// Package eligibility decides whether MFL will allow a roster move, from held feeds only.
// It never guesses: every verdict is eligible, ineligible with the reason, or unknown with the
// reason, and MFL remains the authority. The captured MFL owner pages of 2026-10-10 are the
// test oracle (committed as testdata/answer-key.csv); the rules below reproduce their
// verdicts exactly.
//
// MFL's rules (league settings, Ring1_Gap_Closure.md §8):
//   - IR: the player is classified IR by his NFL team — an injuries-feed status of IR, IR-R,
//     IR-PUP or IR-NFI — and not Suspended, holdout or Reserve/COVID-19.
//   - Taxi: players with less than 3 years of NFL experience.
//
// The experience source is the players export's draft_year (already held in the app's player
// directory): a player drafted in year Y is in his (season−Y+1)-th NFL season, so fewer than
// 3 years of experience means season−Y < 3. The capture agrees on every row: all 10 demotable
// players were drafted 2024 or later, all 28 refused ones 2023 or earlier.
package eligibility

import (
	"fmt"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/domain"
)

// Verdict is what the app believes MFL will do with a move.
type Verdict string

const (
	Eligible   Verdict = "eligible"
	Ineligible Verdict = "ineligible"
	Unknown    Verdict = "unknown"
)

// Direction is one move between the active roster, IR and the taxi squad.
type Direction string

const (
	ToIR    Direction = "to_ir"
	OffIR   Direction = "off_ir"
	ToTaxi  Direction = "to_taxi"
	OffTaxi Direction = "off_taxi"
)

// Source tokens name the feed a decision reads; the snapshot carries each token's provenance.
const (
	SourceInjuries = "injuries"
	SourcePlayers  = "players"
	SourceRosters  = "rosters"
)

// Decision is one direction's verdict. Reason is set for ineligible and unknown, empty for
// eligible; a decision never carries a confidence or score.
type Decision struct {
	Direction Direction `json:"direction"`
	Verdict   Verdict   `json:"verdict"`
	Reason    string    `json:"reason,omitempty"`
	Source    string    `json:"source"`
}

// PlayerVerdicts is one player's two move slots. Each slot holds the decision for the
// direction the player's held roster status makes possible: an active player's IR slot is a
// deactivation verdict, an IR player's an activation verdict.
type PlayerVerdicts struct {
	IR   Decision `json:"ir"`
	Taxi Decision `json:"taxi"`
}

// Capacity is one squad's slot count from the league rules. Held is false when the rulebook
// does not carry the setting, in which case capacity cannot be judged.
type Capacity struct {
	Slots int
	Held  bool
}

// RosterPlayer is one rostered player's held fantasy status.
type RosterPlayer struct {
	ID     string // canonical MFL id
	Status domain.RosterStatus
}

// Franchise is one franchise's roster, for the per-squad occupancy counts.
type Franchise struct {
	ID      string
	Players []RosterPlayer
}

// Injuries is the held MFL injuries feed: one NFL designation per player. Status is MFL's open
// vocabulary; an empty map with a live freshness means nobody is designated.
type Injuries struct {
	Status map[string]string
	Fresh  domain.Freshness
}

// DraftYear is a player's NFL draft year from the players export.
type DraftYear struct {
	Year int
	Held bool
}

// Players is the held players feed as the rules read it: draft years for experience.
type Players struct {
	DraftYear map[string]DraftYear
	Fresh     domain.Freshness
}

// Input is everything the rules read; every field arrives as a plain value.
type Input struct {
	Season   int
	IR       Capacity
	Taxi     Capacity
	Rosters  []Franchise
	Injuries Injuries
	Players  Players
}

// staleReason marks input whose feed is not live under the freshness contract
// (freshness.go: stale is the last good copy after a failed fetch, fail is no data).
const staleReason = "stale"

// mflPlaceholderDraftYear: MFL fills draft_year with 1970 where it holds none (every one of
// the 384 such rows in the 2026 export is a team or positional aggregate), and omits the field
// outright for commissioner-created players. Anything at or below it means "no draft year".
const mflPlaceholderDraftYear = 1970

// taxiExperienceLimit: fewer than this many years of NFL experience may be demoted.
const taxiExperienceLimit = 3

// Evaluate produces the verdicts for every rostered player. A player absent from the map is
// not on any held roster.
func Evaluate(in Input) map[string]PlayerVerdicts {
	onIR, onTaxi := occupancy(in.Rosters)
	out := make(map[string]PlayerVerdicts)
	for _, f := range in.Rosters {
		for _, p := range f.Players {
			if _, seen := out[p.ID]; seen {
				continue
			}
			out[p.ID] = playerVerdicts(in, p, onIR[f.ID], onTaxi[f.ID])
		}
	}
	return out
}

func occupancy(rosters []Franchise) (onIR, onTaxi map[string]int) {
	onIR, onTaxi = map[string]int{}, map[string]int{}
	for _, f := range rosters {
		for _, p := range f.Players {
			switch p.Status {
			case domain.RosterIR:
				onIR[f.ID]++
			case domain.RosterTaxi:
				onTaxi[f.ID]++
			case domain.RosterActive:
			default:
				// An unrecognized status still occupies no squad.
			}
		}
	}
	return onIR, onTaxi
}

func playerVerdicts(in Input, p RosterPlayer, onIR, onTaxi int) PlayerVerdicts {
	if p.Status == domain.RosterIR {
		// The capture offered activation to every on-IR player: MFL gates activation on
		// being on IR, nothing else observed.
		return PlayerVerdicts{IR: eligible(OffIR, SourceRosters),
			Taxi: ineligible(ToTaxi, SourceRosters, "on injured reserve")}
	}
	if p.Status == domain.RosterTaxi {
		// The IR page offers deactivation only on the active roster: taxi players appear
		// solely on the promote table.
		return PlayerVerdicts{IR: ineligible(ToIR, SourceRosters, "on the taxi squad"),
			Taxi: eligible(OffTaxi, SourceRosters)}
	}
	// The active roster, and any unrecognized status, still reads the feeds.
	return PlayerVerdicts{IR: toIR(in, p.ID, onIR), Taxi: toTaxi(in, p.ID, onTaxi)}
}

func toIR(in Input, id string, onIR int) Decision {
	if in.Injuries.Fresh.State != domain.FreshLive {
		return unknown(ToIR, SourceInjuries, staleReason)
	}
	status, designated := in.Injuries.Status[id]
	switch {
	case designated && isIRDesignation(status):
		if !in.IR.Held {
			return unknown(ToIR, SourceRosters, "IR capacity not held")
		}
		if onIR >= in.IR.Slots {
			return ineligible(ToIR, SourceRosters, fmt.Sprintf("IR full (%d/%d)", onIR, in.IR.Slots))
		}
		return eligible(ToIR, SourceInjuries)
	case designated && status == "Suspended":
		return ineligible(ToIR, SourceInjuries, "suspended")
	case designated && status == "Holdout":
		return ineligible(ToIR, SourceInjuries, "holdout")
	default:
		return ineligible(ToIR, SourceInjuries, "not on NFL injured reserve")
	}
}

func toTaxi(in Input, id string, onTaxi int) Decision {
	if in.Players.Fresh.State != domain.FreshLive {
		return unknown(ToTaxi, SourcePlayers, staleReason)
	}
	if in.Season <= 0 {
		return unknown(ToTaxi, SourceRosters, "season not held")
	}
	dy, held := in.Players.DraftYear[id]
	if !held || !dy.Held || dy.Year <= mflPlaceholderDraftYear {
		return unknown(ToTaxi, SourcePlayers, "no draft year held")
	}
	if in.Season-dy.Year >= taxiExperienceLimit {
		return ineligible(ToTaxi, SourcePlayers, fmt.Sprintf(
			"%d or more years of NFL experience (drafted %d)", taxiExperienceLimit, dy.Year))
	}
	if !in.Taxi.Held {
		return unknown(ToTaxi, SourceRosters, "taxi capacity not held")
	}
	if onTaxi >= in.Taxi.Slots {
		return ineligible(ToTaxi, SourceRosters, fmt.Sprintf("taxi full (%d/%d)", onTaxi, in.Taxi.Slots))
	}
	return eligible(ToTaxi, SourcePlayers)
}

// isIRDesignation reports whether MFL's status is an NFL injured-reserve classification.
// The IR-prefixed forms cover variants beyond those seen so far (IR, IR-R, IR-PUP, IR-NFI).
func isIRDesignation(status string) bool {
	return strings.HasPrefix(status, "IR")
}

func eligible(direction Direction, source string) Decision {
	return Decision{Direction: direction, Verdict: Eligible, Source: source}
}

func ineligible(direction Direction, source, reason string) Decision {
	return Decision{Direction: direction, Verdict: Ineligible, Reason: reason, Source: source}
}

func unknown(direction Direction, source, reason string) Decision {
	return Decision{Direction: direction, Verdict: Unknown, Reason: reason, Source: source}
}

// NotChecked renders the draft note for an unknown verdict or unheld rules: MFL, not the
// app, decides.
func NotChecked(reason string) string {
	return "Not checked: " + reason + "; MFL will decide"
}

// IRClassification checks the NFL designation, not the fantasy activation direction or capacity.
func IRClassification(in Injuries, id string) Decision {
	return toIR(Input{Injuries: in, IR: Capacity{Slots: 1, Held: true}}, id, 0)
}

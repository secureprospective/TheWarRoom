package snapshot

import (
	"strconv"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/engine/eligibility"
	"github.com/secureprospective/TheWarRoom/internal/normalize"
)

// Injuries is the held MFL injuries feed as the eligibility rules read it: one NFL
// designation per canonical player id, with the feed's own provenance.
type Injuries struct {
	Status     map[string]string
	Provenance Provenance
}

// Eligibility is the snapshot's held-verdict section: MFL's IR and taxi rules per rostered
// player and direction. Feeds carries each source token's provenance, so a verdict's feed and
// as-of time are one lookup; no verdict repeats them.
type Eligibility struct {
	Players map[string]eligibility.PlayerVerdicts `json:"players"`
	Feeds   map[string]Provenance                 `json:"feeds"`
}

// Slots is the league's squad capacities from its held rules. A slot count that is not held
// keeps Held false, and the rules answer unknown rather than guessing a limit.
type Slots struct {
	IR   eligibility.Capacity
	Taxi eligibility.Capacity
}

// slotSetting reads one capacity setting ("injuredReserve", "taxiSquad") from the rules,
// override-aware through the rulebook's own getter.
func slotSetting(r Rules, key string) eligibility.Capacity {
	raw, ok := r.GetSetting(key)
	v, err := strconv.Atoi(strings.TrimSpace(raw))
	if !ok || err != nil || v < 0 {
		return eligibility.Capacity{}
	}
	return eligibility.Capacity{Slots: v, Held: true}
}

func (s storeSource) Slots() Slots {
	return Slots{IR: slotSetting(s.rules, "injuredReserve"), Taxi: slotSetting(s.rules, "taxiSquad")}
}

// buildEligibility runs the MFL rules over every rostered player. Players holds the whole
// league, so the inspector can show the lines for any rostered player, not only the user's.
func buildEligibility(
	meta Metadata, rosters Sourced[[]Roster], dir Directory, inj Injuries, slots Slots,
) Eligibility {
	input := eligibility.Input{
		Season:  meta.Season,
		IR:      slots.IR,
		Taxi:    slots.Taxi,
		Rosters: mapRosters(rosters.Value),
		Injuries: eligibility.Injuries{
			Status: inj.Status, Fresh: inj.Provenance.Freshness,
		},
		Players: eligibility.Players{
			DraftYear: draftYears(rosters.Value, dir.Lookup), Fresh: dir.Provenance.Freshness,
		},
	}
	return Eligibility{
		Players: eligibility.Evaluate(input),
		Feeds: map[string]Provenance{
			eligibility.SourceRosters:  rosters.Provenance,
			eligibility.SourceInjuries: inj.Provenance,
			eligibility.SourcePlayers:  dir.Provenance,
		},
	}
}

func mapRosters(rosters []Roster) []eligibility.Franchise {
	out := make([]eligibility.Franchise, 0, len(rosters))
	for _, r := range rosters {
		f := eligibility.Franchise{ID: r.FranchiseID, Players: make([]eligibility.RosterPlayer, 0, len(r.Players))}
		for _, p := range r.Players {
			f.Players = append(f.Players, eligibility.RosterPlayer{ID: p.ID.String(), Status: p.RosterStatus})
		}
		out = append(out, f)
	}
	return out
}

// draftYears reads each rostered player's NFL draft year from the directory. The league's own
// directory is the experience source, so a player it does not know stays unknown to the rules.
func draftYears(rosters []Roster, lookup normalize.Lookup) map[string]eligibility.DraftYear {
	out := make(map[string]eligibility.DraftYear)
	for _, r := range rosters {
		for _, p := range r.Players {
			id := p.ID.String()
			if _, seen := out[id]; seen {
				continue
			}
			dy := eligibility.DraftYear{}
			if facts, ok := lookup.Facts(id); ok && facts.HasDraftYear {
				dy = eligibility.DraftYear{Year: facts.DraftYear, Held: true}
			}
			out[id] = dy
		}
	}
	return out
}

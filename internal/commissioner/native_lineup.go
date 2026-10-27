package commissioner

import (
	"fmt"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

const maxLineupStarters = 100

// RawNativeLineup is the lineup a GM asks to have judged. Points are not part of it: this reading says
// whether the lineup is allowed, and what an invalid one costs.
type RawNativeLineup struct {
	League    string   `json:"league"`
	Franchise string   `json:"franchise"`
	Submitted bool     `json:"submitted"`
	Starters  []string `json:"starters"`
}

func (r RawNativeLineup) Validate() error {
	if !strings.HasPrefix(r.League, "native.") || r.League == "native." || strings.TrimSpace(r.League) != r.League ||
		strings.TrimSpace(r.Franchise) == "" || len(r.Starters) > maxLineupStarters || (!r.Submitted && len(r.Starters) != 0) {
		return fmt.Errorf("choose an app league, a franchise and its starters; a lineup that was not set has none")
	}
	seen := map[string]bool{}
	for _, id := range r.Starters {
		if strings.TrimSpace(id) == "" || seen[id] {
			return fmt.Errorf("list each starter once")
		}
		seen[id] = true
	}
	return nil
}

// NativeStarter is one starter of a judged lineup.
type NativeStarter struct {
	Player   string `json:"player"`
	Name     string `json:"name"`
	Position string `json:"position"`
}

// NativeLineupCheck is the plain-words verdict on one franchise's lineup.
type NativeLineupCheck struct {
	League        string                    `json:"league"`
	Name          string                    `json:"name"`
	Revision      int64                     `json:"revision"`
	RosterSeason  int                       `json:"rosterSeason"`
	RulesSeason   int                       `json:"rulesSeason"`
	Franchise     string                    `json:"franchise"`
	FranchiseName string                    `json:"franchiseName"`
	Source        string                    `json:"source"`
	Rules         []RuleLine                `json:"rules"`
	Starters      []NativeStarter           `json:"starters"`
	Verdict       nativecheck.LineupVerdict `json:"verdict"`
}

// NativeLineupCheckReading judges a lineup of active players on one native franchise. A starter must be
// on that franchise's active roster: IR and taxi players cannot start.
func NativeLineupCheckReading(in RawNativeLineup, records []Record, now time.Time, facts nativecheck.Facts) (NativeLineupCheck, error) {
	base, err := ReadNativeBase(in.League, records, now)
	if err != nil {
		return NativeLineupCheck{}, err
	}
	if !base.Assets.Seeded {
		return NativeLineupCheck{}, fmt.Errorf("record the league's starting snapshot before judging a lineup")
	}
	seat, ok := base.Directory.seat(in.Franchise)
	if !ok {
		return NativeLineupCheck{}, fmt.Errorf("choose one of this league's franchises")
	}
	active := map[string]bool{}
	for _, p := range base.playersOf()[in.Franchise] {
		active[p.ID] = p.Status == domain.RosterActive
	}
	out := NativeLineupCheck{League: in.League, Name: Replay(records).Name, Revision: int64(len(records)),
		RosterSeason: base.Assets.Snapshot.Season, RulesSeason: base.Rules.Season, Franchise: in.Franchise,
		FranchiseName: seat.Name, Source: base.source(), Starters: []NativeStarter{},
		Rules: RuleLines(base.Rules, "lineup.positions", "lineup.groups", "lineup.invalidPenalty", "lineup.missedFlag")}
	for _, id := range in.Starters {
		if !active[id] {
			return NativeLineupCheck{}, fmt.Errorf("a starter must be on this franchise's active roster")
		}
	}
	out.Verdict, out.Starters = judgeStarters(base.Rules.Lineup, in.Submitted, in.Starters, facts)
	return out, nil
}

// unjudgedLineup stands in for a verdict when the app does not hold where some starters play.
func unjudgedLineup(unplaced int) nativecheck.LineupVerdict {
	return nativecheck.LineupVerdict{Status: nativecheck.LineupUnjudged, Problems: []nativecheck.LineupProblem{},
		Words: fmt.Sprintf("Can't tell: the position of %d starters is not held", unplaced)}
}

func (d NativeDirectory) seat(franchise string) (NativeDirectorySeat, bool) {
	for _, f := range d.Franchises {
		if f.Franchise == franchise {
			return f, true
		}
	}
	return NativeDirectorySeat{}, false
}

package commissioner

import (
	"time"

	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

// NativeReserveFranchise is one franchise's IR and taxi squads against the league's rules.
type NativeReserveFranchise struct {
	Franchise    string                       `json:"franchise"`
	Name         string                       `json:"name"`
	Abbreviation string                       `json:"abbreviation"`
	IR           nativecheck.SquadUse         `json:"ir"`
	Taxi         nativecheck.SquadUse         `json:"taxi"`
	Players      []nativecheck.ReserveVerdict `json:"players"`
}

// NativeReserveCheck judges every IR and taxi player on every native franchise, and how full each squad is.
type NativeReserveCheck struct {
	Corrections  []Correction             `json:"corrections,omitempty"`
	League       string                   `json:"league"`
	Name         string                   `json:"name"`
	Revision     int64                    `json:"revision"`
	RosterSeason int                      `json:"rosterSeason"`
	RulesSeason  int                      `json:"rulesSeason"`
	Seeded       bool                     `json:"seeded"`
	Source       string                   `json:"source"`
	Rules        []RuleLine               `json:"rules"`
	Franchises   []NativeReserveFranchise `json:"franchises"`
}

// NativeReserveCheckReading is public: every owner may read who must leave a reserve squad. It judges
// this league's rules only; it makes no promise about what MFL would accept.
func NativeReserveCheckReading(league string, records []Record, now time.Time, facts nativecheck.Facts) (NativeReserveCheck, error) {
	base, err := ReadNativeBase(league, records, now)
	if err != nil {
		return NativeReserveCheck{}, err
	}
	changes, err := CorrectionsReading(league, records, now)
	if err != nil {
		return NativeReserveCheck{}, err
	}
	out := NativeReserveCheck{Corrections: changes.Items, League: league, Name: Replay(records).Name, Revision: int64(len(records)),
		RulesSeason: base.Rules.Season, Seeded: base.Assets.Seeded, Source: base.source(),
		Rules: RuleLines(base.Rules, "ir.slots", "ir.eligible", "ir.mustLeaveHealthy", "ir.autoReturn", "taxi.slots",
			"taxi.maximumYears", "taxi.basis", "taxi.autoPromote"),
		Franchises: []NativeReserveFranchise{}}
	if !base.Assets.Seeded {
		return out, nil
	}
	out.RosterSeason = base.Assets.Snapshot.Season
	players := base.playersOf()
	for _, seat := range base.Directory.Franchises {
		check := nativecheck.CheckReserves(base.Rules.IR, base.Rules.Taxi, out.RulesSeason,
			players[seat.Franchise], facts)
		if err := applyReserveExceptions(check.Players, records, now, seat.Franchise, base.Assets.Snapshot.PerSide()); err != nil {
			return NativeReserveCheck{}, err
		}
		out.Franchises = append(out.Franchises, NativeReserveFranchise{Franchise: seat.Franchise, Name: seat.Name,
			Abbreviation: seat.Abbreviation, IR: check.IR, Taxi: check.Taxi, Players: check.Players})
	}
	return out, nil
}

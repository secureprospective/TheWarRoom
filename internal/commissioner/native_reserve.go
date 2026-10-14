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
	base, err := readNativeBase(league, records, now)
	if err != nil {
		return NativeReserveCheck{}, err
	}
	out := NativeReserveCheck{League: league, Name: Replay(records).Name, Revision: int64(len(records)),
		RulesSeason: base.rules.Season, Seeded: base.assets.Seeded, Source: base.source(),
		Rules: ruleLines(base.rules, "ir.slots", "ir.eligible", "ir.mustLeaveHealthy", "taxi.slots",
			"taxi.maximumYears", "taxi.basis"),
		Franchises: []NativeReserveFranchise{}}
	if !base.assets.Seeded {
		return out, nil
	}
	out.RosterSeason = base.assets.Snapshot.Season
	players := base.playersOf()
	for _, seat := range base.directory.Franchises {
		check := nativecheck.CheckReserves(base.rules.IR, base.rules.Taxi, out.RulesSeason,
			players[seat.Franchise], facts)
		out.Franchises = append(out.Franchises, NativeReserveFranchise{Franchise: seat.Franchise, Name: seat.Name,
			Abbreviation: seat.Abbreviation, IR: check.IR, Taxi: check.Taxi, Players: check.Players})
	}
	return out, nil
}

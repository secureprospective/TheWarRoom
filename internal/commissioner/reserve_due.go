package commissioner

import (
	"fmt"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

const ReserveClockPrincipal = "native.reserve.clock"

// DueReserves uses the same eligibility and scoped exceptions as the public reserve check.
// Missing facts never cause an automatic move.
func DueReserves(league string, records []Record, at time.Time, facts nativecheck.Facts) ([]RawWaiverArgs, error) {
	rules, err := RulesNow(records, at)
	if err != nil {
		return nil, err
	}
	check, err := NativeReserveCheckReading(league, records, at, facts)
	if err != nil {
		return nil, err
	}
	due := []RawWaiverArgs{}
	for _, f := range check.Franchises {
		for _, p := range f.Players {
			enabled := rules.IR.AutoReturn
			if p.Squad == "Taxi" {
				enabled = rules.Taxi.AutoPromote
			}
			if enabled.State == leaguerules.IsKnown && enabled.Value && p.Standing == nativecheck.MustLeave {
				due = append(due, RawWaiverArgs{Player: p.Player, Franchise: f.Franchise, Bucket: "active"})
			}
		}
	}
	return due, nil
}

// PrepareDueReserve rechecks the move under the append lock. Automatic returns intentionally
// bypass owner windows and roster/cap admission; they never cut a player to make room.
func PrepareDueReserve(raw *RawCommand, records []Record, at time.Time, facts nativecheck.Facts) error {
	if raw.Command != RosterReserve || raw.Args.Waiver == nil {
		return fmt.Errorf("the reserve clock can only return an ineligible reserve player")
	}
	due, err := DueReserves(raw.League, records, at, facts)
	if err != nil {
		return err
	}
	for _, move := range due {
		if move.Player != raw.Args.Waiver.Player || move.Franchise != raw.Args.Waiver.Franchise || raw.Args.Waiver.Bucket != "active" {
			continue
		}
		assets, err := NativeAssetsReading(raw.League, records, at)
		if err != nil {
			return err
		}
		for _, p := range assets.Snapshot.Players {
			if p.ID == move.Player && p.Franchise == move.Franchise {
				raw.Args.WaiverOutcome = &WaiverOutcome{Player: &p, Season: assets.Snapshot.Season, Source: ReserveClockPrincipal}
				return nil
			}
		}
	}
	return fmt.Errorf("the reserve player is no longer due for automatic return")
}

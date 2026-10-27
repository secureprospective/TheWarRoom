package market

import (
	"fmt"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

// DueReserves uses the same eligibility and scoped exceptions as the public reserve check.
// Missing facts never cause an automatic move.
func DueReserves(league string, records []commissioner.Record, at time.Time, facts nativecheck.Facts) ([]commissioner.RawWaiverArgs, error) {
	rules, err := commissioner.RulesNow(records, at)
	if err != nil {
		//nolint:wrapcheck // Preserve the original refusal text across the package boundary.
		return nil, err
	}
	check, err := commissioner.NativeReserveCheckReading(league, records, at, facts)
	if err != nil {
		//nolint:wrapcheck // Preserve the original refusal text across the package boundary.
		return nil, err
	}
	due := []commissioner.RawWaiverArgs{}
	for _, f := range check.Franchises {
		for _, p := range f.Players {
			enabled := rules.IR.AutoReturn
			if p.Squad == "Taxi" {
				enabled = rules.Taxi.AutoPromote
			}
			if enabled.State == leaguerules.IsKnown && enabled.Value && p.Standing == nativecheck.MustLeave {
				due = append(due, commissioner.RawWaiverArgs{Player: p.Player, Franchise: f.Franchise, Bucket: "active"})
			}
		}
	}
	return due, nil
}

// PrepareDueReserve rechecks the move under the append lock. Automatic returns intentionally
// bypass owner windows and roster/cap admission; they never cut a player to make room.
func PrepareDueReserve(raw *commissioner.RawCommand, records []commissioner.Record, at time.Time, facts nativecheck.Facts) error {
	if raw.Command != commissioner.RosterReserve || raw.Args.Waiver == nil {
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
		assets, err := commissioner.NativeAssetsReading(raw.League, records, at)
		if err != nil {
			//nolint:wrapcheck // Preserve the original refusal text across the package boundary.
			return err
		}
		for _, p := range assets.Snapshot.Players {
			if p.ID == move.Player && p.Franchise == move.Franchise {
				raw.Args.WaiverOutcome = &commissioner.WaiverOutcome{Player: &p, Season: assets.Snapshot.Season, Source: commissioner.ReserveClockPrincipal}
				return nil
			}
		}
	}
	return fmt.Errorf("the reserve player is no longer due for automatic return")
}

package commissioner

import (
	"cmp"
	"fmt"
	"slices"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

// NativeCapYear separates dead money from the other counted commitments. Missing rulings stay absent.
type NativeCapYear struct {
	Season         int           `json:"season"`
	CommittedCents *domain.Money `json:"committedCents"`
	DeadCents      *domain.Money `json:"deadCents"`
	CapCents       *domain.Money `json:"capCents"`
	RoomCents      *domain.Money `json:"roomCents"`
}

// NativeCapRunway projects held flat contracts and dated charges over the league's contract horizon
// using today's cap rules, without assuming renewals, future moves or cap growth. The horizon is the
// longer of the league's maximum contract years and this franchise's last player expiration. Standing
// charges persist; a contract without an end year stays unchanged, as it does at rollover.
func NativeCapRunway(base NativeBase, franchise string) ([]NativeCapYear, error) {
	seed, rules := base.Assets.Snapshot, base.Rules
	cuts := map[string]bool{}
	for _, record := range base.Records {
		cuts[record.Receipt.Entry] = record.Command.Command == RosterCut
	}
	horizon := runwayHorizon(seed.Players, franchise, rules.Season, rules.Contracts.MaximumYears.Value)
	out := make([]NativeCapYear, 0, horizon)
	for season := rules.Season; season < rules.Season+horizon; season++ {
		players := []nativecheck.CapPlayer{}
		charges, liveCharges := []domain.Money{}, []domain.Money{}
		for _, p := range seed.Players {
			if p.Franchise == franchise && cmp.Or(p.ExpirationYear, season) >= season {
				players = append(players, nativecheck.CapPlayer{Salary: p.Salary, Status: p.Status})
			}
		}
		for _, c := range seed.Adjustments {
			if c.Franchise != franchise || !slices.Contains([]int{0, season}, c.Season) {
				continue
			}
			charges = append(charges, c.Amount)
			if c.Kind != deadMoney && !cuts[c.ID] {
				liveCharges = append(liveCharges, c.Amount)
			}
		}
		total, err := nativecheck.CheckCap(rules.Cap, players, charges, time.UTC)
		if err != nil {
			return nil, fmt.Errorf("count cap runway for %s in %d: %w", franchise, season, err)
		}
		live, err := nativecheck.CheckCap(rules.Cap, players, liveCharges, time.UTC)
		if err != nil {
			return nil, fmt.Errorf("count commitments for %s in %d: %w", franchise, season, err)
		}
		dead, counted, err := nativecheck.CountedChange(live, total)
		if err != nil {
			return nil, fmt.Errorf("count dead cap for %s in %d: %w", franchise, season, err)
		}
		row := NativeCapYear{Season: season, CommittedCents: live.CommittedCents,
			CapCents: total.CapCents, RoomCents: total.RoomCents}
		if counted {
			row.DeadCents = &dead
		}
		out = append(out, row)
	}
	return out, nil
}

func runwayHorizon(players []NativePlayer, franchise string, firstSeason, maxYears int) int {
	horizon := max(maxYears, 1)
	for _, p := range players {
		if p.Franchise == franchise {
			horizon = max(horizon, p.ExpirationYear-firstSeason+1)
		}
	}
	return horizon
}

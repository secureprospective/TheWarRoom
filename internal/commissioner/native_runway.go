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
type NativeCapProjection struct {
	Salaries map[string][]nativecheck.SeasonAmount
	Removed  map[string]bool
	Charges  []NativeCharge
}

func NativeCapRunway(base NativeBase, franchise string) ([]NativeCapYear, error) {
	return NativeCapRunwayProjected(base, franchise, NativeCapProjection{})
}

func NativeCapRunwayProjected(base NativeBase, franchise string, projection NativeCapProjection) ([]NativeCapYear, error) {
	seed, rules := base.Assets.Snapshot, base.Rules
	cuts := map[string]bool{}
	for _, record := range base.Records {
		cuts[record.Receipt.Entry] = record.Command.Command == RosterCut || (record.Command.Command == RosterAdd &&
			record.Command.Args.Waiver != nil && record.Command.Args.Waiver.Drop != "")
	}
	horizon := runwayHorizon(seed.Players, franchise, rules.Season, rules.Contracts.MaximumYears.Value)
	for _, schedule := range projection.Salaries {
		horizon = max(horizon, lastScheduleSeason(schedule)-rules.Season+1)
	}
	projectedCharges := append(slices.Clone(seed.Adjustments), projection.Charges...)
	out := make([]NativeCapYear, 0, horizon)
	for season := rules.Season; season < rules.Season+horizon; season++ {
		players := runwayPlayers(seed, franchise, projection, season)
		charges, liveCharges := runwayCharges(projectedCharges, cuts, franchise, season)
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
		row := capRunwayYear(season, live, total, dead, counted, seed.CheckContracts(franchise))
		out = append(out, row)
	}
	return out, nil
}

func runwayPlayers(seed *RawNativeSnapshot, franchise string, projection NativeCapProjection,
	season int) []nativecheck.CapPlayer {
	players := []nativecheck.CapPlayer{}
	for _, player := range seed.Players {
		schedule := projection.Salaries[player.ID]
		if len(schedule) == 0 {
			schedule = player.ContractSchedule
		}
		lastSeason := max(player.ExpirationYear, lastScheduleSeason(schedule))
		if player.Franchise == franchise && !projection.Removed[player.ID] && cmp.Or(lastSeason, season) >= season {
			players = append(players, nativecheck.CapPlayer{Salary: projectedSalary(schedule, season, player.Salary),
				Status: player.Status})
		}
	}
	return players
}

func runwayCharges(projected []NativeCharge, cuts map[string]bool, franchise string, season int) (
	[]domain.Money, []domain.Money) {
	charges, live := []domain.Money{}, []domain.Money{}
	for _, charge := range projected {
		if charge.Franchise != franchise || charge.Season != 0 && charge.Season != season {
			continue
		}
		charges = append(charges, charge.Amount)
		if charge.Kind != deadMoney && !cuts[charge.ID] {
			live = append(live, charge.Amount)
		}
	}
	return charges, live
}

func lastScheduleSeason(schedule []nativecheck.SeasonAmount) int {
	last := 0
	for _, cell := range schedule {
		last = max(last, cell.Season)
	}
	return last
}

func projectedSalary(schedule []nativecheck.SeasonAmount, season int, fallback domain.Money) domain.Money {
	for _, cell := range schedule {
		if cell.Season == season {
			return cell.Cents
		}
	}
	return fallback
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

func capRunwayYear(season int, live, total nativecheck.CapState, dead domain.Money, counted bool, missing error) NativeCapYear {
	row := NativeCapYear{Season: season, CapCents: total.CapCents}
	if missing != nil {
		return row
	}
	row.CommittedCents, row.RoomCents = live.CommittedCents, total.RoomCents
	if counted {
		row.DeadCents = &dead
	}
	return row
}

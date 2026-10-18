package commissioner

import (
	"fmt"
	"math/big"
	"slices"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

func rosterCapAllows(seed *RawNativeSnapshot, rules leaguerules.RuleSet, franchise string,
	facts nativecheck.Facts, zone *time.Location) error {
	roster := []nativecheck.RosterPlayer{}
	players := []nativecheck.CapPlayer{}
	charges := []domain.Money{}
	for _, p := range seed.Players {
		if p.Franchise == franchise {
			roster = append(roster, nativecheck.RosterPlayer{ID: p.ID, Status: p.Status})
			players = append(players, nativecheck.CapPlayer{Salary: p.Salary, Status: p.Status})
		}
	}
	for _, c := range seed.Adjustments {
		if c.Franchise == franchise && (c.Season == 0 || c.Season == rules.Season) {
			charges = append(charges, c.Amount)
		}
	}
	verdict := nativecheck.CheckRoster(rules.Roster, franchise, roster, facts)
	if !verdict.Fits {
		return fmt.Errorf("roster would not be legal: %s", strings.Join(append([]string{verdict.Words},
			append(verdict.Problems, verdict.Notes...)...), "; "))
	}
	capState, err := nativecheck.CheckCap(rules.Cap, players, charges, zone)
	if err != nil {
		return fmt.Errorf("check waiver salary: %w", err)
	}
	if allows, why := nativecheck.CapAllows(capState, 0, zone); !allows {
		return fmt.Errorf("%s", why)
	}
	return nil
}

func waiverClaimAllows(seed *RawNativeSnapshot, rules leaguerules.RuleSet, w *waiverWindow,
	franchise string, facts nativecheck.Facts, zone *time.Location) error {
	if rules.Contracts.ReleaseReSignSameSeason.State != leaguerules.IsKnown {
		return fmt.Errorf("re-signing a released player needs a ruling")
	}
	if franchise == w.Player.Franchise && w.Season == rules.Season && !rules.Contracts.ReleaseReSignSameSeason.Value {
		return fmt.Errorf("you cannot claim a player you released this season")
	}
	if rules.Contracts.ClaimClearsDeadCap.State != leaguerules.IsKnown {
		return fmt.Errorf("waiver dead money relief needs a ruling")
	}
	candidate := *seed
	candidate.Players = slices.Clone(seed.Players)
	candidate.Adjustments = slices.Clone(seed.Adjustments)
	player := w.Player
	player.Franchise, player.Status = franchise, domain.RosterActive
	candidate.Players = append(candidate.Players, player)
	if rules.Contracts.ClaimClearsDeadCap.Value {
		for i := range candidate.Adjustments {
			if candidate.Adjustments[i].ID == w.ID {
				candidate.Adjustments[i].Amount = 0
			}
		}
	}
	return rosterCapAllows(&candidate, rules, franchise, facts, zone)
}

// Dead money counts each remaining flat paid season after the cut season.
func waiverDeadMoney(player NativePlayer, rules leaguerules.RuleSet) (domain.Money, error) {
	rate := rules.Contracts.DeadCap
	if player.Restructured {
		rate = rules.Contracts.RestructuredDeadCap
	}
	if rate.State != leaguerules.IsKnown || rate.Value < 0 || rate.Value > 10000 ||
		rules.Cap.Quantum.State != leaguerules.IsKnown || rules.Cap.Quantum.Value <= 0 ||
		rules.Contracts.FlatSalaries.State != leaguerules.IsKnown || !rules.Contracts.FlatSalaries.Value {
		return 0, fmt.Errorf("release dead money, flat contract salaries and money rounding need a ruling")
	}
	if player.ExpirationYear == 0 {
		return 0, fmt.Errorf("record this player's contract expiration before cutting him")
	}
	years := max(0, player.ExpirationYear-rules.Season)
	value := new(big.Int).Mul(big.NewInt(int64(player.Salary)), big.NewInt(int64(rate.Value)))
	value.Add(value, big.NewInt(5000)).Quo(value, big.NewInt(10000))
	if !value.IsInt64() {
		return 0, fmt.Errorf("release dead money is too large")
	}
	cell, err := nativecheck.SnapCapMoney(domain.Money(value.Int64()), domain.Money(rules.Cap.Quantum.Value))
	if err != nil {
		return 0, fmt.Errorf("round release dead money: %w", err)
	}
	value.Mul(big.NewInt(int64(cell)), big.NewInt(int64(years)))
	if !value.IsInt64() {
		return 0, fmt.Errorf("release dead money is too large")
	}
	return domain.Money(value.Int64()), nil
}

func completeWaiverOrder(order []string, seed *RawNativeSnapshot) error {
	if len(order) != len(seed.Franchises) {
		return fmt.Errorf("waiver order must list every franchise exactly once")
	}
	seen := map[string]bool{}
	for _, id := range order {
		found := false
		for _, f := range seed.Franchises {
			if f.ID == id {
				found = true
			}
		}
		if !found || seen[id] {
			return fmt.Errorf("waiver order must list every franchise exactly once")
		}
		seen[id] = true
	}
	return nil
}

// RecordedStandingsOrder is the pure side of the store's StandingsOrder(league, season, at) seam.
func RecordedStandingsOrder(league string, records []Record, season int, at time.Time) ([]string, error) {
	for _, r := range records {
		if r.Command.League == league && r.Command.Command == NativeSeed && r.Command.Args.Native != nil &&
			!r.Receipt.RecordedAt.After(at) && r.Command.Args.Native.Season == season {
			seed := r.Command.Args.Native
			if len(seed.StandingsOrder) == 0 {
				break
			}
			if err := completeWaiverOrder(seed.StandingsOrder, seed); err != nil {
				return nil, err
			}
			return append([]string{}, seed.StandingsOrder...), nil
		}
	}
	return nil, fmt.Errorf("%s", "Waiver order needs the standings, which are not built yet.")
}

package market

import (
	"fmt"
	"slices"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/governance"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

func (x tradeExecution) legal(t commissioner.TradeState) error {
	if err := checkTradeAssets(t, x.rules.Season); err != nil {
		return err
	}
	hasPicks := slices.ContainsFunc(t.Assets, func(a commissioner.TradeAsset) bool { return a.Player == "" })
	candidate := *x.seed
	candidate.Players, candidate.Picks = slices.Clone(x.seed.Players), slices.Clone(x.seed.Picks)
	if err := commissioner.MoveTradeAssets(&candidate, t.Assets, false); err != nil {
		//nolint:wrapcheck // Preserve the original refusal text across the package boundary.
		return err
	}
	for _, id := range t.Parties {
		f, _ := x.state.Franchise(governance.FranchiseID(id))
		if !f.Abilities["trades"] {
			return fmt.Errorf("trading is turned off for a party")
		}
		if hasPicks && !f.Abilities["pick trades"] {
			return fmt.Errorf("pick trading is turned off for a party")
		}
		rules := x.rules
		if err := lineCardsHeld(&candidate, rules, id); err != nil {
			return err
		}
		if !t.Policy.RosterGate {
			// Keep salary enforcement even when the commissioner disables the roster refusal.
			capState, err := nativeFranchiseCap(&candidate, rules, id, x.zone)
			if err != nil {
				return err
			}
			if allows, why := nativecheck.CapAllows(capState, 0, x.zone); !allows {
				return fmt.Errorf("%s", why)
			}
		} else if err := rosterCapAllows(&candidate, rules, id, x.facts, x.zone); err != nil {
			return err
		}
	}
	return nil
}

func nativeFranchiseCap(seed *commissioner.RawNativeSnapshot, rules leaguerules.RuleSet, id string, zone *time.Location) (nativecheck.CapState, error) {
	players := []nativecheck.CapPlayer{}
	charges := []domain.Money{}
	for _, p := range seed.Players {
		if p.Franchise == id {
			players = append(players, nativecheck.CapPlayer{Salary: p.Salary, Status: p.Status})
		}
	}
	for _, c := range seed.Adjustments {
		if c.Franchise == id && (c.Season == 0 || c.Season == rules.Season) {
			charges = append(charges, c.Amount)
		}
	}
	result, err := nativecheck.CheckCap(rules.Cap, players, charges, zone)
	if err != nil {
		return result, fmt.Errorf("check trade salary: %w", err)
	}
	return result, nil
}

func checkTradeAssets(t commissioner.TradeState, season int) error {
	seen := map[string]bool{}
	for _, a := range t.Assets {
		key := "player." + a.Player
		if a.Player == "" {
			key = fmt.Sprintf("pick.%d.%d.%s", a.Season, a.Round, a.Original)
		}
		if seen[key] {
			return fmt.Errorf("an asset may move only once in a trade")
		}
		seen[key] = true
		if a.Player == "" && (a.Season < season || a.Season > season+t.Policy.FutureYears || a.Round < t.Policy.MinimumRound || a.Round > t.Policy.MaximumRound) {
			return fmt.Errorf("this pick is outside the league's tradable years or rounds")
		}
	}
	return nil
}

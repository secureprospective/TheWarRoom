package market

import (
	"fmt"
	"slices"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/governance"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

func resolveTradePolicy(r leaguerules.TradesRules) (commissioner.TradePolicy, error) {
	if r.Review.State != leaguerules.IsKnown || r.DecideAt.State != leaguerules.IsKnown ||
		r.FutureYears.State != leaguerules.IsKnown || r.Rounds.State != leaguerules.IsKnown ||
		r.RosterGate.State != leaguerules.IsKnown || r.OfferExpiry.State != leaguerules.IsKnown ||
		r.MaximumParties.State != leaguerules.IsKnown {
		return commissioner.TradePolicy{}, fmt.Errorf("trade review, picks, roster gate, offer lifetime and parties need a ruling")
	}
	p := commissioner.TradePolicy{Route: r.Review.Value, DecideAt: r.DecideAt.Value, FutureYears: r.FutureYears.Value,
		MinimumRound: r.Rounds.Value.Min.Value, MaximumRound: r.Rounds.Value.Max.Value, RosterGate: r.RosterGate.Value,
		Expiry: r.OfferExpiry.Value, MaximumParties: r.MaximumParties.Value, Seats: r.DOTSeats.Value}
	if err := p.Validate(r); err != nil {
		//nolint:wrapcheck // Preserve the original refusal text across the package boundary.
		return commissioner.TradePolicy{}, err
	}

	return p, nil
}

func tradeParties(state governance.State, parties []string) []governance.Principal {
	out := []governance.Principal{}
	for _, id := range parties {
		if f, ok := state.Franchise(governance.FranchiseID(id)); ok && f.Seat != "" {
			out = append(out, f.Seat)
		}
	}
	return out
}

func tradeDOT(state governance.State, records []commissioner.Record) []governance.Principal {
	people, _ := commissioner.PeopleInventory(records)
	out := []governance.Principal{}
	for _, p := range people {
		if state.HasRole(governance.Principal(p), governance.DOT) {
			out = append(out, governance.Principal(p))
		}
	}
	return out
}

func tradeReviewers(t *commissioner.TradeState, state governance.State, records []commissioner.Record, now time.Time) ([]governance.Principal, error) {
	parties := tradeParties(state, t.Parties)
	all := tradeDOT(state, records)
	for _, p := range all {
		if slices.Contains(parties, p) && !hasTradeVote(t.Recusals, records, string(p)) {
			t.Recusals = append(t.Recusals, commissioner.TradeVote{Who: string(p), Decision: commissioner.TradeRecuse, At: now, Reason: "A trade party cannot review its own trade"})
		}
	}
	eligible, err := governance.Reviewers(state, parties)
	if len(all) >= 4 && len(eligible) == 0 {
		t.Status, t.Words = "failed", "No eligible DOT reviewer remains; this trade cannot proceed"
	}
	if err != nil {
		return nil, fmt.Errorf("trade reviewers: %w", err)
	}
	if len(all) != t.Policy.Seats {
		return nil, fmt.Errorf("%s: DOT has %d of %d chosen seats", "Needs a ruling", len(all), t.Policy.Seats)
	}
	return eligible, nil
}

func hasTradeVote(votes []commissioner.TradeVote, records []commissioner.Record, principal string) bool {
	return slices.ContainsFunc(votes, func(v commissioner.TradeVote) bool {
		return commissioner.SamePrincipal(records, v.Who, principal)
	})
}

package commissioner

import (
	"fmt"
	"slices"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/governance"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

func resolveTradePolicy(r leaguerules.TradesRules) (TradePolicy, error) {
	if r.Review.State != leaguerules.IsKnown || r.DecideAt.State != leaguerules.IsKnown ||
		r.FutureYears.State != leaguerules.IsKnown || r.Rounds.State != leaguerules.IsKnown ||
		r.RosterGate.State != leaguerules.IsKnown || r.OfferExpiry.State != leaguerules.IsKnown ||
		r.MaximumParties.State != leaguerules.IsKnown {
		return TradePolicy{}, fmt.Errorf("trade review, picks, roster gate, offer lifetime and parties need a ruling")
	}
	p := TradePolicy{Route: r.Review.Value, DecideAt: r.DecideAt.Value, FutureYears: r.FutureYears.Value,
		MinimumRound: r.Rounds.Value.Min.Value, MaximumRound: r.Rounds.Value.Max.Value, RosterGate: r.RosterGate.Value,
		Expiry: r.OfferExpiry.Value, MaximumParties: r.MaximumParties.Value, Seats: r.DOTSeats.Value}
	if err := p.Validate(r); err != nil {
		return TradePolicy{}, err
	}

	return p, nil
}

func (p TradePolicy) Validate(r leaguerules.TradesRules) error {
	if r.Rounds.Value.Min.State != leaguerules.IsKnown || r.Rounds.Value.Max.State != leaguerules.IsKnown ||
		p.invalidLimits() {
		return fmt.Errorf("the trade rules need a valid ruling")
	}
	if !slices.Contains([]leaguerules.ReviewRoute{leaguerules.DOTReview, leaguerules.CommissionerReview, leaguerules.NoReview}, p.Route) {
		return fmt.Errorf("choose who reviews trades")
	}
	if p.Route == leaguerules.DOTReview && (r.DOTSeats.State != leaguerules.IsKnown || p.Seats < 4 || p.Seats > 10 || p.DecideAt > p.Seats) {
		return fmt.Errorf("%s: choose 4 to 10 DOT seats and a decision count within them", "Needs a ruling")
	}
	return nil
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

func tradeDOT(state governance.State, records []Record) []governance.Principal {
	people, _ := peopleInventory(records)
	out := []governance.Principal{}
	for _, p := range people {
		if state.HasRole(governance.Principal(p), governance.DOT) {
			out = append(out, governance.Principal(p))
		}
	}
	return out
}

func tradeReviewers(t *TradeState, state governance.State, records []Record, now time.Time) ([]governance.Principal, error) {
	parties := tradeParties(state, t.Parties)
	all := tradeDOT(state, records)
	for _, p := range all {
		if slices.Contains(parties, p) && !slices.ContainsFunc(t.Recusals, func(v TradeVote) bool { return SamePrincipal(records, v.Who, string(p)) }) {
			t.Recusals = append(t.Recusals, TradeVote{Who: string(p), Decision: TradeRecuse, At: now, Reason: "A trade party cannot review its own trade"})
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

func (p TradePolicy) invalidLimits() bool {
	return p.DecideAt < 1 || p.FutureYears < 0 || p.MinimumRound < 1 || p.MaximumRound < p.MinimumRound ||
		p.Expiry <= 0 || p.MaximumParties < 2 || p.MaximumParties > 3
}

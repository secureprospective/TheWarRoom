package envelope

import (
	"fmt"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/engine/bid"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
)

// BidIntent is the free-agent bid plan's intent.
const BidIntent = "bid.submit"

// BidFacts is what a bid plan holds beside its envelope (the bid_facts row). It names no other
// franchise: the bid to beat is points only, as the owner typed it.
type BidFacts struct {
	Player   playerid.PlayerID `json:"player" ts_type:"string"`
	Salary   domain.Money      `json:"salary"` // annual cents, flat across the years
	Years    int               `json:"years"`
	Points   bid.Points        `json:"points"`
	Snipe    bool              `json:"snipe"`
	Beat     *bid.Points       `json:"beat"`
	Evidence string            `json:"evidence"`
	Closes   string            `json:"closes"` // RFC 3339 UTC when the owner says bidding closes, else empty
}

// BidRequest drafts one bid plan for the owner's franchise.
type BidRequest struct {
	LeagueID    string
	FranchiseID string
	Facts       BidFacts
}

// DraftBid builds a bid plan and runs the bid rules on it, so it is Ready or Blocked at once.
// The target is manual: the owner posts the bid on ProBoards by hand.
func DraftBid(at time.Time, req BidRequest, check BidCheck, generate func() string) (Envelope, error) {
	player := req.Facts.Player
	spec := Spec{
		Intent: BidIntent, LeagueID: req.LeagueID, FranchiseID: req.FranchiseID,
		Subject:  Subject{Players: []playerid.PlayerID{player}},
		Expected: ExpectedChange{Player: player, RosterStatus: domain.RosterActive},
		Gravity:  G3, Undo: Irreversible, Target: Target{Kind: Manual},
	}
	e, err := New(at, spec, generate)
	if err != nil {
		return Envelope{}, fmt.Errorf("draft bid: %w", err)
	}
	checked, err := e.Check(at, snapshot.Snapshot{}, check)
	if err != nil {
		return Envelope{}, fmt.Errorf("draft bid: check: %w", err)
	}
	return checked, nil
}

// BidCheck runs the §6 bid rules on a prepared Input; they read no roster, so the snapshot
// is unused.
type BidCheck struct{ Input bid.Input }

func (c BidCheck) Evaluate(s Spec, _ snapshot.Snapshot) CheckResult {
	if s.Intent != BidIntent {
		return CheckResult{Blocked: true, Note: "bid check does not cover this intent"}
	}
	results := bid.Check(c.Input)
	return CheckResult{Blocked: bid.Blocked(results), Note: bid.Note(results)}
}

// BidPredicate lands a bid by the rosters export: on the owner's roster with the planned
// contract is Landed; on another franchise's roster, or off every roster once the window the
// owner named has closed, is awarded off-app. It carries the plan's facts because the
// contract it looks for lives in bid_facts, not in the envelope.
type BidPredicate struct{ Facts BidFacts }

func (BidPredicate) Sources() []Source { return []Source{Rosters} }

func (p BidPredicate) Evaluate(s Spec, obs Observation) Verdict {
	if s.Intent != BidIntent || obs.LeagueID != s.LeagueID ||
		obs.Rosters.Provenance.Freshness.State == domain.FreshFail {
		return Verdict{Event: Partial, Note: "Not verified: authoritative scoped roster unavailable"}
	}
	mine, elsewhere := false, false
	var held snapshot.RosterPlayer
	for _, r := range obs.Rosters.Value {
		mine = mine || r.FranchiseID == s.FranchiseID
		for _, rp := range r.Players {
			if rp.ID != p.Facts.Player {
				continue
			}
			if r.FranchiseID == s.FranchiseID {
				held = rp
			}
			elsewhere = elsewhere || r.FranchiseID != s.FranchiseID
		}
	}
	switch {
	case !mine:
		return Verdict{Event: Partial, Note: "Not verified: franchise roster missing"}
	case held.ID.String() != "":
		return p.contract(held)
	case elsewhere:
		return Verdict{Event: AwardedElsewhere, Note: "Awarded off-app: he is on another franchise's roster"}
	case p.closed(obs):
		return Verdict{Event: AwardedElsewhere, Note: "Awarded off-app: the bidding window closed and he is not on your roster"}
	}
	return Verdict{Event: NoChange, Note: "No matching MFL change observed"}
}

// contract compares the player now on the owner's roster with the planned contract. MFL states
// an expiration season; the snapshot turns it into years remaining.
func (p BidPredicate) contract(held snapshot.RosterPlayer) Verdict {
	if held.YearsRemaining == nil {
		return Verdict{Event: Partial, Note: "Not verified: on your roster, but the contract length is missing"}
	}
	if held.Salary == p.Facts.Salary && *held.YearsRemaining == p.Facts.Years {
		return Verdict{Event: Match, Note: fmt.Sprintf("Landed: on your roster at %s for %d %s",
			bid.MoneyText(held.Salary), *held.YearsRemaining, yearsWord(*held.YearsRemaining))}
	}
	return Verdict{Event: Partial, Note: fmt.Sprintf(
		"Contract mismatch: on your roster at %s for %d %s, not the planned %s for %d %s",
		bid.MoneyText(held.Salary), *held.YearsRemaining, yearsWord(*held.YearsRemaining),
		bid.MoneyText(p.Facts.Salary), p.Facts.Years, yearsWord(p.Facts.Years))}
}

func yearsWord(n int) string {
	if n == 1 {
		return "year"
	}
	return "years"
}

// closed holds once the rosters were fetched after the window the owner named.
func (p BidPredicate) closed(obs Observation) bool {
	closes, err := time.Parse(time.RFC3339, p.Facts.Closes)
	if err != nil {
		return false
	}
	fetched, err := time.Parse(time.RFC3339, obs.Rosters.Provenance.Freshness.FetchedAt)
	return err == nil && fetched.After(closes)
}

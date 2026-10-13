package envelope

import (
	"fmt"
	"slices"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
)

func IsTrade(intent string) bool {
	return slices.Contains([]string{"trade.accept", "trade.propose", "trade.reject", "trade.revoke"}, intent)
}

func validTradeIntentScope(intent string, t ExpectedTrade, mine string) bool {
	if intent == "trade.accept" || intent == "trade.reject" {
		return validTradeScope(t, mine)
	}
	return t.TradeID != "" && t.Offering == mine && t.Accepting != "" && t.Accepting != mine &&
		len(t.OfferingGives) > 0 && len(t.AcceptingGives) > 0
}

type TradeActCheck struct{ Blocks []string }

func (c TradeActCheck) Evaluate(Spec, snapshot.Snapshot) CheckResult {
	return CheckResult{Blocked: len(c.Blocks) > 0, Note: reasonNote(c.Blocks)}
}

func DraftTradeAct(at time.Time, intent string, req TradeRequest, blocks []string,
	generate func() string) (Envelope, error) {
	subject, err := tradeSubject(req.Trade)
	if err != nil {
		return Envelope{}, fmt.Errorf("trade Act subject: %w", err)
	}
	e, err := New(at, Spec{Intent: intent, LeagueID: req.LeagueID, FranchiseID: req.FranchiseID,
		Subject: subject, Expected: ExpectedChange{Trade: &req.Trade}, Gravity: G3, Undo: Irreversible,
		Target: req.Target, Deadline: req.Deadline}, generate)
	if err != nil {
		return Envelope{}, fmt.Errorf("trade Act: %w", err)
	}
	checked, err := e.Check(at, snapshot.Snapshot{}, TradeActCheck{Blocks: blocks})
	if err != nil {
		return Envelope{}, fmt.Errorf("trade Act check: %w", err)
	}
	return checked, nil
}

type TradeActPredicate struct{}

func (TradeActPredicate) Sources() []Source { return []Source{Transactions, PendingTrades} }

const TradeGoneNote = "No longer pending on MFL. MFL does not record whether it was rejected, withdrawn or expired."

func MatchesPendingTrade(t ExpectedTrade, offer leaguefeed.PendingTrade) bool {
	return t.Offering == offer.Offering && t.Accepting == offer.OfferedTo &&
		sameTokens(t.OfferingGives, leaguefeed.AssetTokens(offer.Gives)) &&
		sameTokens(t.AcceptingGives, leaguefeed.AssetTokens(offer.Gets))
}

func (TradeActPredicate) Evaluate(s Spec, obs Observation) Verdict {
	if reason := tradeObservationReason(s, obs); reason != "" {
		return Verdict{Event: Partial, Note: "Not verified: " + reason}
	}
	for _, row := range obs.Transactions.Value {
		if row.Kind != "TRADE" || row.Trade == nil || row.Unparsed || row.Time.Before(obs.HandedOffAt) ||
			!matchesTrade(*s.Expected.Trade, row) {
			continue
		}
		if s.Intent == "trade.propose" {
			return Verdict{Event: Match, Note: "public TRADE matches both franchises and full asset sets"}
		}
		return Verdict{Event: Partial, Note: "Not verified: MFL records this deal as executed, not rejected or withdrawn."}
	}
	if verdict, found := tradePendingVerdict(s, obs); found {
		return verdict
	}
	if s.Intent != "trade.propose" {
		return Verdict{Event: Partial, Note: TradeGoneNote}
	}
	if obs.TradeProposed {
		return Verdict{Event: AwaitDOT, Note: "In DOT review · seven days; offer left pending after proposal. Acceptance is not separately recorded; it may also have been withdrawn or expired."}
	}
	return Verdict{Event: NoChange, Note: "No matching proposal observed on MFL"}
}

func tradePendingVerdict(s Spec, obs Observation) (Verdict, bool) {
	different := false
	for _, offer := range obs.PendingTrades.Value {
		if s.Intent != "trade.propose" {
			if offer.ID == s.Expected.Trade.TradeID {
				return Verdict{Event: NoChange, Note: "Offer still pending on MFL"}, true
			}
			continue
		}
		if !MatchesPendingTrade(*s.Expected.Trade, offer) {
			different = different || (offer.Offering == s.FranchiseID && offer.OfferedTo == s.Expected.Trade.Accepting)
			continue
		}
		if offer.Proposed.Before(obs.HandedOffAt) {
			return Verdict{Event: Partial, Note: "Not verified: matching offer predates this hand-off"}, true
		}
		return Verdict{Event: NoChange, Note: "Proposed on MFL"}, true
	}
	if different {
		return Verdict{Event: Partial, Note: "MFL shows a different offer"}, true
	}
	return Verdict{}, false
}

// WithdrawTrade removes an unsubmitted local proposal, not an offer on MFL.
func (e Envelope) WithdrawTrade(at time.Time) (Envelope, error) {
	if e.spec.Intent != "trade.propose" {
		return Envelope{}, ErrIllegalTransition{From: e.state, Event: Withdraw}
	}
	for _, entry := range e.audit {
		if entry.To == HandedOff {
			return Envelope{}, ErrIllegalTransition{From: e.state, Event: Withdraw}
		}
	}
	return e.move(at, Withdraw, RemovePlanNote)
}

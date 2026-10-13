package envelope

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/pendingtrades"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/transactions"
	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
)

func TestProposeExactMatchNearMatchAndDOTWindow(t *testing.T) {
	req, _, check, at := tradeFixture(t)
	req.FranchiseID = req.Trade.Offering
	e, err := DraftTradeAct(at, "trade.propose", req, nil, func() string { return "proposal" })
	if err != nil {
		t.Fatal(err)
	}
	handed := at.Add(time.Minute)
	e, err = e.HandOff(handed)
	if err != nil {
		t.Fatal(err)
	}
	obs := tradeObservation(check, at)
	obs.Transactions.Value = nil
	obs.PendingTrades.Value[0].Proposed = handed.Add(time.Second)
	offer := obs.PendingTrades.Value[0]
	near := offer
	near.ID = "near"
	near.Gives = near.Gives[:1]
	obs.PendingTrades.Value = []leaguefeed.PendingTrade{near}
	obs.HandedOffAt = handed
	verdict := (TradeActPredicate{}).Evaluate(e.spec, obs)
	if verdict.Event != Partial || verdict.Note != "MFL shows a different offer" {
		t.Fatal(verdict)
	}
	// The same offer asking the other side for one asset fewer is not the planned deal either.
	nearGets := offer
	nearGets.ID = "near-gets"
	if len(nearGets.Gets) == 0 {
		t.Fatal("fixture offer must ask for assets")
	}
	nearGets.Gets = nearGets.Gets[:len(nearGets.Gets)-1]
	obs.PendingTrades.Value = []leaguefeed.PendingTrade{nearGets}
	if v := (TradeActPredicate{}).Evaluate(e.spec, obs); v.Event != Partial || v.Note != "MFL shows a different offer" {
		t.Fatal("an offer asking for less matched the plan:", v)
	}
	obs.PendingTrades.Value = []leaguefeed.PendingTrade{near, offer}
	verdict = (TradeActPredicate{}).Evaluate(e.spec, obs)
	if verdict.Event != NoChange || verdict.Note != "Proposed on MFL" {
		t.Fatal(verdict)
	}
	now := handed.Add(time.Minute)
	e = observeTradeAct(t, e, obs, now)
	if e.State() != NotYetDone || e.audit[len(e.audit)-1].Note != "Proposed on MFL" {
		t.Fatal(e.Receipt())
	}
	obs.PendingTrades.Value = nil
	entered := now.Add(time.Minute)
	e = observeTradeAct(t, e, obs, entered)
	if e.State() != DOTReview || !strings.Contains(e.audit[len(e.audit)-1].Note, "Acceptance is not separately recorded") {
		t.Fatal(e.Receipt())
	}
	e = observeTradeAct(t, e, obs, entered.Add(7*24*time.Hour+time.Second))
	if e.State() != NotVerified {
		t.Fatal(e.Receipt())
	}
	obs.Transactions.Value = tradeObservation(check, at).Transactions.Value
	obs.Transactions.Value[0].Time = handed.Add(time.Second)
	e = observeTradeAct(t, e, obs, entered.Add(8*24*time.Hour))
	if e.State() != Landed {
		t.Fatal(e.Receipt())
	}
	restored, err := Restore(e.ID(), e.Created(), e.spec, e.audit)
	if err != nil || restored.State() != Landed {
		t.Fatal(restored, err)
	}
}
func observeTradeAct(t *testing.T, e Envelope, obs Observation, at time.Time) Envelope {
	t.Helper()
	fresh := domain.Freshness{State: domain.FreshLive, FetchedAt: at.Format(time.RFC3339)}
	obs.PendingTrades.Provenance.Freshness = fresh
	obs.Transactions.Provenance.Freshness = fresh
	next, err := e.Observe(at, obs, TradeActPredicate{})
	if err != nil {
		t.Fatal(err)
	}
	return next
}
func TestProposalNeverEntersReviewWithoutObservedProposal(t *testing.T) {
	req, _, check, at := tradeFixture(t)
	req.FranchiseID = req.Trade.Offering
	e, err := DraftTradeAct(at, "trade.propose", req, nil, func() string { return "absent" })
	if err != nil {
		t.Fatal(err)
	}
	obs := tradeObservation(check, at)
	obs.Transactions.Value = nil
	obs.PendingTrades.Value = nil
	verdict := (TradeActPredicate{}).Evaluate(e.spec, obs)
	if verdict.Event != NoChange || verdict.Note != "No matching proposal observed on MFL" {
		t.Fatal(verdict)
	}
	obs.PendingTrades.Value = check.PendingTrades.Value
	obs.PendingTrades.Value[0].Proposed = at.Add(-time.Hour)
	if got := (TradeActPredicate{}).Evaluate(e.spec, obs); got.Event != Partial {
		t.Fatal(got)
	}
}
func TestRejectAndRevokeNeverClaimRejectedOrLanded(t *testing.T) {
	for _, intent := range []string{"trade.reject", "trade.revoke"} {
		req, _, check, at := tradeFixture(t)
		if intent == "trade.revoke" {
			req.FranchiseID = req.Trade.Offering
		}
		e, err := DraftTradeAct(at, intent, req, nil, func() string { return intent })
		if err != nil {
			t.Fatal(err)
		}
		obs := tradeObservation(check, at)
		obs.Transactions.Value = nil
		if got := (TradeActPredicate{}).Evaluate(e.spec, obs); got.Event != NoChange {
			t.Fatal(got)
		}
		obs.PendingTrades.Value = nil
		if got := (TradeActPredicate{}).Evaluate(e.spec, obs); got.Event != Partial || got.Note != TradeGoneNote {
			t.Fatal(got)
		}
		obs.Transactions.Value = tradeObservation(check, at).Transactions.Value
		if got := (TradeActPredicate{}).Evaluate(e.spec, obs); got.Event == Match || !strings.Contains(got.Note, "executed") {
			t.Fatal(got)
		}
		obs.PendingTrades.Provenance.Freshness = domain.Freshness{State: domain.FreshFail, Note: "key missing"}
		if got := (TradeActPredicate{}).Evaluate(e.spec, obs); got.Event != Partial || !strings.Contains(got.Note, "key missing") {
			t.Fatal(got)
		}
	}
}
func TestTradeActOnRealTransactionsAndDecodedPendingFixture(t *testing.T) {
	body, err := os.ReadFile("../ingestion/transactions/testdata/transactions.json")
	if err != nil {
		t.Fatal(err)
	}
	raw, err := transactions.Parse(body)
	if err != nil {
		t.Fatal(err)
	}
	rows, err := transactions.ToTransactions(raw)
	if err != nil {
		t.Fatal(err)
	}
	var row leaguefeed.Transaction
	for _, r := range rows {
		if r.Kind == "TRADE" && r.Trade != nil && len(r.Trade.Gave) > 0 && len(r.Trade.CounterpartyGave) > 0 {
			row = r
			break
		}
	}
	at := row.Time.Add(-time.Minute)
	req := TradeRequest{LeagueID: "14432", FranchiseID: row.Franchise, Target: Target{Kind: Mapped, URL: "https://fixture.myfantasyleague.com/2026/options?L=14432&O=05"}, Trade: ExpectedTrade{
		TradeID: "fixture", Offering: row.Franchise, Accepting: row.Trade.Counterparty,
		OfferingGives: leaguefeed.AssetTokens(row.Trade.Gave), AcceptingGives: leaguefeed.AssetTokens(row.Trade.CounterpartyGave)}}
	e, err := DraftTradeAct(at, "trade.propose", req, nil, func() string { return "fixture" })
	if err != nil {
		t.Fatal(err)
	}
	obs := Observation{LeagueID: req.LeagueID, HandedOffAt: at, Transactions: snapshot.Sourced[[]leaguefeed.Transaction]{Value: rows}}
	if got := (TradeActPredicate{}).Evaluate(e.spec, obs); got.Event != Match {
		t.Fatal(got)
	}
	// Encode the documented owner export shape using the fixture's real trade assets.
	pending := map[string]map[string]string{"pendingTrade": {
		"trade_id": "99", "offeringteam": row.Franchise, "offeredto": row.Trade.Counterparty,
		"will_give_up": strings.Join(req.Trade.OfferingGives, ","), "will_receive": strings.Join(req.Trade.AcceptingGives, ","),
		"timestamp": "1791375600", "expires": "1791980400"}}
	wire, err := json.Marshal(map[string]map[string]map[string]string{"pendingTrades": pending})
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := pendingtrades.Parse(wire)
	if err != nil {
		t.Fatal(err)
	}
	offers, err := pendingtrades.ToPendingTrades(decoded)
	if err != nil || len(offers) != 1 {
		t.Fatal(offers, err)
	}
	obs.Transactions.Value = nil
	obs.HandedOffAt = offers[0].Proposed.Add(-time.Second)
	obs.PendingTrades.Value = offers
	if got := (TradeActPredicate{}).Evaluate(e.spec, obs); got.Note != "Proposed on MFL" {
		t.Fatal(got)
	}
}

func TestTradePlanRemovalStopsAtHandOff(t *testing.T) {
	req, _, _, at := tradeFixture(t)
	for _, intent := range []string{"trade.propose", "trade.reject", "trade.revoke"} {
		t.Run(intent, func(t *testing.T) {
			if intent != "trade.reject" {
				req.FranchiseID = req.Trade.Offering
			} else {
				req.FranchiseID = req.Trade.Accepting
			}
			e, err := DraftTradeAct(at, intent, req, nil, func() string { return "plan" })
			if err != nil {
				t.Fatal(err)
			}
			removed, err := e.WithdrawTrade(at.Add(time.Minute))
			if err != nil || removed.State() != Withdrawn || removed.audit[len(removed.audit)-1].Note != RemovePlanNote {
				t.Fatal(removed.Receipt(), err)
			}
			handed, err := e.HandOff(at.Add(time.Minute))
			if err != nil {
				t.Fatal(err)
			}
			if _, err := handed.WithdrawTrade(at.Add(2 * time.Minute)); err == nil {
				t.Fatal("removed a plan already handed off to MFL")
			}
			e.spec.Intent = "trade.accept"
			if _, err := e.WithdrawTrade(at.Add(time.Minute)); err == nil {
				t.Fatal("accept removal changed")
			}
		})
	}
}

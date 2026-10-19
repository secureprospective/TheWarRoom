package main

import (
	"fmt"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
)

type tradeRuleProof struct {
	field, value string
	probe        func(*testing.T, *tradeFixture) string
}

// probeTrade reads a real offered trade, then compensates its actions in reverse so another probe starts
// from exactly the same assets. Refusals also read the native desk to prove they wrote no offer.
func probeTrade(t *testing.T, f *tradeFixture, args commissioner.RawTradeArgs, votes int) string {
	t.Helper()
	before := len(f.trades(t, "owner.one", "franchise.01").Offers)
	offer, err := f.action(t, "owner.two", commissioner.TradeOffer, args)
	if err != nil {
		if len(f.trades(t, "owner.one", "franchise.01").Offers) != before {
			t.Fatal("refused offer was recorded")
		}
		return err.Error()
	}
	receipts := []commissioner.Receipt{offer}
	if votes >= 0 {
		receipts = append(receipts, f.accept(t, offer))
		for i := 0; i < votes; i++ {
			receipts = append(receipts, f.vote(t, offer, []string{"dot.a", "dot.b", "dot.c"}[i], "veto"))
		}
	}
	reading := f.trades(t, "owner.one", "franchise.01").Offers
	last := reading[len(reading)-1]
	result := fmt.Sprintf("%s|%s|%v|%v", last.Status, last.Words, last.ExpiresAt.Sub(last.OfferedAt), last.Policy)
	for i := len(receipts) - 1; i >= 0; i-- {
		f.app.commissionerSession = testCommissionerSession{"creator"}
		f.revert(t, receipts[i])
	}
	return result
}

func TestTradeRulesGovernNativeDeskRevertAndReopen(t *testing.T) {
	t.Parallel()
	for _, proof := range []tradeRuleProof{
		{"trades.review", `"none"`, func(t *testing.T, f *tradeFixture) string { return probeTrade(t, f, tradePlayerOffer(), 0) }},
		{"trades.dotSeats", `5`, func(t *testing.T, f *tradeFixture) string { return probeTrade(t, f, tradePlayerOffer(), 0) }},
		{"trades.decideAt", `2`, func(t *testing.T, f *tradeFixture) string { return probeTrade(t, f, tradePlayerOffer(), 2) }},
		{"trades.offerExpiry", `3600000000000`, func(t *testing.T, f *tradeFixture) string { return probeTrade(t, f, tradePlayerOffer(), -1) }},
		{"trades.maximumParties", `2`, func(t *testing.T, f *tradeFixture) string {
			args := tradePlayerOffer()
			args.Parties = append(args.Parties, "franchise.03")
			args.Assets[1].To = "franchise.03"
			return probeTrade(t, f, args, -1)
		}},
		{"trades.rosterGate", `false`, func(t *testing.T, f *tradeFixture) string { return probeTrade(t, f, tradePlayerOffer(), -1) }},
	} {
		t.Run(proof.field, func(t *testing.T) {
			t.Parallel()
			f := tradeApp(t)
			if proof.field == "trades.rosterGate" {
				f.set(t, "roster.limits", `{"min":0,"max":40}`, "now")
			}
			for cycle := 0; cycle < 2; cycle++ {
				before := proof.probe(t, f)
				f.app.commissionerSession = testCommissionerSession{"creator"}
				change := f.set(t, proof.field, proof.value, "now")
				changed := proof.probe(t, f)
				if before == changed {
					t.Fatalf("%s had no native consequence", proof.field)
				}
				f.restart(t)
				if replay := proof.probe(t, f); replay != changed {
					t.Fatalf("active ruling did not replay: %s / %s", changed, replay)
				}
				f.app.commissionerSession = testCommissionerSession{"creator"}
				f.revert(t, change)
				if restored := proof.probe(t, f); restored != before {
					t.Fatalf("compensation failed: %s / %s", before, restored)
				}
				f.restart(t)
				if restored := proof.probe(t, f); restored != before {
					t.Fatal("compensated ruling did not replay")
				}
			}
		})
	}
}

func tradePickOffer() commissioner.RawTradeArgs {
	return commissioner.RawTradeArgs{Franchise: "franchise.02", Parties: []string{"franchise.01", "franchise.02"},
		Assets: []commissioner.TradeAsset{{From: "franchise.02", To: "franchise.01", Season: 2027, Round: 1, Original: "franchise.04"}}}
}

func TestTradePickRulesGovernNativeDeskRevertAndReopen(t *testing.T) {
	t.Parallel()
	for _, proof := range []struct{ field, value string }{{"trades.futureYears", `0`}, {"trades.rounds", `{"min":2,"max":5}`}} {
		t.Run(proof.field, func(t *testing.T) {
			t.Parallel()
			f := tradePickApp(t)
			probe := func() string { return probeTrade(t, f, tradePickOffer(), -1) }
			for cycle := 0; cycle < 2; cycle++ {
				before := probe()
				f.app.commissionerSession = testCommissionerSession{"creator"}
				change := f.set(t, proof.field, proof.value, "now")
				changed := probe()
				if changed == before {
					t.Fatal("pick restriction had no consequence")
				}
				f.restart(t)
				if probe() != changed {
					t.Fatal("pick restriction did not replay")
				}
				f.app.commissionerSession = testCommissionerSession{"creator"}
				f.revert(t, change)
				if probe() != before {
					t.Fatal("pick compensation failed")
				}
				f.restart(t)
				if probe() != before {
					t.Fatal("compensated pick restriction did not replay")
				}
			}
		})
	}
}

func TestTradeReviewAndAppealGovernNativeDeskRevertAndReopen(t *testing.T) {
	t.Parallel()
	f := tradeApp(t)
	offer := f.offer(t)
	f.accept(t, offer)
	for cycle := 0; cycle < 2; cycle++ {
		first := f.vote(t, offer, "dot.a", "approve")
		second := f.vote(t, offer, "dot.b", "approve")
		third := f.vote(t, offer, "dot.c", "approve")
		if f.only(t).Status != "approved" || f.ownerOf(t, "1000") != "franchise.01" {
			t.Fatal("review approval did not govern assets")
		}
		f.restart(t)
		f.app.commissionerSession = testCommissionerSession{"creator"}
		f.revert(t, third)
		if f.only(t).Status != "review" || len(f.only(t).Votes) != 2 || f.ownerOf(t, "1000") != "franchise.02" {
			t.Fatal("review compensation failed")
		}
		f.app.commissionerSession = testCommissionerSession{"creator"}
		f.revert(t, second)
		f.app.commissionerSession = testCommissionerSession{"creator"}
		f.revert(t, first)
		f.veto(t, offer)
		appeal := f.mustAction(t, "owner.one", commissioner.TradeAppeal, commissioner.RawTradeArgs{ID: offer.Entry, Franchise: "franchise.01"})
		if f.only(t).Status != "appealed" {
			t.Fatal("appeal had no native consequence")
		}
		ruling := f.mustAction(t, "creator", commissioner.TradeAppealRule, commissioner.RawTradeArgs{ID: offer.Entry, Decision: "approve"})
		if f.only(t).Status != "approved" {
			t.Fatal("appeal ruling did not govern")
		}
		f.restart(t)
		f.app.commissionerSession = testCommissionerSession{"creator"}
		f.revert(t, ruling)
		if f.only(t).Status != "appealed" || f.ownerOf(t, "1000") != "franchise.02" {
			t.Fatal("appeal ruling compensation failed")
		}
		f.app.commissionerSession = testCommissionerSession{"creator"}
		f.revert(t, appeal)
		if f.only(t).Status != "vetoed" || len(f.only(t).Appeals) != 0 {
			t.Fatal("appeal compensation failed")
		}
		// Undo the vetoes for the same round trip from a physically reopened database.
		records, err := f.app.leagueRecords(t.Context(), legalityLeague)
		if err != nil {
			t.Fatal(err)
		}
		for i := len(records) - 1; i >= 0; i-- {
			r := records[i]
			if r.Command.Command == commissioner.TradeReview && r.Command.Args.Trade.Decision == "veto" && r.Receipt.RecordedAt.Equal(f.now.Truncate(time.Microsecond)) {
				active := commissioner.Replay(records).Active
				if active[r.Receipt.Entry] {
					f.app.commissionerSession = testCommissionerSession{"creator"}
					f.revert(t, r.Receipt)
				}
			}
		}
		f.restart(t)
		if f.only(t).Status != "review" {
			t.Fatal("review did not restore after reopen")
		}
	}
}

package main

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/envelope"
	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
)

func tradeDraftApp(t *testing.T) (*App, *seasonTransport, leaguefeed.PendingTrade) {
	t.Helper()
	a, transport, _ := lineupDraftApp(t)
	a.targetSnapshot.Franchises.Value = append(a.targetSnapshot.Franchises.Value,
		snapshot.Franchise{ID: "0001", Name: "Other team"})
	assets := func(tokens string) []leaguefeed.Asset {
		out, err := leaguefeed.ParseAssets(tokens)
		if err != nil {
			t.Fatal(err)
		}
		return out
	}
	offer := leaguefeed.PendingTrade{
		ID: "77", Offering: "0001", OfferedTo: "0025",
		Gives: assets("99,FP_0001_2027_1"), Gets: assets("15754"),
		Expires: a.movesNow().Add(time.Hour), Comments: "synthetic offer",
	}
	a.seasonPendingTrades = heldSeasonFeed[[]leaguefeed.PendingTrade]{
		value: []leaguefeed.PendingTrade{offer}, fetchedAt: a.movesNow().Add(-time.Minute),
	}
	return a, transport, offer
}

func TestTargetTradesLocalReading(t *testing.T) {
	a, transport, offer := tradeDraftApp(t)
	other := offer
	other.ID, other.Offering, other.OfferedTo = "88", "0003", "0004"
	outgoing := offer
	outgoing.ID, outgoing.Offering, outgoing.OfferedTo = "66", "0025", "0001"
	later := offer
	later.ID, later.Expires = "1", offer.Expires.Add(time.Minute)
	a.seasonPendingTrades.value = []leaguefeed.PendingTrade{later, offer, other, outgoing}
	a.refreshMu.Lock()
	defer a.refreshMu.Unlock()
	r, err := a.TargetTrades("0025")
	if err != nil || len(r.Offers) != 3 || r.Offers[0].TradeID != "66" ||
		r.Offers[1].TradeID != "77" || r.Offers[2].TradeID != "1" {
		t.Fatalf("reading: %+v %v", r, err)
	}
	incoming, by := r.Offers[1], r.Offers[0]
	if incoming.Direction != "to_you" || incoming.OtherName != "Other team" || incoming.OtherID != "0001" ||
		incoming.Give[0].Token != "15754" || incoming.Give[0].Name == "15754" ||
		incoming.Give[0].Position == "" || incoming.Get[0].Name != "0099" ||
		incoming.Get[1].Token != "FP_0001_2027_1" || incoming.Comments != offer.Comments ||
		by.Direction != "by_you" || by.Give[0].Token != "0099" || by.Get[0].Token != "15754" {
		t.Fatalf("directions/assets: %+v %+v", incoming, by)
	}
	if len(transport.requests) != 0 {
		t.Fatal("reading fetched")
	}
	r.Offers[0].Give[0].Name = "mutated"
	again, err := a.TargetTrades("0025")
	if err != nil || again.Offers[0].Give[0].Name == "mutated" {
		t.Fatal("aliased reading", err)
	}
	for _, fetched := range []time.Time{{}, a.movesNow()} {
		a.seasonPendingTrades.fetchedAt = fetched
		a.seasonPendingTrades.refreshError = "offline"
		failed, err := a.TargetTrades("0025")
		if err != nil || len(failed.Offers) != 0 || failed.Provenance.Freshness.Note != "offline" {
			t.Fatalf("failed offers: %+v %v", failed, err)
		}
		body, err := json.Marshal(failed)
		if err != nil || !strings.Contains(string(body), `"offers":[]`) {
			t.Fatalf("JSON: %s %v", body, err)
		}
	}
	if _, err := a.TargetTrades("unknown"); err == nil {
		t.Fatal("unknown franchise passed")
	}
}

func TestTradeDraftHandOffWatcher(t *testing.T) {
	a, transport, offer := tradeDraftApp(t)
	r, err := a.TargetDraftTradeAccept("0025", offer.ID)
	if err != nil || r.State != envelope.Ready || r.Spec.Deadline == nil ||
		!r.Spec.Deadline.Equal(offer.Expires) || r.Audit[0].Note != "offer 77 from Other team" {
		t.Fatalf("draft: %+v %v", r, err)
	}
	const target = "https://www47.myfantasyleague.com/2026/options?L=14432&O=05"
	opens := 0
	a.openURL = func(ctx context.Context, got string) {
		opens++
		stored, err := a.moves.Get(ctx, r.CorrelationID)
		if got != target || err != nil || stored.State() != envelope.HandedOff {
			t.Fatalf("hand-off: %s %s %v", got, stored.State(), err)
		}
	}
	handed := a.movesNow().Add(time.Minute)
	a.movesNow = func() time.Time { return handed }
	if _, err := a.TargetHandOff(r.CorrelationID); err != nil || opens != 1 {
		t.Fatal("hand-off", err, opens)
	}
	if len(transport.requests) != 0 {
		t.Fatal("draft fetched")
	}
	for i, want := range []envelope.State{
		envelope.NotYetDone, envelope.DOTReview, envelope.DOTReview, envelope.Landed,
	} {
		now := handed.Add(time.Duration(i+1) * time.Minute)
		a.movesNow = func() time.Time { return now }
		requested := []envelope.Source{}
		a.movesRefresh = func(_ context.Context, source envelope.Source) error {
			requested = append(requested, source)
			switch source {
			case envelope.PendingTrades:
				a.seasonPendingTrades.fetchedAt = now
				if i > 0 {
					a.seasonPendingTrades.value = nil
				}
			case envelope.Transactions:
				a.seasonTransactions.fetchedAt = now
				when := handed.Add(-time.Second)
				if i == 3 {
					when = handed.Add(time.Second)
				}
				a.seasonTransactions.value = []leaguefeed.Transaction{{
					Kind: "TRADE", Time: when, Franchise: offer.Offering,
					Trade: &leaguefeed.Trade{
						Counterparty: offer.OfferedTo, Gave: offer.Gives, CounterpartyGave: offer.Gets,
					},
				}}
			case envelope.Rosters, envelope.Lineups:
				t.Fatalf("unexpected feed %s", source)
			}
			return nil
		}
		if err := a.checkMoves(a.ctx); err != nil {
			t.Fatal(err)
		}
		stored, err := a.moves.Get(a.ctx, r.CorrelationID)
		if err != nil || stored.State() != want || len(requested) != 2 {
			t.Fatalf("step %d: %s %v feeds %v", i, stored.State(), err, requested)
		}
		if i == 2 && len(stored.Receipt().Audit) != 4 {
			t.Fatal("watcher should skip saving same-state DOT audit", stored.Receipt().Audit)
		}
	}
}

func TestTradeBindingBlocks(t *testing.T) {
	for _, name := range []string{"failed", "stale failed", "unowned", "expired", "host", "offerer", "gone"} {
		t.Run(name, func(t *testing.T) {
			a, transport, offer := tradeDraftApp(t)
			note := ""
			switch name {
			case "failed", "stale failed":
				a.seasonPendingTrades.refreshError = "offline"
				if name == "failed" {
					a.seasonPendingTrades.fetchedAt = time.Time{}
				}
				note = "pending trades unavailable: offline"
			case "unowned":
				a.targetSnapshot.Rosters.Value = nil
				note = "not on this franchise's roster"
			case "expired":
				a.movesNow = func() time.Time { return offer.Expires.Add(time.Second) }
				note = "deadline passed"
			case "host":
				a.mflClient = nil
				note = "MFL host not yet known"
			case "offerer":
				a.seasonPendingTrades.value[0].Offering = "0025"
				note = "you made this offer; accept is the other owner's"
			case "gone":
				a.seasonPendingTrades.value = nil
				note = "offer 77 is no longer pending"
			}
			r, err := a.TargetDraftTradeAccept("0025", offer.ID)
			if name == "offerer" || name == "gone" {
				if err == nil || !strings.Contains(err.Error(), note) {
					t.Fatal("scope refusal", r, err)
				}
			} else if err != nil || r.State != envelope.Blocked || !strings.Contains(r.Audit[0].Note, note) {
				t.Fatalf("blocked: %+v %v", r, err)
			}
			if len(transport.requests) != 0 {
				t.Fatal("draft fetched")
			}
		})
	}
}

func TestTradeSupersede(t *testing.T) {
	for _, name := range []string{"ready", "handed", "dot", "blocked"} {
		t.Run(name, func(t *testing.T) {
			a, _, offer := tradeDraftApp(t)
			if name == "blocked" {
				a.mflClient = nil
			}
			first, err := a.TargetDraftTradeAccept("0025", offer.ID)
			if err != nil {
				t.Fatal(err)
			}
			if name == "handed" || name == "dot" {
				a.openURL = func(context.Context, string) {}
				if _, err := a.TargetHandOff(first.CorrelationID); err != nil {
					t.Fatal(err)
				}
			}
			if name == "dot" {
				e, err := a.moves.Get(a.ctx, first.CorrelationID)
				if err != nil {
					t.Fatal(err)
				}
				now := a.movesNow().Add(time.Minute)
				fresh := snapshot.Provenance{Freshness: liveFreshness(now)}
				obs := envelope.Observation{LeagueID: "14432"}
				obs.Transactions.Provenance, obs.PendingTrades.Provenance = fresh, fresh
				e, err = e.Observe(now, obs, envelope.TradePredicate{})
				if err != nil {
					t.Fatal(err)
				}
				saveMove(t, a, e)
				a.movesNow = func() time.Time { return now }
			}
			other := offer
			other.ID = "88"
			other.Expires = offer.Expires.Add(time.Hour)
			a.seasonPendingTrades.value = append(a.seasonPendingTrades.value, other)
			unrelated, err := a.TargetDraftTradeAccept("0025", other.ID)
			if err != nil {
				t.Fatal(err)
			}
			second, err := a.TargetDraftTradeAccept("0025", offer.ID)
			if err != nil {
				t.Fatal(err)
			}
			old, err := a.moves.Get(a.ctx, first.CorrelationID)
			want := envelope.Stale
			if name == "blocked" {
				want = envelope.Blocked
			}
			if err != nil || old.State() != want {
				t.Fatalf("superseded: %s %v", old.State(), err)
			}
			if want == envelope.Stale &&
				old.Receipt().Audit[len(old.Receipt().Audit)-1].Note != "superseded by "+second.CorrelationID {
				t.Fatal("missing successor ID")
			}
			kept, err := a.moves.Get(a.ctx, unrelated.CorrelationID)
			if err != nil || kept.State() == envelope.Stale {
				t.Fatal("other trade superseded", err)
			}
		})
	}
}

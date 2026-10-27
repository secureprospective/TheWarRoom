package composition

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/report"
	tradesapp "github.com/secureprospective/TheWarRoom/internal/app/trades"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/envelope"
	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
)

func tradeDraftApp(t *testing.T) (*App, *seasonTransport, leaguefeed.PendingTrade) {
	t.Helper()
	a, transport, _ := lineupDraftApp(t)
	a.core.TargetSnapshot.Franchises.Value = append(a.core.TargetSnapshot.Franchises.Value,
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
		Expires: a.core.MovesNow().Add(time.Hour), Comments: "synthetic offer",
	}
	a.core.SeasonPendingTrades = heldSeasonFeed[[]leaguefeed.PendingTrade]{
		Value: []leaguefeed.PendingTrade{offer}, FetchedAt: a.core.MovesNow().Add(-time.Minute),
	}
	return a, transport, offer
}

func TestTargetTradesLocalReading(t *testing.T) {
	t.Parallel()
	a, transport, offer := tradeDraftApp(t)
	other := offer
	other.ID, other.Offering, other.OfferedTo = "88", "0003", "0004"
	outgoing := offer
	outgoing.ID, outgoing.Offering, outgoing.OfferedTo = "66", "0025", "0001"
	later := offer
	later.ID, later.Expires = "1", offer.Expires.Add(time.Minute)
	a.core.SeasonPendingTrades.Value = []leaguefeed.PendingTrade{later, offer, other, outgoing}
	a.core.RefreshMu.Lock()
	defer a.core.RefreshMu.Unlock()
	r, err := a.tradesAPI().TargetTrades("0025")
	if err != nil || len(r.Offers) != 3 || r.Offers[0].TradeID != "66" ||
		r.Offers[1].TradeID != "77" || r.Offers[2].TradeID != "1" {
		t.Fatalf("reading: %+v %v", r, err)
	}
	incoming, by := r.Offers[1], r.Offers[0]
	if incoming.Direction != "to_you" || incoming.OtherName != "Other team" || incoming.OtherID != "0001" ||
		incoming.Give[0].Token != "15754" || incoming.Give[0].Name == "15754" ||
		incoming.Give[0].Position == "" || incoming.Get[0].Name != "Unnamed asset" ||
		incoming.Get[1].Token != "FP_0001_2027_1" ||
		incoming.Get[1].Name != "2027 round 1 · Other team" || incoming.Comments != offer.Comments ||
		by.Direction != "by_you" || by.Give[0].Token != "0099" || by.Get[0].Token != "15754" {
		t.Fatalf("directions/assets: %+v %+v", incoming, by)
	}
	if len(transport.requests) != 0 {
		t.Fatal("reading fetched")
	}
	r.Offers[0].Give[0].Name = "mutated"
	again, err := a.tradesAPI().TargetTrades("0025")
	if err != nil || again.Offers[0].Give[0].Name == "mutated" {
		t.Fatal("aliased reading", err)
	}
	for _, fetched := range []time.Time{{}, a.core.MovesNow()} {
		a.core.SeasonPendingTrades.FetchedAt = fetched
		a.core.SeasonPendingTrades.RefreshError = "offline"
		failed, err := a.tradesAPI().TargetTrades("0025")
		if err != nil || len(failed.Offers) != 0 || failed.Provenance.Freshness.Note != "offline" {
			t.Fatalf("failed offers: %+v %v", failed, err)
		}
		body, err := json.Marshal(failed)
		if err != nil || !strings.Contains(string(body), `"offers":[]`) {
			t.Fatalf("JSON: %s %v", body, err)
		}
	}
	if _, err := a.tradesAPI().TargetTrades("unknown"); err == nil {
		t.Fatal("unknown franchise passed")
	}
}

func TestTradeDraftHandOffWatcher(t *testing.T) {
	t.Parallel()
	a, transport, offer := tradeDraftApp(t)
	r, err := a.tradesAPI().TargetDraftTradeAccept("0025", offer.ID)
	if err != nil || r.State != envelope.Ready || r.Spec.Deadline == nil ||
		!r.Spec.Deadline.Equal(offer.Expires) || r.Audit[0].Note != "offer 77 from Other team" {
		t.Fatalf("draft: %+v %v", r, err)
	}
	const target = "https://www47.myfantasyleague.com/2026/options?L=14432&O=05"
	opens := 0
	a.core.OpenURL = func(ctx context.Context, got string) {
		opens++
		stored, err := a.core.Moves.Get(ctx, r.CorrelationID)
		if got != target || err != nil || stored.State() != envelope.HandedOff {
			t.Fatalf("hand-off: %s %s %v", got, stored.State(), err)
		}
	}
	handed := a.core.MovesNow().Add(time.Minute)
	a.core.MovesNow = func() time.Time { return handed }
	if _, err := a.platformAPI().TargetHandOff(r.CorrelationID); err != nil || opens != 1 {
		t.Fatal("hand-off", err, opens)
	}
	if len(transport.requests) != 0 {
		t.Fatal("draft fetched")
	}
	for i, want := range []envelope.State{
		envelope.NotYetDone, envelope.DOTReview, envelope.DOTReview, envelope.Landed,
	} {
		now := handed.Add(time.Duration(i+1) * time.Minute)
		a.core.MovesNow = func() time.Time { return now }
		requested := []envelope.Source{}
		a.core.MovesRefresh = func(_ context.Context, source envelope.Source) error {
			requested = append(requested, source)
			switch source {
			case envelope.PendingTrades:
				a.core.SeasonPendingTrades.FetchedAt = now
				if i > 0 {
					a.core.SeasonPendingTrades.Value = nil
				}
			case envelope.Transactions:
				a.core.SeasonTransactions.FetchedAt = now
				when := handed.Add(-time.Second)
				if i == 3 {
					when = handed.Add(time.Second)
				}
				a.core.SeasonTransactions.Value = []leaguefeed.Transaction{{
					Kind: "TRADE", Time: when, Franchise: offer.Offering,
					Trade: &leaguefeed.Trade{
						Counterparty: offer.OfferedTo, Gave: offer.Gives, CounterpartyGave: offer.Gets,
					},
				}}
			case envelope.Rosters, envelope.Contracts, envelope.Lineups:
				t.Fatalf("unexpected feed %s", source)
			}
			return nil
		}
		if err := a.checkMoves(a.core.Ctx); err != nil {
			t.Fatal(err)
		}
		stored, err := a.core.Moves.Get(a.core.Ctx, r.CorrelationID)
		if err != nil || stored.State() != want || len(requested) != 2 {
			t.Fatalf("step %d: %s %v feeds %v", i, stored.State(), err, requested)
		}
		if i == 2 && len(stored.Receipt().Audit) != 4 {
			t.Fatal("watcher should skip saving same-state DOT audit", stored.Receipt().Audit)
		}
	}
}

func TestTradeBindingBlocks(t *testing.T) {
	t.Parallel()
	for _, name := range []string{"failed", "stale failed", "unowned", "expired", "host", "offerer", "gone"} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			a, transport, offer := tradeDraftApp(t)
			note := ""
			switch name {
			case "failed", "stale failed":
				a.core.SeasonPendingTrades.RefreshError = "offline"
				if name == "failed" {
					a.core.SeasonPendingTrades.FetchedAt = time.Time{}
				}
				note = "pending trades unavailable: offline"
			case "unowned":
				a.core.TargetSnapshot.Rosters.Value = nil
				note = "not on this franchise's roster"
			case "expired":
				a.core.MovesNow = func() time.Time { return offer.Expires.Add(time.Second) }
				note = "deadline passed"
			case "host":
				a.core.MFLClient = nil
				note = "MFL host not yet known"
			case "offerer":
				a.core.SeasonPendingTrades.Value[0].Offering = "0025"
				note = "you made this offer; accept is the other owner's"
			case "gone":
				a.core.SeasonPendingTrades.Value = nil
				note = "offer 77 is no longer pending"
			}
			r, err := a.tradesAPI().TargetDraftTradeAccept("0025", offer.ID)
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
	t.Parallel()
	for _, name := range []string{"ready", "handed", "dot", "blocked"} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			a, _, offer := tradeDraftApp(t)
			if name == "blocked" {
				a.core.MFLClient = nil
			}
			first, err := a.tradesAPI().TargetDraftTradeAccept("0025", offer.ID)
			if err != nil {
				t.Fatal(err)
			}
			if name == "handed" || name == "dot" {
				a.core.OpenURL = func(context.Context, string) {}
				if _, err := a.platformAPI().TargetHandOff(first.CorrelationID); err != nil {
					t.Fatal(err)
				}
			}
			if name == "dot" {
				e, err := a.core.Moves.Get(a.core.Ctx, first.CorrelationID)
				if err != nil {
					t.Fatal(err)
				}
				now := a.core.MovesNow().Add(time.Minute)
				fresh := snapshot.Provenance{Freshness: liveFreshness(now)}
				obs := envelope.Observation{LeagueID: "14432"}
				obs.Transactions.Provenance, obs.PendingTrades.Provenance = fresh, fresh
				e, err = e.Observe(now, obs, envelope.TradePredicate{})
				if err != nil {
					t.Fatal(err)
				}
				saveMove(t, a, e)
				a.core.MovesNow = func() time.Time { return now }
			}
			other := offer
			other.ID = "88"
			other.Expires = offer.Expires.Add(time.Hour)
			a.core.SeasonPendingTrades.Value = append(a.core.SeasonPendingTrades.Value, other)
			unrelated, err := a.tradesAPI().TargetDraftTradeAccept("0025", other.ID)
			if err != nil {
				t.Fatal(err)
			}
			second, err := a.tradesAPI().TargetDraftTradeAccept("0025", offer.ID)
			if err != nil {
				t.Fatal(err)
			}
			old, err := a.core.Moves.Get(a.core.Ctx, first.CorrelationID)
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
			kept, err := a.core.Moves.Get(a.core.Ctx, unrelated.CorrelationID)
			if err != nil || kept.State() == envelope.Stale {
				t.Fatal("other trade superseded", err)
			}
		})
	}
}

func TestTradesBindingGoldens(t *testing.T) {
	t.Parallel()
	a, transport, offer := tradeDraftApp(t)
	reading, err := a.tradesAPI().TargetTrades("0025")
	if err != nil {
		t.Fatal(err)
	}
	checkGolden(t, "trades", "TargetTrades", reading)
	receipt, err := a.tradesAPI().TargetDraftTradeAccept("0025", offer.ID)
	if err != nil {
		t.Fatal(err)
	}
	receipt.CorrelationID = "fixture-correlation"
	checkGolden(t, "trades", "TargetDraftTradeAccept", receipt)
	builder := a.tradesAPI().TargetTradeBuilderRead("invalid")
	checkGolden(t, "trades", "TargetTradeBuilder-invalid", builder)
	if len(transport.requests) != 0 {
		t.Fatal("binding fetched")
	}
}

func TestTradesDenialGoldens(t *testing.T) {
	t.Parallel()
	f := tradeApp(t)
	f.app.core.LocalZone = time.UTC
	denials := map[string]string{}
	for _, who := range []string{"stranger", "owner.one"} {
		_, err := f.action(t, who, commissioner.TradeOffer, tradePlayerOffer())
		denials[who+"/offer-franchise.02"] = errorWords(err)
		_, err = f.action(t, who, commissioner.TradeOverride, commissioner.RawTradeArgs{ID: "absent", Decision: "veto"})
		denials[who+"/override"] = errorWords(err)
	}
	f.app.core.CommissionerSession = testCommissionerSession{"creator"}
	f.set(t, "calendar.windows", `{"closed":{"start":{"month":1,"day":1},"opens":[]}}`, "now")
	_, err := f.action(t, "owner.two", commissioner.TradeOffer, tradePlayerOffer())
	denials["closed/offer"] = errorWords(err)
	f.app.core.CommissionerSession = testCommissionerSession{"creator"}
	f.set(t, "calendar.windows", `{"native":{"start":{"month":1,"day":1},"opens":["trade"]}}`, "now")
	f.set(t, "calendar.tradeDeadline", `"2026-01-01T00:00:00Z"`, "now")
	_, err = f.action(t, "owner.two", commissioner.TradeOffer, tradePlayerOffer())
	denials["deadline/offer"] = errorWords(err)
	a, transport, offer := tradeDraftApp(t)
	_, err = a.tradesAPI().TargetDraftTradeAccept("0001", offer.ID)
	denials["local/offerer-accept"] = errorWords(err)
	_, held, err := a.tradeState("0025")
	if err != nil {
		t.Fatal(err)
	}
	_, err = a.draftTradeExit(a.core.Ctx, tradesapp.RawTradeBuilder{Mine: "0025", ID: offer.ID, Action: "revoke"}, held)
	denials["local/wrong-revoke"] = errorWords(err)
	checkGolden(t, "trades", "denials", denials)
	if len(transport.requests) != 0 {
		t.Fatal("denial fetched")
	}
}

func TestTradeBuilderPopulatedGolden(t *testing.T) {
	t.Parallel()
	a, transport := realTradeBuilderApp(t)
	reading := a.tradesAPI().TargetTradeBuilderPreview(realDeal())
	if reading.Status != report.ReadingOK || reading.Value.Preview == nil {
		t.Fatal(reading)
	}
	reading.Value.Teams = reading.Value.Teams[:2]
	for i := range reading.Value.Teams {
		reading.Value.Teams[i].Assets = reading.Value.Teams[i].Assets[:2]
	}
	for i := range reading.Value.Preview.Teams {
		reading.Value.Preview.Teams[i].Roster = reading.Value.Preview.Teams[i].Roster[:2]
	}
	checkGolden(t, "trades", "TargetTradeBuilder-preview", reading)
	if len(transport.requests) != 0 {
		t.Fatal("builder fetched")
	}
}

func TestTradesReadDuringHeldPublication(t *testing.T) {
	t.Parallel()
	a, transport, offer := tradeDraftApp(t)
	start, done := make(chan struct{}), make(chan struct{})
	go func() {
		defer close(done)
		<-start
		for i := 0; i < 100; i++ {
			next := offer
			next.Comments = strconv.Itoa(i)
			a.core.SeasonMu.Lock()
			a.core.SeasonPendingTrades.Value = []leaguefeed.PendingTrade{next}
			a.core.SeasonMu.Unlock()
		}
	}()
	close(start)
	for i := 0; i < 100; i++ {
		reading, err := a.tradesAPI().TargetTrades("0025")
		if err != nil || len(reading.Offers) != 1 {
			t.Errorf("held trade publication: %+v %v", reading, err)
			break
		}
		reading.Offers[0].Get[0].Name = "detached result"
	}
	<-done
	reading, err := a.tradesAPI().TargetTrades("0025")
	if err != nil || reading.Offers[0].Comments != "99" || reading.Offers[0].Get[0].Name == "detached result" {
		t.Fatal(reading, err)
	}
	if len(transport.requests) != 0 {
		t.Fatal("held publication fetched")
	}
}

package envelope

import (
	"encoding/json"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
)

func tradeFixture(t *testing.T) (TradeRequest, snapshot.Snapshot, TradeCheck, time.Time) {
	t.Helper()
	_, at, snap := testEnvelope(t)
	parse := func(token string) []leaguefeed.Asset {
		assets, err := leaguefeed.ParseAssets(token)
		if err != nil {
			t.Fatal(err)
		}
		return assets
	}
	offer := leaguefeed.PendingTrade{
		ID: "77", Offering: "0002", OfferedTo: "0001",
		Gives: parse("99,FP_0002_2027_1"), Gets: parse("0531"), Expires: at.Add(time.Hour),
	}
	req := TradeRequest{
		LeagueID: "1", FranchiseID: "0001", Deadline: &offer.Expires,
		Target: Target{Kind: Mapped, URL: "https://fixture.invalid/options?O=05"},
		Trade: ExpectedTrade{
			TradeID: offer.ID, Offering: offer.Offering, Accepting: offer.OfferedTo,
			OfferingGives: leaguefeed.AssetTokens(offer.Gives), AcceptingGives: leaguefeed.AssetTokens(offer.Gets),
		},
	}
	check := TradeCheck{
		At: at, OfferingName: "Other team",
		PendingTrades: snapshot.Sourced[[]leaguefeed.PendingTrade]{
			Value:      []leaguefeed.PendingTrade{offer},
			Provenance: snapshot.Provenance{Freshness: domain.Freshness{State: domain.FreshLive}},
		},
	}
	return req, snap, check, at
}

func TestTradeSpecValidation(t *testing.T) {
	req, snap, check, at := tradeFixture(t)
	e, err := DraftTrade(at, req, snap, check, func() string { return "trade" })
	if err != nil || e.State() != Ready || e.spec.Gravity != G2 || e.spec.Undo != Irreversible {
		t.Fatalf("draft: %+v %v", e, err)
	}
	for _, tc := range []struct {
		name   string
		mutate func(*Spec)
	}{
		{"missing", func(s *Spec) { s.Expected.Trade = nil }},
		{"id", func(s *Spec) { s.Expected.Trade.TradeID = "" }},
		{"offering", func(s *Spec) { s.Expected.Trade.Offering = "" }},
		{"accepting", func(s *Spec) { s.Expected.Trade.Accepting = "" }},
		{"same", func(s *Spec) { s.Expected.Trade.Offering = s.Expected.Trade.Accepting }},
		{"scope", func(s *Spec) { s.FranchiseID = "0002" }},
		{"give", func(s *Spec) { s.Expected.Trade.OfferingGives = nil }},
		{"get", func(s *Spec) { s.Expected.Trade.AcceptingGives = nil }},
		{"players", func(s *Spec) { s.Subject.Players = nil }},
		{"picks", func(s *Spec) { s.Subject.Picks = nil }},
		{"status", func(s *Spec) { s.Expected.RosterStatus = domain.RosterIR }},
		{"player", func(s *Spec) { s.Expected.Player = s.Subject.Players[0] }},
		{"lineup", func(s *Spec) { s.Expected.Lineup = &ExpectedLineup{Week: 1} }},
		{"invalid asset", func(s *Spec) { s.Expected.Trade.AcceptingGives = []string{"bad"} }},
		{"money", func(s *Spec) { s.Expected.Trade.AcceptingGives = []string{"BB_1.00"} }},
		{"legacy mixed", func(s *Spec) { s.Intent = "roster.ir" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := e.Receipt().Spec
			tc.mutate(&s)
			if err := validateSpec(s); err == nil {
				t.Fatal("invalid spec passed")
			}
		})
	}
	ir, _, _ := testEnvelope(t)
	if err := validateSpec(ir.spec); err != nil {
		t.Fatal(err)
	}
	lineReq, lineSnap, lineCheck := lineupFixture(t)
	if _, err := DraftLineup(at, lineReq, lineSnap, lineCheck, func() string { return "line" }); err != nil {
		t.Fatal(err)
	}
	r := e.Receipt()
	r.Spec.Expected.Trade.OfferingGives[0] = "mutated"
	if e.Receipt().Spec.Expected.Trade.OfferingGives[0] == "mutated" {
		t.Fatal("aliased expected trade")
	}
}

func TestTradeChecks(t *testing.T) {
	for _, name := range []string{"pass", "gone", "fail", "stale fail", "offerer", "expired",
		"unowned", "pick", "roster fail", "changed", "host"} {
		t.Run(name, func(t *testing.T) {
			req, snap, check, at := tradeFixture(t)
			e, err := DraftTrade(at, req, snap, check, func() string { return "trade" })
			if err != nil {
				t.Fatal(err)
			}
			s := e.Receipt().Spec
			note := ""
			switch name {
			case "gone":
				check.PendingTrades.Value = nil
				note = "offer 77 is no longer pending"
			case "fail", "stale fail":
				check.PendingTrades.Provenance.Freshness = domain.Freshness{State: domain.FreshFail, Note: "offline"}
				if name == "stale fail" {
					check.PendingTrades.Provenance.Freshness.State = domain.FreshStale
				}
				note = "pending trades unavailable: offline"
			case "offerer":
				s.FranchiseID = "0002"
				note = "you made this offer; accept is the other owner's"
			case "expired":
				check.At = at.Add(2 * time.Hour)
				note = "offer 77 expired"
			case "unowned":
				snap.Rosters.Value = nil
				note = "asset 0531 is not on this franchise's roster"
			case "pick":
				offer := &check.PendingTrades.Value[0]
				offer.Gets = append(slices.Clone(offer.Gets), offer.Gives[1:]...)
				s.Expected.Trade.AcceptingGives = leaguefeed.AssetTokens(offer.Gets)
				note = "offer 77 from Other team; MFL checks pick ownership"
			case "roster fail":
				snap.Rosters.Provenance.Freshness.State = domain.FreshFail
				note = "roster unavailable"
			case "changed":
				check.PendingTrades.Value[0].Gives = check.PendingTrades.Value[0].Gets
				note = "pending offer differs"
			case "host":
				check.Blocks = []string{"MFL host not yet known"}
				note = "MFL host not yet known"
			}
			got := check.Evaluate(s, snap)
			if name == "pass" || name == "pick" {
				want := note
				if name == "pass" {
					want = "offer 77 from Other team"
				}
				if got.Blocked || got.Note != want {
					t.Fatal(got)
				}
			} else if !got.Blocked || !strings.Contains(got.Note, note) {
				t.Fatal(got)
			}
		})
	}
}

func TestTradeBlockedArrays(t *testing.T) {
	req, snap, check, at := tradeFixture(t)
	req.Trade.OfferingGives = []string{"99"}
	check.PendingTrades.Provenance.Freshness.State = domain.FreshFail
	e, err := DraftTrade(at, req, snap, check, func() string { return "blocked" })
	if err != nil || e.State() != Blocked {
		t.Fatal(e, err)
	}
	body, err := json.Marshal(e.Receipt())
	if err != nil || strings.Contains(string(body), ":null") || !strings.Contains(string(body), `"picks":[]`) {
		t.Fatalf("JSON: %s %v", body, err)
	}
	cloned := cloneSpec(Spec{Expected: ExpectedChange{Trade: &ExpectedTrade{}}})
	if cloned.Expected.Trade.AcceptingGives == nil || cloned.Expected.Trade.OfferingGives == nil {
		t.Fatal("nil clone arrays")
	}
}

func tradeObservation(check TradeCheck, at time.Time) Observation {
	offer := check.PendingTrades.Value[0]
	obs := Observation{LeagueID: "1", HandedOffAt: at, PendingTrades: check.PendingTrades}
	obs.Transactions.Value = []leaguefeed.Transaction{{
		Kind: "TRADE", Time: at.Add(time.Second), Franchise: offer.Offering,
		Trade: &leaguefeed.Trade{Counterparty: offer.OfferedTo, Gave: offer.Gives, CounterpartyGave: offer.Gets},
	}}
	return obs
}

func TestTradePredicateVerdicts(t *testing.T) {
	for _, name := range []string{"match", "reverse", "old", "equal", "pending", "gone", "league",
		"transactions", "pending fail", "stale fail", "partial assets", "other franchise", "unparsed"} {
		t.Run(name, func(t *testing.T) {
			req, snap, check, at := tradeFixture(t)
			e, err := DraftTrade(at, req, snap, check, func() string { return "trade" })
			if err != nil {
				t.Fatal(err)
			}
			obs := tradeObservation(check, at)
			row := &obs.Transactions.Value[0]
			want := Match
			switch name {
			case "reverse":
				row.Franchise, row.Trade.Counterparty = row.Trade.Counterparty, row.Franchise
				row.Trade.Gave, row.Trade.CounterpartyGave = row.Trade.CounterpartyGave, row.Trade.Gave
			case "old":
				row.Time = at.Add(-time.Second)
				want = NoChange
			case "equal":
				row.Time = at
			case "pending":
				obs.Transactions.Value = nil
				want = NoChange
			case "gone":
				obs.Transactions.Value, obs.PendingTrades.Value = nil, nil
				want = AwaitDOT
			case "league":
				obs.LeagueID = "other"
				want = Partial
			case "transactions":
				obs.Transactions.Provenance.Freshness.State = domain.FreshFail
				want = Partial
			case "pending fail", "stale fail":
				obs.PendingTrades.Provenance.Freshness = domain.Freshness{State: domain.FreshFail, Note: "offline"}
				if name == "stale fail" {
					obs.PendingTrades.Provenance.Freshness.State = domain.FreshStale
				}
				want = Partial
			case "partial assets":
				row.Trade.Gave = row.Trade.Gave[:1]
				want = NoChange
			case "other franchise":
				row.Franchise = "0003"
				want = NoChange
			case "unparsed":
				row.Unparsed = true
				want = NoChange
			}
			got := (TradePredicate{}).Evaluate(e.spec, obs)
			if got.Event != want || (want == Partial && !strings.HasPrefix(got.Note, "Not verified: ")) {
				t.Fatal(got)
			}
			if want == AwaitDOT && got.Note !=
				"offer left MFL's pending list; awaiting DOT and commissioner (or declined there)" {
				t.Fatal(got)
			}
		})
	}
	if !reflect.DeepEqual((TradePredicate{}).Sources(), []Source{Transactions, PendingTrades}) {
		t.Fatal("wrong sources")
	}
}

func TestObserveTradeStagesAndCutoff(t *testing.T) {
	req, snap, check, at := tradeFixture(t)
	e, err := DraftTrade(at, req, snap, check, func() string { return "trade" })
	if err != nil {
		t.Fatal(err)
	}
	handed := at.Add(time.Minute)
	e, err = e.HandOff(handed)
	if err != nil {
		t.Fatal(err)
	}
	obs := tradeObservation(check, at)
	obs.HandedOffAt = handed.Add(-time.Hour) // Observe must overwrite caller input.
	for i, want := range []State{NotYetDone, DOTReview, DOTReview, Landed} {
		now := handed.Add(time.Duration(i+1) * time.Minute)
		fresh := domain.Freshness{State: domain.FreshLive, FetchedAt: now.Format(time.RFC3339)}
		obs.Transactions.Provenance.Freshness, obs.PendingTrades.Provenance.Freshness = fresh, fresh
		if i > 0 {
			obs.PendingTrades.Value = nil
		}
		if i == 3 {
			obs.Transactions.Value[0].Time = handed.Add(time.Second)
		}
		e, err = e.Observe(now, obs, TradePredicate{})
		if err != nil || e.State() != want {
			t.Fatalf("step %d: %s %v", i, e.State(), err)
		}
		if i == 2 && e.audit[len(e.audit)-1].Event != NoChange {
			t.Fatal("repeated DOT must audit no_change")
		}
	}
	if len(e.audit) != 6 {
		t.Fatal("NoChange in DOT must append an audit entry", e.audit)
	}
	restored, err := Restore(e.ID(), e.Created(), e.spec, e.audit)
	if err != nil || !reflect.DeepEqual(restored, e) {
		t.Fatal("replay failed", err)
	}
	ir, _, _ := testEnvelope(t)
	if (IRPredicate{}).Evaluate(ir.spec, observeAt(snap, at)).Event == AwaitDOT {
		t.Fatal("IR returned AwaitDOT")
	}
	lineReq, lineSnap, lineCheck := lineupFixture(t)
	line, err := DraftLineup(at, lineReq, lineSnap, lineCheck, func() string { return "line" })
	if err != nil || (LineupPredicate{}).Evaluate(line.spec, Observation{}).Event == AwaitDOT {
		t.Fatal("lineup returned AwaitDOT", err)
	}
}

func TestTradeAssetSetsCanonicalAndOrderIndependent(t *testing.T) {
	req, snap, check, at := tradeFixture(t)
	req.Trade.OfferingGives = []string{"DP_4_9", "99", "FP_0002_2027_1"}
	assets, err := leaguefeed.ParseAssets("FP_0002_2027_1,DP_04_09,99")
	if err != nil {
		t.Fatal(err)
	}
	check.PendingTrades.Value[0].Gives = assets
	e, err := DraftTrade(at, req, snap, check, func() string { return "trade" })
	if err != nil || e.State() != Ready {
		t.Fatal(e, err)
	}
	obs := tradeObservation(check, at)
	if got := (TradePredicate{}).Evaluate(e.spec, obs); got.Event != Match {
		t.Fatal(got)
	}
	obs.HandedOffAt = time.Time{}
	if got := (TradePredicate{}).Evaluate(e.spec, obs); got.Event != Partial {
		t.Fatal("unscoped timestamp landed", got)
	}
}

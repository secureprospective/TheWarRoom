package main

import (
	"context"
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/lab"
	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
	"github.com/secureprospective/TheWarRoom/internal/store/personal"
)

func liveOffersTestApp(t *testing.T, path string) *App {
	t.Helper()
	p, err := db.Open(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := p.Close(); err != nil {
			t.Error(err)
		}
	})
	a := NewApp()
	a.pools, a.season = p, 2026
	if err := personal.New(p).Initialize(context.Background()); err != nil {
		t.Fatal(err)
	}
	return a
}

func offerTestScope() RawOfferScope { return RawOfferScope{From: 2026, To: 2026, Direction: "all"} }
func offerOutcomeCount(r Reading[OffersView], id string) int {
	count := 0
	for _, st := range r.Value.Stretches {
		for _, o := range st.Outcomes {
			if o.ID == id {
				var n int
				_, _ = fmt.Sscanf(strings.TrimPrefix(o.Text, o.Label+" "), "%d", &n)
				count += n
			}
		}
	}
	return count
}

func TestLiveOfferHistoryRestartAndDedup(t *testing.T) {
	path := filepath.Join(t.TempDir(), "league.db")
	a := liveOffersTestApp(t, path)
	ctx := context.Background()
	ts := time.Date(2026, 10, 12, 13, 0, 0, 0, time.UTC)
	at := ts.Add(time.Hour)
	p := leaguefeed.PendingTrade{ID: "9001", Offering: "0025", OfferedTo: "0001", Proposed: ts}
	before := a.TargetOffers(offerTestScope())
	if err := a.recordOffers(ctx, []leaguefeed.PendingTrade{p}, at, nil); err != nil {
		t.Fatal(err)
	}
	r := a.TargetOffers(offerTestScope())
	if r.Count != before.Count+1 || offerOutcomeCount(r, "pending") != 1 ||
		!strings.Contains(r.Value.Summary, "1 open offer") ||
		!strings.Contains(r.Cutoff, at.Format(time.RFC3339)) || r.Provenance.Source != r.Cutoff ||
		r.Provenance.Freshness.State != FreshLive || !strings.Contains(before.Value.Summary, "No open offers.") {
		t.Fatalf("live offer missing or undated: %+v", r)
	}
	for _, st := range r.Value.Stretches {
		if st.ID == "midseason" && !slices.ContainsFunc(st.Outcomes, func(o OfferOutcomeView) bool {
			return o.ID == "pending" && o.Text == "Open offers 1"
		}) {
			t.Fatalf("proposal placed in wrong stretch: %+v", st)
		}
	}
	rows, _, err := personal.New(a.pools).Offers(ctx, "0025")
	if err != nil || len(rows) != 1 || rows[0].Timestamp != ts.Unix() || rows[0].FirstSeen != at.Unix() {
		t.Fatalf("proposal time/first seen: %+v %v", rows, err)
	}

	if err := a.recordOffers(ctx, []leaguefeed.PendingTrade{p}, at.Add(time.Hour), nil); err != nil {
		t.Fatal(err)
	}
	rows, _, err = personal.New(a.pools).Offers(ctx, "0025")
	if err != nil || len(rows) != 1 || rows[0].FirstSeen != at.Unix() || rows[0].LastSeen != at.Add(time.Hour).Unix() {
		t.Fatalf("repeat observation lost first/last seen: %+v %v", rows, err)
	}
	r = a.TargetOffers(offerTestScope())
	if err := a.pools.Close(); err != nil {
		t.Fatal(err)
	}
	restarted := liveOffersTestApp(t, path)
	after := restarted.TargetOffers(offerTestScope())
	if after.Count != r.Count || offerOutcomeCount(after, "pending") != 1 || after.Cutoff != r.Cutoff {
		t.Fatalf("restart lost history: %+v", after)
	}
	d, err := restarted.lab()
	if err != nil {
		t.Fatal(err)
	}
	merged := *d
	merged.Offers = slices.Clone(d.Offers)
	duplicate, err := liveOffer(d, 2026, ts.Unix(), "0025", "0001", "accepted")
	if err != nil {
		t.Fatal(err)
	}
	merged.Offers = append(merged.Offers, duplicate)
	restarted.labData = func() (*lab.Data, error) { return &merged, nil }
	dedup := restarted.TargetOffers(offerTestScope())
	if dedup.Count != r.Count || offerOutcomeCount(dedup, "pending") != 0 {
		t.Fatalf("archive/live overlap counted twice: %+v", dedup)
	}
}

func TestLiveOfferEndingEvidenceAndFailedRead(t *testing.T) {
	for _, tc := range []struct {
		name, kind, sender, receiver string
		delta                        time.Duration
		want                         string
	}{
		{"matching", "TRADE", "0025", "0001", time.Minute, "accepted"},
		{"reverse", "TRADE", "0001", "0025", time.Minute, "accepted"},
		{"none", "", "", "", 0, "unknown"},
		{"other teams", "TRADE", "0025", "0002", time.Minute, "unknown"},
		{"before proposal", "TRADE", "0025", "0001", -time.Second, "unknown"},
		{"after first miss", "TRADE", "0025", "0001", 3 * time.Hour, "unknown"},
		{"not completed", "TRADE_PROPOSAL", "0025", "0001", time.Minute, "unknown"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a := liveOffersTestApp(t, filepath.Join(t.TempDir(), "league.db"))
			ctx := context.Background()
			ts := time.Date(2026, 10, 12, 13, 0, 0, 0, time.UTC)
			p := leaguefeed.PendingTrade{ID: "9001", Offering: "0025", OfferedTo: "0001", Proposed: ts}
			before := a.TargetOffers(offerTestScope())
			if err := a.recordOffers(ctx, []leaguefeed.PendingTrade{p}, ts.Add(time.Hour), nil); err != nil {
				t.Fatal(err)
			}
			if err := a.recordOffers(ctx, nil, ts.Add(90*time.Minute), fmt.Errorf("offline")); err == nil {
				t.Fatal("failed read lost error")
			}
			failed := a.TargetOffers(offerTestScope())
			if offerOutcomeCount(failed, "pending") != 1 || strings.Contains(failed.Cutoff, "14:30:00") {
				t.Fatal("failed read changed pending offer or check time")
			}
			a.seasonTransactions.value = []leaguefeed.Transaction{{Kind: tc.kind, Franchise: tc.sender,
				Time: ts.Add(tc.delta), Trade: &leaguefeed.Trade{Counterparty: tc.receiver}}}
			if err := a.recordOffers(ctx, nil, ts.Add(2*time.Hour), nil); err != nil {
				t.Fatal(err)
			}
			ended := a.TargetOffers(offerTestScope())
			if offerOutcomeCount(ended, "pending") != 0 ||
				offerOutcomeCount(ended, tc.want) != offerOutcomeCount(before, tc.want)+1 {
				t.Fatalf("ending should be %s: %+v", tc.want, ended)
			}
			rows, _, err := personal.New(a.pools).Offers(ctx, "0025")
			if err != nil || rows[0].FirstMissed != ts.Add(2*time.Hour).Unix() {
				t.Fatal("first miss not saved", err)
			}
		})
	}
}

func TestPendingRefreshRecordsHistory(t *testing.T) {
	a, transport, _, _ := seasonTestApp(t)
	if err := personal.New(a.pools).Initialize(a.ctx); err != nil {
		t.Fatal(err)
	}
	if err := a.keyStore.Set(a.ctx, "synthetic-stored-key"); err != nil {
		t.Fatal(err)
	}
	at := time.Date(2026, 10, 13, 14, 9, 0, 0, time.UTC)
	a.clockNow = func() time.Time { return at }
	transport.bodies["pendingTrades"] = `{"pendingTrades":{"pendingTrade":{
 "trade_id":"9001","offeringteam":"0025","offeredto":"0001","timestamp":"1791810000",
 "expires":"1791982800","will_give_up":"0042,","will_receive":"0816,"}}}`
	a.refreshPendingTrades(a.ctx)
	r := a.TargetOffers(offerTestScope())
	if offerOutcomeCount(r, "pending") != 1 || !strings.Contains(r.Cutoff, at.Format(time.RFC3339)) {
		t.Fatalf("refresh did not save history: %+v, feed error %s", r, a.seasonPendingTrades.refreshError)
	}
	transport.failure = "pendingTrades"
	at = at.Add(time.Hour)
	a.refreshPendingTrades(a.ctx)
	if offerOutcomeCount(a.TargetOffers(offerTestScope()), "pending") != 1 {
		t.Fatal("failed fetch ended offer")
	}
}

func TestLiveOfferCalendarMatchesArchive(t *testing.T) {
	d, err := lab.Load()
	if err != nil {
		t.Fatal(err)
	}
	for _, o := range d.Offers {
		got, err := liveOffer(d, o.Year, int64(o.Ts), o.By, o.To, o.Outcome)
		if err != nil || got.Season != o.Season || got.Phase != o.Phase {
			t.Fatalf("calendar diverged at %d: %+v vs %+v, %v", o.Ts, got, o, err)
		}
	}
	for _, tc := range []struct {
		date   string
		season int
		phase  string
	}{
		{"2026-12-07T04:59:00Z", 2026, "closed"},
		{"2026-12-08T05:00:00Z", 2027, "closed"},
		{"2027-01-05T05:00:00Z", 2027, "winter"},
	} {
		at, err := time.Parse(time.RFC3339, tc.date)
		if err != nil {
			t.Fatal(err)
		}
		got, err := liveOffer(d, 2026, at.Unix(), "0025", "0001", "pending")
		if err != nil || got.Season != tc.season || got.Phase != tc.phase {
			t.Fatal(got, err)
		}
	}
}

func TestLiveOfferCutoffAndOwnerScope(t *testing.T) {
	a := liveOffersTestApp(t, filepath.Join(t.TempDir(), "league.db"))
	d, err := a.lab()
	if err != nil {
		t.Fatal(err)
	}
	cutoff, err := time.Parse(time.RFC3339, d.ArchiveThrough)
	if err != nil {
		t.Fatal(err)
	}
	before := a.TargetOffers(offerTestScope())
	pending := []leaguefeed.PendingTrade{
		{ID: "1", Offering: "0025", OfferedTo: "0001", Proposed: cutoff},
		{ID: "2", Offering: "0025", OfferedTo: "0001", Proposed: cutoff.Add(-time.Second)},
		{ID: "3", Offering: "0002", OfferedTo: "0001", Proposed: cutoff.Add(time.Hour)},
		{ID: "4", Offering: "0001", OfferedTo: "0025", Proposed: cutoff.Add(time.Hour)},
	}
	if err := a.recordOffers(context.Background(), pending, cutoff.Add(2*time.Hour), nil); err != nil {
		t.Fatal(err)
	}
	r := a.TargetOffers(offerTestScope())
	if r.Count != before.Count+1 || offerOutcomeCount(r, "pending") != 1 {
		t.Fatal("cutoff or owner leak", r)
	}
	scope := offerTestScope()
	scope.Direction = "sent"
	if offerOutcomeCount(a.TargetOffers(scope), "pending") != 0 {
		t.Fatal("received offer counted as sent")
	}
	scope.Direction = "received"
	if offerOutcomeCount(a.TargetOffers(scope), "pending") != 1 {
		t.Fatal("received offer missing")
	}
	scope.Team = "0002"
	if offerOutcomeCount(a.TargetOffers(scope), "pending") != 0 {
		t.Fatal("counterparty filter ignored")
	}
	rows, _, err := personal.New(a.pools).Offers(context.Background(), "0025")
	if err != nil || len(rows) != 3 {
		t.Fatal("recorded someone else's private offer", rows, err)
	}
}

func TestLiveOfferAfterDeadlineRemainsVisible(t *testing.T) {
	a := liveOffersTestApp(t, filepath.Join(t.TempDir(), "league.db"))
	at := time.Date(2026, 11, 3, 12, 0, 0, 0, time.UTC)
	p := leaguefeed.PendingTrade{ID: "9001", Offering: "0025", OfferedTo: "0001", Proposed: at}
	if err := a.recordOffers(context.Background(), []leaguefeed.PendingTrade{p}, at, nil); err != nil {
		t.Fatal(err)
	}
	r := a.TargetOffers(offerTestScope())
	if offerOutcomeCount(r, "pending") != 1 {
		t.Fatal("post-deadline offer vanished", r)
	}
	if !slices.ContainsFunc(r.Value.Stretches, func(st OfferStretchView) bool {
		return st.ID == "closed" && st.Label == "After the trade deadline"
	}) {
		t.Fatal("post-deadline stretch missing", r)
	}
}

func TestLiveOfferFromUnheldYearKeepsHistoryOnScreen(t *testing.T) {
	a := liveOffersTestApp(t, filepath.Join(t.TempDir(), "league.db"))
	ctx := context.Background()
	ts := time.Date(2027, 3, 2, 15, 0, 0, 0, time.UTC)
	p := leaguefeed.PendingTrade{ID: "9101", Offering: "0025", OfferedTo: "0001", Proposed: ts}
	if err := personal.New(a.pools).ObserveOffers(ctx, "0025", 2027, []leaguefeed.PendingTrade{p}, nil,
		ts.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	r := a.TargetOffers(offerTestScope())
	if r.Status != ReadingOK || r.AbsenceReason != nil ||
		!strings.Contains(r.Cutoff, "1 offer from a league year without held dates not placed in a stretch yet") {
		t.Fatalf("an offer from a year without held dates must not fail the history: %+v", r)
	}
}

func TestOfferHistorySaveFailureLeavesDeskRead(t *testing.T) {
	a, transport, _, _ := seasonTestApp(t)
	if err := a.keyStore.Set(a.ctx, "synthetic-stored-key"); err != nil {
		t.Fatal(err)
	}
	var logged []string
	a.fallbackLog = func(line string) { logged = append(logged, line) }
	a.labData = func() (*lab.Data, error) { return nil, fmt.Errorf("lab unavailable") }
	transport.bodies["pendingTrades"] = `{"pendingTrades":{"pendingTrade":{
 "trade_id":"9001","offeringteam":"0025","offeredto":"0001","timestamp":"1791810000",
 "expires":"1791982800","will_give_up":"0042,","will_receive":"0816,"}}}`
	a.refreshPendingTrades(a.ctx)
	if a.seasonPendingTrades.refreshError != "" || len(a.seasonPendingTrades.value) != 1 {
		t.Fatalf("a failed history save marked the desk's MFL read failed: %q %+v",
			a.seasonPendingTrades.refreshError, a.seasonPendingTrades.value)
	}
	if len(logged) != 1 || !strings.Contains(logged[0], "offers: archive owner") {
		t.Fatalf("history save failure not logged once: %q", logged)
	}
}

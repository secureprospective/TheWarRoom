package envelope

import (
	"reflect"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
)

func tradeInReview(t *testing.T) (Envelope, Observation, time.Time) {
	t.Helper()
	req, snap, check, at := tradeFixture(t)
	e, err := DraftTrade(at, req, snap, check, func() string { return "window" })
	if err != nil {
		t.Fatal(err)
	}
	e, err = e.HandOff(at.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	// Entry is deliberately much later than hand-off.
	entered := at.Add(2 * 24 * time.Hour)
	obs := tradeObservation(check, at)
	obs.Transactions.Value = nil
	obs.PendingTrades.Value = nil
	e = observeTradeWindow(t, e, obs, entered)
	return e, obs, entered
}

func observeTradeWindow(t *testing.T, e Envelope, obs Observation, at time.Time) Envelope {
	t.Helper()
	fresh := domain.Freshness{State: domain.FreshLive, FetchedAt: at.Format(time.RFC3339)}
	obs.Transactions.Provenance.Freshness = fresh
	obs.PendingTrades.Provenance.Freshness = fresh
	next, err := e.Observe(at, obs, TradePredicate{})
	if err != nil {
		t.Fatal(err)
	}
	return next
}

func TestTradeReviewWindow(t *testing.T) {
	for _, tc := range []struct {
		name    string
		elapsed time.Duration
		want    State
	}{
		{"before", 6*24*time.Hour + 23*time.Hour, DOTReview},
		{"equal", 7 * 24 * time.Hour, DOTReview},
		{"after", 7*24*time.Hour + time.Second, NotVerified},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e, obs, entered := tradeInReview(t)
			// A repeated DOT no_change must not reset the entry time.
			e = observeTradeWindow(t, e, obs, entered.Add(time.Hour))
			next := observeTradeWindow(t, e, obs, entered.Add(tc.elapsed))
			if next.State() != tc.want {
				t.Fatalf("state %s, want %s", next.State(), tc.want)
			}
			if tc.want == NotVerified {
				last := next.audit[len(next.audit)-1]
				want := "Not verified: no executed trade 7 days after acceptance " +
					"(declined, vetoed or still in review on MFL)"
				if last.Event != Partial || last.Note != want {
					t.Fatal(last)
				}
			}
		})
	}
}

func TestTradeReviewUnchangedThenLands(t *testing.T) {
	e, obs, entered := tradeInReview(t)
	at := entered.Add(7*24*time.Hour + time.Second)
	e = observeTradeWindow(t, e, obs, at)
	next := observeTradeWindow(t, e, obs, at.Add(time.Hour))
	if next.State() != NotVerified || !reflect.DeepEqual(next.audit, e.audit) {
		t.Fatal("unchanged evidence moved or appended audit", next.Receipt())
	}
	_, _, check, created := tradeFixture(t)
	obs.Transactions.Value = tradeObservation(check, created).Transactions.Value
	obs.Transactions.Value[0].Time = e.handedAt().Add(time.Second)
	next = observeTradeWindow(t, next, obs, at.Add(2*time.Hour))
	if next.State() != Landed {
		t.Fatal("later TRADE did not land", next.Receipt())
	}
}

// Inside the window a stale-read Not verified returns to DOTReview, and the window stays anchored
// at the first entry; past it, the plan stays Not verified with no new audit row.
func TestTradeNotVerifiedInsideWindowReentersReview(t *testing.T) {
	e, obs, entered := tradeInReview(t)
	e, err := e.move(entered.Add(time.Hour), Partial, "old partial evidence")
	if err != nil {
		t.Fatal(err)
	}
	back := observeTradeWindow(t, e, obs, entered.Add(2*time.Hour))
	if back.State() != DOTReview {
		t.Fatal("absent offer inside the window did not re-enter DOTReview", back.Receipt())
	}
	expired := observeTradeWindow(t, back, obs, entered.Add(7*24*time.Hour+time.Second))
	if expired.State() != NotVerified {
		t.Fatal("re-entry restarted the window", expired.Receipt())
	}
	next := observeTradeWindow(t, expired, obs, entered.Add(8*24*time.Hour))
	if !reflect.DeepEqual(next, expired) {
		t.Fatal("absent offer bounced out of NotVerified after the window", next.Receipt())
	}
	_, _, check, _ := tradeFixture(t)
	obs.PendingTrades.Value = check.PendingTrades.Value
	next = observeTradeWindow(t, e, obs, entered.Add(3*time.Hour))
	if next.State() != NotYetDone || next.audit[len(next.audit)-1].Event != NoChange {
		t.Fatal("pending offer did not preserve NoChange behavior", next.Receipt())
	}
}

func TestRestoreOldTradeReviewHistory(t *testing.T) {
	e, _, entered := tradeInReview(t)
	// Old Observe could bounce NotVerified back into DOTReview and append no_change there.
	for i, event := range []Event{Partial, AwaitDOT, NoChange} {
		next, err := e.move(entered.Add(time.Duration(i+1)*time.Hour), event, "old evidence")
		if err != nil {
			t.Fatal(err)
		}
		e = next
	}
	restored, err := Restore(e.ID(), e.Created(), e.spec, e.audit)
	if err != nil || !reflect.DeepEqual(restored, e) {
		t.Fatal("old history replay", err)
	}
}

// A held-stale feed before DOT review is Not verified, but it must not freeze the plan: once the
// offer leaves the pending list the plan still enters DOTReview (no window has started).
func TestTradeFeedFailureStillEntersReview(t *testing.T) {
	req, snap, check, at := tradeFixture(t)
	e, err := DraftTrade(at, req, snap, check, func() string { return "feedfail" })
	if err != nil {
		t.Fatal(err)
	}
	if e, err = e.HandOff(at.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	failed := tradeObservation(check, at)
	stamp := at.Add(2 * time.Minute).Format(time.RFC3339)
	failed.Transactions.Provenance.Freshness = domain.Freshness{State: domain.FreshLive, FetchedAt: stamp}
	failed.PendingTrades.Provenance.Freshness = domain.Freshness{
		State: domain.FreshStale, Note: "refresh failed", FetchedAt: stamp}
	if e, err = e.Observe(at.Add(2*time.Minute), failed, TradePredicate{}); err != nil {
		t.Fatal(err)
	}
	if e.State() != NotVerified {
		t.Fatalf("after feed failure: state %s, want %s", e.State(), NotVerified)
	}
	obs := tradeObservation(check, at)
	obs.Transactions.Value = nil
	obs.PendingTrades.Value = nil
	next := observeTradeWindow(t, e, obs, at.Add(time.Hour))
	if next.State() != DOTReview {
		t.Fatalf("state %s, want %s", next.State(), DOTReview)
	}
}

package history

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

func TestAutomaticStatFeedUsesDurableApprovalLedger(t *testing.T) {
	f := &statResults{resultsFixture: resultsTest(t), t: t, revisions: map[string][]leaguerules.Decimal{}, silent: map[string]bool{}}
	f.installFeed()
	if err := f.s.ProcessStatFeed(t.Context(), "native.test", 2026); err != nil {
		t.Fatal(err)
	}
	before := revision(t, f.s)
	key := fmt.Sprint(f.starter(f.a), "/", 2)
	f.revisions[key] = append(f.revisions[key], 30000)
	if err := f.s.ProcessStatFeed(t.Context(), "native.test", 2026); err != nil {
		t.Fatal(err)
	}
	if got := f.score(t, f.a, 2); got.Points == nil || *got.Points != 45000 {
		t.Fatalf("live score: %+v", got)
	}
	if revision(t, f.s) != before+1 {
		t.Fatal("move not recorded once")
	}
	f.reopen(t)
	if err := f.s.ProcessStatFeed(t.Context(), "native.test", 2026); err != nil {
		t.Fatal(err)
	}
	if revision(t, f.s) != before+1 {
		t.Fatal("repeat duplicated ledger move")
	}
	q := f.queue(t)
	if len(q.Open) != 0 || len(q.Decided) != 7 || q.Decided[0].By != "feed: newest live" {
		t.Fatalf("durable ledger: %+v", q)
	}
	raw := statCommand(commissioner.StatApprove, commissioner.RawStatChange{Item: q.Decided[0].Entry}, revision(t, f.s), "spoof")
	if _, err := f.s.ExecuteCommissioner(t.Context(), raw, commissioner.StatFeedPrincipal, nil); err == nil {
		t.Fatal("feed identity allowed outside worker")
	}
}

func TestAutomaticStatFeedAdoptsSeasonWithinReadBudget(t *testing.T) {
	f := statResultsTest(t)
	const count = 1200
	generation := 0
	f.s.statWeeks = func(context.Context, int) ([]commissioner.StatWeek, error) {
		out := make([]commissioner.StatWeek, count)
		for i := range out {
			player := fmt.Sprint(20000 + i)
			if generation > 0 {
				f.revisions[player+"/1"] = []leaguerules.Decimal{30000}
			}
			out[i] = commissioner.StatWeek{Player: player, Week: 1, Evidence: f.evidenceFor(player, 1)}
		}
		return out, nil
	}
	for generation = 0; generation < 2; generation++ {
		before := revision(t, f.s)
		ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
		began := time.Now()
		err := f.s.ProcessStatFeed(ctx, "native.test", 2026)
		cancel()
		t.Logf("generation %d: %s", generation, time.Since(began))
		if err != nil {
			t.Fatal(err)
		}
		// One ledger entry per approval batch, not per stat line.
		want := before + int64((count+commissioner.MaxStatBatch-1)/commissioner.MaxStatBatch)
		if got := revision(t, f.s); got != want {
			t.Fatalf("adopted revision %d, want %d", got, want)
		}
	}
	before := revision(t, f.s)
	if err := f.s.ProcessStatFeed(t.Context(), "native.test", 2026); err != nil {
		t.Fatal(err)
	}
	if got := revision(t, f.s); got != before {
		t.Fatal("unchanged season duplicated approvals")
	}
}

func TestAutomaticStatFeedRechecksAdmissionAndRollsBackBatch(t *testing.T) {
	f := statResultsTest(t)
	before := revision(t, f.s)
	calls := 0
	ctx := commissioner.WithCredentialAdmission(t.Context(), admissionFunc(func(_ context.Context, access commissioner.Access) error {
		calls++
		if access.Principal != commissioner.StatFeedPrincipal || access.Operation != commissioner.StatApprove {
			t.Fatalf("unexpected feed authority: %+v", access)
		}
		if calls == 1 {
			return fmt.Errorf("feed admission revoked")
		}
		return nil
	}))
	if err := f.s.ProcessStatFeed(ctx, "native.test", 2026); err == nil || calls != 1 {
		t.Fatalf("revocation not rechecked: calls %d, error %v", calls, err)
	}
	if got := revision(t, f.s); got != before {
		t.Fatalf("refused batch left partial approvals: revision %d, want %d", got, before)
	}
	if err := f.s.ProcessStatFeed(t.Context(), "native.test", 2026); err != nil {
		t.Fatal("refused batch prevented retry", err)
	}
	// Six weeks are one batch per feed reason.
	if got := revision(t, f.s); got < before+1 || got > before+2 {
		t.Fatalf("successful retry did not adopt the six weeks as batches: revision %d", got)
	}
	adopted := revision(t, f.s)
	if err := f.s.ProcessStatFeed(t.Context(), "native.test", 2026); err != nil || revision(t, f.s) != adopted {
		t.Fatalf("retry re-approved adopted weeks: %v", err)
	}
}

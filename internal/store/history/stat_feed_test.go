package history

import (
	"fmt"
	"testing"

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

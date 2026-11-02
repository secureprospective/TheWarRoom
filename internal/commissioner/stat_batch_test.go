package commissioner

import (
	"reflect"
	"slices"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/fantasyscore/source"
)

func batchWeeks() []StatWeek {
	weeks := []StatWeek{}
	for w := 1; w <= 4; w++ {
		versions := []source.LineRevision{tackleVersion(0, "LB", 10000, 5), tackleVersion(1, "LB", 20000, 6)}
		for i := range versions {
			held := *versions[i].Held.Week
			held.Week = w
			versions[i].Held.Week = &held
			versions[i].Digest = source.LineDigest(held)
		}
		if w%2 == 0 {
			versions[1].Held.Authority = source.NFLVerse
		}
		weeks = append(weeks, StatWeek{Player: "0099", Week: w, Evidence: source.NativeEvidence{GSIS: "00-0023459",
			Held: &versions[len(versions)-1].Held, Lines: versions}})
	}
	return weeks
}

func feedLedger(t *testing.T, weeks []StatWeek, batched bool) *ledgerBuilder {
	t.Helper()
	b := seededLedger(t)
	moves, err := FeedStatMoves("native.test", b.records, weeks, 2026)
	if err != nil || len(moves) != len(weeks) {
		t.Fatalf("moves: %+v %v", moves, err)
	}
	if batched {
		moves = BatchFeedMoves(moves)
		if len(moves) != 2 {
			t.Fatalf("one entry per reason expected: %+v", moves)
		}
	}
	for _, raw := range moves {
		if err := raw.Args.Validate(raw.Command); err != nil {
			t.Fatal(err)
		}
		b.add(raw, seasonMoment(10))
		b.records[len(b.records)-1].Principal = StatFeedPrincipal
	}
	return b
}

func TestBatchedFeedApprovalsReplayToTheSameState(t *testing.T) {
	weeks := batchWeeks()
	lines, batch := feedLedger(t, weeks, false), feedLedger(t, weeks, true)
	for _, week := range weeks {
		scope := source.RawNativeScope{Player: "0099", Season: 2026, Week: week.Week}
		want, err := NativePlayerScoreReading("native.test", scope, lines.records, week.Evidence, seasonMoment(20))
		if err != nil {
			t.Fatal(err)
		}
		got, err := NativePlayerScoreReading("native.test", scope, batch.records, week.Evidence, seasonMoment(20))
		if err != nil || !reflect.DeepEqual(want.Points, got.Points) {
			t.Fatalf("week %d: per-line %+v, batched %+v %v", week.Week, want.Points, got.Points, err)
		}
	}
	a, err := foldStatEffects(lines.records)
	if err != nil {
		t.Fatal(err)
	}
	b, err := foldStatEffects(batch.records)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(a.baselines, b.baselines) || !reflect.DeepEqual(a.rejected, b.rejected) ||
		!reflect.DeepEqual(a.corrections, b.corrections) || len(a.changes) != len(b.changes) {
		t.Fatalf("effects differ:\n%+v\n%+v", a, b)
	}
	for key, changes := range a.changes {
		other := b.changes[key]
		if len(changes) != len(other) || changes[0].What != other[0].What || changes[0].Reason != other[0].Reason ||
			changes[0].Status != other[0].Status {
			t.Fatalf("provenance %v: %+v vs %+v", key, changes, other)
		}
	}
	qa, qb := queueOf(t, lines, weeks...), queueOf(t, batch, weeks...)
	if len(qa.Open) != len(qb.Open) || len(qa.Decided) != len(qb.Decided) {
		t.Fatalf("queues differ: %+v vs %+v", qa, qb)
	}
	// Batches group a pass by reason, so the newest-first list may interleave differently; each decision is the same.
	byWeek := func(d StatDecision) int { return d.Week }
	slices.SortFunc(qa.Decided, func(x, y StatDecision) int { return byWeek(x) - byWeek(y) })
	slices.SortFunc(qb.Decided, func(x, y StatDecision) int { return byWeek(x) - byWeek(y) })
	for i := range qa.Decided {
		x, y := qa.Decided[i], qb.Decided[i]
		x.Entry, y.Entry, x.At = "", "", y.At
		if !reflect.DeepEqual(x, y) {
			t.Fatalf("decision %d: %+v vs %+v", i, x, y)
		}
	}
	for _, records := range [][]Record{lines.records, batch.records} {
		if again, err := FeedStatMoves("native.test", records, weeks, 2026); err != nil || len(again) != 0 {
			t.Fatalf("feed re-approved: %+v %v", again, err)
		}
	}
	if _, err := Reading("native.test", "creator", batch.records, allowAll{}); err != nil {
		t.Fatalf("ledger reading of a batch: %v", err)
	}
}

func TestStatBatchIsFeedApprovalOnly(t *testing.T) {
	weeks := batchWeeks()
	b := seededLedger(t)
	moves, err := FeedStatMoves("native.test", b.records, weeks, 2026)
	if err != nil {
		t.Fatal(err)
	}
	items := []string{moves[0].Args.Stat.Item, moves[1].Args.Stat.Item}
	for _, bad := range []RawStatChange{
		{Items: []string{items[0], items[0]}},
		{Items: items, Item: items[0]},
		{Items: []string{"stat.nope"}},
		{Items: []string{}},
	} {
		if _, err := DecodeStatChange(RawArgs{Stat: &bad}, StatApprove); err == nil {
			t.Fatalf("accepted %+v", bad)
		}
	}
	if _, err := DecodeStatChange(RawArgs{Stat: &RawStatChange{Items: items}}, StatReject); err == nil {
		t.Fatal("batched rejection accepted")
	}
	raw := RawCommand{League: "native.test", Command: StatApprove, Args: RawArgs{Stat: &RawStatChange{Items: items}},
		Request: "manual", Effective: "now", Visibility: "league"}
	if err := CheckStatAppend(raw, b.records, StatFacts{}, seasonMoment(20)); err == nil {
		t.Fatal("manual batch accepted")
	}
	if err := CheckFeedStatAppend(raw, b.records, StatFacts{}); err == nil {
		t.Fatal("unsplit feed batch accepted")
	}
}

func TestGovernanceStateMemoMatchesFoldAndFollowsChanges(t *testing.T) {
	b := feedLedger(t, batchWeeks(), false)
	want, err := governanceFold(b.records)
	if err != nil {
		t.Fatal(err)
	}
	held := WithReplayCache(b.records)
	for range 2 {
		got, err := GovernanceState(held)
		if err != nil || !reflect.DeepEqual(got, want) {
			t.Fatalf("memo differs: %v", err)
		}
	}
	if memoFor(held) == nil || len(memoFor(held).views) != 1 {
		t.Fatal("memo not held")
	}
	// Verified identities drop the projection memo; the governance memo must survive that.
	if memoFor(WithoutClosedBooks(held)) == nil {
		t.Fatal("identity view lost the governance memo")
	}
	edited := slices.Clone(held)
	edited[len(edited)-1].Canonical = "changed"
	if memoFor(edited) != nil {
		t.Fatal("edited payload reused the memo")
	}
	if memoFor(held[:len(held)-1]) != nil {
		t.Fatal("truncated ledger reused the memo")
	}
}

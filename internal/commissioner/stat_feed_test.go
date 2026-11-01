package commissioner

import (
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/fantasyscore/source"
)

func TestFeedCountedRevisionFollowsLiveThenNFLVerse(t *testing.T) {
	b := seededLedger(t)
	versions := []source.LineRevision{tackleVersion(0, "LB", 10000, 5), tackleVersion(1, "LB", 20000, 6), tackleVersion(2, "LB", 30000, 7), tackleVersion(3, "LB", 40000, 8)}
	versions[2].Held.Authority = source.NFLVerse
	scope := source.RawNativeScope{Player: "0099", Season: 2026, Week: 1}
	for i, want := range []int64{15000, 30000, 45000, 45000} {
		week := statWeek(versions[:i+1]...)
		moves, err := FeedStatMoves("native.test", b.records, []StatWeek{week}, 2026)
		if err != nil {
			t.Fatal(err)
		}
		expected := 1
		if i == 3 {
			expected = 0
		}
		if len(moves) != expected {
			t.Fatalf("step %d moves: %+v", i, moves)
		}
		for _, raw := range moves {
			if err := CheckFeedStatAppend(raw, b.records, StatFacts{Versions: versions[:i+1]}); err != nil {
				t.Fatal(err)
			}
			b.add(raw, seasonMoment(10+i))
			b.records[len(b.records)-1].Principal = StatFeedPrincipal
		}
		score, err := NativePlayerScoreReading("native.test", scope, b.records, week.Evidence, seasonMoment(20))
		if err != nil || score.Points == nil || int64(*score.Points) != want {
			t.Fatalf("step %d: %+v %v", i, score, err)
		}
	}
	q := queueOf(t, b, statWeek(versions...))
	if len(q.Open) != 0 || len(q.Decided) != 3 {
		t.Fatalf("ledger: %+v", q)
	}
	for i, d := range q.Decided {
		want := "feed: newest live"
		if i == 0 {
			want = "feed: nflverse final"
		}
		if d.By != want || d.Reason != want || d.At.IsZero() || d.Entry == "" {
			t.Fatalf("provenance: %+v", d)
		}
	}
	moves, err := FeedStatMoves("native.test", b.records, []StatWeek{statWeek(versions...)}, 2026)
	if err != nil || len(moves) != 0 {
		t.Fatalf("repeat: %+v %v", moves, err)
	}
	b.statCommand(StatCorrect, RawStatChange{Player: "0099", Season: 2026, Week: 1, Stat: "tackle", Value: decimalOf(90000), Source: "commissioner"}, 12)
	versions = append(versions, tackleVersion(4, "LB", 50000, 9))
	versions[4].Held.Authority = source.NFLVerse
	score, err := NativePlayerScoreReading("native.test", scope, b.records, statWeek(versions...).Evidence, seasonMoment(20))
	if err != nil || score.Points == nil || *score.Points != 135000 {
		t.Fatalf("correction lost: %+v %v", score, err)
	}
	b.statCommand(StatApprove, RawStatChange{Item: itemOf(versions[0])}, 13)
	selected, err := countedVersion(versions, StatBaseline{Seq: 0, Digest: versions[0].Digest, Entry: "manual"})
	if err != nil || selected.Seq != 0 {
		t.Fatalf("manual pin: %+v %v", selected, err)
	}
	moves, err = FeedStatMoves("native.test", b.records, []StatWeek{statWeek(versions...)}, 2026)
	if err != nil || len(moves) != 0 {
		t.Fatalf("manual override replaced: %+v %v", moves, err)
	}
}

func TestLaterLiveEvidenceGapsDoNotDowngradeNFLVerseFinal(t *testing.T) {
	b := seededLedger(t)
	final, live := tackleVersion(0, "LB", 10000, 5), tackleVersion(1, "LB", 20000, 6)
	final.Held.Authority = source.NFLVerse
	live.Held.Missing = []string{"Live details are incomplete"}
	week := statWeek(final, live)
	week.Evidence.Missing = live.Held.Missing
	got, err := NativePlayerScoreReading("native.test", source.RawNativeScope{Player: "0099", Season: 2026, Week: 1}, b.records, week.Evidence, seasonMoment(20))
	if err != nil || got.Points == nil || *got.Points != 15000 || len(got.Missing) != 0 {
		t.Fatalf("later live downgraded final: %+v %v", got, err)
	}
}

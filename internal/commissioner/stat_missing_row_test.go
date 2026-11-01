package commissioner

import (
	"os"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/fantasyscore/source"
)

func TestMissingWeeklyRowStaysInQueueAfterFeedCountsIt(t *testing.T) {
	b := seededLedger(t)

	body, err := os.ReadFile("../fantasyscore/source/testdata/missing_tackle.csv")
	if err != nil {
		t.Fatal(err)
	}
	plays, err := source.ReadPlays(strings.NewReader(string(body)))
	if err != nil {
		t.Fatal(err)
	}
	rows, err := source.MissingPlayWeeks(nil, plays)
	if err != nil || len(rows) != 1 {
		t.Fatalf("%+v %v", rows, err)
	}
	if err := source.AttachPlays(rows, plays); err != nil {
		t.Fatal(err)
	}
	held := source.SelectHeldWeek(source.RawStatScope{GSIS: "00-0023459", Season: 2026, Week: 1}, rows, "fixture-missing-weekly-row", seasonMoment(5))
	held.Missing = nil
	v := source.DistinctLines([]source.HeldWeek{held})[0]
	week := statWeek(v)
	week.Evidence.ListedPosition = "LB"
	moves, err := FeedStatMoves("native.test", b.records, []StatWeek{week}, 2026)
	if err != nil || len(moves) != 1 {
		t.Fatalf("%+v %v", moves, err)
	}
	for _, raw := range moves {
		if err := CheckFeedStatAppend(raw, b.records, StatFacts{Versions: week.Evidence.Versions()}); err != nil {
			t.Fatal(err)
		}
		b.add(raw, seasonMoment(6))
		b.records[len(b.records)-1].Principal = StatFeedPrincipal
	}
	out := queueOf(t, b, week)
	if len(out.Open) != 1 || !strings.Contains(out.Open[0].Words, "nflverse missing row") {
		t.Fatalf("%+v", out)
	}
	score, err := NativePlayerScoreReading("native.test", source.RawNativeScope{Player: "0099", Season: 2026, Week: 1}, b.records, week.Evidence, seasonMoment(20))
	if err != nil || score.Points == nil || *score.Points != 15000 {
		t.Fatalf("%+v %v", score, err)
	}
	b.statCommand(StatApprove, RawStatChange{Item: out.Open[0].ID}, 6)
	if got := queueOf(t, b, week); len(got.Open) != 0 {
		t.Fatalf("manual approval did not clear review: %+v", got)
	}
}

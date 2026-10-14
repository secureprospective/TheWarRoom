package commissioner

import (
	"math"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/fantasyscore"
	"github.com/secureprospective/TheWarRoom/internal/fantasyscore/source"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

func completeScoreEvidence() source.NativeEvidence {
	return source.NativeEvidence{GSIS: "00-0023459", Held: &source.HeldWeek{SHA256: "held-revision",
		FetchedAt: seasonMoment(4), Week: &source.PlayerWeek{Position: "LB", Season: 2026, Week: 1,
			Line: fantasyscore.StatLine{Quantities: map[leaguerules.StatID]leaguerules.Decimal{"tackle": 10000}}}}}
}

func TestNativePlayerScoreCompleteAndIncompleteNeverShareTotal(t *testing.T) {
	b := newLedger(t, PresetLegacy)
	b.seed(2026, nil, seasonMoment(2))
	evidence := completeScoreEvidence()
	scope := source.RawNativeScope{Player: "0099", Season: 2026, Week: 1}
	out, err := NativePlayerScoreReading("native.test", scope, b.records, evidence, seasonMoment(5))
	if err != nil || out.Status != "Complete" || out.Points == nil || *out.Points != 15000 ||
		out.PartialPoints != nil || out.Position != leaguerules.LB || out.Words != "1.5 points" ||
		len(out.Items) != 1 || out.Items[0].Words != "Tackle: 1 recorded; 1.5 points" {
		t.Fatalf("complete reading: %+v %v", out, err)
	}
	evidence.Missing = []string{"Play-by-play is needed for long plays and return touchdowns"}
	out, err = NativePlayerScoreReading("native.test", scope, b.records, evidence, seasonMoment(5))
	if err != nil || out.Status != "Incomplete" || out.Points != nil || out.PartialPoints == nil ||
		*out.PartialPoints != 15000 || !strings.Contains(out.Words, "counted so far; incomplete") {
		t.Fatalf("missing evidence became final: %+v %v", out, err)
	}
}

func TestNativePlayerScorePositionVocabularyAndUnresolvedPosition(t *testing.T) {
	for _, tc := range []struct {
		feed string
		want leaguerules.Position
	}{
		{"K", leaguerules.PK}, {"NT", leaguerules.DT}, {"ILB", leaguerules.LB},
		{"OLB", leaguerules.LB}, {"MLB", leaguerules.LB}, {"SS", leaguerules.S}, {"FS", leaguerules.S},
		{"DT", leaguerules.DT}, {"DL", ""}, {"DB", ""}, {"", ""},
	} {
		if got := scoreFeedPosition(tc.feed); got != tc.want {
			t.Fatalf("%s: %s, want %s", tc.feed, got, tc.want)
		}
	}
	b := newLedger(t, PresetLegacy)
	b.seed(2026, nil, seasonMoment(2))
	evidence := completeScoreEvidence()
	evidence.Held.Week.Position = "DL"
	out, err := NativePlayerScoreReading("native.test", source.RawNativeScope{Player: "0099", Season: 2026, Week: 1},
		b.records, evidence, seasonMoment(5))
	if err != nil || out.Status != "Needs a ruling" || out.Points != nil || out.PartialPoints != nil ||
		!strings.Contains(out.Words, "position") {
		t.Fatalf("ambiguous position guessed: %+v %v", out, err)
	}
}

func TestNativePlayerScoreDeltaOverflowAndUnusedStatWords(t *testing.T) {
	out := NativePlayerScore{Position: leaguerules.LB}
	line := fantasyscore.StatLine{Quantities: map[leaguerules.StatID]leaguerules.Decimal{"tackle": 10000}}
	if _, err := out.count(leaguerules.LegacyNFL().Scoring, line, math.MaxInt64); err == nil {
		t.Fatal("total plus adjustments overflowed")
	}
	item := nativeScoreItem(fantasyscore.LineItem{Stat: "sack", Label: "Sacks", Quantity: 10000, Unused: true})
	if item.Points != nil || item.Words != "Sacks: 1 recorded; not used for league scoring" {
		t.Fatalf("unused stat became zero: %+v", item)
	}
}

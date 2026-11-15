package commissioner

import (
	"math"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/fantasyscore"
	"github.com/secureprospective/TheWarRoom/internal/fantasyscore/source"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
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
		len(out.Items) != 1 || out.Items[0].Words != "Tackle: 1 recorded; 1.5 points" || out.Line["tackle"] != 10000 {
		t.Fatalf("complete reading: %+v %v", out, err)
	}
	evidence.Missing = []string{"Play-by-play is needed for long plays and return touchdowns"}
	out, err = NativePlayerScoreReading("native.test", scope, b.records, evidence, seasonMoment(5))
	if err != nil || out.Status != "Incomplete" || out.Points != nil || out.PartialPoints == nil ||
		*out.PartialPoints != 15000 || !strings.Contains(out.Words, "counted so far; incomplete") {
		t.Fatalf("missing evidence became final: %+v %v", out, err)
	}
}

func TestNativePlayerScoreRetainsCountedRevisionGaps(t *testing.T) {
	b := newLedger(t, PresetLegacy)
	b.seed(2026, nil, seasonMoment(2))
	evidence := completeScoreEvidence()
	old := *evidence.Held
	old.Missing = []string{"The counted revision lacks play-by-play"}
	evidence.Lines = []source.LineRevision{{Seq: 0, Digest: source.LineDigest(*old.Week), Held: old},
		{Seq: 1, Digest: source.LineDigest(*evidence.Held.Week), Held: *evidence.Held}}
	b.statCommand(StatApprove, RawStatChange{Item: itemOf(evidence.Lines[0])}, 4)
	out, err := NativePlayerScoreReading("native.test", source.RawNativeScope{Player: "0099", Season: 2026, Week: 1},
		b.records, evidence, seasonMoment(5))
	if err != nil || out.Status != "Incomplete" || out.Points != nil || out.PartialPoints == nil ||
		!strings.Contains(out.Words, old.Missing[0]) {
		t.Fatalf("new evidence hid the counted revision's gaps: %+v %v", out, err)
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

func TestNativeScoreUsesLeaguePlayerPositionBeforeFeedPosition(t *testing.T) {
	b := seededLedger(t)
	evidence := completeScoreEvidence()
	evidence.Held.Week.Position = "DB"
	evidence.ListedPosition = "S"
	scope := source.RawNativeScope{Player: "0099", Season: 2026, Week: 1}
	out, err := NativePlayerScoreReading("native.test", scope, b.records, evidence, seasonMoment(20))
	if err != nil || out.Position != leaguerules.S || out.Points == nil || !strings.Contains(out.PositionWhy, "player record") {
		t.Fatalf("record fallback: %+v %v", out, err)
	}
	evidence.Held.Week.Position = "LB"
	evidence.ListedPosition = "DE"
	out, err = NativePlayerScoreReading("native.test", scope, b.records, evidence, seasonMoment(20))
	if err != nil || out.Position != leaguerules.DE || out.Points == nil {
		t.Fatalf("league position replaced by feed: %+v %v", out, err)
	}
	want, _, err := fantasyscore.Score(leaguerules.LegacyNFL().Scoring, leaguerules.DE, evidence.Held.Week.Line)
	if err != nil || *out.Points != want.Value {
		t.Fatalf("score did not use DE rules: %+v, want %+v, %v", out, want, err)
	}
	evidence.Held.Week.Position = "DL"
	evidence.ListedPosition = "DL"
	out, err = NativePlayerScoreReading("native.test", scope, b.records, evidence, seasonMoment(20))
	if err != nil || out.Points != nil || out.Position != "" {
		t.Fatalf("ambiguous record guessed: %+v %v", out, err)
	}
	evidence.ListedPosition = "DE"
	b.add(RawCommand{League: "native.test", Command: PlayerPosition, Args: RawArgs{Score: &RawScoreChange{Player: "0099", Position: leaguerules.DT}}, Request: "position", Effective: "now", Visibility: "league"}, seasonMoment(10))
	out, err = NativePlayerScoreReading("native.test", scope, b.records, evidence, seasonMoment(20))
	if err != nil || out.Position != leaguerules.DT || !strings.Contains(out.PositionWhy, "commissioner") {
		t.Fatalf("manual position lost: %+v %v", out, err)
	}
}

func TestLeaguePositionGovernsLineupLegality(t *testing.T) {
	b := seededLedger(t)
	evidence := completeScoreEvidence()
	evidence.ListedPosition = "DE"
	score, err := NativePlayerScoreReading("native.test", source.RawNativeScope{Player: "0099", Season: 2026, Week: 1},
		b.records, evidence, seasonMoment(20))
	if err != nil || score.Points == nil {
		t.Fatalf("score: %+v %v", score, err)
	}
	rules := leaguerules.LegacyNFL().Lineup
	rules.Positions = leaguerules.Known(map[leaguerules.Position]leaguerules.Bounds{
		leaguerules.DE: {Min: leaguerules.Known(1), Max: leaguerules.Known(1)},
	})
	rules.Groups = leaguerules.Known(map[leaguerules.LineupGroup]leaguerules.Bounds{})
	out := FranchiseWeekScore{Starters: []StarterScore{{Player: "0099", Position: score.Position, Points: score.Points}}}
	_, err = out.judge(rules, *score.Points)
	if err != nil || out.Lineup.Status != nativecheck.LineupValid || out.Zeroed != "" {
		t.Fatalf("league DE judged as feed LB: %+v %v", out, err)
	}
	if evidence.Held.Week.Position != "LB" || evidence.ListedPosition != "DE" {
		t.Fatal("scoring mutated held position evidence")
	}
}

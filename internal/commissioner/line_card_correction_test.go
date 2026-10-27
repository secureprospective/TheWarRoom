package commissioner

import (
	"slices"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/fantasyscore/source"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

// The commissioner corrects a card's score through the same ledgered paths as a player's: a stat
// correction replaces a line component, a score change adds to the total, and both are listed.
func TestLineCardScoreTakesCommissionerCorrections(t *testing.T) {
	b := newLedger(t, PresetDefault32)
	seed, err := Default32SeedCommand(b.records, "native.test", "start", "Start", 1, seasonMoment(2))
	if err != nil {
		t.Fatal(err)
	}
	b.add(seed, seasonMoment(2))
	evidence := source.NativeEvidence{OLine: realLineGames(t)}
	scope := source.RawNativeScope{Player: pitCard, Season: 2026, Week: 1}
	read := func() NativePlayerScore {
		t.Helper()
		got, err := NativePlayerScoreReading("native.test", scope, b.records, evidence, seasonMoment(20))
		if err != nil || got.Points == nil {
			t.Fatalf("score: %v %+v", err, got)
		}
		return got
	}
	if got := read(); *got.Points != 13300 {
		t.Fatalf("held PIT week 1: %d, study 13300", *got.Points)
	}
	hits := leaguerules.Decimal(400000)
	correct := RawCommand{League: "native.test", Command: StatCorrect, Request: "hits", Effective: "now",
		Visibility: "league", Args: RawArgs{Stat: &RawStatChange{Player: pitCard, Season: 2026, Week: 1,
			Stat: "line_qb_hits", Value: &hits, Source: "Gamebook"}}}
	if err := CheckStatAppend(correct, b.records, StatFacts{}, seasonMoment(5)); err != nil {
		t.Fatalf("a line component is correctable on a card: %v", err)
	}
	other := correct
	other.Args = RawArgs{Stat: &RawStatChange{Player: "1000", Season: 2026, Week: 1, Stat: "line_qb_hits", Value: &hits, Source: "Gamebook"}}
	if err := CheckStatAppend(other, b.records, StatFacts{}, seasonMoment(5)); err == nil {
		t.Fatal("a line component is not a player's statistic")
	}
	hitsEntry := b.add(correct, seasonMoment(5))
	corrected := read()
	if *corrected.Points == 13300 || !slices.ContainsFunc(corrected.Changes, func(c ScoreChange) bool { return c.Entry == hitsEntry }) {
		t.Fatalf("the correction must change the score and be listed: %+v", corrected)
	}
	one := leaguerules.Decimal(10000)
	scoreEntry := b.add(RawCommand{League: "native.test", Command: ScorePlayer, Request: "plus", Effective: "now",
		Visibility: "league", Reason: "Ruling", Args: RawArgs{Score: &RawScoreChange{Player: pitCard, Season: 2026,
			Week: 1, Delta: &one}}}, seasonMoment(6))
	changed := read()
	if *changed.Points != *corrected.Points+one ||
		!slices.ContainsFunc(changed.Changes, func(c ScoreChange) bool { return c.Entry == scoreEntry }) {
		t.Fatalf("a score change adds to the card's total and is listed: %+v", changed)
	}
}

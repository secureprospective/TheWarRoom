package commissioner

import (
	"context"
	"slices"
	"strings"
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

// cardFeed names the card for a preview; a card has no held NFL weeks.
type cardFeed struct{}

func (cardFeed) Weeks(context.Context) ([]StatWeek, error) { return nil, nil }
func (cardFeed) Name(string) string                        { return "PIT O-Line" }

// A card correction reads like a player's: the preview's "before" is the held line game's value under the
// component's plain-English label, and entering the value already held is refused as a change of nothing.
func TestLineCardCorrectionReadsTheHeldLineGame(t *testing.T) {
	b := newLedger(t, PresetDefault32)
	seed, err := Default32SeedCommand(b.records, "native.test", "start", "Start", 1, seasonMoment(2))
	if err != nil {
		t.Fatal(err)
	}
	b.add(seed, seasonMoment(2))
	lines := realLineGames(t)
	held, ok := lineCardWeek("PIT", 1, lines.Games)
	if !ok {
		t.Fatal("PIT week 1 is held")
	}
	hits := held.Quantities["line_qb_hits"]
	correct := func(value leaguerules.Decimal) RawCommand {
		return RawCommand{League: "native.test", Command: StatCorrect, Request: "hits", Effective: "now",
			Visibility: "league", Args: RawArgs{Stat: &RawStatChange{Player: pitCard, Season: 2026, Week: 1,
				Stat: "line_qb_hits", Value: &value, Source: "Gamebook"}}}
	}
	facts := StatFacts{OLine: lines}
	if err := CheckStatAppend(correct(hits), b.records, facts, seasonMoment(5)); err == nil ||
		!strings.Contains(err.Error(), "already counts") {
		t.Fatalf("entering the held value changes nothing and must be refused: %v", err)
	}
	more := hits + 10000
	if err := CheckStatAppend(correct(more), b.records, facts, seasonMoment(5)); err != nil {
		t.Fatalf("a different value is a correction: %v", err)
	}
	preview, err := StatChangePreview(context.Background(), "native.test", b.records, nil, cardFeed{},
		StatWeek{Player: pitCard, Week: 1, Evidence: source.NativeEvidence{OLine: lines}}, correct(more), seasonMoment(5))
	if err != nil {
		t.Fatal(err)
	}
	want := "quarterback hits allowed " + quantityWords(hits, true) + " → " + scorePointsWords(more)
	if !strings.Contains(preview.Words, want) {
		t.Fatalf("preview %q, want %q", preview.Words, want)
	}
}

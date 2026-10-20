package history

import (
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

// A running clock must not make a reading resolve the season's rules again for every starter it scores.
func TestNativeReadingsResolveTheirRulesOncePerLedger(t *testing.T) {
	t.Parallel()
	f := bracketTest(t)
	f.do(t, commissioner.BracketBuild, rulesBuild())
	ticks := 0
	f.s.SetClock(func() time.Time {
		ticks++
		return t0().Add(time.Duration(ticks) * time.Millisecond)
	})
	records, err := f.s.nativeRecords(t.Context(), bracketLeague)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		name string
		read func(src *nativeSources) error
	}{
		{"standings", func(src *nativeSources) error {
			_, err := commissioner.NativeStandingsReading(t.Context(), bracketLeague, records, src, 2026, src.now)
			return err
		}},
		{"bracket", func(src *nativeSources) error {
			_, err := commissioner.NativeBracketReading(t.Context(), bracketLeague, records, src, 2026, src.now,
				commissioner.LineupFacts{})
			return err
		}},
	} {
		src := f.s.nativeSources(bracketLeague, 2026)
		if err := c.read(src); err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if got := src.rules.Resolved(); got != 1 {
			t.Errorf("%s: 32 franchises' starters were scored under %d reads of the rules, want 1", c.name, got)
		}
	}
}

// Each reading starts a new memo, so a rule recorded after one reading is in force for the next.
func TestNativeReadingsDoNotCarryRulesFromAnEarlierReading(t *testing.T) {
	t.Parallel()
	f := bracketTest(t)
	f.bump["franchise.05"] = map[int]leaguerules.Decimal{1: 10002}
	points := func() leaguerules.Decimal {
		results, err := f.s.NativeResults(t.Context(), bracketLeague, 2026, 1)
		if err != nil {
			t.Fatal(err)
		}
		for _, score := range results.Scores {
			if score.Franchise == "franchise.05" && score.Points != nil {
				return *score.Points
			}
		}
		t.Fatal("franchise.05 has no complete score in week 1")
		return 0
	}
	if got := points(); got != 15000 {
		t.Fatalf("with points rounded to the cent: %d, want 15000", got)
	}
	f.setRule(t, "scoring.roundStatPoints", "false")
	if got := points(); got != 15003 {
		t.Fatalf("after the rounding rule was switched off: %d, want 15003", got)
	}
}

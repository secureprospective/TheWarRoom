package mflsource

import (
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/ingestion/league"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

// MFL's draft kind sets the later rounds, the base fills the rest, and a setting the league does not use
// (keepers in a dynasty league) is not listed as waiting.
func TestSettingsRunAsTheLeagueRuns(t *testing.T) {
	t.Parallel()
	r, _, notes := Settings(league.RawConfig{KeeperType: "dynasty"}, leaguerules.LegacyNFL(), "SAME")
	if r.Draft.LaterPattern.Value != leaguerules.Linear || r.Draft.LaterPattern.State != leaguerules.IsKnown {
		t.Errorf("MFL's SAME draft must read as linear: %+v", r.Draft.LaterPattern)
	}
	if r.Contracts.MaximumYears.State != leaguerules.IsKnown {
		t.Errorf("an unheld setting was not filled from the base: %+v", r.Contracts.MaximumYears)
	}
	for _, n := range notes {
		if strings.Contains(n, "Keepers") || strings.Contains(n, "Later round draft pattern") {
			t.Errorf("listed as waiting: %q", n)
		}
	}
}

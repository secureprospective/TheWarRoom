package history

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
)

func TestRolloverArchivesDecidedFinalThroughHistoryLinkAndReopens(t *testing.T) {
	f := bracketTest(t)
	f.do(t, commissioner.BracketBuild, rulesBuild())
	if c := f.bracket(t).Champion; c == nil || c.Franchise != franchiseID(32) {
		t.Fatalf("final not decided: %+v", c)
	}
	f.run(t, f.link(t, "champion.2026", franchiseID(3)))
	prepareChampionRollover(t, f)
	for pass := range 2 {
		if pass == 1 {
			f.reopen(t)
		}
		raw := commissioner.RawCommand{League: bracketLeague, Command: commissioner.SeasonRollover,
			Args: commissioner.RawArgs{Rollover: &commissioner.RawRolloverArgs{To: 2027}}, Expected: f.rev(t),
			Request: fmt.Sprint("rollover.", f.rev(t)), Reason: "Close the decided season", Effective: "now", Visibility: "league"}
		receipt := f.run(t, raw)
		assertArchivedChampion(t, f, franchiseID(3))
		f.reopen(t)
		assertArchivedChampion(t, f, franchiseID(3))
		f.undo(t, receipt.Entry)
		f.reopen(t)
		assets, err := commissioner.NativeAssetsReading(bracketLeague, scoreRecords(t, f.s))
		if err != nil || slices.ContainsFunc(assets.Snapshot.Champions, func(c commissioner.NativeChampion) bool { return c.Season == 2026 }) {
			t.Fatal("rollover compensation left the champion archived", err)
		}
		if c := f.bracket(t).Champion; c == nil || c.Franchise != franchiseID(3) {
			t.Fatal("compensation lost decided champion")
		}
	}
}

func prepareChampionRollover(t *testing.T, f *bracketFixture) {
	t.Helper()
	for _, ruling := range []struct{ field, value string }{
		{"calendar.boundary", `"start_of_day"`}, {"calendar.tradeDeadline", `"2026-11-09T05:00:00Z"`},
	} {
		f.run(t, ruleCommand(ruling.field, ruling.value, "now", "Set the rollover dates", f.rev(t)))
	}
}

func assertArchivedChampion(t *testing.T, f *bracketFixture, franchise string) {
	t.Helper()
	assets, err := commissioner.NativeAssetsReading(bracketLeague, scoreRecords(t, f.s))
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, c := range assets.Snapshot.Champions {
		if c.Season == 2026 {
			found = true
			if c.Franchise != franchise {
				t.Fatalf("archived wrong champion: %+v", c)
			}
		}
	}
	if !found {
		t.Fatal("decided final was not archived")
	}
}

func TestRolloverUndecidedFinalSaysNoChampion(t *testing.T) {
	f := bracketTest(t)
	prepareChampionRollover(t, f)
	raw := commissioner.RawCommand{League: bracketLeague, Command: commissioner.SeasonRollover,
		Args: commissioner.RawArgs{Rollover: &commissioner.RawRolloverArgs{To: 2027, DryRun: true}}, Expected: f.rev(t),
		Request: "undecided-preview", Reason: "Preview the new year", Effective: "now", Visibility: "league"}
	receipt := f.run(t, raw)
	if !strings.Contains(strings.Join(receipt.Plan.Lines, " "), "No champion is decided for season 2026") {
		t.Fatal("undecided final disclosure missing", receipt.Plan.Lines)
	}
}

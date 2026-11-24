package history

import (
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/model"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

// Computer picks follow the league's draft.autopickRanking: a one-season league takes this season's best
// (now_ppg), a dynasty league the long-term best. Once the one-season startup draft closes there is no
// rookie draft to run or schedule.
func TestAutoPickRankingFollowsTheLeagueSetting(t *testing.T) {
	t.Parallel()
	facts := nativecheck.Facts{
		"1001": {Name: "Young QB", Position: leaguerules.QB, Team: "KC"},
		"1002": {Name: "Veteran QB", Position: leaguerules.QB, Team: "BUF"},
	}
	for _, c := range []struct{ preset, want string }{
		{commissioner.PresetOneSeason12, "1002"}, {commissioner.PresetDefault32, "1001"},
	} {
		f := newAutoPickFixture(t, "native.rank"+c.preset, c.preset, 2, 1, "",
			[]string{"franchise.01", "franchise.02"}, facts)
		run := NewModelRun{Refresh: true, Season: 2026, AsOf: f.c.now,
			Params: ParamSet{Params: map[string]float64{"a": 1}, Measures: []string{}}, Engine: "test", InputsHash: "in",
			Scores: []ModelScore{
				{MFLID: "1001", Position: "QB", Value: model.Value{DynastyPPG: 30, NowPPG: 10}},
				{MFLID: "1002", Position: "QB", Value: model.Value{DynastyPPG: 12, NowPPG: 25}},
			}}
		if _, _, err := f.s.WriteModelRun(t.Context(), run); err != nil {
			t.Fatal(err)
		}
		f.openStartup()
		if err := f.s.ProcessDueAutoPicks(t.Context(), f.league, f.c.now, facts, nil); err != nil {
			t.Fatal(err)
		}
		if got := f.draftReading("franchise.01").Draft.Picks[0].Player; got != c.want {
			t.Fatalf("%s: first computer pick %s, want %s", c.preset, got, c.want)
		}
		if c.preset != commissioner.PresetOneSeason12 {
			continue
		}
		if err := f.s.CloseFinishedDrafts(t.Context(), f.league, facts); err != nil {
			t.Fatal(err)
		}
		records, err := f.s.nativeRecords(t.Context(), f.league)
		if err != nil {
			t.Fatal(err)
		}
		jobs, err := commissioner.JobsReading(f.league, records, f.c.now.Add(time.Hour))
		if err != nil {
			t.Fatal(err)
		}
		for _, job := range jobs.Jobs {
			if job.ID == "rookie_draft" && !strings.HasPrefix(job.Words, "No rookie draft is left") {
				t.Fatalf("one-season rookie draft: %+v", job)
			}
		}
	}
}

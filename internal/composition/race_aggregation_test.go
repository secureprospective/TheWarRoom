package composition

import (
	"strconv"
	"testing"
	"time"

	raceapp "github.com/secureprospective/TheWarRoom/internal/app/race"
	"github.com/secureprospective/TheWarRoom/internal/app/season"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/league"
	"github.com/secureprospective/TheWarRoom/internal/m2service"
	"github.com/secureprospective/TheWarRoom/internal/model"
	"github.com/secureprospective/TheWarRoom/internal/normalize"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
	"github.com/secureprospective/TheWarRoom/internal/store/params"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

func TestRaceAggregationModesReachCalculationAndHeldRead(t *testing.T) {
	a, transport, _, _ := isolatedSeasonTestApp(t)
	a.core.Lookup, a.core.HasLookup, a.core.LookupAt = normalize.Lookup{}, true, time.Now()
	cfg := realRace(t, "2026").config
	cfg.Starters = league.Starters{Count: "2", IDPStarters: "0", Positions: []league.PositionLimit{{Name: "QB", Limit: "1"}, {Name: "WR", Limit: "1"}}}
	if _, err := a.core.Rulebook.Sync(t.Context(), cfg); err != nil {
		t.Fatal(err)
	}
	if _, err := a.core.League.Replace(t.Context(), state.MirrorSnapshot{Season: 2026, Players: []state.MirrorPlayer{
		{MFLID: "0042", FranchiseID: "0001", RosterStatus: domain.RosterActive},
		{MFLID: "0043", FranchiseID: "0001", RosterStatus: domain.RosterActive},
		{MFLID: "0044", FranchiseID: "0001", RosterStatus: domain.RosterActive},
	}}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := a.core.History.WriteModelRun(t.Context(), history.NewModelRun{Season: 2026, AsOf: time.Now(), Engine: "fixture", InputsHash: "agg-modes", Params: history.ParamSet{Params: params.DefaultSet().Model()}, Scores: []history.ModelScore{
		{MFLID: "0042", Position: "QB", Value: model.Value{NowPPG: 100, DynastyPPG: 100}},
		{MFLID: "0043", Position: "QB", Value: model.Value{NowPPG: 90, DynastyPPG: 90}},
		{MFLID: "0044", Position: "WR", Value: model.Value{NowPPG: 10, DynastyPPG: 10}},
	}}); err != nil {
		t.Fatal(err)
	}
	a.refreshRace(t.Context())
	requests := len(transport.requests)
	api := raceapp.New(raceHost{a.core}, a.raceHeld)
	for _, tc := range []struct {
		mode  string
		value float64
	}{{m2service.AggSum, 200}, {m2service.AggLineup, 110}, {m2service.AggTopN, 190}, {"", 110}} {
		t.Run(tc.mode, func(t *testing.T) {
			held := a.raceHeld.Snapshot()
			for _, view := range []string{season.ViewSeason, season.ViewFranchise} {
				result := held.Power[season.PowerKey(view, tc.mode)]
				if !result.OK || result.AggMode != m2service.ResolveAggMode(tc.mode, m2service.AggLineup) {
					t.Fatalf("%s: ok=%v mode=%q error=%s", view, result.OK, result.AggMode, result.Error)
				}
				found := false
				for _, row := range result.Rows {
					if row.FranchiseID == "0001" {
						found = true
						if row.RosterValue != tc.value {
							t.Fatalf("%s roster = %v, want %v", view, row.RosterValue, tc.value)
						}
					}
				}
				if !found {
					t.Fatal("missing held franchise", view)
				}
			}
			board, err := api.TargetRace(0, "", "0001", tc.mode)
			if err != nil {
				t.Fatal(err)
			}
			for id, column := range map[string]int{"M-008": 4, "C-02": 3} {
				found := false
				for _, report := range board.Board.Reports {
					if report.ID != id {
						continue
					}
					for _, row := range report.Rows {
						if !row.Mine {
							continue
						}
						found = true
						if row.Cells[column] != strconv.FormatFloat(tc.value, 'f', 1, 64) {
							t.Fatalf("%s roster = %s, want %v", id, row.Cells[column], tc.value)
						}
					}
				}
				if !found {
					t.Fatal("missing report franchise", id)
				}
			}
			if len(transport.requests) != requests {
				t.Fatal("mode read fetched instead of reading held data")
			}
		})
	}
}

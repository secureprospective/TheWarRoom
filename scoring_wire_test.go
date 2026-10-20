package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/ingestion/leagueschedule"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/leaguestandings"
	"github.com/secureprospective/TheWarRoom/internal/model"
	"github.com/secureprospective/TheWarRoom/internal/normalize"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
	"github.com/secureprospective/TheWarRoom/internal/store/params"
)

// The retired M1/M2 bindings have no live methods; Race consumes this root entry point.
func TestScoringWireGolden(t *testing.T) {
	a := targetTestApp(t)
	a.lookup, a.hasLookup = normalize.Lookup{}, true
	schedule := func(context.Context) []leagueschedule.RawScheduleWeek { return nil }
	standings := []leaguestandings.RawStanding{{FranchiseID: "0001", H2HW: "2", H2HL: "1"}}
	for _, populated := range []bool{false, true} {
		if populated {
			_, _, err := a.history.WriteModelRun(a.ctx, history.NewModelRun{Season: a.season, AsOf: time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC), Engine: "fixture", InputsHash: "fixture", Params: history.ParamSet{Params: params.DefaultSet().Model()}, Scores: []history.ModelScore{{MFLID: "0042", Position: "WR", Value: model.Value{NowPPG: 12, DynastyPPG: 15}}}})
			if err != nil {
				t.Fatal(err)
			}
		}
		for _, view := range []string{ViewSeason, ViewFranchise} {
			name := view
			if populated {
				name += "-populated"
			}
			result := a.powerRankings(a.ctx, standings, Freshness{}, schedule, .4, true, defaultAgg(view), view)
			if !result.OK {
				t.Fatal(result.Error)
			}
			body := sourcesWire(t, result)
			path := filepath.Join("testdata", "wire", "scoring", name+".json")
			if os.Getenv("TWR_WRITE_WIRE") == "1" {
				if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, body, 0o600); err != nil {
					t.Fatal(err)
				}
			}
			held, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if string(held) != string(body) {
				t.Errorf("%s scoring wire changed", name)
			}
		}
	}
}

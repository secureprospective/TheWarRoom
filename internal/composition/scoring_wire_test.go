package composition

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/season"

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
	a.core.Lookup, a.core.HasLookup = normalize.Lookup{}, true
	schedule := func(context.Context) []leagueschedule.RawScheduleWeek { return nil }
	standings := []leaguestandings.RawStanding{{FranchiseID: "0001", H2HW: "2", H2HL: "1"}}
	for _, populated := range []bool{false, true} {
		if populated {
			_, _, err := a.core.History.WriteModelRun(a.core.Ctx, history.NewModelRun{Season: a.core.Season, AsOf: time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC), Engine: "fixture", InputsHash: "fixture", Params: history.ParamSet{Params: params.DefaultSet().Model()}, Scores: []history.ModelScore{{MFLID: "0042", Position: "WR", Value: model.Value{NowPPG: 12, DynastyPPG: 15}}}})
			if err != nil {
				t.Fatal(err)
			}
		}
		for _, view := range []string{ViewSeason, ViewFranchise} {
			name := view
			if populated {
				name += "-populated"
			}
			result := a.powerRankings(a.core.Ctx, standings, Freshness{}, schedule, .4, true, defaultAgg(view), view)
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

func TestScoringWireTwins(t *testing.T) {
	t.Parallel()
	row := season.PowerRow{Rank: 1, FranchiseID: "0001", Name: "Name", RankDelta: 2, DeltaOK: true,
		PowerScore: .8, RosterZ: .2, MFLPerfZ: .3, AgeZ: .4, RosterValue: 12, Results: .6, Age: 25,
		CapRoom: 10, DeadCap: 2, HasCap: true, Luck: .1, HasLuck: true, ProjW: 8, ProjL: 5, HasProj: true,
		H2HW: 3, H2HL: 2, H2HT: 1, AllPlayW: 9, AllPlayL: 8, AllPlayT: 7, PF: 100, PA: 90, PP: 120, Pwr: 4, AltPwr: 5}
	b := season.PowerRankingsResult{OK: true, Error: "error", Label: "label", Season: 2026, View: ViewSeason,
		Weight: .5, WeightAuto: true, AgeWeight: .2, ModelRunID: 2, PreviousRunID: 1, AggMode: "sum", StarterN: 21,
		Freshness: Freshness{State: FreshStale, FetchedAt: "at", Note: "note"}, Performance: "points for",
		WeeksScored: 3, SeasonWeeks: 13, Rows: []season.PowerRow{row}}
	for _, b := range []season.PowerRankingsResult{b, {}, {Rows: []season.PowerRow{}}} {
		if string(sourcesWire(t, b)) != string(sourcesWire(t, wirePower(b))) {
			t.Fatal("scoring wire twin lost a field or slice shape")
		}
	}
}

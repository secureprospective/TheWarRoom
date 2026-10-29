package composition

import (
	"context"
	"testing"
	"time"

	scoringapp "github.com/secureprospective/TheWarRoom/internal/app/scoring"
	"github.com/secureprospective/TheWarRoom/internal/app/season"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/draftreports"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/leagueschedule"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/leaguestandings"
	"github.com/secureprospective/TheWarRoom/internal/m2service"
	"github.com/secureprospective/TheWarRoom/internal/measures"
	"github.com/secureprospective/TheWarRoom/internal/model"
	"github.com/secureprospective/TheWarRoom/internal/normalize"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
	"github.com/secureprospective/TheWarRoom/internal/store/params"
)

type pickListing struct{}

func (pickListing) IDs() []string { return []string{"0042"} }
func (pickListing) Facts(string) (normalize.PlayerFacts, bool) {
	return normalize.PlayerFacts{Name: "Rookie", Position: domain.PosWR, HasDraftYear: true, DraftYear: 2026, DraftRound: 1, DraftPick: 5}, true
}

func TestPowerPickSettingRebuildsHeldFranchiseOnlyAndNeverPlayerValues(t *testing.T) {
	a := targetTestApp(t)
	a.core.Params = params.New(a.core.Pools)
	if err := a.core.Params.Initialize(t.Context()); err != nil {
		t.Fatal(err)
	}
	a.core.Lookup, a.core.HasLookup = normalize.Lookup{}, true
	if _, err := a.core.History.Ingest(t.Context(), measures.Batch{Source: "nflverse", Facts: []measures.Fact{{IDType: "mfl", ID: "0042", Field: "draft_picks.pick", Raw: "5"}}}); err != nil {
		t.Fatal(err)
	}
	if _, err := a.recordStore().Write(t.Context(), pickListing{}, time.Now()); err != nil {
		t.Fatal(err)
	}
	run, _, err := a.core.History.WriteModelRun(t.Context(), history.NewModelRun{Season: 2026, AsOf: time.Now(), Engine: "fixture", InputsHash: "fixture", Params: history.ParamSet{Params: params.DefaultSet().Model()}, Scores: []history.ModelScore{{MFLID: "0042", Position: "WR", Value: model.Value{NowPPG: 8, DynastyPPG: 10}}}})
	if err != nil {
		t.Fatal(err)
	}
	a.core.SeasonFuturePicks.Value = []draftreports.FuturePick{{Year: 2027, Round: 1, Original: "0001", Owner: "0001"}}
	schedule := func(context.Context) []leagueschedule.RawScheduleWeek { return nil }
	standings := []leaguestandings.RawStanding{{FranchiseID: "0001"}}
	build := func(view string) season.PowerRankingsResult {
		return racePowerResult(a.powerRankings(t.Context(), standings, Freshness{}, schedule, 0, true, scoringapp.DefaultAgg(view), view))
	}
	franchise, now := build(ViewFranchise), build(ViewSeason)
	if !franchise.OK || franchise.PickWeight != .6 || !franchise.Rows[0].HasPicks || franchise.Rows[0].PickValue <= 0 || now.Rows[0].HasPicks || now.PickWeight != 0 {
		t.Fatal(franchise, now)
	}
	power := map[string]season.PowerRankingsResult{ViewFranchise: franchise, ViewSeason: now}
	for _, mode := range []string{m2service.AggSum, m2service.AggTopN} {
		power[season.PowerKey(ViewFranchise, mode)] = a.powerRankings(t.Context(), standings, Freshness{}, schedule, 0, true, mode, ViewFranchise)
	}
	a.raceHeld.PublishStandings(season.Feed[[]leaguestandings.RawStanding]{Value: standings}, power, true)
	if result := (controlHost{app: a}).SetParam(params.KeyPowerPickWeight, "", 0); !result.OK {
		t.Fatal(result)
	}
	held := a.raceHeld.Snapshot()
	for _, mode := range []string{m2service.AggLineup, m2service.AggSum, m2service.AggTopN} {
		if got := held.Power[season.PowerKey(ViewFranchise, mode)]; !got.OK || got.PickWeight != 0 || got.Rows[0].RosterValue != 10 {
			t.Fatal("mode kept stale pick settings", mode, got)
		}
	}
	zero := held.Power[ViewFranchise]
	if zero.PickWeight != 0 || zero.Rows[0].RosterValue != 10 || held.Power[ViewSeason].Rows[0] != now.Rows[0] {
		t.Fatal("setting did not change only franchise", held)
	}
	if result := (controlHost{app: a}).ResetParam(params.KeyPowerPickWeight, ""); !result.OK {
		t.Fatal(result)
	}
	if restored := a.raceHeld.Snapshot().Power[ViewFranchise]; restored.PickWeight != .6 || restored.Rows[0].RosterValue != franchise.Rows[0].RosterValue {
		t.Fatal(restored)
	}
	a.core.SeasonFuturePicks.Value = nil
	missing := build(ViewFranchise)
	if !missing.OK || missing.Rows[0].HasPicks || missing.Rows[0].RosterValue != 10 {
		t.Fatal("missing inventory failed or changed board", missing)
	}
	scores, err := a.core.History.ModelScores(t.Context(), run.ID)
	if err != nil || len(scores) != 1 || scores[0].DynastyPPG != 10 || scores[0].NowPPG != 8 {
		t.Fatal("pick settings moved player values", scores, err)
	}
}

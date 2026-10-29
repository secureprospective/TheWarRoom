// Package scoring reads model inputs and builds the power boards without owning app state.
package scoring

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/normalize"

	"github.com/secureprospective/TheWarRoom/internal/app/season"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/leagueschedule"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/leaguestandings"
	"github.com/secureprospective/TheWarRoom/internal/m2service"
	"github.com/secureprospective/TheWarRoom/internal/powerrankings"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

// History supplies the model runs and scores used by a board.
type History interface {
	LatestRun(context.Context, int, history.RunKind) (history.Run, bool, error)
	PreviousRun(context.Context, history.Run) (history.Run, bool, error)
	ModelScores(context.Context, int64) ([]history.ModelScore, error)
}

// Source supplies live reads in their original order; it owns no held state.
type Source struct {
	Picks       func(context.Context, history.Run) m2service.PickCapital
	History     History
	Season      int
	State       func() state.Reader
	Rules       m2service.FranchiseSource
	Ages        func(context.Context) func(string) float64
	SeasonWeeks func() int
	DeadCap     func() map[string]domain.Money
}

func (a Source) powerFailure(err error, view string, weight float64, auto bool, mode string) season.PowerRankingsResult {
	return season.PowerRankingsResult{Error: err.Error(), Season: a.Season, View: view, Weight: weight,
		WeightAuto: auto, AggMode: mode, Freshness: domain.Freshness{State: domain.FreshFail, Note: err.Error()}}
}

// Build builds the M2 board from standings already read; schedule is read only when a
// model run exists to rank.
func (a Source) Build(
	ctx context.Context, standings []leaguestandings.RawStanding, fresh domain.Freshness,
	schedule func(context.Context) []leagueschedule.RawScheduleWeek, weight float64, autoWeight bool, mode, view string,
) season.PowerRankingsResult {
	fail := func(err error) season.PowerRankingsResult { return a.powerFailure(err, view, weight, autoWeight, mode) }
	runs, err := a.modelRuns(ctx)
	if err != nil {
		return fail(err)
	}
	played := weeksScored(standings)
	wt := boardWeights(view, weight, autoWeight, played)
	res := season.PowerRankingsResult{OK: true, Season: a.Season, View: view, Weight: wt.Roster,
		WeightAuto: autoWeight && view == season.ViewSeason, AgeWeight: wt.Age, AggMode: mode, Freshness: fresh,
		SeasonWeeks: a.SeasonWeeks(), WeeksScored: played, Rows: []season.PowerRow{}}
	if len(runs) == 0 {
		return res
	}
	svc, err := m2service.New(a.State(), a.Rules)
	if err != nil {
		return fail(err)
	}
	ages := a.Ages(ctx)
	boards := make([]m2service.Board, len(runs))
	var now []m2service.PlayerValue
	for i, run := range runs {
		values, err := a.playerValues(ctx, run.ID, ages)
		if err != nil {
			return fail(err)
		}
		if i == 0 {
			now = values[season.ViewSeason]
		}
		picks := m2service.PickCapital{}
		if view == season.ViewFranchise && a.Picks != nil {
			picks = a.Picks(ctx, run)
		}
		if boards[i], err = svc.BuildBoard(standings, values[view], wt, mode, picks); err != nil {
			return fail(err)
		}
	}
	outlooks, err := svc.Outlooks(m2service.OutlookInputs{Standings: standings, Now: now,
		Schedule: schedule(ctx), SeasonWeeks: res.SeasonWeeks, DeadCap: a.DeadCap()})
	if err != nil {
		return fail(err)
	}
	fillBoard(&res, runs, boards, outlooks)
	return res
}

func DefaultAgg(view string) string { return season.DefaultAgg(view) }

// boardWeights are the view's blend weights. The franchise ranks on the roster and its age
// alone; this season blends the roster with results at the caller's weight, or with auto the
// weight the weeks played give.
func boardWeights(view string, weight float64, auto bool, played int) powerrankings.Weights {
	switch {
	case view == season.ViewFranchise:
		return powerrankings.Weights{Roster: 1, Age: powerrankings.FranchiseAgeWeight}
	case auto:
		return powerrankings.Weights{Roster: powerrankings.AutoRosterWeight(played)}
	}
	return powerrankings.Weights{Roster: weight}
}

// fillBoard puts the latest board in res, each rank's move against the previous one and each
// franchise's context columns.
func fillBoard(res *season.PowerRankingsResult, runs []history.Run, boards []m2service.Board, outlooks map[string]m2service.Outlook) {
	res.Label = fmt.Sprintf("Roster value: model run #%d, league points per game (%s)", runs[0].ID, viewMeasure(res.View))
	res.ModelRunID, res.Weight, res.AggMode, res.StarterN = runs[0].ID, boards[0].Weight, boards[0].Mode, boards[0].StarterN
	res.Performance, res.PickWeight = boards[0].Performance, boards[0].PickWeight
	was := map[string]int{}
	if len(boards) > 1 {
		res.PreviousRunID = runs[1].ID
		for _, r := range boards[1].Rows {
			was[r.FranchiseID] = r.Rank
		}
	}
	res.Rows = powerRows(boards[0].Rows, was, outlooks)
}

func viewMeasure(view string) string {
	if view == season.ViewFranchise {
		return "dynasty"
	}
	return "on-field-now"
}

// modelRuns returns the season's latest model run and the one before it, newest first; none
// before the first engine run.
func (a Source) modelRuns(ctx context.Context) ([]history.Run, error) {
	latest, ok, err := a.History.LatestRun(ctx, a.Season, history.RunModel)
	if err != nil || !ok {
		return nil, err //nolint:wrapcheck // the history store's errors are already labelled
	}
	runs := []history.Run{latest}
	if prev, ok, err := a.History.PreviousRun(ctx, latest); err != nil {
		return nil, err //nolint:wrapcheck // the history store's errors are already labelled
	} else if ok {
		runs = append(runs, prev)
	}
	return runs, nil
}

// playerValues is every player's value from a model run in each view, with the position he was
// valued at and his age.
func (a Source) playerValues(ctx context.Context, runID int64, age func(string) float64) (map[string][]m2service.PlayerValue, error) {
	scores, err := a.History.ModelScores(ctx, runID)
	if err != nil {
		return nil, fmt.Errorf("power rankings: %w", err)
	}
	now := make([]m2service.PlayerValue, len(scores))
	dyn := make([]m2service.PlayerValue, len(scores))
	for i, s := range scores {
		now[i] = m2service.PlayerValue{MFLID: s.MFLID, Position: s.Position, InputStatus: s.InputStatus, Value: s.NowPPG, Age: age(s.MFLID)}
		dyn[i] = now[i]
		dyn[i].Value = s.DynastyPPG
	}
	return map[string][]m2service.PlayerValue{season.ViewSeason: now, season.ViewFranchise: dyn}, nil
}

// powerRows formats the board, with each rank's move from was (the previous board's ranks) and
// each franchise's context columns.
func powerRows(rows []m2service.Row, was map[string]int, outlooks map[string]m2service.Outlook) []season.PowerRow {
	out := make([]season.PowerRow, 0, len(rows))
	for _, r := range rows {
		o := outlooks[r.FranchiseID]
		row := season.PowerRow{
			ReducedInputs: r.ReducedInputs, Unavailable: r.Unavailable, ValueReason: r.Reason,
			Rank: r.Rank, FranchiseID: r.FranchiseID, Name: r.Name,
			PowerScore: r.PowerScore, RosterZ: r.RosterZ, MFLPerfZ: r.MFLPerfZ, AgeZ: r.AgeZ,
			RosterValue: r.RosterValue, Results: r.Results, PickValue: r.PickValue, HasPicks: r.HasPicks,
			H2HW: r.H2HW, H2HL: r.H2HL, H2HT: r.H2HT,
			AllPlayW: r.AllPlayW, AllPlayL: r.AllPlayL, AllPlayT: r.AllPlayT,
			PF: r.PF, PA: r.PA, PP: r.PP, Pwr: r.Pwr, AltPwr: r.AltPwr,
			CapRoom: o.CapRoom, DeadCap: o.DeadCap, HasCap: o.HasCap, Luck: o.Luck, HasLuck: o.HasLuck,
			ProjW: o.ProjW, ProjL: o.ProjL, HasProj: o.HasProj,
		}
		if !math.IsNaN(r.Age) { // JSON has no NaN; 0 reads as unknown
			row.Age = r.Age
		}
		if prev, ok := was[r.FranchiseID]; ok {
			row.RankDelta, row.DeltaOK = prev-r.Rank, true
		}
		out = append(out, row)
	}
	return out
}

// weeksScored is how many weeks MFL's standings hold: a franchise plays one head-to-head game a
// week. It reads the standings themselves, so the phase shows before the first model run. A blank
// or unreadable count reads as 0; it affects the label only.
func weeksScored(standings []leaguestandings.RawStanding) int {
	most := 0
	for _, st := range standings {
		games := 0
		for _, n := range []string{st.H2HW, st.H2HL, st.H2HT} {
			v, _ := strconv.Atoi(strings.TrimSpace(n))
			games += v
		}
		most = max(most, games)
	}
	return most
}

// RemainingSchedule omits the projected-record column when schedule evidence is unavailable.
func RemainingSchedule(ctx context.Context, read func(context.Context) ([]leagueschedule.RawScheduleWeek, domain.Freshness, error)) []leagueschedule.RawScheduleWeek {
	weeks, _, err := read(ctx)
	if err != nil {
		return nil
	}
	return weeks
}

func PlayerAges(lk normalize.Lookup, err error, today time.Time) func(string) float64 {
	return func(mflID string) float64 {
		if err != nil {
			return math.NaN()
		}
		f, ok := lk.Facts(mflID)
		if !ok || !f.HasBirthdate {
			return math.NaN()
		}
		return today.Sub(time.Unix(f.Birthdate, 0)).Hours() / (24 * 365.2425)
	}
}

func SeasonWeeks(last string) int {
	n, err := strconv.Atoi(last)
	if err != nil {
		return 0
	}
	return n
}

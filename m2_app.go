package main

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/ingestion/leagueschedule"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/leaguestandings"
	"github.com/secureprospective/TheWarRoom/internal/m2service"
	"github.com/secureprospective/TheWarRoom/internal/powerrankings"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

// m2Timeout bounds GetPowerRankings: one MFL standings fetch plus a read of two model runs.
const m2Timeout = 120 * time.Second

// PowerRow is one franchise's power-ranking row: the blended score, its two components, and
// MFL's report columns from the same standings call. Display values only. RankDelta is the move
// since the board built from the previous model run; DeltaOK is false when there is none.
type PowerRow struct {
	Rank        int    `json:"rank"`
	FranchiseID string `json:"franchiseID"`
	Name        string `json:"name"`
	RankDelta   int    `json:"rankDelta"`
	DeltaOK     bool   `json:"deltaOK"`

	PowerScore float64 `json:"powerScore"` // weighted z-blend, scaled to [0,1]
	RosterZ    float64 `json:"rosterZ"`    // 0 = league average
	MFLPerfZ   float64 `json:"mflPerfZ"`
	AgeZ       float64 `json:"ageZ"`

	RosterValue float64 `json:"rosterValue"`
	Results     float64 `json:"results"` // what the blend read, in [0,1]; Performance names it
	Age         float64 `json:"age"`     // the counted players' value-weighted age; 0 when none is known

	// Context columns, display only (m2service.Outlook); each Has* says the column was read.
	CapRoom float64 `json:"capRoom"`
	DeadCap float64 `json:"deadCap"`
	HasCap  bool    `json:"hasCap"`
	Luck    float64 `json:"luck"`
	HasLuck bool    `json:"hasLuck"`
	ProjW   float64 `json:"projW"`
	ProjL   float64 `json:"projL"`
	HasProj bool    `json:"hasProj"`

	// MFL report columns, from leagueStandings.
	H2HW     int     `json:"h2hW"`
	H2HL     int     `json:"h2hL"`
	H2HT     int     `json:"h2hT"`
	AllPlayW int     `json:"allPlayW"`
	AllPlayL int     `json:"allPlayL"`
	AllPlayT int     `json:"allPlayT"`
	PF       float64 `json:"pf"`
	PA       float64 `json:"pa"`
	PP       float64 `json:"pp"`
	Pwr      float64 `json:"pwr"`
	AltPwr   float64 `json:"altPwr"`
}

// PowerRankingsResult echoes the view, weights, aggregation mode and N actually applied, so the
// controls and the rows never disagree. WeightAuto is true when the roster weight came from the
// weeks played rather than the slider. Zero rows means the league has no model run yet.
// WeeksScored is how many weeks the MFL standings hold, read from the head-to-head records, and
// SeasonWeeks the league's last regular-season week: the standings are final when they meet. Phase is separate from
// Freshness: an offseason board is fresh data about a finished season. An empty Phase means the
// phase read failed, which affects the label only.
type PowerRankingsResult struct {
	OK         bool    `json:"ok"`
	Error      string  `json:"error"`
	Label      string  `json:"label"`
	Season     int     `json:"season"`
	View       string  `json:"view"`
	Weight     float64 `json:"weight"`
	WeightAuto bool    `json:"weightAuto"`
	AgeWeight  float64 `json:"ageWeight"`
	// ModelRunID is the model run ranked; PreviousRunID the one the deltas compare with (0: none).
	ModelRunID    int64      `json:"modelRunID"`
	PreviousRunID int64      `json:"previousRunID"`
	AggMode       string     `json:"aggMode"`
	StarterN      int        `json:"starterN"`
	Freshness     Freshness  `json:"freshness"`
	Performance   string     `json:"performance"` // which result the blend read: all-play, points for, none
	WeeksScored   int        `json:"weeksScored"`
	SeasonWeeks   int        `json:"seasonWeeks"`
	Rows          []PowerRow `json:"rows"`
}

// The two M2 views (plan Stage 8): this season ranks on-field-now blended with the standings;
// the franchise ranks dynasty and the roster's age, because this season's record says little
// about the next five. Each view counts the roster its own way by default: this season the
// lineup the rules allow, the franchise the whole roster, whose young bench grows into starters
// (docs/modules/M2_Power_Ranking_Factors.md).
const (
	ViewSeason    = "season"
	ViewFranchise = "franchise"
)

// GetPowerRankings builds the M2 board: each franchise's roster value in the view, from the
// season's latest model run, blended (this season only) with the MFL standings, with team names
// and the context columns joined and each rank's move since the previous model run. This season's
// roster weight is the caller's, or with autoWeight the one the weeks played give. A failed
// standings fetch falls back to the last good cache and is labelled stale; only a failure with
// nothing cached is fatal.
func (a *App) GetPowerRankings(weight float64, autoWeight bool, aggMode, view string) PowerRankingsResult {
	if view != ViewFranchise {
		view = ViewSeason
	}
	mode := m2service.ResolveAggMode(aggMode, defaultAgg(view))
	fail := func(err error) PowerRankingsResult {
		return PowerRankingsResult{Error: err.Error(), Season: a.season, View: view, Weight: weight,
			WeightAuto: autoWeight, AggMode: mode, Freshness: Freshness{State: FreshFail, Note: err.Error()}}
	}
	if err := a.ready(); err != nil {
		return fail(err)
	}
	ctx, cancel := context.WithTimeout(a.ctx, m2Timeout)
	defer cancel()

	runs, err := a.modelRuns(ctx)
	if err != nil {
		return fail(err)
	}
	standings, fresh, err := a.standingsOrArchive(ctx)
	if err != nil {
		return fail(fmt.Errorf("power rankings: %w", err))
	}
	svc, err := m2service.New(a.league.Reader(), a.rulebook)
	if err != nil {
		return fail(err)
	}
	played := weeksScored(standings)
	wt := boardWeights(view, weight, autoWeight, played)
	ages := a.ages(ctx)
	boards := make([]m2service.Board, len(runs))
	var now []m2service.PlayerValue
	for i, run := range runs {
		values, err := a.playerValues(ctx, run.ID, ages)
		if err != nil {
			return fail(err)
		}
		if i == 0 {
			now = values[ViewSeason]
		}
		if boards[i], err = svc.BuildBoard(standings, values[view], wt, mode); err != nil {
			return fail(err)
		}
	}
	res := PowerRankingsResult{OK: true, Season: a.season, View: view, Weight: wt.Roster,
		WeightAuto: autoWeight && view == ViewSeason, AgeWeight: wt.Age, AggMode: mode, Freshness: fresh,
		SeasonWeeks: a.seasonWeeks(), WeeksScored: played, Rows: []PowerRow{}}
	if len(boards) == 0 {
		return res
	}
	outlooks, err := svc.Outlooks(m2service.OutlookInputs{Standings: standings, Now: now,
		Schedule: a.remainingSchedule(ctx), SeasonWeeks: res.SeasonWeeks, DeadCap: a.league.DeadCap()})
	if err != nil {
		return fail(err)
	}
	fillBoard(&res, runs, boards, outlooks)
	return res
}

// defaultAgg is how a view counts the roster unless the caller chose: this season the lineup the
// rules allow, the franchise the whole roster.
func defaultAgg(view string) string {
	if view == ViewFranchise {
		return m2service.AggSum
	}
	return m2service.AggLineup
}

// boardWeights are the view's blend weights. The franchise ranks on the roster and its age
// alone; this season blends the roster with results at the caller's weight, or with auto the
// weight the weeks played give.
func boardWeights(view string, weight float64, auto bool, played int) powerrankings.Weights {
	switch {
	case view == ViewFranchise:
		return powerrankings.Weights{Roster: 1, Age: powerrankings.FranchiseAgeWeight}
	case auto:
		return powerrankings.Weights{Roster: powerrankings.AutoRosterWeight(played)}
	}
	return powerrankings.Weights{Roster: weight}
}

// remainingSchedule is the league schedule for the projected record, nil when it cannot be read:
// a missing column must not stop the board.
func (a *App) remainingSchedule(ctx context.Context) []leagueschedule.RawScheduleWeek {
	weeks, _, err := a.leagueScheduleOrArchive(ctx)
	if err != nil {
		return nil
	}
	return weeks
}

// ages reads each player's age today from the players directory. When the directory cannot be
// read every age is unknown (NaN), and the board counts none.
func (a *App) ages(ctx context.Context) func(mflID string) float64 {
	lk, err := a.directory(ctx)
	today := time.Now()
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

// fillBoard puts the latest board in res, each rank's move against the previous one and each
// franchise's context columns.
func fillBoard(res *PowerRankingsResult, runs []history.Run, boards []m2service.Board, outlooks map[string]m2service.Outlook) {
	res.Label = fmt.Sprintf("Roster value: model run #%d, league points per game (%s)", runs[0].ID, viewMeasure(res.View))
	res.ModelRunID, res.Weight, res.AggMode, res.StarterN = runs[0].ID, boards[0].Weight, boards[0].Mode, boards[0].StarterN
	res.Performance = boards[0].Performance
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
	if view == ViewFranchise {
		return "dynasty"
	}
	return "on-field-now"
}

// modelRuns returns the season's latest model run and the one before it, newest first; none
// before the first Score League.
func (a *App) modelRuns(ctx context.Context) ([]history.Run, error) {
	latest, ok, err := a.history.LatestRun(ctx, a.season, history.RunModel)
	if err != nil || !ok {
		return nil, err //nolint:wrapcheck // the history store's errors are already labelled
	}
	runs := []history.Run{latest}
	if prev, ok, err := a.history.PreviousRun(ctx, latest); err != nil {
		return nil, err //nolint:wrapcheck // the history store's errors are already labelled
	} else if ok {
		runs = append(runs, prev)
	}
	return runs, nil
}

// playerValues is every player's value from a model run in each view, with the position he was
// valued at and his age.
func (a *App) playerValues(ctx context.Context, runID int64, age func(string) float64) (map[string][]m2service.PlayerValue, error) {
	scores, err := a.history.ModelScores(ctx, runID)
	if err != nil {
		return nil, fmt.Errorf("power rankings: %w", err)
	}
	now := make([]m2service.PlayerValue, len(scores))
	dyn := make([]m2service.PlayerValue, len(scores))
	for i, s := range scores {
		now[i] = m2service.PlayerValue{MFLID: s.MFLID, Position: s.Position, Value: s.NowPPG, Age: age(s.MFLID)}
		dyn[i] = now[i]
		dyn[i].Value = s.DynastyPPG
	}
	return map[string][]m2service.PlayerValue{ViewSeason: now, ViewFranchise: dyn}, nil
}

// powerRows formats the board, with each rank's move from was (the previous board's ranks) and
// each franchise's context columns.
func powerRows(rows []m2service.Row, was map[string]int, outlooks map[string]m2service.Outlook) []PowerRow {
	out := make([]PowerRow, 0, len(rows))
	for _, r := range rows {
		o := outlooks[r.FranchiseID]
		row := PowerRow{
			Rank: r.Rank, FranchiseID: r.FranchiseID, Name: r.Name,
			PowerScore: r.PowerScore, RosterZ: r.RosterZ, MFLPerfZ: r.MFLPerfZ, AgeZ: r.AgeZ,
			RosterValue: r.RosterValue, Results: r.Results,
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

// seasonWeeks is the league's last regular-season week, 0 when unreadable.
func (a *App) seasonWeeks() int {
	last, _ := a.rulebook.GetSetting("lastRegularSeasonWeek")
	n, err := strconv.Atoi(last)
	if err != nil {
		return 0
	}
	return n
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

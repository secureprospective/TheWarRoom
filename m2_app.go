package main

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/ingestion/leaguestandings"
	"github.com/secureprospective/TheWarRoom/internal/m2service"
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

	RosterValue float64 `json:"rosterValue"`
	Results     float64 `json:"results"` // what the blend read, in [0,1]; Performance names it

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

// PowerRankingsResult echoes the view, weight, aggregation mode and N actually applied, so the
// controls and the rows never disagree. Zero rows means the league has no model run yet.
// WeeksScored is how many weeks the MFL standings hold, read from the head-to-head records, and
// SeasonWeeks the league's last regular-season week: the standings are final when they meet. Phase is separate from
// Freshness: an offseason board is fresh data about a finished season. An empty Phase means the
// phase read failed, which affects the label only.
type PowerRankingsResult struct {
	OK     bool    `json:"ok"`
	Error  string  `json:"error"`
	Label  string  `json:"label"`
	Season int     `json:"season"`
	View   string  `json:"view"`
	Weight float64 `json:"weight"`
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
// the franchise ranks dynasty, the roster alone, because this season's record says little
// about the next five.
const (
	ViewSeason    = "season"
	ViewFranchise = "franchise"
)

// GetPowerRankings builds the M2 board: each franchise's roster value in the view, from the
// season's latest model run, blended (this season only) with the MFL standings at the caller's
// weight, with team names joined and each rank's move since the previous model run. A failed
// standings fetch falls back to the last good cache and is labelled stale; only a failure with
// nothing cached is fatal.
func (a *App) GetPowerRankings(weight float64, aggMode, view string) PowerRankingsResult {
	mode := m2service.ResolveAggMode(aggMode)
	if view != ViewFranchise {
		view = ViewSeason
	}
	if view == ViewFranchise {
		weight = 1
	}
	fail := func(err error) PowerRankingsResult {
		return PowerRankingsResult{Error: err.Error(), Season: a.season, View: view, Weight: weight, AggMode: mode,
			Freshness: Freshness{State: FreshFail, Note: err.Error()}}
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
	boards := make([]m2service.Board, len(runs))
	for i, run := range runs {
		values, err := a.playerValues(ctx, run.ID, view)
		if err != nil {
			return fail(err)
		}
		if boards[i], err = svc.BuildBoard(standings, values, weight, aggMode); err != nil {
			return fail(err)
		}
	}
	res := PowerRankingsResult{OK: true, Season: a.season, View: view, Weight: weight, AggMode: mode,
		Freshness: fresh, SeasonWeeks: a.seasonWeeks(), WeeksScored: weeksScored(standings), Rows: []PowerRow{}}
	if len(boards) == 0 {
		return res
	}
	fillBoard(&res, runs, boards)
	return res
}

// fillBoard puts the latest board in res, each rank's move against the previous one.
func fillBoard(res *PowerRankingsResult, runs []history.Run, boards []m2service.Board) {
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
	res.Rows = powerRows(boards[0].Rows, was)
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

// playerValues is every player's value in the view, from a model run.
func (a *App) playerValues(ctx context.Context, runID int64, view string) ([]m2service.PlayerValue, error) {
	scores, err := a.history.ModelScores(ctx, runID)
	if err != nil {
		return nil, fmt.Errorf("power rankings: %w", err)
	}
	out := make([]m2service.PlayerValue, len(scores))
	for i, s := range scores {
		out[i] = m2service.PlayerValue{MFLID: s.MFLID, Value: s.NowPPG}
		if view == ViewFranchise {
			out[i].Value = s.DynastyPPG
		}
	}
	return out, nil
}

// powerRows formats the board, with each rank's move from was (the previous board's ranks).
func powerRows(rows []m2service.Row, was map[string]int) []PowerRow {
	out := make([]PowerRow, 0, len(rows))
	for _, r := range rows {
		row := PowerRow{
			Rank: r.Rank, FranchiseID: r.FranchiseID, Name: r.Name,
			PowerScore: r.PowerScore, RosterZ: r.RosterZ, MFLPerfZ: r.MFLPerfZ,
			RosterValue: r.RosterValue, Results: r.Results,
			H2HW: r.H2HW, H2HL: r.H2HL, H2HT: r.H2HT,
			AllPlayW: r.AllPlayW, AllPlayL: r.AllPlayL, AllPlayT: r.AllPlayT,
			PF: r.PF, PA: r.PA, PP: r.PP, Pwr: r.Pwr, AltPwr: r.AltPwr,
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

package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/playerscores"
	"github.com/secureprospective/TheWarRoom/internal/modelrun"
	"github.com/secureprospective/TheWarRoom/internal/normalize"
	"github.com/secureprospective/TheWarRoom/internal/rankings"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

// m1Timeout bounds the M1 IPC calls: ScoreLeague fetches two MFL exports and scores ~1,200
// players.
const m1Timeout = 120 * time.Second

// proxyLabel must accompany every score surface while base points are MFL's YTD totals
// (Christopher's ruling); the UI renders it verbatim.
func (a *App) proxyLabel() string {
	if a.season == 0 { // startup failed before the season was parsed
		return "BasePoints: MFL YTD fantasy points (proxy) — L2 pending"
	}
	return fmt.Sprintf("BasePoints: MFL %d YTD fantasy points (proxy) — L2 pending", a.season-1)
}

// loadSeasonPoints keeps history holding MFL's season totals for every league season: each
// earlier season once, the last one again (so late stat corrections land), and the current one
// every pass, which is empty before its first game. A failed fetch is recorded against the
// source and returned as a warning: scoring runs from what history holds, and source health
// decides whether the measure counts as lost.
func (a *App) loadSeasonPoints(ctx context.Context) (warning string, err error) {
	var warnings []string
	for y := ingestion.LeagueFirstSeason; y <= a.season; y++ {
		if y < a.season-1 {
			held, err := a.seasonPointsHeld(ctx, y)
			if err != nil {
				return "", err
			}
			if held {
				continue
			}
		}
		batch, ferr := playerscores.Fetch(ctx, a.mflClient, strconv.Itoa(y), ingestion.LeagueID, y)
		switch {
		case ferr == nil:
			if _, ierr := a.history.Ingest(ctx, batch); ierr != nil {
				return "", fmt.Errorf("app: store %d season points: %w", y, ierr)
			}
		case y == a.season && errors.Is(ferr, playerscores.ErrEmptyScores):
		default:
			if lerr := a.history.LoadFailed(ctx, playerscores.Source, ferr); lerr != nil {
				return "", errors.Join(ferr, lerr)
			}
			warnings = append(warnings, fmt.Sprintf("%d: %v", y, ferr))
		}
	}
	if len(warnings) > 0 {
		return "MFL fantasy points not refreshed (" + strings.Join(warnings, "; ") + "): scored from the last points held", nil
	}
	return "", nil
}

// seasonPointsHeld reports whether history already holds MFL's totals for season.
func (a *App) seasonPointsHeld(ctx context.Context, season int) (bool, error) {
	feats, err := a.history.Features(ctx, history.FeatureQuery{AsOf: time.Now(), Season: season,
		Measures: []string{rankings.BaseMeasure}})
	if err != nil {
		return false, fmt.Errorf("app: read %d season points: %w", season, err)
	}
	return len(feats) > 0, nil
}

// ScoreLeagueResult is the board's scoring report (run, scored, zero-base, exclusions with
// reasons) and the model run's, plus the proxy label. Warning reports a source that failed this
// pass.
type ScoreLeagueResult struct {
	OK      bool            `json:"ok"`
	Error   string          `json:"error"`
	Warning string          `json:"warning"`
	Label   string          `json:"label"`
	Report  rankings.Report `json:"report"`
	Model   modelrun.Report `json:"model"`
}

// ScoreLeague scores all 32 rosters as a board run and a model run in history. A pass that matches the latest
// board exactly writes nothing and reports unchanged.
func (a *App) ScoreLeague() ScoreLeagueResult {
	if err := a.ready(); err != nil {
		return ScoreLeagueResult{Error: err.Error(), Label: a.proxyLabel()}
	}
	ctx, cancel := context.WithTimeout(a.ctx, m1Timeout)
	defer cancel()
	fail := func(err error) ScoreLeagueResult {
		return ScoreLeagueResult{Error: err.Error(), Label: a.proxyLabel()}
	}

	lk, err := a.directory(ctx)
	if err != nil {
		return fail(err)
	}
	warning, err := a.loadSeasonPoints(ctx)
	if err != nil {
		return fail(err)
	}
	// One crosswalk fetch serves the player directory and every scouting signal. A directory
	// write failure is logged, not fatal: the board does not read the directory yet.
	cw, err := a.fetchCrosswalk(ctx)
	if err != nil {
		return fail(err)
	}
	if _, err := a.linkCrosswalk(ctx, cw, lk); err != nil {
		log.Printf("the war room: %v", err)
	}
	// A scouting fetch failure fails the run, so a signal-less league is visible. The exception
	// is an unconfigured CFBD key (in buildScoutingDirectory).
	scout, err := a.buildScoutingDirectory(ctx, lk, cw)
	if err != nil {
		return fail(err)
	}
	runner, err := rankings.New(a.league.Reader(), lk, scout, a.rulebook, a.history, buildLabel())
	if err != nil {
		return fail(err)
	}
	set := a.params.Snapshot() // the board and the model score with the same params
	rep, err := runner.Run(ctx, rankings.RunSpec{
		Kind: history.RunBoard, Season: a.season, AsOf: time.Now(),
		Params: set, Measures: rankings.BoardMeasures(),
	})
	if err != nil {
		return fail(err)
	}
	modelRep, err := a.runModel(ctx, lk, set)
	if err != nil {
		return fail(err)
	}
	names := a.rulebook.FranchiseNames()
	for i := range rep.Excluded {
		rep.Excluded[i].FranchiseName = domain.FranchiseLabel(names, rep.Excluded[i].FranchiseID)
	}
	return ScoreLeagueResult{OK: true, Warning: warning, Label: a.proxyLabel(), Report: rep, Model: modelRep}
}

// latestBoard returns the season's board run. ok is false before the first ScoreLeague.
func (a *App) latestBoard(ctx context.Context) (history.Run, bool, error) {
	run, ok, err := a.history.LatestRun(ctx, a.season, history.RunBoard)
	if err != nil {
		return history.Run{}, false, fmt.Errorf("app: read board: %w", err)
	}
	return run, ok, nil
}

// describeMeasures pairs each measure name with its dictionary meaning.
func (a *App) describeMeasures(names []string) []MissingMeasure {
	out := make([]MissingMeasure, len(names))
	for i, n := range names {
		m, _ := a.history.Registry().Measure(n)
		out[i] = MissingMeasure{Name: n, Meaning: m.Meaning}
	}
	return out
}

// priorRanks maps each player on the board before run to their rank there. It is nil when run
// is the first board.
func (a *App) priorRanks(ctx context.Context, run history.Run) (map[string]int, error) {
	prev, ok, err := a.history.PreviousRun(ctx, run)
	if err != nil || !ok {
		return nil, err //nolint:wrapcheck // display-only; the caller drops it
	}
	scores, err := a.history.RunScores(ctx, prev.ID)
	if err != nil {
		return nil, err //nolint:wrapcheck // display-only; the caller drops it
	}
	ranks := make(map[string]int, len(scores))
	for i, s := range scores {
		ranks[s.MFLID] = i + 1
	}
	return ranks, nil
}

// RankRow is one board row: the stored score joined with identity and contract. CapEff is
// AdjustedScore per $M of salary; CapEffOK is false at $0 salary so the UI shows a dash.
type RankRow struct {
	Rank          int     `json:"rank"`
	MFLID         string  `json:"mflID"`
	Name          string  `json:"name"`
	Position      string  `json:"position"`
	FranchiseID   string  `json:"franchiseID"`
	FranchiseName string  `json:"franchiseName"`
	Salary        float64 `json:"salary"`

	BasePoints    float64 `json:"basePoints"`
	AdjustedScore float64 `json:"adjustedScore"`

	CapEff   float64 `json:"capEff"`
	CapEffOK bool    `json:"capEffOK"`

	// RankDelta is the move since the previous board; positive is up. DeltaOK is false
	// when there is no previous board: "unknown" is not "held position".
	RankDelta int  `json:"rankDelta"`
	DeltaOK   bool `json:"deltaOK"`

	// The two measurables from the season's latest model run, in league points per game;
	// ModelOK is false when that run did not value the player.
	NowPPG     float64 `json:"nowPPG"`
	DynastyPPG float64 `json:"dynastyPPG"`
	ModelOK    bool    `json:"modelOK"`
}

// MissingMeasure is a measure the board's model reads that no active source fed when it ran.
type MissingMeasure struct {
	Name    string `json:"name"`
	Meaning string `json:"meaning"`
}

// RankingsResult is the latest board run in final ranking order. The UI never re-sorts;
// filtering is client-side. MissingMeasures non-empty means the run scored on a reduced set.
type RankingsResult struct {
	OK              bool             `json:"ok"`
	Error           string           `json:"error"`
	Warning         string           `json:"warning"` // non-fatal, e.g. names unavailable offline
	Label           string           `json:"label"`
	Season          int              `json:"season"`
	RunID           int64            `json:"runID"` // 0 before the first ScoreLeague
	AsOf            string           `json:"asOf"`  // RFC 3339
	MissingMeasures []MissingMeasure `json:"missingMeasures"`
	// Freshness is always live here: the board is stored engine output. A names outage is reported
	// in Warning, not as staleness.
	Freshness Freshness `json:"freshness"`
	Rows      []RankRow `json:"rows"`
	// ModelRunID is the model run the measurables come from; 0 when there is none.
	ModelRunID int64 `json:"modelRunID"`
}

// GetRankings reads the latest board run and joins display fields. It never scores: empty Rows
// means ScoreLeague hasn't run this season.
func (a *App) GetRankings() RankingsResult {
	if err := a.ready(); err != nil {
		return RankingsResult{Error: err.Error(), Label: a.proxyLabel()}
	}
	ctx, cancel := context.WithTimeout(a.ctx, m1Timeout)
	defer cancel()

	run, ok, err := a.latestBoard(ctx)
	if err != nil {
		return RankingsResult{Error: err.Error(), Label: a.proxyLabel()}
	}
	if !ok {
		return RankingsResult{OK: true, Label: a.proxyLabel(), Season: a.season,
			MissingMeasures: []MissingMeasure{}, Freshness: localFreshness(), Rows: []RankRow{}}
	}
	scores, err := a.history.RunScores(ctx, run.ID)
	if err != nil {
		return RankingsResult{Error: err.Error(), Label: a.proxyLabel()}
	}
	// Names are display-only: an MFL outage must not blank a board that sits in SQLite. Unknown ids
	// render a fallback and the warning says why.
	var warning string
	lk, err := a.directory(ctx)
	if err != nil {
		lk = normalize.Lookup{}
		warning = "player names unavailable (players-DB fetch failed: " + err.Error() + ") — scores are persisted and complete"
	}

	// Prior ranks feed the movement indicator only; a failure here must never fail the board.
	prior, perr := a.priorRanks(ctx, run)
	if perr != nil {
		prior = nil
	}

	rows := a.rankRows(scores, lk, prior)
	modelRun, err := a.joinModel(ctx, rows)
	if err != nil {
		warning = strings.TrimPrefix(warning+"; "+err.Error(), "; ")
	}
	return RankingsResult{
		OK: true, Warning: warning, Label: a.proxyLabel(),
		Season: a.season, RunID: run.ID, AsOf: run.AsOf.Format(time.RFC3339),
		MissingMeasures: a.describeMeasures(run.MissingMeasures),
		Freshness:       localFreshness(), Rows: rows, ModelRunID: modelRun,
	}
}

// rankRows joins each stored score, in run order, with its display fields and its move since
// the prior board (nil when there is none).
func (a *App) rankRows(scores []history.Score, lk normalize.Lookup, prior map[string]int) []RankRow {
	names := a.rulebook.FranchiseNames()
	rows := make([]RankRow, 0, len(scores))
	for i, s := range scores {
		row := RankRow{
			Rank:          i + 1, // stored order is the ranking
			MFLID:         s.MFLID,
			BasePoints:    s.BasePoints,
			AdjustedScore: s.AdjustedScore,
		}
		// Only a player on the prior board gets a delta. Treating a missing prior rank as 0 would
		// invent a huge jump for every new player.
		if was, ok := prior[s.MFLID]; ok {
			row.RankDelta, row.DeltaOK = was-row.Rank, true
		}
		if f, ok := lk.Facts(s.MFLID); ok {
			row.Name, row.Position = f.Name, string(f.Position)
		} else {
			row.Name = "(unknown id " + s.MFLID + ")"
		}
		if p, ok := a.league.Reader().Player(s.MFLID); ok {
			row.FranchiseID, row.FranchiseName = p.FranchiseID, domain.FranchiseLabel(names, p.FranchiseID)
			row.Salary = p.CapSalary.Millions()
			if row.Salary > 0 {
				row.CapEff, row.CapEffOK = s.AdjustedScore/row.Salary, true
			}
		}
		rows = append(rows, row)
	}
	return rows
}

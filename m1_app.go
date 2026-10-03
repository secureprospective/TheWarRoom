package main

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/playerscores"
	"github.com/secureprospective/TheWarRoom/internal/normalize"
	"github.com/secureprospective/TheWarRoom/internal/rankings"
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

// m1Ready reports whether the stores came up.
func (a *App) m1Ready() error {
	if a.startupErr != nil {
		return a.startupErr
	}
	if a.rulebook == nil || a.state == nil || a.output == nil || a.params == nil {
		return fmt.Errorf("stores not initialized")
	}
	return nil
}

// basePoints fetches the proxy season's YTD totals. Validate makes parse failures impossible, so
// one here is drift and fails loudly.
func (a *App) basePoints(ctx context.Context) (map[string]float64, error) {
	raw, err := playerscores.Fetch(ctx, a.mflClient, ingestion.SeasonYear, ingestion.LeagueID, strconv.Itoa(a.season-1))
	if err != nil {
		return nil, fmt.Errorf("app: fetch YTD proxy scores: %w", err)
	}
	base := make(map[string]float64, len(raw))
	for _, r := range raw {
		v, perr := strconv.ParseFloat(r.Score, 64)
		if perr != nil {
			return nil, fmt.Errorf("app: proxy score %s=%q unparseable despite Validate: %w", r.ID, r.Score, perr)
		}
		base[r.ID] = v
	}
	return base, nil
}

// ScoreLeagueResult is the scoring report (scored, skipped, zero-base, exclusions with reasons)
// plus the proxy label.
type ScoreLeagueResult struct {
	OK     bool            `json:"ok"`
	Error  string          `json:"error"`
	Label  string          `json:"label"`
	Report rankings.Report `json:"report"`
}

// ScoreLeague scores all 32 rosters and stores the board under the active config version.
// Re-running a scored (season, config) writes nothing and reports skippedExisting.
func (a *App) ScoreLeague() ScoreLeagueResult {
	if err := a.m1Ready(); err != nil {
		return ScoreLeagueResult{Error: err.Error(), Label: a.proxyLabel()}
	}
	ctx, cancel := context.WithTimeout(a.ctx, m1Timeout)
	defer cancel()

	// Check skip-if-present before the two MFL fetches, so a re-run doesn't pay for data it would
	// discard. The Runner re-checks; this is only the fast path.
	ver, err := a.rulebook.ActiveVersion(ctx)
	if err != nil {
		return ScoreLeagueResult{Error: err.Error(), Label: a.proxyLabel()}
	}
	if existing, serr := a.output.Reader().Scores(ctx, a.season, ver); serr == nil && len(existing) > 0 {
		return ScoreLeagueResult{OK: true, Label: a.proxyLabel(), Report: rankings.Report{
			Season: a.season, ConfigVersion: ver, SkippedExisting: true, Existing: len(existing),
		}}
	}

	lk, err := a.directory(ctx)
	if err != nil {
		return ScoreLeagueResult{Error: err.Error(), Label: a.proxyLabel()}
	}
	base, err := a.basePoints(ctx)
	if err != nil {
		return ScoreLeagueResult{Error: err.Error(), Label: a.proxyLabel()}
	}
	asm, err := a.assembler()
	if err != nil {
		return ScoreLeagueResult{Error: err.Error(), Label: a.proxyLabel()}
	}
	// A scouting fetch failure fails the run, so a signal-less league is visible. The exception
	// is an unconfigured CFBD key (below).
	scout, err := a.buildScoutingDirectory(ctx, lk)
	if err != nil {
		return ScoreLeagueResult{Error: err.Error(), Label: a.proxyLabel()}
	}
	runner, err := rankings.New(a.state.Reader(), lk, scout, base, a.rulebook, a.output.Writer(), asm, rankings.Registry(a.rubrics()))
	if err != nil {
		return ScoreLeagueResult{Error: err.Error(), Label: a.proxyLabel()}
	}
	rep, err := runner.Run(ctx, a.season, time.Now())
	if err != nil {
		return ScoreLeagueResult{Error: err.Error(), Label: a.proxyLabel()}
	}
	names := a.rulebook.FranchiseNames()
	for i := range rep.Excluded {
		rep.Excluded[i].FranchiseName = domain.FranchiseLabel(names, rep.Excluded[i].FranchiseID)
	}
	return ScoreLeagueResult{OK: true, Label: a.proxyLabel(), Report: rep}
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

	// RankDelta is the move since the previous scored config; positive is up. DeltaOK is false
	// when there is no previous board: "unknown" is not "held position".
	RankDelta int  `json:"rankDelta"`
	DeltaOK   bool `json:"deltaOK"`
}

// RankingsResult holds every stored row for the active (season, config) in final ranking
// order. The UI never re-sorts; filtering is client-side.
type RankingsResult struct {
	OK            bool   `json:"ok"`
	Error         string `json:"error"`
	Warning       string `json:"warning"` // non-fatal, e.g. names unavailable offline
	Label         string `json:"label"`
	Season        int    `json:"season"`
	ConfigVersion int    `json:"configVersion"`
	// Freshness is always live here: the board is stored engine output. A names outage is reported
	// in Warning, not as staleness.
	Freshness Freshness `json:"freshness"`
	Rows      []RankRow `json:"rows"`
}

// GetRankings reads the stored board and joins display fields. It never scores: empty Rows
// means ScoreLeague hasn't run for the active config.
func (a *App) GetRankings() RankingsResult {
	if err := a.m1Ready(); err != nil {
		return RankingsResult{Error: err.Error(), Label: a.proxyLabel()}
	}
	ctx, cancel := context.WithTimeout(a.ctx, m1Timeout)
	defer cancel()

	ver, err := a.rulebook.ActiveVersion(ctx)
	if err != nil {
		return RankingsResult{Error: err.Error(), Label: a.proxyLabel()}
	}
	scores, err := a.output.Reader().Scores(ctx, a.season, ver)
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
	prior, priorOK, perr := a.output.Reader().PriorRanks(ctx, a.season, ver)
	if perr != nil {
		priorOK = false
	}

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
		if priorOK {
			if was, ok := prior[s.MFLID]; ok {
				row.RankDelta, row.DeltaOK = was-row.Rank, true
			}
		}
		if f, ok := lk.Facts(s.MFLID); ok {
			row.Name, row.Position = f.Name, string(f.Position)
		} else {
			row.Name = "(unknown id " + s.MFLID + ")"
		}
		if p, ok := a.state.Reader().Player(s.MFLID); ok {
			row.FranchiseID, row.FranchiseName = p.FranchiseID, domain.FranchiseLabel(names, p.FranchiseID)
			row.Salary = p.CapSalary.Millions()
			if row.Salary > 0 {
				row.CapEff, row.CapEffOK = s.AdjustedScore/row.Salary, true
			}
		}
		rows = append(rows, row)
	}
	return RankingsResult{
		OK: true, Warning: warning, Label: a.proxyLabel(),
		Season: a.season, ConfigVersion: ver,
		Freshness: localFreshness(), Rows: rows,
	}
}

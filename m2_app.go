package main

import (
	"context"
	"fmt"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/m2service"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

// m2Timeout bounds GetPowerRankings: one MFL standings fetch plus a read of the M1 board.
const m2Timeout = 120 * time.Second

// PowerRow is one franchise's power-ranking row: the blended score, its two components, and
// MFL's report columns from the same standings call. Display values only.
type PowerRow struct {
	Rank        int    `json:"rank"`
	FranchiseID string `json:"franchiseID"`
	Name        string `json:"name"`

	PowerScore float64 `json:"powerScore"` // weighted z-blend, scaled to [0,1]
	ScoutingZ  float64 `json:"scoutingZ"`  // 0 = league average
	MFLPerfZ   float64 `json:"mflPerfZ"`

	ScoutingScore float64 `json:"scoutingScore"`
	AllPlayWinPct float64 `json:"allPlayWinPct"`

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

// PowerRankingsResult echoes the weight, aggregation mode and N actually applied, so the slider
// and the rows never disagree. Zero rows means M1 hasn't been scored. Phase is separate from
// Freshness: an offseason board is fresh data about a finished season. An empty Phase means the
// phase read failed, which affects the label only.
type PowerRankingsResult struct {
	OK        bool       `json:"ok"`
	Error     string     `json:"error"`
	Label     string     `json:"label"`
	Season    int        `json:"season"`
	Weight    float64    `json:"weight"`
	AggMode   string     `json:"aggMode"`
	StarterN  int        `json:"starterN"`
	Freshness Freshness  `json:"freshness"`
	Phase     string     `json:"phase"`
	Rows      []PowerRow `json:"rows"`
}

// GetPowerRankings builds the M2 board: MFL standings plus each franchise's stored M1 score,
// blended at the caller's weight (clamped), with team names joined. A failed standings fetch
// falls back to the last good cache and is labelled stale; only a failure with nothing cached
// is fatal. The orchestration lives in internal/m2service; this is the adapter.
func (a *App) GetPowerRankings(weight float64, aggMode string) PowerRankingsResult {
	// Resolve the mode first so every error path echoes the normalized mode.
	mode := m2service.ResolveAggMode(aggMode)
	fail := func(err error) PowerRankingsResult {
		return PowerRankingsResult{
			Error: err.Error(), Label: a.proxyLabel(), Season: a.season,
			Weight: weight, AggMode: mode,
			Freshness: Freshness{State: FreshFail, Note: err.Error()},
		}
	}
	if err := a.ready(); err != nil {
		return fail(err)
	}
	ctx, cancel := context.WithTimeout(a.ctx, m2Timeout)
	defer cancel()

	var scores []history.Score
	run, ok, err := a.latestBoard(ctx)
	if err != nil {
		return fail(err)
	}
	if ok {
		if scores, err = a.history.RunScores(ctx, run.ID); err != nil {
			return fail(err)
		}
	}

	standings, fresh, err := a.standingsOrArchive(ctx)
	if err != nil {
		return fail(fmt.Errorf("power rankings: %w", err))
	}

	svc, err := m2service.New(a.league.Reader(), a.rulebook)
	if err != nil {
		return fail(err)
	}
	board, err := svc.BuildBoard(standings, scores, weight, aggMode)
	if err != nil {
		return fail(err)
	}

	rows := make([]PowerRow, 0, len(board.Rows))
	for _, r := range board.Rows {
		rows = append(rows, PowerRow{
			Rank: r.Rank, FranchiseID: r.FranchiseID, Name: r.Name,
			PowerScore: r.PowerScore, ScoutingZ: r.ScoutingZ, MFLPerfZ: r.MFLPerfZ,
			ScoutingScore: r.ScoutingScore, AllPlayWinPct: r.AllPlayWinPct,
			H2HW: r.H2HW, H2HL: r.H2HL, H2HT: r.H2HT,
			AllPlayW: r.AllPlayW, AllPlayL: r.AllPlayL, AllPlayT: r.AllPlayT,
			PF: r.PF, PA: r.PA, PP: r.PP, Pwr: r.Pwr, AltPwr: r.AltPwr,
		})
	}
	return PowerRankingsResult{
		OK: true, Label: a.proxyLabel(),
		Season: a.season, Weight: board.Weight,
		AggMode: board.Mode, StarterN: board.StarterN,
		Freshness: fresh, Phase: a.currentPhaseLabel(),
		Rows: rows,
	}
}

// currentPhaseLabel reads the season phase for labelling only; a failure returns "". It uses
// its own context because the caller's is usually dead after a failed fetch, which is exactly
// when the "final" label matters.
func (a *App) currentPhaseLabel() string {
	ctx, cancel := context.WithTimeout(a.fallbackParent(), fallbackTimeout)
	defer cancel()
	ph, err := a.whatif.CurrentPhase(ctx)
	if err != nil {
		return ""
	}
	return string(ph)
}

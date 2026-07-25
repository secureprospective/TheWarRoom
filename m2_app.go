package main

import (
	"context"
	"fmt"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/m2service"
)

// m2Timeout bounds the M2 IPC call. GetPowerRankings does at most one MFL
// leagueStandings fetch plus a read of the persisted M1 board — the same order of
// work as GetRankings, so it shares its budget.
const m2Timeout = 120 * time.Second

// PowerRow is one franchise's Power-Ranking row: the blended score and its two
// normalized components joined with identity and the MFL report columns that come
// free in the same standings call. Fully typed for ifaceguard; every float is a
// display value, never a stored competing truth.
type PowerRow struct {
	Rank        int    `json:"rank"`
	FranchiseID string `json:"franchiseID"`
	Name        string `json:"name"`

	PowerScore float64 `json:"powerScore"` // weighted z-blend, min-max'd to [0,1]
	ScoutingZ  float64 `json:"scoutingZ"`  // standardized scouting component (0 = league avg)
	MFLPerfZ   float64 `json:"mflPerfZ"`   // standardized all-play component

	ScoutingScore float64 `json:"scoutingScore"` // raw aggregated AdjustedScore
	AllPlayWinPct float64 `json:"allPlayWinPct"` // raw all-play win rate [0,1]

	// MFL report passthrough columns (display + sort), from leagueStandings.
	H2HW     int     `json:"h2hW"`
	H2HL     int     `json:"h2hL"`
	H2HT     int     `json:"h2hT"`
	AllPlayW int     `json:"allPlayW"`
	AllPlayL int     `json:"allPlayL"`
	AllPlayT int     `json:"allPlayT"`
	PF       float64 `json:"pf"`
	PA       float64 `json:"pa"`
	PP       float64 `json:"pp"`
	Pwr      float64 `json:"pwr"`    // MFL Power Rank (unnormalized display column)
	AltPwr   float64 `json:"altPwr"` // MFL Alternate Power Rank (display column)
}

// PowerRankingsResult is the GetPowerRankings IPC payload. Weight echoes back the
// clamped scouting weight actually applied so the UI slider and the rows never
// disagree. AggMode echoes the scouting aggregation used ("sum" or "topn") and
// StarterN the N applied for top-N (0 when not applicable). A zero-rows OK result
// means the M1 board has not been scored yet.
//
// Freshness carries the B-5 degradation contract: live when the standings fetch
// succeeded, stale when the board is built from the last-known-good cache after a failed
// fetch. Phase is the current season phase, carried alongside — NOT folded into Freshness —
// because an offseason board is fresh data about a finished season, not stale data (see
// the Freshness doc comment). An empty Phase means the phase read itself failed, which
// degrades the label only and never the board.
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

// GetPowerRankings composes the M2 blended board: it fetches MFL leagueStandings,
// aggregates the persisted M1 AdjustedScore per franchise, blends them with the
// caller's scouting weight (default powerrankings.DefaultScoutingWeight, clamped in
// Blend), and joins the MFL report columns + real team names for display. It is
// READ-ONLY as far as league state goes — no transactions, no staged confirm.
//
// B-5 DEGRADATION: an MFL standings failure is NO LONGER fatal. It used to be, because
// the all-play component is the whole 40% of the blend and had no persisted fallback
// (unlike M1 scores, which sit in SQLite). standings_cache now provides that fallback:
// a successful fetch is cached, and a failed fetch falls back to the last-known-good copy
// and labels the result stale. Only a failure with NOTHING cached is still fatal — that
// is the honest "no data at all" case, not a degradation. The single cache write is why
// this is no longer literally read-only; it touches no league state.
//
// Session 43 (SLIM_MAP §6.1): this method is now the whole adapter — validate →
// fetch IO (standings + scores) → route into m2service.Service → format the Wails
// DTO. The aggregation/blend/join orchestration lives in internal/m2service,
// mirroring how ScoreLeague delegates to rankings.Runner.
func (a *App) GetPowerRankings(weight float64, aggMode string) PowerRankingsResult {
	fail := func(err error) PowerRankingsResult {
		return PowerRankingsResult{
			Error: err.Error(), Label: a.proxyLabel(), Season: a.season,
			Weight: weight, AggMode: aggMode,
			Freshness: Freshness{State: FreshFail, Note: err.Error()},
		}
	}
	if err := a.m1Ready(); err != nil {
		return fail(err)
	}
	ctx, cancel := context.WithTimeout(a.ctx, m2Timeout)
	defer cancel()

	ver, err := a.rulebook.ActiveVersion(ctx)
	if err != nil {
		return fail(err)
	}
	scores, err := a.output.Reader().Scores(ctx, a.season, ver)
	if err != nil {
		return fail(err)
	}

	standings, fresh, err := a.standingsOrCache(ctx)
	if err != nil {
		return fail(fmt.Errorf("power rankings: %w", err))
	}

	svc, err := m2service.New(a.state.Reader(), a.rulebook)
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

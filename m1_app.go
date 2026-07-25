package main

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/playerscores"
	"github.com/secureprospective/TheWarRoom/internal/normalize"
	"github.com/secureprospective/TheWarRoom/internal/rankings"
)

// m1Timeout bounds the M1 IPC calls. ScoreLeague may fetch two MFL exports and
// score ~1200 players; GetRankings may lazily fetch the players DB once. Both far
// exceed the 3 s ping budget, neither may hang the frontend forever.
const m1Timeout = 120 * time.Second

// proxyLabel is the honest-labeling string for the BasePoints placeholder
// (Christopher 2026-06-28, option (b)): it MUST accompany every score surface
// until the real L2 base-scoring block ships. The UI renders it verbatim.
func (a *App) proxyLabel() string {
	if a.season == 0 { // startup failed before the season parsed; don't render "MFL -1"
		return "BasePoints: MFL YTD fantasy points (proxy) — L2 pending"
	}
	return fmt.Sprintf("BasePoints: MFL %d YTD fantasy points (proxy) — L2 pending", a.season-1)
}

// m1Ready reports whether the store floor came up. One place, one message shape.
func (a *App) m1Ready() error {
	if a.startupErr != nil {
		return a.startupErr
	}
	if a.rulebook == nil || a.state == nil || a.output == nil || a.params == nil {
		return fmt.Errorf("stores not initialized")
	}
	return nil
}

// basePoints fetches the proxy season's YTD totals and parses them into the
// orchestrator's map. Parse failures are impossible-by-Validate, so one here is
// drift worth failing loud on, not skipping.
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

// ScoreLeagueResult is the ScoreLeague IPC payload: the orchestrator's full
// report (scored / skipped / zero-base / exclusions with reasons) plus the proxy
// label the UI must display. Fully typed (ifaceguard).
type ScoreLeagueResult struct {
	OK     bool            `json:"ok"`
	Error  string          `json:"error"`
	Label  string          `json:"label"`
	Report rankings.Report `json:"report"`
}

// ScoreLeague runs the M1 scoring pass: all 32 rosters through the engine,
// persisted to B6 stamped with the ACTIVE config version. Re-running an
// already-scored (season, config) reports skippedExisting and writes nothing
// (the gate-checked skip-if-present UX) — score under a NEW config by changing
// the rulebook active version, never by mutating persisted rows (DECISION-010).
func (a *App) ScoreLeague() ScoreLeagueResult {
	if err := a.m1Ready(); err != nil {
		return ScoreLeagueResult{Error: err.Error(), Label: a.proxyLabel()}
	}
	ctx, cancel := context.WithTimeout(a.ctx, m1Timeout)
	defer cancel()

	// Skip-if-present is checked BEFORE the two live MFL fetches (GLM M1 review):
	// a re-run of an already-scored (season, config) must not pay a playerScores +
	// players-DB fetch it is about to discard. The Runner re-checks internally —
	// this is the app-layer fast path, not the guard of record.
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
	// S-Phase 0/1: build the scouting profiles (RAS + SchoolTier) and inject
	// them through the ScoutingDirectory port. A fetch failure surfaces loudly —
	// a signal-less league should be visible, matching the app's fail-loud
	// posture (graceful degradation is a separate Christopher decision, not the
	// v1 default; the one exception is an UNCONFIGURED CFBD key — see below).
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
	return ScoreLeagueResult{OK: true, Label: a.proxyLabel(), Report: rep}
}

// RankRow is one board row: the B6 persisted score joined with identity (players
// DB) and contract state (B3c). CapEff is AdjustedScore per $M of effective
// salary (the AD-20 cap-efficiency view); CapEffOK is false for a $0 salary so
// the UI shows "—" instead of an infinity.
type RankRow struct {
	Rank        int     `json:"rank"`
	MFLID       string  `json:"mflID"`
	Name        string  `json:"name"`
	Position    string  `json:"position"`
	FranchiseID string  `json:"franchiseID"`
	Salary      float64 `json:"salary"`

	BasePoints    float64 `json:"basePoints"`
	AdjustedScore float64 `json:"adjustedScore"`

	// B-5: AgePull, L4Combined and CapTier were REMOVED here. They moved into the
	// Contextual Inspector (B-4), which reads its own PlayerScoreDTO — note that DTO
	// uses the SAME field names, so a bare grep for them looks like they are still in
	// use. Nothing on the board read them any more, and they were serializing on every
	// row of every board read (~1200 rows). Per-player diagnostics belong to the
	// Inspector; the board carries only what it renders.

	CapEff   float64 `json:"capEff"`
	CapEffOK bool    `json:"capEffOK"`

	// RankDelta is the player's movement since the previous scored config: positive =
	// moved UP the board. DeltaOK is false when there is no previous board to compare
	// against (first-ever scoring run), and the UI renders that as ABSENT rather than as
	// a delta of zero — "held position" and "unknown" are different claims (§1).
	RankDelta int  `json:"rankDelta"`
	DeltaOK   bool `json:"deltaOK"`
}

// RankingsResult is the GetRankings IPC payload: every persisted row for the
// active (season, config) in FINAL RANKING ORDER (B6's ORDER BY encodes the L6
// tiebreak — the UI never re-sorts). Position/franchise filtering is a client-
// side view over these rows.
type RankingsResult struct {
	OK            bool   `json:"ok"`
	Error         string `json:"error"`
	Warning       string `json:"warning"` // non-fatal degradation (e.g. names unavailable offline)
	Label         string `json:"label"`
	Season        int    `json:"season"`
	ConfigVersion int    `json:"configVersion"`
	// Freshness is always LOCAL-live here: the board is persisted engine output read from
	// SQLite, so it is authoritative regardless of network state. The one network touch is
	// display-only name resolution, and its failure is a PARTIAL degradation reported in
	// Warning — not a staleness claim about the scores, which are complete either way.
	Freshness Freshness `json:"freshness"`
	Rows      []RankRow `json:"rows"`
}

// GetRankings reads the persisted board back from B6 and joins display fields.
// It performs NO scoring — an empty Rows means ScoreLeague has not run for the
// active config yet, which the UI states plainly.
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
	// Name resolution is DISPLAY-ONLY here, so a directory failure must not hide a
	// fully-persisted board (GLM M1 review): on a warm DB the first GetRankings is
	// the first network touch, and an MFL outage would otherwise blank scores that
	// sit in SQLite. Degrade — the zero Lookup misses every id, the per-row
	// "(unknown id …)" fallback renders, and the warning states why.
	var warning string
	lk, err := a.directory(ctx)
	if err != nil {
		lk = normalize.Lookup{}
		warning = "player names unavailable (players-DB fetch failed: " + err.Error() + ") — scores are persisted and complete"
	}

	// Prior ranks back the §1 movement indicator. A failure here degrades the DELTA ONLY —
	// the board is fully persisted and complete without it, so a history read must never be
	// able to fail a board. No prior board (first run) is the same non-event: priorOK false.
	prior, priorOK, perr := a.output.Reader().PriorRanks(ctx, a.season, ver)
	if perr != nil {
		priorOK = false
	}

	rows := make([]RankRow, 0, len(scores))
	for i, s := range scores {
		row := RankRow{
			Rank:          i + 1, // B6 order IS the ranking (L6 encoded in the ORDER BY)
			MFLID:         s.MFLID,
			BasePoints:    s.BasePoints,
			AdjustedScore: s.AdjustedScore,
		}
		// A player present on the prior board gets a delta; one who is NOT (a new
		// acquisition, or a player the prior config did not score) gets none. Treating an
		// absent prior rank as 0 would invent a huge fake jump for every new player.
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
			row.FranchiseID = p.FranchiseID
			// Money → float millions at the display edge (cap-efficiency = score per $M).
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

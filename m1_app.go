package main

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/crosswalk"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/playerscores"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/ras"
	"github.com/secureprospective/TheWarRoom/internal/normalize"
	"github.com/secureprospective/TheWarRoom/internal/rankings"
	"github.com/secureprospective/TheWarRoom/internal/scouting/assembly"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
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
	// S-Phase 0: build the RAS scouting profiles and inject them through the
	// ScoutingDirectory port. A fetch failure surfaces loudly — a RAS-less
	// league should be visible, matching the app's fail-loud posture (graceful
	// degradation is a separate Christopher decision, not the v1 default).
	scout, err := a.buildRASDirectory(ctx, lk)
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

// rasFetchTimeout bounds each RAS-pipeline HTTP call (combine.csv + the
// crosswalk). It is a per-request BACKSTOP — the ScoreLeague context deadline
// (m1Timeout) still bounds the whole pass. Generous by design: combine.csv is
// a multi-season release (~5 MB) and the crosswalk ~1 MB, both off the MFL
// transport on static CDNs; a tight timeout would fail on a cold cache.
const rasFetchTimeout = 90 * time.Second

// buildRASDirectory wires the S-Phase 0 RAS pipeline: collect rostered MFL
// ids, fetch combine + crosswalk, compute per-position RAS-equivalent (§3),
// and wrap the result as a rankings ScoutingDirectory. The PositionLookup
// adapter rides on the SAME cached normalize.Lookup the orchestrator already
// builds — position is a players-DB fact, never re-fetched. The rubric-
// resolved position (composition.ResolveRubricPosition) is identical to the
// players-DB position this phase: no PassRushSnapShare is wired, so the
// DE/LB EDGE router is a passthrough today (deferred per the §3 brief note).
func (a *App) buildRASDirectory(ctx context.Context, lk normalize.Lookup) (rankings.MapScoutingDirectory, error) {
	rosterMFLIDs := collectRosterMFLIDs(a.state.Reader())
	client := &http.Client{Timeout: rasFetchTimeout}
	profiles, err := assembly.BuildRAS(ctx, client, ras.SourceURL, crosswalk.SourceURL, rosterMFLIDs, lookupPosAdapter{lk: lk})
	if err != nil {
		return rankings.MapScoutingDirectory{}, fmt.Errorf("app: build RAS scouting directory: %w", err)
	}
	return rankings.NewMapScoutingDirectory(profiles), nil
}

// collectRosterMFLIDs walks every franchise roster and returns the set of
// rostered MFL ids — the population the RAS pipeline scores against. Order-
// independent: BuildRAS treats the slice as a set, and the cohort math is
// order-independent (see internal/scouting/assembly/ras_math.go). Duplicates
// across franchises are deduped defensively; in practice a player is on
// exactly one roster (state's invariant).
func collectRosterMFLIDs(st state.Reader) []string {
	seen := make(map[string]struct{})
	out := make([]string, 0, 64)
	for _, fid := range st.Franchises() {
		roster, ok := st.Roster(fid)
		if !ok {
			continue
		}
		for _, p := range roster {
			if _, dup := seen[p.MFLID]; dup {
				continue
			}
			seen[p.MFLID] = struct{}{}
			out = append(out, p.MFLID)
		}
	}
	return out
}

// lookupPosAdapter adapts the existing normalize.Lookup to assembly's narrow
// PositionLookup port. It returns false on an unknown id OR an aggregate (Facts
// already collapses both to ok=false), keeping the assembly package free of any
// normalize import.
type lookupPosAdapter struct {
	lk normalize.Lookup
}

func (a lookupPosAdapter) Position(mflID string) (domain.Position, bool) {
	facts, ok := a.lk.Facts(mflID)
	if !ok {
		return "", false
	}
	return facts.Position, true
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
	AgePull       float64 `json:"agePull"`
	L4Combined    float64 `json:"l4Combined"`
	CapTier       string  `json:"capTier"`
	AdjustedScore float64 `json:"adjustedScore"`

	CapEff   float64 `json:"capEff"`
	CapEffOK bool    `json:"capEffOK"`
}

// RankingsResult is the GetRankings IPC payload: every persisted row for the
// active (season, config) in FINAL RANKING ORDER (B6's ORDER BY encodes the L6
// tiebreak — the UI never re-sorts). Position/franchise filtering is a client-
// side view over these rows.
type RankingsResult struct {
	OK            bool      `json:"ok"`
	Error         string    `json:"error"`
	Warning       string    `json:"warning"` // non-fatal degradation (e.g. names unavailable offline)
	Label         string    `json:"label"`
	Season        int       `json:"season"`
	ConfigVersion int       `json:"configVersion"`
	Rows          []RankRow `json:"rows"`
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

	rows := make([]RankRow, 0, len(scores))
	for i, s := range scores {
		row := RankRow{
			Rank:          i + 1, // B6 order IS the ranking (L6 encoded in the ORDER BY)
			MFLID:         s.MFLID,
			BasePoints:    s.BasePoints,
			AgePull:       s.AgePull,
			L4Combined:    s.Layer4Output.Combined,
			CapTier:       string(s.CapTier),
			AdjustedScore: s.AdjustedScore,
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
	return RankingsResult{OK: true, Warning: warning, Label: a.proxyLabel(), Season: a.season, ConfigVersion: ver, Rows: rows}
}

package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/crosswalk"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/playerscores"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/ras"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/schooltier"
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

// rasFetchTimeout bounds each scouting-pipeline HTTP call (combine.csv, the
// crosswalk, and the CFBD /teams call). It is a per-request BACKSTOP — the
// ScoreLeague context deadline (m1Timeout) still bounds the whole pass.
// Generous by design: combine.csv is a multi-season release (~5 MB) and the
// crosswalk ~1 MB, both off the MFL transport on static CDNs; a tight timeout
// would fail on a cold cache.
const rasFetchTimeout = 90 * time.Second

// cfbdEnvVar is the environment variable carrying the CollegeFootballData
// bearer token — the credential the SchoolTier (and later CollegeShare) signal
// needs. It is read at wire time, never stored in the repo. When it is UNSET the
// school-tier signal is skipped rather than failing the board (see below).
const cfbdEnvVar = "CFBD_API_KEY"

// buildScoutingDirectory wires the scouting pipeline and wraps the merged result
// as a rankings ScoutingDirectory. Both signals ride on the SAME cached
// normalize.Lookup the orchestrator already built (position + college are
// players-DB facts, never re-fetched) via one scoutLookupAdapter.
//
//   - S-Phase 0 (RAS): fetch combine + crosswalk, compute per-position
//     RAS-equivalent (§3). A fetch failure surfaces loudly. (The rubric-resolved
//     position equals the players-DB position this phase — no PassRushSnapShare is
//     wired, so the DE/LB EDGE router is a passthrough today.)
//   - S-Phase 1 (SchoolTier): fetch CFBD /teams, join each rostered id →
//     college → tier, and MERGE into the per-player Profile. This needs a CFBD
//     API key; when the key is UNCONFIGURED the signal is SKIPPED (every player →
//     SchoolUnset → Data-Parity neutral) rather than failing the whole board, so
//     an environment without the key still ranks on RAS. A key-PRESENT fetch
//     failure (network / 401 / parse) still surfaces loudly — a missing key and a
//     broken fetch are different conditions.
func (a *App) buildScoutingDirectory(ctx context.Context, lk normalize.Lookup) (rankings.MapScoutingDirectory, error) {
	rosterMFLIDs := collectRosterMFLIDs(a.state.Reader())
	client := &http.Client{Timeout: rasFetchTimeout}
	adapter := scoutLookupAdapter{lk: lk}

	profiles, err := assembly.BuildRAS(ctx, client, ras.SourceURL, crosswalk.SourceURL, rosterMFLIDs, adapter)
	if err != nil {
		return rankings.MapScoutingDirectory{}, fmt.Errorf("app: build RAS scouting directory: %w", err)
	}

	if key := strings.TrimSpace(os.Getenv(cfbdEnvVar)); key != "" {
		year, cerr := strconv.Atoi(ingestion.SeasonYear)
		if cerr != nil {
			return rankings.MapScoutingDirectory{}, fmt.Errorf("app: season year %q not numeric: %w", ingestion.SeasonYear, cerr)
		}
		tiers, terr := assembly.BuildSchoolTier(ctx, client, schooltier.TeamsURL, key, year, rosterMFLIDs, adapter)
		if terr != nil {
			return rankings.MapScoutingDirectory{}, fmt.Errorf("app: build school-tier scouting directory: %w", terr)
		}
		// Merge tiers into the per-player Profile map. A tier-only player (school
		// known, no combine) gets a fresh Profile carrying just SchoolTier — its
		// HasRAS stays false, so the RAS rubric still imputes the fallback.
		for pid, tier := range tiers {
			p := profiles[pid]
			p.MFLID = pid
			p.SchoolTier = tier
			profiles[pid] = p
		}
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

// scoutLookupAdapter adapts the existing normalize.Lookup to assembly's narrow
// scouting ports — PositionLookup (RAS) and SchoolLookup (SchoolTier) — so both
// signals read the SAME cached players-DB facts and the assembly package stays
// free of any normalize import. Facts returns ok=false on an unknown id OR an
// aggregate (collapsed there), which both ports treat as an ordinary miss.
type scoutLookupAdapter struct {
	lk normalize.Lookup
}

func (a scoutLookupAdapter) Position(mflID string) (domain.Position, bool) {
	facts, ok := a.lk.Facts(mflID)
	if !ok {
		return "", false
	}
	return facts.Position, true
}

// College returns the player's raw MFL college name. ok=false when the player is
// unknown/aggregate OR MFL carries no college for them (team-D rows, some deep
// database players) — the school-tier join treats an absent college as a neutral
// miss (SchoolUnset downstream).
func (a scoutLookupAdapter) College(mflID string) (string, bool) {
	facts, ok := a.lk.Facts(mflID)
	if !ok || facts.College == "" {
		return "", false
	}
	return facts.College, true
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

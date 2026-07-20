package assembly

import (
	"context"
	"fmt"
	"math"
	"net/http"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/crosswalk"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/pfrcoverage"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
	"github.com/secureprospective/TheWarRoom/internal/scouting"
)

// BuildCoverage fetches the crosswalk + the PFR advanced-defense coverage-allowed feed,
// joins each rostered CB/S MFL id → gsis → RawCoverage, and returns the ONE engine-ready
// coverage anchor value scouting.Profile.Coverage carries — a [0,1] score where HIGHER is
// BETTER (the raw signal, PFR passer-rating-allowed, is INVERTED here: a defender who
// surrenders a low passer rating is covering well). A player NOT in the returned map has no
// coverage signal (clean miss — the film composite stays Data-Parity neutral for him).
//
// This is the FILM calibration pass's first wired weight (Thread C, C-4 step 1). The C-1
// live evidence made it a signal worth wiring: PFR passer-rating-allowed is INDEPENDENT of
// Madden coverage grades at CB/S (|r|<0.30), so it is a GENUINE additive signal, not a
// Madden restatement — it earns a real (0.20) film-budget weight rather than a token one.
// The 0.20 blend itself lives UPSTREAM (rankings.applyScouting builds the film composite);
// this leaf owns only the raw→normalized inversion, so the engine keeps consuming a single
// [0,1] FilmComposite and this package imports no engine/store.
//
// HARD BOUNDARY (scouting.NGSCoverage doc): the coverage anchor applies at CB and S ONLY.
// A non-CB/S defender that somehow carries a coverage row is skipped — the anchor must not
// bleed to any other position.
//
// NORMALIZATION (coverageBest/coverageWorst): the raw passer-rating-allowed is mapped
// linearly onto [0,1] and inverted, anchored on the C-1 CB/S CoverRatingAllowed population
// (min≈32–40, median≈93–94, p95≈135–141): best=40.0 → 1.0, worst=145.0 → 0.0, clamped. The
// band is chosen so the population MEDIAN lands at ≈0.50 (CB 92.6 → 0.50, S 94.3 → 0.48),
// matching the engine film S-curve's 0.50 neutral inflection — an average coverage season
// is neutral, not a lift or a drag.
//
// SAMPLE FLOOR (coverageTargetFloor): a defender targeted only a handful of times has a
// noisy passer-rating-allowed (one long TD swings it 100+ points). Below the floor the row
// is treated as absent (clean miss → neutral film) rather than fed as a real signal, the
// same fidelity posture veteranfilm takes with its rate floors.
//
// It is a COMPOSITION leaf: it imports Layer-1 fetchers (pfrcoverage, crosswalk) +
// domain/playerid/scouting, produces a typed value, and imports no engine / store /
// normalize-write / database/sql.
//
// ZERO-LEAK (hard constraint): RawCoverage carries only coverage-allowed counting/rate
// stats — no fantasy points / projected volume / MFL scoring column exists on it to bind.
//
// Failures: a genuine fetch failure (network/HTTP/parse, a crosswalk that resolved zero
// entries, or a feed that resolved zero targeted defenders) is surfaced loudly — a
// coverage-less league should be visible, matching BuildRAS/BuildCollegeDefense. A
// player-level miss (no gsis, no coverage row, a non-CB/S position, an absent rating, a
// sub-floor target count) is ordinary and never an error.
func BuildCoverage(
	ctx context.Context,
	client *http.Client,
	sourceURL string,
	cw crosswalk.Map,
	season string,
	rosterMFLIDs []string,
	pos PositionLookup,
) (map[playerid.PlayerID]float64, error) {
	if client == nil {
		return nil, fmt.Errorf("assembly: BuildCoverage requires a non-nil *http.Client")
	}
	if pos == nil {
		return nil, fmt.Errorf("assembly: BuildCoverage requires a non-nil PositionLookup")
	}

	raw, err := pfrcoverage.Fetch(ctx, client, sourceURL, season, cw.PFRMap())
	if err != nil {
		return nil, fmt.Errorf("assembly: fetch coverage: %w", err)
	}

	out := make(map[playerid.PlayerID]float64, len(rosterMFLIDs))
	for _, mfl := range rosterMFLIDs {
		pid, err := playerid.New(mfl)
		if err != nil {
			continue // malformed — an upstream layer should have caught this; skip
		}
		gsis, ok := cw.Lookup(pid)
		if !ok {
			continue // no MFL→gsis mapping — ordinary miss
		}
		rc, ok := raw[gsis]
		if !ok {
			continue // no coverage row — ordinary miss
		}
		position, ok := pos.Position(mfl)
		if !ok {
			continue // no resolved position → cannot apply the CB/S boundary — ordinary miss
		}
		norm, ok := normalizeCoverage(rc, position)
		if !ok {
			continue // non-CB/S, absent rating, or below the target floor — neutral
		}
		out[pid] = norm
	}
	return out, nil
}

const (
	// coverageBest / coverageWorst anchor the passer-rating-allowed → [0,1] inversion on the
	// C-1 CB/S population (see BuildCoverage doc). A rating at or below coverageBest is elite
	// coverage (→1.0); at or above coverageWorst is poor coverage (→0.0); the population
	// median (≈93) lands at ≈0.50, the engine film S-curve's neutral inflection.
	coverageBest  = 40.0
	coverageWorst = 145.0

	// coverageTargetFloor is the minimum season targets for a passer-rating-allowed to be
	// trusted as a real signal (below it a single play dominates the rate). A sub-floor
	// defender is a clean miss (neutral film), not a noisy input.
	coverageTargetFloor = 10
)

// normalizeCoverage maps one defender's raw coverage line to the engine-ready [0,1] anchor
// (higher = better), or reports ok=false when the row carries no usable coverage signal:
// a non-CB/S position (the hard anchor boundary), an absent passer-rating-allowed (nil =
// undefined, never faked from a blank), or a target count below coverageTargetFloor (too
// noisy to trust). A non-finite rating is also rejected so it can never poison the film
// composite's finite-range invariant.
func normalizeCoverage(rc pfrcoverage.RawCoverage, pos domain.Position) (float64, bool) {
	if pos != domain.PosCB && pos != domain.PosS {
		return 0, false // coverage anchor is CB/S ONLY (hard boundary)
	}
	if rc.PasserRatingAllowed == nil {
		return 0, false // undefined rating (no/too-few targets upstream) — absent, not 0.0
	}
	if rc.Targets < coverageTargetFloor {
		return 0, false // too few targets — noisy, treat as absent
	}
	rating := *rc.PasserRatingAllowed
	if math.IsNaN(rating) || math.IsInf(rating, 0) {
		return 0, false
	}
	// Invert + clamp: low rating allowed → high score.
	norm := (coverageWorst - rating) / (coverageWorst - coverageBest)
	if norm < 0 {
		norm = 0
	} else if norm > 1 {
		norm = 1
	}
	return norm, true
}

// CoverageGroup wraps a normalized coverage anchor as the non-nil scouting.NGSCoverage
// group the merge step writes into Profile.Coverage (present at CB/S only). Exported so the
// app's thin merge loop stays a copy — mirroring how every other S-Phase signal lands.
func CoverageGroup(norm float64) *scouting.NGSCoverage {
	return &scouting.NGSCoverage{CoverageMetrics: norm}
}

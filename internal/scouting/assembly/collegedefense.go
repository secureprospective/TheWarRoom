package assembly

import (
	"context"
	"fmt"
	"math"
	"net/http"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/collegedefense"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/crosswalk"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
)

// BuildCollegeDefense fetches the crosswalk + the season DEFENSIVE college-production
// feed, joins each rostered MFL id → gsis → RawCollegeDefense, and collapses the five
// raw within-team component shares (tackle / sack / TFL / passes-defended / interception)
// into the ONE monolithic position-defined value scouting.Profile.CollegeProductionShare
// holds. It returns that value keyed by playerid.PlayerID; a player NOT in the returned
// map has no defensive college-share signal (clean miss — the rankings side treats
// "absent from map" and "HasCollegeProductionShare=false" identically, and the rubric's
// Data-Parity path neutralizes the sub-signal).
//
// The collapse is NOT a provisional v1 call (unlike the offense BuildCollegeShare). The
// IDP rubrics already LOCK which within-team component shares define the slot per position
// (fetcher header + each §4 rubric), and the combine of a multi-share definition is the
// equal-weight AVERAGE of the named components (explicit at S "average of… INTs and…
// tackles" / LB "averaged across the three event classes"; CB/DT/DE are strictly parallel):
//   - CB → mean(PassDefShare, InterceptionShare)              (CB_Rubric §4)
//   - S  → mean(InterceptionShare, TackleShare)               (S_Rubric §4)
//   - LB → mean(TackleShare, SackShare, TFLShare)             (LB_Rubric §4)
//   - DT → mean(TFLShare, SackShare)                          (DT_Rubric §4)
//   - DE → mean(TFLShare, SackShare)                          (DE_Rubric §4)
//   - Every offense position (QB/RB/WR/TE) + K → absent here: their college-production
//     share is the SEPARATE offense collegeshare feed, not this one. FLAG is unclassified.
//
// The engine consumes the RAW averaged share and applies the per-position §4 normalization
// curve (engine.ScoutingInput.CollegeShare is documented "raw… the rubric maps it to its
// index") — exactly the offense posture: assembly emits raw, the engine normalizes.
//
// This is BuildCollegeShare's defensive sibling and joins on the same RAS SHAPE (gsis-keyed
// raw records via the crosswalk), not on a college NAME. It is a COMPOSITION leaf: it
// imports Layer-1 fetchers (collegedefense, crosswalk) + domain/playerid, produces a typed
// value, and imports no engine / store / normalize-write / database/sql.
//
// Failures: a genuine fetch failure (network/HTTP/parse, a crosswalk that resolved zero
// entries, or a feed that resolved zero gsis-keyed records) is surfaced loudly — a
// defense-share-less league should be visible, matching BuildRAS/BuildCollegeShare. A
// player-level miss (no gsis, no defensive row, a non-defensive position, a non-finite
// share) is ordinary and never an error.
//
// ZERO-LEAK (hard constraint): RawCollegeDefense carries only clean college defensive
// box-score counts and their within-team shares — no fantasy points / projected volume /
// MFL scoring column (CFBD PPA is never fetched) exists on it to bind.
func BuildCollegeDefense(
	ctx context.Context,
	client *http.Client,
	statsURL, crosswalkURL, apiKey string,
	year int,
	rosterMFLIDs []string,
	pos PositionLookup,
) (map[playerid.PlayerID]float64, error) {
	if client == nil {
		return nil, fmt.Errorf("assembly: BuildCollegeDefense requires a non-nil *http.Client")
	}
	if pos == nil {
		return nil, fmt.Errorf("assembly: BuildCollegeDefense requires a non-nil PositionLookup")
	}

	cw, err := crosswalk.Fetch(ctx, client, crosswalkURL)
	if err != nil {
		return nil, fmt.Errorf("assembly: fetch crosswalk: %w", err)
	}
	raw, err := collegedefense.Fetch(ctx, client, statsURL, apiKey, year, cw.GSISForESPN)
	if err != nil {
		return nil, fmt.Errorf("assembly: fetch college defense: %w", err)
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
			continue // no college-defense row — ordinary miss
		}
		position, ok := pos.Position(mfl)
		if !ok {
			continue // no resolved position → cannot collapse — ordinary miss
		}
		share, ok := collapseCollegeDefense(rc, position)
		if !ok {
			continue // position has no source in this feed (offense / K) — neutral
		}
		out[pid] = share
	}
	return out, nil
}

// collapseCollegeDefense reduces the five raw within-team component shares to the single
// position-defined value the engine consumes (the equal-weight average documented on
// BuildCollegeDefense). ok=false means this position has no defensive college-share source
// in this feed (offense positions draw from the separate offense collegeshare feed; K has
// none) — an ordinary skip, not an error. A non-finite blend (defensive against an upstream
// NaN/Inf) is also skipped so it never poisons the PlayerSpec's finite-share invariant.
func collapseCollegeDefense(rc collegedefense.RawCollegeDefense, pos domain.Position) (float64, bool) {
	var share float64
	switch pos {
	case domain.PosCB:
		share = mean(rc.PassDefShare, rc.InterceptionShare)
	case domain.PosS:
		share = mean(rc.InterceptionShare, rc.TackleShare)
	case domain.PosLB:
		share = mean(rc.TackleShare, rc.SackShare, rc.TFLShare)
	case domain.PosDT, domain.PosDE:
		share = mean(rc.TFLShare, rc.SackShare)
	case domain.PosQB, domain.PosRB, domain.PosWR, domain.PosTE,
		domain.PosK, domain.PosFlag:
		// Offense positions draw their college-production share from the separate
		// offense collegeshare feed; K has no college-production share; FLAG is
		// unclassified. No defensive source here.
		return 0, false
	}
	if math.IsNaN(share) || math.IsInf(share, 0) {
		return 0, false
	}
	return share, true
}

// mean is the equal-weight average of the named component shares — the locked IDP combine.
func mean(vals ...float64) float64 {
	var sum float64
	for _, v := range vals {
		sum += v
	}
	return sum / float64(len(vals))
}

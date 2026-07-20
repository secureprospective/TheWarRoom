package assembly

import (
	"context"
	"fmt"
	"math"
	"net/http"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/collegeshare"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/crosswalk"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
)

// BuildCollegeShare fetches the crosswalk + the season college-production feed,
// joins each rostered MFL id → gsis → RawCollegeShare, and collapses the three
// raw within-team shares (reception / receiving-yard / rushing-yard) into the ONE
// monolithic position-defined value scouting.Profile.CollegeProductionShare holds.
// It returns that value keyed by playerid.PlayerID; a player NOT in the returned
// map has no college-share signal (clean miss — the rankings side treats "absent
// from map" and "HasCollegeProductionShare=false" identically, and the rubric's
// Data-Parity path neutralizes the sub-signal).
//
// The v1 collapse (Christopher-approved 2026-07-20, PROVISIONAL — real calibration
// is a deferred, decision-gated pass) is position-defined:
//   - WR / TE → ReceivingYardShare (the classic yardage-dominator signal).
//   - RB      → 0.70·RushingYardShare + 0.30·ReceivingYardShare (rushing-dominant
//     but credits receiving backs).
//   - QB      → no source in this feed (it carries no passing share) → absent →
//     the QB rubric's college-share sub-signal stays Data-Parity neutral in v1.
//   - Every defensive position → absent here: the IDP college-production share is a
//     SEPARATE feed (the collegedefense fetcher, a later phase), not this one.
//
// This is the SchoolTier assembler's sibling but joins on the RAS SHAPE (gsis-keyed
// raw records via the crosswalk), not on a college NAME. It is a COMPOSITION leaf:
// it imports Layer-1 fetchers (collegeshare, crosswalk) + domain/playerid, produces
// a typed value, and imports no engine / store / normalize-write / database/sql.
//
// Failures: a genuine fetch failure (network/HTTP/parse, a crosswalk that resolved
// zero entries, or a feed that resolved zero gsis-keyed records) is surfaced loudly
// — a college-share-less league should be visible, matching BuildRAS's posture. A
// player-level miss (no gsis, no college row, a non-collapsible position, a
// non-finite share) is ordinary and never an error.
//
// ZERO-LEAK (hard constraint): RawCollegeShare carries only clean college box-score
// counts and their within-team shares — no fantasy points / projected volume / MFL
// scoring column exists on it to bind.
func BuildCollegeShare(
	ctx context.Context,
	client *http.Client,
	statsURL, crosswalkURL, apiKey string,
	year int,
	rosterMFLIDs []string,
	pos PositionLookup,
) (map[playerid.PlayerID]float64, error) {
	if client == nil {
		return nil, fmt.Errorf("assembly: BuildCollegeShare requires a non-nil *http.Client")
	}
	if pos == nil {
		return nil, fmt.Errorf("assembly: BuildCollegeShare requires a non-nil PositionLookup")
	}

	cw, err := crosswalk.Fetch(ctx, client, crosswalkURL)
	if err != nil {
		return nil, fmt.Errorf("assembly: fetch crosswalk: %w", err)
	}
	raw, err := collegeshare.Fetch(ctx, client, statsURL, apiKey, year, cw.GSISForESPN)
	if err != nil {
		return nil, fmt.Errorf("assembly: fetch college share: %w", err)
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
			continue // no college-production row — ordinary miss
		}
		position, ok := pos.Position(mfl)
		if !ok {
			continue // no resolved position → cannot collapse — ordinary miss
		}
		share, ok := collapseCollegeShare(rc, position)
		if !ok {
			continue // position has no source in this feed (QB / defense) — neutral
		}
		out[pid] = share
	}
	return out, nil
}

// collapseCollegeShare reduces the three raw within-team shares to the single
// position-defined value the engine consumes (the v1 rule documented on
// BuildCollegeShare). ok=false means this position has no offense college-share
// source in this feed (QB carries no passing share; defensive positions draw from
// the separate collegedefense feed) — an ordinary skip, not an error. A non-finite
// blend (defensive against an upstream NaN/Inf) is also skipped so it never poisons
// the PlayerSpec's finite-share invariant.
func collapseCollegeShare(rc collegeshare.RawCollegeShare, pos domain.Position) (float64, bool) {
	var share float64
	switch pos {
	case domain.PosWR, domain.PosTE:
		share = rc.ReceivingYardShare
	case domain.PosRB:
		share = 0.70*rc.RushingYardShare + 0.30*rc.ReceivingYardShare
	case domain.PosQB, domain.PosK, domain.PosDE, domain.PosDT,
		domain.PosLB, domain.PosCB, domain.PosS, domain.PosFlag:
		// QB carries no passing share in this feed; K has no college-production
		// share; every defensive position draws from the separate collegedefense
		// feed (a later phase); FLAG is unclassified. No offense source here.
		return 0, false
	}
	if math.IsNaN(share) || math.IsInf(share, 0) {
		return 0, false
	}
	return share, true
}

// Package ingestion is Layer 1: it fetches raw data from the MFL API and
// schema-validates the response SHAPE. It transforms NOTHING — converting Raw*
// records into domain types is normalize's job (WF 1C, B3). depguard
// (layer1-no-upward-import) forbids importing engine/store/transactions/api or
// database/sql from here; a violation fails the build.
//
// This file holds the cross-fetcher boundary helpers every Fetch shares:
// player-ID validation (RISK-003 enforcement site #1) and team-aggregate
// filtering. Each fetcher lives in its own subpackage (rosters/, schedule/, …)
// and imports these so the boundary rules have exactly one implementation.
package ingestion

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/playerid"
)

// League identity for Phase 1 (single league, current season). These are
// configuration, not engine calibration — they become user/config-driven when
// the app serves more than the Legacy NFL league. The HOST is deliberately
// absent: it is discovered at runtime by mfl.Client.DiscoverHost, never
// hardcoded (project hard constraint, B1 carry-forward).
const (
	LeagueID   = "14432"
	SeasonYear = "2026"
)

// ValidatePlayerID is RISK-003 enforcement call site #1: every player ID crossing
// the ingestion boundary must be a well-formed MFL ID or the record is rejected
// here, before anything downstream sees it. It returns the playerid.PlayerID so a
// caller may keep it, but ingestion stores the RAW string — normalize re-derives
// the typed ID at call site #2 (AD-06: one impl, two sites). The struct-wrap means
// there is no other way to mint a PlayerID, so a downstream int conversion cannot
// compile.
func ValidatePlayerID(raw string) (playerid.PlayerID, error) {
	id, err := playerid.New(raw)
	if err != nil {
		return playerid.PlayerID{}, fmt.Errorf("ingestion: %w", err)
	}
	return id, nil
}

// Team-aggregate ID block. MFL encodes team/positional aggregate "players"
// (team defenses, TMQB, ST, …) in a reserved numeric ID range, documented in
// docs/data-layer/MFL_API_Reference.md. We drop them at ingestion so they never
// reach normalization or the engine. NOTE: the rosters endpoint carries no
// position field, so position-based aggregate filtering (Coach, PN, TMWR, …)
// happens at B3 after the players-cache join — here we filter by ID range only.
const (
	teamAggregateLowID  = 151
	teamAggregateHighID = 782
)

// IsTeamAggregateID reports whether a raw MFL player ID falls inside the reserved
// team-aggregate block (0151–0782). A non-numeric ID is not an aggregate; it is
// caught later by ValidatePlayerID.
func IsTeamAggregateID(raw string) bool {
	n, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		return false
	}
	return n >= teamAggregateLowID && n <= teamAggregateHighID
}

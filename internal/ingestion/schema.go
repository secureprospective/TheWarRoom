// Package ingestion is Layer 1: fetchers that pull raw data and validate its shape. They
// transform nothing; normalize builds domain types. depguard forbids importing engine, store,
// transactions or database/sql from here.
//
// External sources add fields without versioning, so decoders tolerate unknown fields and
// Validate asserts the fields actually used; rejecting extras would turn a harmless upstream
// addition into an outage.
//
// This root package holds the helpers fetchers share; each fetcher is a subpackage.
package ingestion

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/playerid"
)

// LeagueID is the league this app serves. The host is never hardcoded (mfl.Client.DiscoverHost
// finds it), and neither is the season (league.Discover reads it from MFL).
const LeagueID = "14432"

// LeagueFirstSeason is the league's first season on MFL: its scoring history starts here.
const LeagueFirstSeason = 2021

// ValidatePlayerID rejects a malformed MFL id at the boundary. Fetchers keep the raw string;
// normalize re-derives the PlayerID.
func ValidatePlayerID(raw string) (playerid.PlayerID, error) {
	id, err := playerid.New(raw)
	if err != nil {
		return playerid.PlayerID{}, fmt.Errorf("ingestion: %w", err)
	}
	return id, nil
}

// MFL reserves an id range for team and positional aggregates (team defenses, TMQB, ST;
// docs/data-layer/MFL_API_Reference.md). They are dropped here by id; aggregates outside the
// range are caught by position in normalize, because rosters carry no position.
const (
	teamAggregateLowID  = 151
	teamAggregateHighID = 782
)

// IsTeamAggregateID reports an id in the reserved range 0151–0782. A non-numeric id is left
// for ValidatePlayerID.
func IsTeamAggregateID(raw string) bool {
	n, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		return false
	}
	return n >= teamAggregateLowID && n <= teamAggregateHighID
}

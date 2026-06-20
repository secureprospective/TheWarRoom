// Package crosswalk is the Layer 1 fetcher for the MFL-ID -> gsis_id crosswalk —
// the foundation leaf of B2b-Fetch-Offense. Every nflverse scouting source keys on
// nflverse's gsis_id, but scouting.Profile is keyed by the MFL PlayerID; without a
// crosswalk none of those signals can attach to a rostered player. The nflverse
// players.csv does NOT carry an mfl_id column (verified live, 2026-06-19), so the
// map is sourced from dynastyprocess db_playerids.csv, which carries both ids.
//
// DIRECTION: the map is keyed by MFL PlayerID, valued by gsis_id. This is the join
// the engine actually performs — for each ROSTERED MFL id, find where its nflverse
// stats live (by gsis) — and it is the side that is unique. gsis->MFL is genuinely
// one-to-many (MFL keeps duplicate player records; found live: gsis 00-0031320 ->
// mfl 12459 AND 12571), so an mfl-keyed map is correct where a gsis-keyed one would
// falsely collide. A single MFL id resolving to two different gsis IS a real
// integrity error and fails loud.
//
// The external-HTTP-CSV plumbing (fetch, byte cap, by-name column binding, the "NA"
// missing-cell sentinel) lives in the shared ingestion helpers (extcsv.go); this
// fetcher owns only the crosswalk-specific shape: its two columns, the MFL->gsis
// map type, and the conflict/empty integrity checks.
package crosswalk

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
)

// SourceURL is the dynastyprocess maintained player-id crosswalk (raw CSV). It is
// a data-source address — not a secret, not an environment-specific host — so it
// is a documented constant; Fetch takes the URL as an argument so tests can point
// at a fixture server and a future config can override the source.
const SourceURL = "https://raw.githubusercontent.com/dynastyprocess/data/master/files/db_playerids.csv"

// The two columns this fetcher depends on, located by NAME in the header (the
// source carries 30+ columns whose order is not guaranteed).
const (
	colMFLID = "mfl_id"
	colGSIS  = "gsis_id"
)

// errEmpty guards a crosswalk that resolved zero entries. The map is never
// legitimately empty (the source lists tens of thousands of players); an empty map
// would make every downstream MFL->gsis join miss silently — every player
// "unknown" — so we surface it loudly instead of returning an empty Map.
var errEmpty = errors.New("crosswalk: source resolved zero MFL->gsis entries")

// Map is the resolved MFL PlayerID -> gsis_id crosswalk. The backing map is
// unexported so a caller cannot construct a half-built crosswalk or mutate one;
// the only way to obtain a Map is Fetch, and the only way to read it is Lookup.
type Map struct {
	byMFL map[playerid.PlayerID]string
}

// Lookup returns the nflverse gsis_id for an MFL PlayerID and whether it was
// present. A missing id is an ordinary miss — not every MFL player maps to an
// nflverse gsis (e.g. commissioner-created players) — not an error; the caller
// decides whether an unmapped player is skippable.
func (m Map) Lookup(id playerid.PlayerID) (string, bool) {
	gsis, ok := m.byMFL[id]
	return gsis, ok
}

// Len reports the number of resolved crosswalk entries, for callers/tests to
// sanity-check coverage against the source size.
func (m Map) Len() int { return len(m.byMFL) }

// Fetch retrieves the crosswalk CSV from url using client and returns the resolved
// MFL->gsis Map. client and url are injected (not package globals) so the fetcher
// is unit-testable against a fixture server and survives a source move; pass
// SourceURL and an http.Client with a timeout for production use. Rows missing
// either id (empty or "NA") are skipped as ordinary non-matches; a present-but-
// malformed MFL id fails loud (RISK-003, via ingestion.ValidatePlayerID); a single
// MFL id mapping to two different gsis is a source-integrity error and fails loud.
func Fetch(ctx context.Context, client *http.Client, url string) (Map, error) {
	records, err := ingestion.FetchCSV(ctx, client, url, ingestion.DefaultMaxCSVBytes)
	if err != nil {
		return Map{}, fmt.Errorf("crosswalk: %w", err)
	}
	if len(records) == 0 {
		return Map{}, fmt.Errorf("crosswalk: %q returned no rows (not even a header)", url)
	}

	cols, err := ingestion.CSVColumns(records[0], colMFLID, colGSIS)
	if err != nil {
		return Map{}, fmt.Errorf("crosswalk: %w", err)
	}
	mflIdx, gsisIdx := cols[colMFLID], cols[colGSIS]

	out := make(map[playerid.PlayerID]string)
	for _, rec := range records[1:] {
		gsis := strings.TrimSpace(rec[gsisIdx])
		rawMFL := strings.TrimSpace(rec[mflIdx])
		if ingestion.IsMissing(gsis) || ingestion.IsMissing(rawMFL) {
			continue // player exists in only one of the two id systems — not a match
		}

		id, err := ingestion.ValidatePlayerID(rawMFL)
		if err != nil {
			return Map{}, fmt.Errorf("crosswalk: mfl_id %q: %w", rawMFL, err)
		}

		if existing, dup := out[id]; dup && existing != gsis {
			return Map{}, fmt.Errorf("crosswalk: MFL id %s maps to conflicting gsis %q and %q",
				id.String(), existing, gsis)
		}
		out[id] = gsis
	}

	if len(out) == 0 {
		return Map{}, errEmpty
	}
	return Map{byMFL: out}, nil
}

// Package crosswalk is the Layer 1 fetcher for the MFL-ID -> gsis_id crosswalk —
// the foundation leaf of B2b-Fetch-Offense. Every nflverse scouting source keys on
// nflverse's gsis_id, but scouting.Profile is keyed by the MFL PlayerID; without a
// crosswalk none of those signals can attach to a rostered player. The nflverse
// players.csv does NOT carry an mfl_id column (verified live, 2026-06-19), so the
// map is sourced from dynastyprocess db_playerids.csv, which carries both ids.
//
// DIRECTION: the map is keyed by MFL PlayerID, valued by gsis_id. This is the
// join the engine actually performs — for each ROSTERED MFL id, find where its
// nflverse stats live (by gsis) — and it is the side that is unique. gsis->MFL is
// genuinely one-to-many (MFL keeps duplicate player records; found live: gsis
// 00-0031320 -> mfl 12459 AND 12571), so an mfl-keyed map is correct where a
// gsis-keyed one would falsely collide. A single MFL id resolving to two different
// gsis IS a real integrity error and fails loud.
//
// This is NOT an MFL endpoint: it is a static external CSV fetched over plain
// net/http, so it does not use mfl.Client (no host discovery, no MFL rate limit,
// no MFL error envelope). It still inherits the ingestion discipline — parse at
// the boundary, fail loud on a malformed in-band id, reject an empty result — and
// it lives under internal/ingestion/ so the layer1-no-upward-import depguard rule
// covers it for free (a Layer-1 fetcher must never import the engine/stores/etc.).
//
// FIRST INSTANCE of the external-HTTP-CSV pattern (Codex M17): the nflverse offense
// fetchers that follow clone this transport + by-name column-binding shape. Review
// it before they inherit it.
package crosswalk

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
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

// maxBodyBytes caps the response read so a redirected, runaway, or hostile source
// cannot exhaust memory at the boundary (Codex M15). db_playerids.csv is a few MB;
// 64 MiB is ample headroom while still bounding the read.
const maxBodyBytes = 64 << 20

// The two columns this fetcher depends on, located by NAME in the header. The
// source carries 30+ columns whose order is not guaranteed — binding by position
// is exactly the fragility that bit the source recon, so we bind by name and fail
// loud if either column disappears.
const (
	colMFLID = "mfl_id"
	colGSIS  = "gsis_id"
)

// naSentinel is how the R-generated source encodes a missing cell: data.table /
// readr export a missing value as the literal string "NA", not an empty field
// (found live — multiple rows carry gsis_id "NA" with different mfl_ids). Both an
// empty cell and "NA" mean "this player has no id in that system" → not a match.
const naSentinel = "NA"

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
// gsis->MFL Map. client and url are injected (not package globals) so the fetcher
// is unit-testable against a fixture server and survives a source move; pass
// SourceURL and an http.Client with a timeout for production use. ctx bounds both
// the request and the parse.
func Fetch(ctx context.Context, client *http.Client, url string) (Map, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return Map{}, fmt.Errorf("crosswalk: build request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return Map{}, fmt.Errorf("crosswalk: fetch %s: %w", url, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return Map{}, fmt.Errorf("crosswalk: unexpected status %d from %s", resp.StatusCode, url)
	}

	return parse(ctx, io.LimitReader(resp.Body, maxBodyBytes))
}

// parse reads the crosswalk CSV, binds the two depended-on columns by name, and
// builds the MFL->gsis map. Rows missing either id are skipped as ordinary
// non-matches; a present-but-malformed MFL id fails loud (RISK-003, via
// ingestion.ValidatePlayerID); a single MFL id mapping to two different gsis_ids is
// a source-integrity error and also fails loud. csv.Reader enforces a constant
// field count, so a short/long row errors here rather than producing a bad index.
func parse(ctx context.Context, r io.Reader) (Map, error) {
	cr := csv.NewReader(r)

	header, err := cr.Read()
	if err != nil {
		return Map{}, fmt.Errorf("crosswalk: read header: %w", err)
	}
	// Strip a leading UTF-8 BOM from the first header cell — an invisible byte
	// from some CSV exporters would otherwise make "mfl_id" fail to match (the
	// invisible-byte class, Technical pillar Debugging Discipline).
	if len(header) > 0 {
		header[0] = strings.TrimPrefix(header[0], "\ufeff")
	}

	mflIdx, err := columnIndex(header, colMFLID)
	if err != nil {
		return Map{}, err
	}
	gsisIdx, err := columnIndex(header, colGSIS)
	if err != nil {
		return Map{}, err
	}

	out := make(map[playerid.PlayerID]string)
	for {
		select {
		case <-ctx.Done():
			return Map{}, fmt.Errorf("crosswalk: parse cancelled: %w", ctx.Err())
		default:
		}

		rec, err := cr.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return Map{}, fmt.Errorf("crosswalk: read row: %w", err)
		}

		gsis := strings.TrimSpace(rec[gsisIdx])
		rawMFL := strings.TrimSpace(rec[mflIdx])
		if isMissing(gsis) || isMissing(rawMFL) {
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

// columnIndex returns the position of the named column in the header, or an error
// naming the missing column. Binding by name then validating presence is the
// boundary's shape check: it tolerates new or reordered columns in the external
// file but fails loud the moment a column we depend on disappears.
func columnIndex(header []string, name string) (int, error) {
	for i, h := range header {
		if strings.TrimSpace(h) == name {
			return i, nil
		}
	}
	return 0, fmt.Errorf("crosswalk: source missing required column %q", name)
}

// isMissing reports whether a cell is absent: an empty field or the R "NA"
// sentinel the source uses for a player with no id in that system.
func isMissing(s string) bool { return s == "" || s == naSentinel }

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
// SECOND BRIDGE — espn_id -> gsis_id. The same db_playerids.csv also carries an
// espn_id column, and CFBD's player endpoints key on that espn-style athlete id
// (verified live 2026-06-21: CFBD playerId 4431611 = Caleb Williams = espn_id ->
// mfl 16579 / gsis 00-0039918). collegeshare (a college-production signal for
// rookies the MFL->gsis map can't reach directly) needs to attach CFBD player ids
// to a gsis. Rather than download the file twice, this one Fetch builds BOTH maps in
// a single pass. The espn_id column is OPTIONAL here, not required: the crosswalk's
// foundation job is MFL->gsis, and a future source that drops espn_id must NOT break
// that foundation — so a missing espn_id column yields an empty espn bridge, and the
// fail-loud-on-empty check lives in collegeshare (where the dependency is real), not
// here. An espn id resolving to two different gsis is AMBIGUOUS source data, not a
// hard error: live, 4 of ~7900 espn ids carry two distinct players' gsis (a
// dynastyprocess id-assignment slip — espn 2582138 is tagged to both Kyle Carter and
// David Morgan). Mirroring the RAS combine pfr-collision policy (Christopher,
// 2026-06-21 — drop-ambiguous-and-continue, NOT fail-loud, NOT first-wins), the
// ambiguous espn id is dropped from the bridge entirely so collegeshare gets a clean
// miss rather than a silently-wrong gsis. The MFL side stays fail-loud: it is the
// foundation, has zero live conflicts, and a conflict there would be real corruption.
//
// The external-HTTP-CSV plumbing (fetch, byte cap, by-name column binding, the "NA"
// missing-cell sentinel) lives in the shared ingestion helpers (extcsv.go); this
// fetcher owns only the crosswalk-specific shape: its columns, the two map types,
// and the conflict/empty integrity checks.
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

// The columns this fetcher reads, located by NAME in the header (the source carries
// 30+ columns whose order is not guaranteed). mfl_id and gsis_id are required;
// espn_id is optional (see the package doc — it feeds the secondary espn->gsis
// bridge and must not be allowed to break the foundation MFL->gsis map).
const (
	colMFLID = "mfl_id"
	colGSIS  = "gsis_id"
	colESPN  = "espn_id"
)

// errEmpty guards a crosswalk that resolved zero entries. The map is never
// legitimately empty (the source lists tens of thousands of players); an empty map
// would make every downstream MFL->gsis join miss silently — every player
// "unknown" — so we surface it loudly instead of returning an empty Map.
var errEmpty = errors.New("crosswalk: source resolved zero MFL->gsis entries")

// Map is the resolved crosswalk: an MFL PlayerID -> gsis_id map (the foundation) and
// an espn_id -> gsis_id bridge (for CFBD-keyed signals). Both backing maps are
// unexported so a caller cannot construct a half-built crosswalk or mutate one; the
// only way to obtain a Map is Fetch, and the only way to read it is the Lookup
// accessors.
type Map struct {
	byMFL  map[playerid.PlayerID]string
	byESPN map[string]string
}

// Lookup returns the nflverse gsis_id for an MFL PlayerID and whether it was
// present. A missing id is an ordinary miss — not every MFL player maps to an
// nflverse gsis (e.g. commissioner-created players) — not an error; the caller
// decides whether an unmapped player is skippable.
func (m Map) Lookup(id playerid.PlayerID) (string, bool) {
	gsis, ok := m.byMFL[id]
	return gsis, ok
}

// GSISForESPN returns the nflverse gsis_id for a CFBD/ESPN athlete id and whether it
// was present. CFBD's player endpoints key on this espn-style id; bridging it to gsis
// lets a college-production signal attach to the same gsis the engine joins every
// other offense signal on. A miss is ordinary (not every espn id is in the source,
// and a college-only player never drafted has no gsis) — not an error.
func (m Map) GSISForESPN(espnID string) (string, bool) {
	gsis, ok := m.byESPN[espnID]
	return gsis, ok
}

// Len reports the number of resolved MFL->gsis entries, for callers/tests to
// sanity-check coverage against the source size.
func (m Map) Len() int { return len(m.byMFL) }

// LenESPN reports the number of resolved espn->gsis entries, for callers/tests to
// sanity-check the bridge coverage (collegeshare fails loud if it is implausibly low).
func (m Map) LenESPN() int { return len(m.byESPN) }

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

	// CSVColumns strips a BOM from records[0][0] in place and fails loud if either
	// required column is absent. Run it FIRST, then locate the optional espn_id
	// against the now-BOM-stripped header.
	cols, err := ingestion.CSVColumns(records[0], colMFLID, colGSIS)
	if err != nil {
		return Map{}, fmt.Errorf("crosswalk: %w", err)
	}
	mflIdx, gsisIdx := cols[colMFLID], cols[colGSIS]
	espnIdx := optionalColumn(records[0], colESPN) // -1 if the source omits espn_id

	byMFL := make(map[playerid.PlayerID]string)
	byESPN := make(map[string]string)
	poisonedESPN := make(map[string]bool) // espn ids dropped for resolving to 2+ gsis
	for _, rec := range records[1:] {
		gsis := strings.TrimSpace(rec[gsisIdx])
		if ingestion.IsMissing(gsis) {
			continue // no gsis: this row feeds neither bridge (both target gsis)
		}

		if err := addMFL(byMFL, strings.TrimSpace(rec[mflIdx]), gsis); err != nil {
			return Map{}, err
		}
		if espnIdx >= 0 {
			addESPN(byESPN, poisonedESPN, strings.TrimSpace(rec[espnIdx]), gsis)
		}
	}

	if len(byMFL) == 0 {
		return Map{}, errEmpty
	}
	return Map{byMFL: byMFL, byESPN: byESPN}, nil
}

// optionalColumn returns the index of name in header, or -1 if absent. Unlike
// ingestion.CSVColumns it never errors — it is for a column the crosswalk reads when
// present but does not require (espn_id; see the package doc).
func optionalColumn(header []string, name string) int {
	for i, h := range header {
		if strings.TrimSpace(h) == name {
			return i
		}
	}
	return -1
}

// addMFL inserts an MFL->gsis entry, skipping a missing/NA mfl id (the player exists
// in only one id system — not a match), failing loud on a malformed id (RISK-003) and
// on a single mfl id mapping to two different gsis.
func addMFL(byMFL map[playerid.PlayerID]string, rawMFL, gsis string) error {
	if ingestion.IsMissing(rawMFL) {
		return nil
	}
	id, err := ingestion.ValidatePlayerID(rawMFL)
	if err != nil {
		return fmt.Errorf("crosswalk: mfl_id %q: %w", rawMFL, err)
	}
	if existing, dup := byMFL[id]; dup && existing != gsis {
		return fmt.Errorf("crosswalk: MFL id %s maps to conflicting gsis %q and %q",
			id.String(), existing, gsis)
	}
	byMFL[id] = gsis
	return nil
}

// addESPN inserts an espn->gsis entry. A missing/NA espn id is skipped. An espn id
// that resolves to two DIFFERENT gsis is ambiguous source data (live: 4 of ~7900
// espn ids tag two distinct players) — per the RAS combine-collision precedent it is
// dropped from the bridge entirely and recorded as poisoned so a later row cannot
// resurrect it, giving collegeshare a clean miss rather than a silently-wrong gsis. An
// identical repeat (same espn -> same gsis, from MFL's duplicate records) is deduped.
func addESPN(byESPN map[string]string, poisoned map[string]bool, espn, gsis string) {
	if ingestion.IsMissing(espn) || poisoned[espn] {
		return
	}
	if existing, dup := byESPN[espn]; dup && existing != gsis {
		delete(byESPN, espn)
		poisoned[espn] = true
		return
	}
	byESPN[espn] = gsis
}

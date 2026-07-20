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
// THIRD BRIDGE — pfr_id -> gsis_id. db_playerids.csv also carries a pfr_id column
// (Pro-Football-Reference's player id), the key three nflverse scouting sources are
// joined on: snap_counts (touchshare), combine (ras), and pfr advanced defense
// (pfrcoverage). Each of those fetchers used to build its OWN pfr->gsis map from this
// same file in a duplicated live-test helper (Codex M17); the bridge is promoted here
// so the file is read once and the dedup policy is defined in one place. Like espn_id
// it is OPTIONAL (a future source dropping it must not break the MFL->gsis foundation)
// and AMBIGUOUS (live: 3 of ~7800 pfr ids tag two distinct players' gsis — CartKy01,
// HarrAl00, MillSt00). Mirroring the espn/RAS drop-ambiguous precedent, such a pfr id
// is dropped from the bridge entirely (a clean miss beats a silently-wrong gsis),
// replacing the old helpers' silent last-write-wins. Consumers take the bridge as a
// map[string]string via PFRMap() (a defensive copy), matching their existing
// per-row-lookup shape.
//
// FOURTH BRIDGE — (name, birthdate) -> gsis_id. The Madden feed (ingestion/madden)
// carries no gsis/espn/pfr id — only a player's name, team, position, and birthdate —
// so its records are keyed to a TheWarRoom player by a normalized name + birthdate
// match (birthdate disambiguates same-name players). db_playerids.csv carries a name
// and a birthdate column alongside gsis_id, so the same one-file read that builds the
// three id bridges also indexes (normalized name | ISO birthdate) -> gsis. This is
// promoted here from a duplicated live-test helper for the same reason as PFRMap
// (Codex M17: read the file once, define the match in one place). Like the id bridges
// the columns are OPTIONAL (a future source dropping them must not break the MFL->gsis
// foundation) and the index is drop-ambiguous: a (name|birth) key resolving to two
// different gsis is dropped rather than guessed. Consumers take it as a GSISResolver-
// shaped closure via MaddenResolver().
//
// The external-HTTP-CSV plumbing (fetch, byte cap, by-name column binding, the "NA"
// missing-cell sentinel) lives in the shared ingestion helpers (extcsv.go); this
// fetcher owns only the crosswalk-specific shape: its columns, the map types, and the
// conflict/empty integrity checks.
package crosswalk

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

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
	colPFR   = "pfr_id"
	colName  = "name"      // optional — feeds the (name|birth)->gsis Madden resolver
	colBirth = "birthdate" // optional — feeds the (name|birth)->gsis Madden resolver
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
	byMFL       map[playerid.PlayerID]string
	byESPN      map[string]string
	byPFR       map[string]string
	byNameBirth map[string]string // (normName|isoBirth) -> gsis, for the Madden resolver
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

// PFRMap returns the pfr_id -> gsis_id bridge as a fresh map the caller owns. It is a
// defensive copy so a consumer cannot mutate the crosswalk's internal state; the
// pfr-keyed scouting fetchers (touchshare, ras, pfrcoverage) take this map and resolve
// each row's pfr_id through it. A copy is cheap relative to the network fetch that
// built the Map and is requested once per pipeline run.
func (m Map) PFRMap() map[string]string {
	out := make(map[string]string, len(m.byPFR))
	for k, v := range m.byPFR {
		out[k] = v
	}
	return out
}

// LenPFR reports the number of resolved pfr->gsis entries, for callers/tests to
// sanity-check bridge coverage against the source size.
func (m Map) LenPFR() int { return len(m.byPFR) }

// MaddenResolver returns a GSISResolver-shaped closure that maps a Madden player's raw
// name + birthdate to an nflverse gsis_id (ok=false on a miss). ingestion/madden takes
// exactly this func to key its otherwise-idless records; building it here (from the same
// db_playerids read that built the id bridges) keeps the brittle name+birthdate match
// defined in one place. The closure reads the crosswalk's internal index but cannot
// mutate it (the map is captured read-only through the closure), so no defensive copy is
// needed. An empty index (the source omitted the name/birthdate columns) yields a
// resolver that always misses — the caller's madden.Fetch then fails loud on zero
// resolved records, surfacing the gap rather than silently attaching nothing.
func (m Map) MaddenResolver() func(fullName, birthdate string) (string, bool) {
	index := m.byNameBirth
	return func(fullName, birthdate string) (string, bool) {
		g, ok := index[nameBirthKey(fullName, birthdate)]
		return g, ok
	}
}

// LenMaddenResolver reports the number of resolved (name|birth)->gsis entries, for
// callers/tests to sanity-check resolver coverage against the source size.
func (m Map) LenMaddenResolver() int { return len(m.byNameBirth) }

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
	espnIdx := optionalColumn(records[0], colESPN)   // -1 if the source omits espn_id
	pfrIdx := optionalColumn(records[0], colPFR)     // -1 if the source omits pfr_id
	nameIdx := optionalColumn(records[0], colName)   // -1 if the source omits name
	birthIdx := optionalColumn(records[0], colBirth) // -1 if the source omits birthdate

	byMFL := make(map[playerid.PlayerID]string)
	byESPN := make(map[string]string)
	byPFR := make(map[string]string)
	byNameBirth := make(map[string]string)
	poisonedESPN := make(map[string]bool)      // espn ids dropped for resolving to 2+ gsis
	poisonedPFR := make(map[string]bool)       // pfr ids dropped for resolving to 2+ gsis
	poisonedNameBirth := make(map[string]bool) // name|birth keys dropped for resolving to 2+ gsis
	for _, rec := range records[1:] {
		gsis := strings.TrimSpace(rec[gsisIdx])
		if ingestion.IsMissing(gsis) {
			continue // no gsis: this row feeds no bridge (all four target gsis)
		}

		if err := addMFL(byMFL, strings.TrimSpace(rec[mflIdx]), gsis); err != nil {
			return Map{}, err
		}
		if espnIdx >= 0 {
			addBridge(byESPN, poisonedESPN, strings.TrimSpace(rec[espnIdx]), gsis)
		}
		if pfrIdx >= 0 {
			addBridge(byPFR, poisonedPFR, strings.TrimSpace(rec[pfrIdx]), gsis)
		}
		if nameIdx >= 0 && birthIdx >= 0 {
			name, birth := normName(rec[nameIdx]), isoBirth(rec[birthIdx])
			if name != "" && birth != "" { // birthdate is the disambiguator — never a partial key
				addBridge(byNameBirth, poisonedNameBirth, name+"|"+birth, gsis)
			}
		}
	}

	if len(byMFL) == 0 {
		return Map{}, errEmpty
	}
	return Map{byMFL: byMFL, byESPN: byESPN, byPFR: byPFR, byNameBirth: byNameBirth}, nil
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

// nameBirthKey builds the Madden resolver's lookup key from a raw name + birthdate,
// applying the same normalization the index was built with (so a query and an indexed
// row that describe the same player collide). It is the read-side mirror of the loop in
// Fetch. A key whose name or birth is empty after normalization can never match an
// indexed entry (those require both non-empty), so it is a guaranteed miss.
func nameBirthKey(fullName, birthdate string) string {
	return normName(fullName) + "|" + isoBirth(birthdate)
}

var nonAlpha = regexp.MustCompile(`[^a-z]`)

// normName lowercases and strips every non-letter (spaces, punctuation, Jr./III, apostrophes)
// so "T.J. Watt" and "TJ Watt" collide. Both the EA name and the db_playerids name pass
// through it.
func normName(s string) string { return nonAlpha.ReplaceAllString(strings.ToLower(s), "") }

// isoBirth normalizes both EA's M/D/YYYY and db_playerids' ISO YYYY-MM-DD to YYYY-MM-DD,
// or "" if unparseable (an unparseable birthdate yields a guaranteed non-match, never a
// partial key).
func isoBirth(b string) string {
	for _, layout := range []string{"2006-01-02", "1/2/2006"} {
		if d, err := time.Parse(layout, strings.TrimSpace(b)); err == nil {
			return d.Format("2006-01-02")
		}
	}
	return ""
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

// addBridge inserts a secondary-id -> gsis entry into one of the optional bridges
// (espn or pfr — both share this drop-ambiguous policy, so the logic lives once;
// Codex M17). A missing/NA id is skipped. An id that resolves to two DIFFERENT gsis is
// ambiguous source data (live: 4 of ~7900 espn ids and 3 of ~7800 pfr ids tag two
// distinct players) — per the RAS combine-collision precedent it is dropped from the
// bridge entirely and recorded as poisoned so a later row cannot resurrect it, giving
// the consumer a clean miss rather than a silently-wrong gsis. An identical repeat
// (same id -> same gsis, from MFL's duplicate records) is deduped.
func addBridge(bridge map[string]string, poisoned map[string]bool, id, gsis string) {
	if ingestion.IsMissing(id) || poisoned[id] {
		return
	}
	if existing, dup := bridge[id]; dup && existing != gsis {
		delete(bridge, id)
		poisoned[id] = true
		return
	}
	bridge[id] = gsis
}

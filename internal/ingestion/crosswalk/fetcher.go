// Package crosswalk maps MFL player ids to nflverse gsis ids, which every nflverse source keys
// on. nflverse's players.csv has no mfl_id (verified 2026-06-19), so the source is
// DynastyProcess db_playerids.csv. One read builds four indexes:
//
//   - MFL -> gsis, the foundation. Keyed by MFL id because gsis -> MFL is one-to-many (MFL
//     keeps duplicate records: gsis 00-0031320 is mfl 12459 and 12571). One MFL id with two
//     gsis is corruption and fails the fetch.
//   - espn -> gsis, for CFBD player ids (CFBD playerId is the espn id).
//   - pfr -> gsis, for snap counts, combine and PFR advanced defense.
//   - (name, birthdate) -> gsis, for Madden, which carries no id.
//
// The last three read optional columns, so a source dropping one cannot break the MFL map,
// and they drop any key that resolves to two different gsis (live: 4 of ~7900 espn ids, 3 of
// ~7800 pfr ids). A clean miss beats a mis-attributed player.
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

// SourceURL is the DynastyProcess crosswalk. Fetch takes the URL so tests can use a fixture.
const SourceURL = "https://raw.githubusercontent.com/dynastyprocess/data/master/files/db_playerids.csv"

// Columns are bound by name; mfl_id and gsis_id are required, the rest optional.
const (
	colMFLID = "mfl_id"
	colGSIS  = "gsis_id"
	colESPN  = "espn_id"
	colPFR   = "pfr_id"
	colName  = "name"
	colBirth = "birthdate"
)

// errEmpty: the source lists tens of thousands of players, and an empty map would make every
// join miss silently.
var errEmpty = errors.New("crosswalk: source resolved zero MFL->gsis entries")

// Map is the resolved crosswalk. Its maps are unexported, so a Map only comes from Fetch and
// is read through the accessors.
type Map struct {
	byMFL       map[playerid.PlayerID]string
	byESPN      map[string]string
	byPFR       map[string]string
	byNameBirth map[string]string // (normName|isoBirth) -> gsis, for the Madden resolver
}

// Lookup returns the gsis id for an MFL id. A miss is ordinary (commissioner-created players
// have none).
func (m Map) Lookup(id playerid.PlayerID) (string, bool) {
	gsis, ok := m.byMFL[id]
	return gsis, ok
}

// GSISForESPN returns the gsis id for a CFBD/ESPN athlete id. A miss is ordinary.
func (m Map) GSISForESPN(espnID string) (string, bool) {
	gsis, ok := m.byESPN[espnID]
	return gsis, ok
}

// Len is the number of MFL -> gsis entries.
func (m Map) Len() int { return len(m.byMFL) }

// LenESPN is the number of espn -> gsis entries.
func (m Map) LenESPN() int { return len(m.byESPN) }

// PFRMap returns a copy of the pfr -> gsis index for the caller to own.
func (m Map) PFRMap() map[string]string {
	out := make(map[string]string, len(m.byPFR))
	for k, v := range m.byPFR {
		out[k] = v
	}
	return out
}

// LenPFR is the number of pfr -> gsis entries.
func (m Map) LenPFR() int { return len(m.byPFR) }

// MaddenResolver returns a closure mapping a raw name and birthdate to a gsis id, the key
// ingestion/madden needs. If the source lacked those columns it always misses, and madden
// fails on zero resolved records.
func (m Map) MaddenResolver() func(fullName, birthdate string) (string, bool) {
	index := m.byNameBirth
	return func(fullName, birthdate string) (string, bool) {
		g, ok := index[nameBirthKey(fullName, birthdate)]
		return g, ok
	}
}

// LenMaddenResolver is the number of (name, birthdate) -> gsis entries.
func (m Map) LenMaddenResolver() int { return len(m.byNameBirth) }

// Fetch reads the crosswalk CSV and builds the Map. A row missing either id is skipped; a
// malformed MFL id fails.
func Fetch(ctx context.Context, client *http.Client, url string) (Map, error) {
	records, err := ingestion.FetchCSV(ctx, client, url, ingestion.DefaultMaxCSVBytes)
	if err != nil {
		return Map{}, fmt.Errorf("crosswalk: %w", err)
	}
	if len(records) == 0 {
		return Map{}, fmt.Errorf("crosswalk: %q returned no rows (not even a header)", url)
	}

	// CSVColumns strips the BOM from the header in place, so find the optional columns after it.
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
			continue
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
			if name != "" && birth != "" {
				addBridge(byNameBirth, poisonedNameBirth, name+"|"+birth, gsis)
			}
		}
	}

	if len(byMFL) == 0 {
		return Map{}, errEmpty
	}
	return Map{byMFL: byMFL, byESPN: byESPN, byPFR: byPFR, byNameBirth: byNameBirth}, nil
}

// optionalColumn returns name's index in header, or -1.
func optionalColumn(header []string, name string) int {
	for i, h := range header {
		if strings.TrimSpace(h) == name {
			return i
		}
	}
	return -1
}

// nameBirthKey builds the resolver key with the same normalization as the index. An empty part
// can never match.
func nameBirthKey(fullName, birthdate string) string {
	return normName(fullName) + "|" + isoBirth(birthdate)
}

var nonAlpha = regexp.MustCompile(`[^a-z]`)

// normName lowercases and strips non-letters, so "T.J. Watt" matches "TJ Watt".
func normName(s string) string { return nonAlpha.ReplaceAllString(strings.ToLower(s), "") }

// isoBirth normalizes M/D/YYYY (EA) and YYYY-MM-DD to YYYY-MM-DD; unparseable gives "", a
// guaranteed miss.
func isoBirth(b string) string {
	for _, layout := range []string{"2006-01-02", "1/2/2006"} {
		if d, err := time.Parse(layout, strings.TrimSpace(b)); err == nil {
			return d.Format("2006-01-02")
		}
	}
	return ""
}

// addMFL adds an MFL -> gsis entry. A missing id is skipped; a malformed one, or one MFL id
// with two gsis, is an error.
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

// addBridge adds an entry to an optional index. An id seen with two different gsis is dropped
// and marked poisoned so a later row cannot restore it; an identical repeat is ignored.
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

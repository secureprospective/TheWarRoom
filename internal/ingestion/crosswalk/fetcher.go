// Package crosswalk maps MFL player ids to nflverse gsis ids, which every nflverse source keys
// on. nflverse's players.csv has no mfl_id (verified 2026-06-19), so the source is
// DynastyProcess db_playerids.csv. One read builds three indexes:
//
//   - MFL -> gsis, the foundation. Keyed by MFL id because gsis -> MFL is one-to-many (MFL
//     keeps duplicate records: gsis 00-0031320 is mfl 12459 and 12571). One MFL id with two
//     gsis is corruption and fails the fetch.
//   - pfr -> gsis, for snap counts, combine and PFR advanced defense.
//
// The pfr index reads an optional column, so a source dropping it cannot break the MFL map,
// and it drops any key that resolves to two different gsis (live: 3 of ~7800 pfr ids). A
// clean miss beats a mis-attributed player.
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

// SourceURL is the DynastyProcess crosswalk. Fetch takes the URL so tests can use a fixture.
const SourceURL = "https://raw.githubusercontent.com/dynastyprocess/data/master/files/db_playerids.csv"

// Columns are bound by name; mfl_id and gsis_id are required, the rest optional.
const (
	colMFLID = "mfl_id"
	colGSIS  = "gsis_id"
	colPFR   = "pfr_id"
	colName  = "name"
)

// errEmpty: the source lists tens of thousands of players, and an empty map would make every
// join miss silently.
var errEmpty = errors.New("crosswalk: source resolved zero MFL->gsis entries")

// Map is the resolved crosswalk. Its maps are unexported, so a Map only comes from Fetch and
// is read through the accessors.
type Map struct {
	byMFL   map[playerid.PlayerID]string
	byPFR   map[string]string
	entries map[string]Entry // canonical MFL id -> every row naming it, gsis or not
}

// Lookup returns the gsis id for an MFL id. A miss is ordinary (commissioner-created players
// have none).
func (m Map) Lookup(id playerid.PlayerID) (string, bool) {
	gsis, ok := m.byMFL[id]
	return gsis, ok
}

// PFRMap returns a copy of the pfr -> gsis index for the caller to own.
func (m Map) PFRMap() map[string]string {
	out := make(map[string]string, len(m.byPFR))
	for k, v := range m.byPFR {
		out[k] = v
	}
	return out
}

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
	idCols := idColumns(records[0])
	pfrIdx := optionalColumn(records[0], colPFR)   // -1 if the source omits pfr_id
	nameIdx := optionalColumn(records[0], colName) // -1 if the source omits name

	byMFL := make(map[playerid.PlayerID]string)
	byPFR, poisonedPFR := map[string]string{}, map[string]bool{}
	entries := make(map[string]Entry)
	for _, rec := range records[1:] {
		if err := addEntry(entries, rec, mflIdx, nameIdx, idCols); err != nil {
			return Map{}, err
		}
		gsis := strings.TrimSpace(rec[gsisIdx])
		if ingestion.IsMissing(gsis) {
			continue
		}

		if err := addMFL(byMFL, strings.TrimSpace(rec[mflIdx]), gsis); err != nil {
			return Map{}, err
		}
		if pfrIdx >= 0 {
			addBridge(byPFR, poisonedPFR, strings.TrimSpace(rec[pfrIdx]), gsis)
		}
	}

	if len(byMFL) == 0 {
		return Map{}, errEmpty
	}
	return Map{byMFL: byMFL, byPFR: byPFR, entries: entries}, nil
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

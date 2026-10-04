package crosswalk

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/ingestion"
)

// Entry is what DynastyProcess says about one MFL id: the player's name and every other id it
// carries, keyed by id type ("gsis", "espn", "pfr", …).
type Entry struct {
	Name string
	IDs  map[string]string
}

// Link maps one source id onto an MFL id: a row of the player directory.
type Link struct {
	IDType, IDValue, MFLID string
}

// Why an MFL id gets no links.
const (
	ReasonOtherPlayer = "DynastyProcess has this MFL id as another player"
	ReasonSharedGSIS  = "its NFL id is shared with another MFL id in DynastyProcess"
)

// Entry returns DynastyProcess's row for an MFL id, gsis or not.
func (m Map) Entry(mflID string) (Entry, bool) {
	e, ok := m.entries[mflID]
	return e, ok
}

// Links returns the player directory: every id DynastyProcess carries, pointed at its MFL id. A
// clean miss beats a mis-attributed player, so two guards apply. An id DynastyProcess ties to two
// MFL ids links to neither. And no id links to an MFL id that MFL lists under a different name:
// MFL reuses low ids for team units and commissioner-created players, which DynastyProcess still
// maps to the player who held the id first. mflName reports MFL's name for an id; an id MFL does
// not list (a retired player) links on DynastyProcess's word. rejected says why an MFL id got
// no gsis link when a guard stopped it.
func (m Map) Links(mflName func(mflID string) (string, bool)) (links []Link, rejected map[string]string) {
	owners := map[[2]string]int{}
	for _, e := range m.entries {
		for t, v := range e.IDs {
			owners[[2]string{t, v}]++
		}
	}
	rejected = map[string]string{}
	for _, mflID := range slices.Sorted(maps.Keys(m.entries)) {
		e := m.entries[mflID]
		if name, listed := mflName(mflID); listed && !sameName(name, e.Name) {
			rejected[mflID] = ReasonOtherPlayer
			continue
		}
		for _, t := range slices.Sorted(maps.Keys(e.IDs)) {
			v := e.IDs[t]
			if owners[[2]string{t, v}] > 1 {
				if t == idTypeGSIS {
					rejected[mflID] = ReasonSharedGSIS
				}
				continue
			}
			links = append(links, Link{IDType: t, IDValue: v, MFLID: mflID})
		}
	}
	return links, rejected
}

// idTypeGSIS is the id type of nflverse's gsis id, the key a match is measured on.
const idTypeGSIS = "gsis"

// idColumns maps each "<type>_id" column but mfl_id to its index, keyed by type.
func idColumns(header []string) map[string]int {
	out := map[string]int{}
	for i, h := range header {
		h = strings.TrimSpace(h)
		if t, ok := strings.CutSuffix(h, "_id"); ok && h != colMFLID && t != "" {
			out[t] = i
		}
	}
	return out
}

// addEntry records one row under its MFL id. A row without an MFL id is skipped; a malformed one
// fails the fetch, as addMFL does. Two rows for one MFL id merge, and an id type on which they
// disagree is dropped from the entry.
func addEntry(entries map[string]Entry, rec []string, mflIdx, nameIdx int, idCols map[string]int) error {
	raw := strings.TrimSpace(rec[mflIdx])
	if ingestion.IsMissing(raw) {
		return nil
	}
	id, err := ingestion.ValidatePlayerID(raw)
	if err != nil {
		return fmt.Errorf("crosswalk: mfl_id %q: %w", raw, err)
	}
	e, seen := entries[id.String()]
	if !seen {
		e = Entry{IDs: map[string]string{}}
		if nameIdx >= 0 {
			e.Name = strings.TrimSpace(rec[nameIdx])
		}
	}
	for t, i := range idCols {
		v := strings.TrimSpace(rec[i])
		if ingestion.IsMissing(v) {
			continue
		}
		if prev, dup := e.IDs[t]; dup && prev != v {
			delete(e.IDs, t)
			continue
		}
		e.IDs[t] = v
	}
	entries[id.String()] = e
	return nil
}

// sameName compares MFL's "Last, First" with DynastyProcess's "First Last": lowercased, letters
// only, generational suffixes dropped.
func sameName(mfl, dp string) bool {
	if last, first, ok := strings.Cut(mfl, ","); ok {
		mfl = first + " " + last
	}
	return nameKey(mfl) == nameKey(dp)
}

func nameKey(s string) string {
	var b strings.Builder
	for _, tok := range strings.Fields(strings.ToLower(s)) {
		tok = nonAlpha.ReplaceAllString(tok, "")
		switch tok {
		case "jr", "sr", "ii", "iii", "iv", "v":
			continue
		}
		b.WriteString(tok)
	}
	return b.String()
}

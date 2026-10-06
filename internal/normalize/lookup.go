// Package normalize turns raw MFL feeds into typed domain records by joining the rosters feed
// with the players database. It does no I/O and imports only ingestion, playerid and domain.
//
// Lookup indexes the players database by canonical id and classifies each position once, so a
// roster join is a map read.
package normalize

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/players"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
)

// lookupEntry is one player's resolved facts; raw MFL codes do not survive into it.
type lookupEntry struct {
	PlayerFacts
	isAggregate bool // "Def", "TMWR", …
}

// Lookup is the players database keyed by canonical player id.
type Lookup struct {
	byID map[string]lookupEntry
}

// entry returns the resolved player for a canonical id, and whether it exists.
func (l Lookup) entry(id playerid.PlayerID) (lookupEntry, bool) {
	e, ok := l.byID[id.String()]
	return e, ok
}

// NewLookup builds the index from the raw players feed, canonicalizing ids and classifying
// positions. MFL reserves ids 151–782 for team aggregates; an unrecognized record in that
// range is flagged for review and the build continues, so one odd id cannot halt the league.
func NewLookup(raws []players.RawPlayer) (Lookup, error) {
	posMap := newPositionMap()
	aggSet := newAggregateSet()
	byID := make(map[string]lookupEntry, len(raws))

	for _, rp := range raws {
		id, err := playerid.New(rp.ID)
		if err != nil {
			return Lookup{}, fmt.Errorf("normalize: players id %q: %w", rp.ID, err)
		}
		pos, isAgg := classifyPosition(rp.Position, posMap, aggSet)

		// The rosters fetcher already drops ids in the reserved range, so this flag is the
		// players-side signal that the reserved-range assumption slipped.
		if !isAgg && ingestion.IsTeamAggregateID(rp.ID) {
			pos = domain.PosFlag
		}

		entry := lookupEntry{
			PlayerFacts: PlayerFacts{
				Name:     rp.Name,
				Team:     rp.Team,
				Position: pos,
				IsRookie: rp.Status == "R",
				College:  strings.TrimSpace(rp.College),
			},
			isAggregate: isAgg,
		}
		// The fetcher validated the birthdate; absent stays absent and the consumer decides.
		if bd := strings.TrimSpace(rp.Birthdate); bd != "" {
			v, perr := strconv.ParseInt(bd, 10, 64)
			if perr != nil {
				return Lookup{}, fmt.Errorf("normalize: players id %q birthdate %q: %w", rp.ID, rp.Birthdate, perr)
			}
			entry.Birthdate, entry.HasBirthdate = v, true
		}
		// MFL sends "0" for undrafted, so only a positive year counts. Placeholder years like 1970
		// are the consumer's to judge.
		if dy := strings.TrimSpace(rp.DraftYear); dy != "" {
			v, perr := strconv.Atoi(dy)
			if perr != nil {
				return Lookup{}, fmt.Errorf("normalize: players id %q draft year %q: %w", rp.ID, rp.DraftYear, perr)
			}
			if v > 0 {
				entry.DraftYear, entry.HasDraftYear = v, true
			}
		}
		byID[id.String()] = entry
	}
	return Lookup{byID: byID}, nil
}

// PlayerFacts is the players-DB fact set the M1 runner reads per rostered id. It carries no
// contract fields: contract state lives in the state store and must not be copied from a
// static feed.
type PlayerFacts struct {
	Name         string
	Team         string
	Position     domain.Position
	IsRookie     bool
	Birthdate    int64  // epoch seconds
	HasBirthdate bool   // commissioner-created players lack one
	DraftYear    int    // the §6 experience source
	HasDraftYear bool   // false for undrafted ("0")
	College      string // raw; the SchoolTier source
}

// Facts resolves a canonical id. ok is false for an unknown id or a team aggregate, which
// callers treat the same.
func (l Lookup) Facts(id string) (PlayerFacts, bool) {
	pid, err := playerid.New(id)
	if err != nil {
		return PlayerFacts{}, false
	}
	e, ok := l.entry(pid)
	if !ok || e.isAggregate {
		return PlayerFacts{}, false
	}
	return e.PlayerFacts, true
}

// Name returns MFL's name for any id it lists, team aggregates included: the crosswalk checks a
// reused id against it.
func (l Lookup) Name(id string) (string, bool) {
	pid, err := playerid.New(id)
	if err != nil {
		return "", false
	}
	e, ok := l.entry(pid)
	return e.Name, ok
}

// IDs returns every player's canonical id, team aggregates excluded, sorted.
func (l Lookup) IDs() []string {
	out := make([]string, 0, len(l.byID))
	for id, e := range l.byID {
		if !e.isAggregate {
			out = append(out, id)
		}
	}
	slices.Sort(out)
	return out
}

// classifyPosition maps a raw MFL code onto the engine set. An aggregate returns true so the
// join can reject it; "XX" and unknown codes become PosFlag.
func classifyPosition(raw string, posMap map[string]domain.Position, aggSet map[string]struct{}) (domain.Position, bool) {
	code := strings.TrimSpace(raw)
	if _, ok := aggSet[code]; ok {
		return "", true
	}
	if pos, ok := posMap[code]; ok {
		return pos, false
	}
	return domain.PosFlag, false
}

// PositionFromMFL maps a raw MFL code using the same table as NewLookup, for callers without
// a Lookup (roster-limit enforcement). Aggregates and unknown codes return PosFlag; ok is false
// only for an aggregate.
func PositionFromMFL(raw string) (domain.Position, bool) {
	return classifyPosition(raw, newPositionMap(), newAggregateSet())
}

// newPositionMap is the MFL code -> engine position table. XX is absent on purpose so it
// becomes PosFlag.
func newPositionMap() map[string]domain.Position {
	return map[string]domain.Position{
		"QB":   domain.PosQB,
		"RB":   domain.PosRB,
		"WR":   domain.PosWR,
		"TE":   domain.PosTE,
		"PK":   domain.PosK, // MFL uses PK, engine uses K
		"DE":   domain.PosDE,
		"EDGE": domain.PosDE, // OQ-004: MFL labels edge rushers DE; no separate EDGE
		"DT":   domain.PosDT,
		"LB":   domain.PosLB,
		"CB":   domain.PosCB,
		"S":    domain.PosS,
	}
}

// newAggregateSet is the set of MFL team and positional aggregate codes, which are never
// players.
func newAggregateSet() map[string]struct{} {
	return map[string]struct{}{
		"PN":    {},
		"Coach": {},
		"Def":   {},
		"ST":    {},
		"Off":   {},
		"TMQB":  {},
		"TMRB":  {},
		"TMWR":  {},
		"TMTE":  {},
		"TMPK":  {},
		"TMPN":  {},
		"TMDL":  {},
		"TMLB":  {},
		"TMDB":  {},
	}
}

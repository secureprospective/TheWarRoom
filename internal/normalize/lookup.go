// Package normalize is the B3 Raw* → typed-domain boundary (WF 1C): it joins the
// rosters feed with the players database and produces domain.Roster / PlayerRecord.
// It is a PURE transformation — no I/O, no fetching, no scoring. It is Layer 1
// (depguard layer1-no-upward-import covers it): it imports ingestion + playerid +
// domain only, never store/engine/db.
//
// This file holds the cross-reference Lookup: the players database indexed by
// canonical player id, with each player's position classified ONCE at build time
// (so a roster join is a map read, not a re-classification per row). Building the
// Lookup is also where the reserved-range policy runs (B2 review item #1).
package normalize

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/players"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
)

// lookupEntry is one player's resolved facts from the players database: position
// already mapped to the engine set, rookie flag already derived. The raw MFL codes
// do not survive into here.
type lookupEntry struct {
	name         string
	position     domain.Position
	isAggregate  bool // a team/positional aggregate ("Def", "TMWR", …) — never rostered
	team         string
	isRookie     bool
	birthdate    int64  // epoch seconds, valid only when hasBirthdate
	hasBirthdate bool   // MFL DETAILS=1 birthdate present (commissioner-created players lack it)
	draftYear    int    // MFL DETAILS=1 draft year, valid only when hasDraftYear
	hasDraftYear bool   // a REAL draft year is present (the "0" undrafted sentinel is dropped here)
	college      string // MFL DETAILS=1 college name, raw (empty when absent); the SchoolTier source
}

// Lookup is the players database keyed by canonical player id (playerid form), the
// read side of the B3 cross-reference join.
type Lookup struct {
	byID map[string]lookupEntry
}

// entry returns the resolved player for a canonical id, and whether it exists.
func (l Lookup) entry(id playerid.PlayerID) (lookupEntry, bool) {
	e, ok := l.byID[id.String()]
	return e, ok
}

// NewLookup builds the cross-reference index from the raw players feed. It
// canonicalizes every id through playerid.New (RISK-003) and classifies each
// position once. It also applies the reserved-range policy (B2 review item #1):
// MFL's id range [151,782] is RESERVED for team aggregates, so a record there that
// we did NOT recognize as a known aggregate is an anomaly — either a real player the
// rosters ID-filter is wrongly dropping, or an unrecognized aggregate code. Such a
// record is FLAGGED for admin review and the build continues (Christopher, B3): one
// odd id must not halt the whole league's normalization.
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

		// Reserved-range policy (Christopher, B3): [151,782] is reserved for team
		// aggregates, so a NON-aggregate record there is an anomaly. FLAG it for
		// admin review and CONTINUE rather than halt the whole league's
		// normalization — one odd id must not block the pipeline. NOTE: the rosters
		// fetcher independently drops roster ids in this range, so such a record is
		// not currently reachable through a roster join; this flag is the
		// players-DB-side signal that the reserved-range assumption slipped.
		if !isAgg && ingestion.IsTeamAggregateID(rp.ID) {
			pos = domain.PosFlag
		}

		entry := lookupEntry{
			name:        rp.Name,
			position:    pos,
			isAggregate: isAgg,
			team:        rp.Team,
			isRookie:    rp.Status == "R",
			college:     strings.TrimSpace(rp.College),
		}
		// Birthdate (DETAILS=1) is typed HERE — the Raw→domain boundary. The fetcher
		// already validated a present value parses; absent stays absent (the M1
		// consumer owns the missing-birthdate policy, this join just carries the fact).
		if bd := strings.TrimSpace(rp.Birthdate); bd != "" {
			v, perr := strconv.ParseInt(bd, 10, 64)
			if perr != nil {
				return Lookup{}, fmt.Errorf("normalize: players id %q birthdate %q: %w", rp.ID, rp.Birthdate, perr)
			}
			entry.birthdate, entry.hasBirthdate = v, true
		}
		// Draft year (DETAILS=1) is typed HERE. MFL sends "0" for the undrafted/unknown — a
		// non-year — so only a positive value is a candidate draft year; its season-relative
		// plausibility (MFL's "1970" epoch placeholder, future years) is the §6 consumer's policy.
		if dy := strings.TrimSpace(rp.DraftYear); dy != "" {
			v, perr := strconv.Atoi(dy)
			if perr != nil {
				return Lookup{}, fmt.Errorf("normalize: players id %q draft year %q: %w", rp.ID, rp.DraftYear, perr)
			}
			if v > 0 {
				entry.draftYear, entry.hasDraftYear = v, true
			}
		}
		byID[id.String()] = entry
	}
	return Lookup{byID: byID}, nil
}

// PlayerFacts is the per-player DISPLAY/derivation fact set the M1 orchestrator
// reads for each rostered id: identity fields from the players DB plus the raw
// birthdate for age derivation. It is deliberately NOT domain.PlayerRecord — no
// roster/contract fields, because runtime contract state is B3c's job and must
// never be duplicated out of a static players-feed join.
type PlayerFacts struct {
	Name         string
	Position     domain.Position
	IsRookie     bool
	Birthdate    int64 // epoch seconds, valid only when HasBirthdate
	HasBirthdate bool
	DraftYear    int    // MFL draft year, valid only when HasDraftYear; the §6 experience source
	HasDraftYear bool   // a real (positive) draft year is present — undrafted/created players lack it
	College      string // MFL college name, raw (empty when MFL has none); the SchoolTier join source
}

// Facts resolves one canonical player id to its players-DB facts. ok is false when
// the id is unknown OR resolves to a team aggregate — an aggregate is never a
// scorable player, so a caller treats both identically ("not in the players DB").
func (l Lookup) Facts(id string) (PlayerFacts, bool) {
	pid, err := playerid.New(id)
	if err != nil {
		return PlayerFacts{}, false
	}
	e, ok := l.entry(pid)
	if !ok || e.isAggregate {
		return PlayerFacts{}, false
	}
	return PlayerFacts{
		Name:         e.name,
		Position:     e.position,
		IsRookie:     e.isRookie,
		Birthdate:    e.birthdate,
		HasBirthdate: e.hasBirthdate,
		DraftYear:    e.draftYear,
		HasDraftYear: e.hasDraftYear,
		College:      e.college,
	}, true
}

// classifyPosition maps a raw MFL position code onto the engine set. Aggregate
// codes return (_, true) so the join can reject a roster that references one;
// "PK"→K and "EDGE"→DE are the two real remaps (OQ-004: MFL labels edge rushers
// DE); "XX" and any unknown code become PosFlag for admin review. The maps are
// passed in (built once by NewLookup) so this stays allocation-free per call.
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

// newPositionMap is the raw-MFL-code → engine-position table. Real codes pass
// through; PK→K and EDGE→DE are the remaps; XX is intentionally absent so it falls
// through to PosFlag in classifyPosition.
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

// newAggregateSet is the set of MFL team/positional aggregate codes that are never
// real players and must be filtered (they reach normalize only via the players DB,
// never via a roster, since rosters drops the aggregate ID range upstream).
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

// Package players is the Layer 1 fetcher for the MFL players endpoint (WF 1B):
// the master id → name/position/team/rookie lookup that the B3 cross-reference
// join reads. It clones the reviewed rosters/schedule pattern: fetch →
// schema-validate the SHAPE → return RAW records, transforming nothing (position
// stays the raw MFL code "PK"/"EDGE"/"XX"; rookie stays the raw "R" status —
// normalize maps both at B3).
//
// players is a LEAGUE-scoped call (www47 host, L param, DiscoverHost-first like
// rosters) — NOT the global load-balanced `api` feed. This is deliberate and was
// found the hard way at B3: the global feed OMITS commissioner-created players. In
// this league an owner can bid on someone not yet in MFL's database; the
// commissioner then creates a league-local player record (ids 0816/0820/0835/0838
// live: Gosnell/Roberts/Childress/Robinson). A roster references those ids, but the
// global feed cannot name them — so the join fails. The league feed is a superset
// (2621 vs 2578 live) that includes the created players, so every rostered id
// resolves. MFL caps this endpoint at once per day — caching is the caller's
// responsibility; the fetcher just fetches.
//
// (FUTURE, deferred to the refresh/sync layer: when MFL later assigns an official
// id to a created player, the commissioner manually swaps it on the roster. An
// auto-matching "reconciliation ramp" could detect that on refresh and replace the
// created id with the official one — name-matching, so it needs its own review.)
//
// Aggregate filtering is deliberately NOT done here. The rosters fetcher must
// filter by ID range because it has no position field; the players endpoint
// carries position, and position-based aggregate filtering (Coach, PN, TM*, Def,
// ST, Off) is normalize's single responsibility (WF 1C). Doing it in two places
// would split one rule across two layers. This fetcher therefore returns every
// record raw and validates only the boundary it owns: the player ID (RISK-003)
// and the presence of a position (every downstream filter/map keys off it).
package players

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/mfl"
)

// errEmptyPlayers guards a glitch/empty payload (zero players). The MFL player
// database is never legitimately empty for a season, so we surface it rather than
// return an empty slice — an empty lookup cache downstream would fail every
// roster join silently (every player "unknown") instead of loudly.
var errEmptyPlayers = errors.New("players: response contained zero players")

// RawPlayer is one player-database row exactly as MFL returns it. Every field is
// the raw string MFL sends: Position keeps the MFL code (normalize maps PK→K,
// EDGE→DE, XX→FLAG at B3) and Status is the raw rookie flag ("R" when present,
// empty otherwise — normalize derives IsRookie). The fetcher transforms nothing.
type RawPlayer struct {
	ID       string // MFL player ID, string, may carry leading zeros ("0531")
	Name     string // "Last, First"
	Position string // raw MFL position code; mapped at B3
	Team     string // 3-letter NFL code, or "FA" for free agent
	Status   string // "R" for a 2026 rookie; empty otherwise
	// Birthdate is the raw epoch-seconds string DETAILS=1 adds ("239432400"), or
	// empty when MFL has none — commissioner-created players and some deep-database
	// rows legitimately lack it (M1 recon: 6 of 1232 rostered). The consumer (M1
	// age derivation) owns the absent-birthdate policy; the fetcher passes it raw.
	Birthdate string
	// DraftYear is the raw draft-year string DETAILS=1 adds ("2020"), the source for
	// the §6 free-agency min-salary floor (experience = season − draft year). MFL
	// sends the field for every player but uses sentinels for the undrafted/unknown —
	// "0" and its epoch placeholder "1970" — so a present value is NOT necessarily a
	// real draft year. The consumer (the §6 signing handler) owns the sentinel /
	// plausibility policy; the fetcher passes it raw.
	DraftYear string
	// College is the raw college-name string DETAILS=1 adds ("Ohio State", "Miami
	// (FL)"), the source for the SchoolTier scouting signal (S-Phase 1). MFL omits it
	// for team-defense / coach rows and some deep-database players (~15% league-wide);
	// absent is legitimate and the consumer (the scouting assembly's school-tier join)
	// treats a missing/unmatched college as SchoolUnset → Data-Parity neutral, so this
	// fetcher validates nothing about it and passes it raw. MFL's college vocabulary
	// does not match CFBD's school vocabulary 1:1 ("Miami (FL)" vs "Miami"); that
	// reconciliation belongs to the join, not here.
	College string
}

// Validate checks the raw record's SHAPE before anything downstream runs. It does
// not convert types (normalize's job) and it does not judge the position value —
// it only rejects records that cannot be valid: the player ID must pass the
// ingestion boundary (RISK-003 site #1) and a position must be present (every
// downstream filter and mapping keys off it). Name/Team are validated at
// normalize, after the position filter has removed aggregate records that
// legitimately carry no name.
func (p RawPlayer) Validate() error {
	if _, err := ingestion.ValidatePlayerID(p.ID); err != nil {
		return fmt.Errorf("players: %w", err)
	}
	if strings.TrimSpace(p.Position) == "" {
		return fmt.Errorf("players: record %s (%q) missing position", p.ID, p.Name)
	}
	// A PRESENT birthdate must be a parseable epoch-seconds integer; absent is
	// legitimate (see the field comment) and judged by the consumer, not here.
	if bd := strings.TrimSpace(p.Birthdate); bd != "" {
		if _, err := strconv.ParseInt(bd, 10, 64); err != nil {
			return fmt.Errorf("players: record %s (%q) non-numeric birthdate %q: %w", p.ID, p.Name, p.Birthdate, err)
		}
	}
	// A PRESENT draft year must be a parseable integer; absent is legitimate and the
	// "0"/"1970" sentinels are numeric (the consumer judges plausibility, not here).
	if dy := strings.TrimSpace(p.DraftYear); dy != "" {
		if _, err := strconv.Atoi(dy); err != nil {
			return fmt.Errorf("players: record %s (%q) non-numeric draft year %q: %w", p.ID, p.Name, p.DraftYear, err)
		}
	}
	return nil
}

// playersEnvelope mirrors the MFL players JSON. Unknown fields are tolerated (MFL
// is an external API we do not control and adds fields without versioning — the
// internal/schema unknown-field policy); correctness comes from Validate asserting
// the fields we depend on. player[] uses MFLList so the single-element collapse
// (MFL emits a bare object for a one-element array) cannot crash the decode.
type playersEnvelope struct {
	Players struct {
		Player ingestion.MFLList[playerBlock] `json:"player"`
	} `json:"players"`
}

type playerBlock struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Position  string `json:"position"`
	Team      string `json:"team"`
	Status    string `json:"status"`
	Birthdate string `json:"birthdate"`  // epoch seconds; present only with DETAILS=1
	DraftYear string `json:"draft_year"` // draft year; present with DETAILS=1 ("0"/"1970" = undrafted sentinel)
	College   string `json:"college"`    // college name; present with DETAILS=1, omitted for team-D/coach rows
}

// Fetch retrieves the league's player database for the given season+league and
// returns shape-validated RawPlayer records. Like rosters, it discovers the league
// host FIRST (the ingestion layer owns "discover before any league-specific call"),
// then issues one league-scoped call so commissioner-created players are included.
// year and leagueID are explicit arguments (ingestion.SeasonYear / ingestion.LeagueID
// are the canonical Phase-1 values) so the fetcher is unit-testable and survives
// season rollover. Rate-limiting and 429 backoff are inherited.
func Fetch(ctx context.Context, c *mfl.Client, year, leagueID string) ([]RawPlayer, error) {
	if err := c.DiscoverHost(ctx, year, leagueID); err != nil {
		return nil, fmt.Errorf("players: discover host: %w", err)
	}

	// DETAILS=1 adds the extended per-player fields; M1 needs birthdate (the age
	// input for L3 decay). The base fields are unchanged, so existing consumers
	// (the B3 cross-reference join) see the same shape plus one optional column.
	resp, err := c.Do(ctx, mfl.Request{
		Type:   "players",
		Year:   year,
		Params: map[string]string{"L": leagueID, "DETAILS": "1"},
	})
	if err != nil {
		return nil, fmt.Errorf("players: fetch: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("players: unexpected status %d", resp.StatusCode)
	}

	var env playersEnvelope
	if err := json.Unmarshal(resp.Body, &env); err != nil {
		return nil, fmt.Errorf("players: decode: %w", err)
	}
	if len(env.Players.Player) == 0 {
		return nil, errEmptyPlayers
	}

	return flatten(ctx, env)
}

// flatten walks the decoded envelope into RawPlayer records, validating every
// record's shape. A malformed record fails LOUD (returns an error) rather than
// being silently dropped, consistent with rosters/schedule. It honors ctx
// cancellation so a shutdown mid-parse returns promptly instead of churning
// through the full ~2,500-record database.
func flatten(ctx context.Context, env playersEnvelope) ([]RawPlayer, error) {
	out := make([]RawPlayer, 0, len(env.Players.Player))
	for _, pb := range env.Players.Player {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("players: flatten cancelled: %w", ctx.Err())
		default:
		}

		rp := RawPlayer(pb)
		if err := rp.Validate(); err != nil {
			return nil, err
		}
		out = append(out, rp)
	}
	return out, nil
}

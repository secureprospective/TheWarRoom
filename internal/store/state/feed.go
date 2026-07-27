package state

import (
	"context"
	"fmt"
	"strings"
)

// FeedEvent is one row of the Activity / Transaction Feed — a single chronological event
// projected from ONE row of an append-only ledger table. The read-model is a UNION across
// the existing append-only event tables (trade_notes, player_status_events, dead_cap_ledger,
// cap_relief_ledger, contract_year_changes): one source row → one feed event, no collapsing
// and no dedup across tables, so the feed is a faithful historical river of every recorded
// ledger mutation. NO writes, NO schema change — this is read-only against tables that
// already exist for the transaction engine's own audit spine.
//
// The Session-D event grammar renders this substrate as a single time-ordered stream
// (one 2px-spine row per event); the `Kind` drives the spine's semantic hue, the
// `FranchiseIDs` slice backs the subject line, and `TradeRationale` / `TradePicksNote`
// surface Session-3's persisted trade text on TRADE rows. Acquisition provenance
// (MFL's "Player Acquired Info" concept — draft / trade / waiver / free-agent-signing)
// is DERIVED from the source table + Kind + Reason here in the read-model: no manual field
// was added to any write path (the groundwork did not exist; this is the home for it).
type FeedEvent struct {
	// Source is the ledger table the row was projected from — "trade_notes",
	// "player_status_events", "dead_cap_ledger", "cap_relief_ledger", or
	// "contract_year_changes". Drives the per-source id namespace (the per-row ID is only
	// unique within its source table) and the frontend's "data origin" affordance.
	Source string
	// ID is the source-table row's primary key (trade_notes.id, dead_cap_ledger.id,
	// contract_year_changes.id, or the INTEGER seq of player_status_events / cap_relief_ledger
	// cast to TEXT). Unique only within Source; the frontend composes a stable React key as
	// Source + ":" + ID.
	ID string
	// Kind is the coarse event classification the spine renders. Derived from the source
	// table + its discriminator column (status / source / reason text):
	//   TRADE              — trade_notes row
	//   RELEASE            — player_status_events status=FREE_AGENT (cut / waived / expired)
	//   RETIREMENT         — player_status_events status=RETIRED
	//   DEATH              — player_status_events status=DECEASED
	//   DEAD_CAP           — dead_cap_ledger row (the money side of a §8/§12/§13 charge)
	//   CAP_RELIEF         — cap_relief_ledger row (§13 commissioner credit)
	//   SIGN               — contract_year_changes source="signing" (§6 free-agency signing)
	//   EXTENSION          — contract_year_changes source="extension" (§10 extension years)
	//   RESTRUCTURE        — contract_year_changes source="op" + reason matches §11
	//   TAG                — contract_year_changes source="op" + reason matches §9
	//   WAIVER_VOID        — contract_year_changes source="op" + reason matches §8 void
	//   CONTRACT_CHANGE    — contract_year_changes source="op", uncategorized (future op)
	// seed rows (source="seed") are excluded — those are initial migrations, not events.
	// A player_status_events row whose status is none of FREE_AGENT/RETIRED/DECEASED (a
	// future domain.PlayerStatus the feed CASE was not updated for, or a raw bogus insert)
	// surfaces as a drift ERROR from Feed — it is never silently bucketed as a release.
	Kind string
	// Timestamp is the RFC3339 the source row was written at. The feed is ordered by this
	// descending; ties are broken by Source then by the numeric value of the seq-cast IDs
	// (LENGTH-then-lex, so seq 10 does not sort above seq 2) for deterministic rendering.
	Timestamp string
	// MFLID is the player id the event concerns. Empty for franchise-only events
	// (trade_notes does not single out a player — its involved_franchises carries both
	// sides — and cap_relief_ledger has no player). IDs are the MFL string form (leading
	// zeros intact, per RISK-003); a stale commissioner-created id that no longer resolves
	// in the players-DB is rendered as-is and never causes a hard-fail (OQ-013).
	MFLID string
	// FranchiseIDs are the franchises the event touched. trade_notes splits its
	// comma-joined involved_franchises (both sending AND receiving franchises — Session 3
	// bug-fix); the dead_cap / cap_relief ledgers carry a single franchise_id; the
	// player_status_events and contract_year_changes tables are player-keyed only and
	// carry NO franchise (a reliable at-time franchise would require a historical roster
	// snapshot this codebase does not keep), so those rows return an empty slice and the
	// frontend renders the player id alone.
	FranchiseIDs []string
	// Reason is the raw audit text the source row carries — the immutable audit spine the
	// ledger exists to preserve. The frontend renders it as the predicate line.
	Reason string
	// Provenance is the derived acquisition category (MFL's "Player Acquired Info"
	// vocabulary): "trade", "waiver", "free-agent-signing", "draft", or "" when the event
	// is not an acquisition (a release, a contract mutation, a cap-relief credit, etc.).
	// Derived from Source + Kind + Reason; "draft" never fires today (no draft handler or
	// draft table exists yet — it is reserved for the future rookie-draft surface).
	Provenance string
	// TradeRationale is the TRADE row's persisted rationale (Session 3: required,
	// validated non-empty before a tx opens). Empty for every non-TRADE event.
	TradeRationale string
	// TradePicksNote is the TRADE row's Alpha-scope free-text picks note (Session 3:
	// unvalidated by design — no pick-ownership ledger yet). Empty for every non-TRADE event.
	TradePicksNote string
}

// feedKindDiscriminators are the literal substrings the contract_year_changes reason text
// is matched against to classify a source="op" row. They are the exact reason prefixes the
// transaction handlers write (contracts.go §11 restructure, contracts.go §9 tag, deadcap.go
// §8 waiver void) — kept here as a single source of truth so a handler reason change and the
// feed classifier cannot drift apart silently.
//
// All three are lowercase by convention (the handlers format them that way). Matching is
// case-INSENSITIVE on both sides to keep SQL and Go in lockstep without depending on that
// convention: the SQL CASE uses LIKE (SQLite LIKE is ASCII case-insensitive by default), and
// the Go classifiers fold the reason through strings.ToLower before Contains. A handler that
// ever wrote an uppercase variant would therefore still classify consistently across both.
const (
	opReasonRestructure = "§11 restructure"
	opReasonTag         = "§9 franchise tag"
	opReasonWaiverVoid  = "waiver-cut §8"
)

// classifyContractChangeKind maps a contract_year_changes row onto its feed Kind via the
// source column + the reason text. source="seed" rows must be filtered out BEFORE this is
// called (the caller's WHERE clause excludes them); the function is total over the rest.
// source="signing" → SIGN; source="extension" → EXTENSION; source="op" is disambiguated by
// reason text (case-insensitively, mirroring the SQL CASE's LIKE), defaulting to
// CONTRACT_CHANGE when no prefix matches (a future op kind).
//
// NOTE: this function is the Go-side mirror of the feedSQL contract_year_changes CASE — the
// production feed reads the Kind straight out of SQL, and this function exists so a test
// (TestClassifyContractChangeKind) can pin the SQL↔Go contract and surface drift.
func classifyContractChangeKind(source, reason string) string {
	switch source {
	case "signing":
		return "SIGN"
	case "extension":
		return "EXTENSION"
	default:
		r := strings.ToLower(reason)
		switch {
		case strings.Contains(r, opReasonRestructure):
			return "RESTRUCTURE"
		case strings.Contains(r, opReasonTag):
			return "TAG"
		case strings.Contains(r, opReasonWaiverVoid):
			return "WAIVER_VOID"
		default:
			return "CONTRACT_CHANGE"
		}
	}
}

// deriveProvenance maps a (Kind, Reason) pair onto MFL's "Player Acquired Info"
// vocabulary. An event that is not an acquisition returns "". The mapping is conservative:
// only kinds whose semantics unambiguously match an acquisition category are labeled, so a
// future handler that introduces a new kind surfaces as "" rather than a mislabel.
//
//   - TRADE                         → "trade"
//   - SIGN                          → "free-agent-signing"
//   - DEAD_CAP / RELEASE with a §8  → "waiver" (the player was acquired-via-waiver at some
//     prior point and is now being released — the cut is the
//     waiver event of record)
//
// RETIREMENT / DEATH / CAP_RELIEF / EXTENSION / RESTRUCTURE / TAG / WAIVER_VOID /
// CONTRACT_CHANGE → "" (not acquisitions).
func deriveProvenance(kind, reason string) string {
	switch kind {
	case "TRADE":
		return "trade"
	case "SIGN":
		return "free-agent-signing"
	case "DEAD_CAP", "RELEASE":
		if strings.Contains(strings.ToLower(reason), opReasonWaiverVoid) {
			return "waiver"
		}
		// A natural §14 UFA-expiry release carries no §8 marker — the player was simply
		// exposed to free agency, not acquired via waiver. Leave provenance empty.
		return ""
	default:
		return ""
	}
}

// feedSQL is the single UNION ALL across the append-only event tables. Every branch projects
// to the same nine columns so the rows scan into one FeedEvent shape. The trailing ORDER BY
// applies to the unioned set (SQLite scopes ORDER BY / LIMIT to the whole result set when the
// UNION is the top-level statement). LIMIT is bound once at the end.
//
// Each branch's columns, in order:
//  1. source-table label
//  2. the source row's primary key, as TEXT
//  3. the coarse Kind
//  4. timestamp (RFC3339)
//  5. mfl_id (empty when the table has no player)
//  6. franchises-raw (the comma-joined form; split in Go)
//  7. reason
//  8. trade rationale (TRADE branch only)
//  9. trade picks note (TRADE branch only)
//
// The contract_year_changes branch CASE-expression mirrors classifyContractChangeKind so the
// SQL and the Go classifier never drift; if a future reason prefix lands, the default
// CONTRACT_CHANGE bucket keeps the row visible rather than dropping it.
//
// The player_status_events branch CASE enumerates every domain.PlayerStatus (FREE_AGENT,
// RETIRED, DECEASED) and uses ELSE 'UNKNOWN' — NOT a silent RELEASE. The Go scan layer turns a
// 'UNKNOWN' kind into a loud drift error (mirroring CurrentStatus's fail-loud-on-unknown-status
// contract), so a future PlayerStatus addition surfaces immediately rather than miscoloring the
// row as a release. The three mapped statuses are kept in lockstep with domain.PlayerStatus by
// the TestFeed_UnknownStatusSurfacesDrift contract.
const feedSQL = `
SELECT source, id, kind, ts, mfl_id, franchises_raw, reason, trade_rationale, trade_picks_note FROM (
    SELECT 'trade_notes'              AS source,
           id                         AS id,
           'TRADE'                    AS kind,
           created_at                 AS ts,
           ''                         AS mfl_id,
           involved_franchises        AS franchises_raw,
           ''                         AS reason,
           rationale                  AS trade_rationale,
           picks_note                 AS trade_picks_note
    FROM trade_notes WHERE league_id = ?1

    UNION ALL

    SELECT 'player_status_events'     AS source,
           CAST(seq AS TEXT)          AS id,
           CASE status WHEN 'FREE_AGENT' THEN 'RELEASE'
                       WHEN 'RETIRED'    THEN 'RETIREMENT'
                       WHEN 'DECEASED'   THEN 'DEATH'
                       ELSE 'UNKNOWN'
           END                        AS kind,
           at                         AS ts,
           mfl_id                     AS mfl_id,
           ''                         AS franchises_raw,
           reason                     AS reason,
           ''                         AS trade_rationale,
           ''                         AS trade_picks_note
    FROM player_status_events WHERE league_id = ?1

    UNION ALL

    SELECT 'dead_cap_ledger'          AS source,
           id                         AS id,
           'DEAD_CAP'                 AS kind,
           created_at                 AS ts,
           mfl_id                     AS mfl_id,
           franchise_id               AS franchises_raw,
           reason                     AS reason,
           ''                         AS trade_rationale,
           ''                         AS trade_picks_note
    FROM dead_cap_ledger WHERE league_id = ?1

    UNION ALL

    SELECT 'cap_relief_ledger'        AS source,
           CAST(seq AS TEXT)          AS id,
           'CAP_RELIEF'               AS kind,
           created_at                 AS ts,
           ''                         AS mfl_id,
           franchise_id               AS franchises_raw,
           reason                     AS reason,
           ''                         AS trade_rationale,
           ''                         AS trade_picks_note
    FROM cap_relief_ledger WHERE league_id = ?1

    UNION ALL

    SELECT 'contract_year_changes'    AS source,
           id                         AS id,
           CASE
               WHEN source = 'signing'   THEN 'SIGN'
               WHEN source = 'extension' THEN 'EXTENSION'
               WHEN reason LIKE '%' || ?2 || '%' THEN 'RESTRUCTURE'
               WHEN reason LIKE '%' || ?3 || '%' THEN 'TAG'
               WHEN reason LIKE '%' || ?4 || '%' THEN 'WAIVER_VOID'
               ELSE 'CONTRACT_CHANGE'
           END                        AS kind,
           changed_at                 AS ts,
           mfl_id                     AS mfl_id,
           ''                         AS franchises_raw,
           reason                     AS reason,
           ''                         AS trade_rationale,
           ''                         AS trade_picks_note
    FROM contract_year_changes
    WHERE league_id = ?1 AND source <> 'seed'
)
-- LENGTH(id) DESC precedes id DESC so the seq-cast ids of player_status_events and
-- cap_relief_ledger sort NUMERICALLY within a same-timestamp group (a bare id DESC would be
-- lexicographic and misorder seq 10 above seq 2). For the text-id sources (trade_notes,
-- dead_cap_ledger, contract_year_changes) LENGTH-then-lex is a different but still-deterministic
-- tiebreaker, and the tiebreak only fires within a single source (source DESC groups first).
ORDER BY ts DESC, source DESC, LENGTH(id) DESC, id DESC
LIMIT ?5`

// Feed reads the Activity / Transaction Feed — every row from every append-only event
// table in this league, projected to one chronological stream. The query is a single UNION
// ALL (above) executed on the read pool; this function only scans its rows and splits the
// comma-joined franchise column. NO writes; the read pool is the same one every sibling
// read-only method (CalendarEvents, FreeAgents, LedgerCells) uses.
//
// `limit` caps the row count (most-recent first); a non-positive value falls back to a sane
// default. The cap is the only pagination v1 offers — full historical scroll is a follow-up
// once the operator surface asks for it. The caller (the IPC layer) resolves player names
// and franchise names through the players / rulebook directories; this read returns ids only.
//
// An empty result (a fresh league with no executed transactions yet) returns a non-nil empty
// slice, never nil — the frontend renders an empty state, not a null.
func (s *Store) Feed(ctx context.Context, limit int) ([]FeedEvent, error) {
	if limit <= 0 {
		limit = 500
	}
	rows, err := s.pools.Read().QueryContext(ctx, feedSQL,
		s.leagueID,
		opReasonRestructure,
		opReasonTag,
		opReasonWaiverVoid,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("state: feed: %w", err)
	}
	defer func() { _ = rows.Close() }()

	out := make([]FeedEvent, 0)
	for rows.Next() {
		var (
			source, id, kind, ts, mflID, franchisesRaw, reason string
			tradeRationale, tradePicksNote                     string
		)
		if err := rows.Scan(
			&source, &id, &kind, &ts, &mflID, &franchisesRaw, &reason,
			&tradeRationale, &tradePicksNote,
		); err != nil {
			return nil, fmt.Errorf("state: feed scan: %w", err)
		}
		// 'UNKNOWN' is the player_status_events CASE ELSE — a status the feed does not
		// recognize (a future domain.PlayerStatus the CASE was not updated for, or a raw
		// insert of a bogus status). Fail loud rather than silently mislabel the row; this
		// mirrors CurrentStatus's drift contract. mfl_id is included so the row is locatable.
		if kind == "UNKNOWN" {
			return nil, fmt.Errorf("state: feed: player_status_events row for mfl_id %q classified as UNKNOWN (status drift — add the status to the feed CASE)", mflID)
		}
		out = append(out, FeedEvent{
			Source:         source,
			ID:             id,
			Kind:           kind,
			Timestamp:      ts,
			MFLID:          mflID,
			FranchiseIDs:   splitFranchises(franchisesRaw),
			Reason:         reason,
			Provenance:     deriveProvenance(kind, reason),
			TradeRationale: tradeRationale,
			TradePicksNote: tradePicksNote,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("state: feed iterate: %w", err)
	}
	return out, nil
}

// splitFranchises splits trade_notes' comma-joined involved_franchises column into a slice.
// Empty input → nil (so the JSON-marshalled DTO renders an empty array, not [""]). Whitespace
// around each id is trimmed (defense-in-depth; LogTradeNote joins without spaces today, but a
// future writer is not bound by that).
func splitFranchises(joined string) []string {
	joined = strings.TrimSpace(joined)
	if joined == "" {
		return nil
	}
	parts := strings.Split(joined, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if id := strings.TrimSpace(p); id != "" {
			out = append(out, id)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

package state

import (
	"context"
	"fmt"
	"strings"
)

// FeedEvent is one activity-feed event, projected from one row of an append-only ledger
// (trade_notes, player_status_events, dead_cap_ledger, cap_relief_ledger, contract_year_changes).
// One row, one event: no collapsing, no dedup, no writes. Acquisition provenance is derived here
// from the source, kind and reason.
type FeedEvent struct {
	// Source is the ledger table the row came from; IDs are unique only within it.
	Source string
	// ID is the source row's key as text; the frontend's React key is Source + ":" + ID.
	ID string
	// Kind classifies the event:
	//   TRADE                            trade_notes
	//   RELEASE / RETIREMENT / DEATH     player_status_events FREE_AGENT / RETIRED / DECEASED
	//   DEAD_CAP, CAP_RELIEF             the two cap ledgers
	//   SIGN, EXTENSION                  contract_year_changes source signing / extension
	//   RESTRUCTURE / TAG / WAIVER_VOID  source op, by reason text; CONTRACT_CHANGE otherwise
	// Seed rows are excluded. An unrecognized player status is a drift error, never a release.
	Kind string
	// Timestamp orders the feed, newest first. Ties break by Source, then by numeric ID
	// (length-then-lexical, so seq 10 sorts above seq 2).
	Timestamp string
	// MFLID is empty for franchise-only events (trades, cap relief). A stale commissioner-created id
	// is returned as-is.
	MFLID string
	// FranchiseIDs are the franchises the event touched. Status and contract rows carry none: no
	// historical roster snapshot exists to say which franchise held the player then.
	FranchiseIDs []string
	// Reason is the row's raw audit text.
	Reason string
	// Provenance is the acquisition category ("trade", "waiver", "free-agent-signing"), or "".
	// "draft" is reserved; nothing produces it yet.
	Provenance string
	// TradeRationale is set on TRADE rows only.
	TradeRationale string
	// TradePicksNote is set on TRADE rows only (unvalidated free text).
	TradePicksNote string
}

// The reason prefixes the transaction handlers write, used to classify source="op" rows,
// case-insensitively.
const (
	opReasonRestructure = "§11 restructure"
	opReasonTag         = "§9 franchise tag"
	opReasonWaiverVoid  = "waiver-cut §8"
)

// classifyContractChangeKind gives a contract_year_changes row its Kind from the row's source and
// reason. Seed rows are filtered out in SQL.
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

// deriveProvenance labels only unambiguous acquisitions: TRADE is "trade", SIGN is
// "free-agent-signing", and a §8 release or dead-cap charge is "waiver". Everything else, and any
// future kind, is "".
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
		// A §14 UFA expiry carries no §8 marker: not a waiver.
		return ""
	default:
		return ""
	}
}

// feedSQL is one UNION ALL across the append-only ledgers. Every branch projects the same nine
// columns: source, id, kind, timestamp, mfl_id, franchises (comma-joined), reason, trade rationale,
// trade picks note. The contract branch projects its raw source as kind; Feed classifies it in Go.
// The status branch's ELSE is 'UNKNOWN', which Feed turns into a drift error.
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
           source                     AS kind,
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
LIMIT ?2`

// Feed returns up to limit events across the ledgers, newest first (non-positive limit: a
// default). Ids only; the caller resolves names. An empty league returns an empty, non-nil slice.
func (s *Store) Feed(ctx context.Context, limit int) ([]FeedEvent, error) {
	if limit <= 0 {
		limit = 500
	}
	rows, err := s.pools.Read().QueryContext(ctx, feedSQL, s.leagueID, limit)
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
		if source == "contract_year_changes" {
			kind = classifyContractChangeKind(kind, reason)
		}
		// A status the CASE doesn't know: fail loudly rather than mislabel the row.
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

// splitFranchises splits the comma-joined column, trimming spaces. Empty input returns nil.
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

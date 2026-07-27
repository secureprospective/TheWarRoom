package main

import (
	"context"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/normalize"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

// This file groups the ACTIVITY / TRANSACTION FEED IPC surface — the read-only GetFeed method
// that backs the Session-D event grammar's backward-looking historical river (the "Feed/Pulse"
// facet from the Session-E facet map). Split from transactions_app.go to keep both within the
// 400-line file cap (AD-14/AD-17), matching the per-op-family split convention
// (transactions_calendar_app.go, etc.). The feed writes nothing: it reads the same append-only
// ledger tables the transaction engine already persists, projected by internal/store/state.Feed.

// FeedEventDTO is one chronological feed event as it crosses the IPC boundary — the read-side
// companion to state.FeedEvent with display-only joins (player name/position, franchise name)
// resolved server-side so the frontend never re-derives them. StableKey is the React key
// (Source + ":" + ID), pre-computed because every list row needs one and computing it in JS
// from two strings is exactly the kind of per-row fiddle the spine grammar exists to avoid.
//
// Money is deliberately NOT carried: dead_cap_ledger stores dead_cap_cents, but the feed's job
// is the event spine, not a re-derivation of cap figures (the dedicated FranchiseHQ cap view
// owns the dollar rendering). The Reason text already carries the human-readable audit label.
type FeedEventDTO struct {
	StableKey      string   `json:"stableKey"`
	Source         string   `json:"source"`
	ID             string   `json:"id"`
	Kind           string   `json:"kind"`
	Timestamp      string   `json:"timestamp"`
	MFLID          string   `json:"mflID"`
	PlayerName     string   `json:"playerName"`
	PlayerPosition string   `json:"playerPosition"`
	PlayerUnknown  bool     `json:"playerUnknown"`
	FranchiseIDs   []string `json:"franchiseIDs"`
	FranchiseNames []string `json:"franchiseNames"`
	Reason         string   `json:"reason"`
	Provenance     string   `json:"provenance"`
	TradeRationale string   `json:"tradeRationale,omitempty"`
	TradePicksNote string   `json:"tradePicksNote,omitempty"`
	// Session 2 correction state — the reconciled/net-state projection. TxID is this event's
	// logical id (Source + ":" + ID) keyed to the transaction_corrections ledger. CorrectionStatus
	// is the LATEST correction row's status for this entry (CORRECTED / REVERSED), or "" when the
	// entry has no correction (POSTED — the implicit default). The correction's note/reason/by/at
	// are carried alongside so the feed can render BOTH the original and the correction (append-only
	// honest — the original is never hidden). The IPC stamps these by joining Store.Corrections.
	TxID             string `json:"txID"`
	CorrectionStatus string `json:"correctionStatus,omitempty"`
	CorrectionReason string `json:"correctionReason,omitempty"`
	CorrectionNote   string `json:"correctionNote,omitempty"`
	CorrectedBy      string `json:"correctedBy,omitempty"`
	CorrectedAt      string `json:"correctedAt,omitempty"`
}

// FeedResult is the typed IPC outcome: OK plus the chronological events, or OK=false with a
// human-readable Detail. A failed players-DB / rulebook directory lookup does NOT fail the
// result — the feed renders fine with ids only, so a DirectoryWarning is carried alongside a
// successful read (mirroring the m4_app resolveDirectory degradation contract).
type FeedResult struct {
	OK               bool           `json:"ok"`
	Events           []FeedEventDTO `json:"events"`
	Detail           string         `json:"detail"`
	DirectoryWarning string         `json:"directoryWarning,omitempty"`
}

// GetFeed reads the Activity / Transaction Feed — every recent event across the append-only
// ledger tables (trade_notes, player_status_events, dead_cap_ledger, cap_relief_ledger,
// contract_year_changes), in strict reverse-chronological order. The read is read-only and
// executes on the read pool; it never touches the writer.
//
// Player names + positions are resolved through the cached players-DB Lookup (the LEAGUE-scoped
// feed that includes commissioner-created players — OQ-013). A stale created-id that no longer
// resolves (e.g. the commissioner already swapped it for MFL's official id and the created-id
// has dropped out of the league feed) renders as PlayerUnknown=true with the raw id displayed,
// NEVER as a hard-fail: the historical ledger row references the id that was live at the time
// of the event, and that history is preserved verbatim. This is the natural home for OQ-013's
// read-side reconciliation seam; a future auto-matching ramp (the WRITE that swaps the id) plugs
// in upstream of here, and once it has run, the same Lookup call resolves the new official id.
//
// Franchise names are resolved through the rulebook's franchise directory; an unknown id falls
// back to the id itself. The directory fetches are best-effort: a failure surfaces as
// DirectoryWarning but does not blank the feed.
func (a *App) GetFeed() FeedResult {
	if a.startupErr != nil {
		return FeedResult{Detail: a.startupErr.Error()}
	}
	if a.state == nil {
		return FeedResult{Detail: "state store not initialized"}
	}
	if err := a.state.Err(); err != nil {
		return FeedResult{Detail: "state is stale after a failed reload: " + err.Error()}
	}
	ctx, cancel := context.WithTimeout(a.ctx, 10*time.Second)
	defer cancel()

	events, err := a.state.Feed(ctx, 0) // 0 → the store's default cap (most-recent first)
	if err != nil {
		return FeedResult{Detail: err.Error()}
	}

	// Session 2 reconciled-projection join: read every correction row and reduce to latest-per-tx_id
	// (the current correction state of each logical entry). Each feed event is then stamped with its
	// correction state so the frontend renders BOTH the original and the correction (append-only
	// honest — a REVERSED entry shows struck-through with the correction banner; a CORRECTED entry
	// shows the original plus the amended note). A read failure here is best-effort: the feed still
	// renders (uncorrected), with a directory-style warning, never a hard-fail.
	corr, corrErr := a.state.Corrections(ctx)
	latest := latestCorrectionByTxID(corr)

	// Display-only joins — best-effort, never fail the feed.
	dir, dirWarning := a.resolveDirectory(ctx)
	var franchiseNames map[string]string
	if a.rulebook != nil {
		franchiseNames = a.rulebook.FranchiseNames()
	}

	out := make([]FeedEventDTO, len(events))
	for i, e := range events {
		out[i] = feedEventDTO(e, latest, dir, franchiseNames)
	}
	res := FeedResult{OK: true, Events: out}
	if dirWarning != "" {
		res.DirectoryWarning = dirWarning
	}
	if corrErr != nil {
		// A correction-read failure is best-effort: the feed renders uncorrected (POSTED) with a
		// warning, never a hard-fail (the historical river is still the point).
		w := "correction ledger unreadable: " + corrErr.Error()
		if res.DirectoryWarning != "" {
			res.DirectoryWarning += " · " + w
		} else {
			res.DirectoryWarning = w
		}
	}
	return res
}

// feedEventDTO builds one FeedEventDTO from a raw state.FeedEvent, stamping the display-only joins
// (franchise names, player name/position via the OQ-013 Lookup) and the Session 2 reconciled
// correction state (from latest, the tx_id → latest-correction-row map). Split out of GetFeed to
// keep it under the function-length gate — this is the per-row body of that loop, unchanged.
func feedEventDTO(e state.FeedEvent, latest map[string]state.CorrectionRow, dir normalize.Lookup, franchiseNames map[string]string) FeedEventDTO {
	txID := e.Source + ":" + e.ID
	dto := FeedEventDTO{
		StableKey:      txID,
		Source:         e.Source,
		ID:             e.ID,
		Kind:           e.Kind,
		Timestamp:      e.Timestamp,
		MFLID:          e.MFLID,
		FranchiseIDs:   e.FranchiseIDs,
		FranchiseNames: resolveFranchiseNames(e.FranchiseIDs, franchiseNames),
		Reason:         e.Reason,
		Provenance:     e.Provenance,
		TradeRationale: e.TradeRationale,
		TradePicksNote: e.TradePicksNote,
		TxID:           txID,
	}
	// Stamp the latest correction row for this entry (the reconciled state). Corrections returns
	// seq-ascending, so latestCorrectionByTxID keeps the newest per tx_id; an entry with no
	// correction row stays all-zero (POSTED — the implicit default the frontend renders clean).
	if c, ok := latest[txID]; ok {
		dto.CorrectionStatus = c.Status
		dto.CorrectionReason = c.Reason
		dto.CorrectionNote = c.Note
		dto.CorrectedBy = c.Commissioner
		dto.CorrectedAt = c.CreatedAt
	}
	// OQ-013 reconciliation seam: resolve the player id through the players-DB Lookup.
	// The Lookup is built from MFL's LEAGUE feed, so commissioner-created ids that are
	// still live resolve normally; a stale created-id (swapped out after MFL assigned an
	// official id) returns ok=false and surfaces as PlayerUnknown rather than failing.
	// When a future refresh/sync layer swaps the historical row's id, the SAME call here
	// will resolve the new official id with no rendering change.
	if e.MFLID != "" {
		if facts, ok := dir.Facts(e.MFLID); ok {
			dto.PlayerName = facts.Name
			dto.PlayerPosition = string(facts.Position)
		} else {
			dto.PlayerUnknown = true
		}
	}
	return dto
}

// latestCorrectionByTxID reduces a seq-ascending correction slice to the LATEST correction row per
// tx_id — the reconciled/net-state projection (the current correction state of each logical entry).
// Iterating in seq order and overwriting map[tx_id] = row keeps the newest per tx_id; a nil/empty
// input yields an empty map (every entry is POSTED — the implicit default).
func latestCorrectionByTxID(rows []state.CorrectionRow) map[string]state.CorrectionRow {
	out := make(map[string]state.CorrectionRow, len(rows))
	for _, r := range rows {
		out[r.TxID] = r // seq-ascending → last write wins = latest
	}
	return out
}

// resolveFranchiseNames maps each franchise id to its display name from the rulebook directory,
// falling back to the raw id when the directory carries no entry (a rulebook version that
// predates the franchise directory, or an unknown id). Mirrors GetFranchises' id-fallback rule.
// Returns nil when ids is empty so the JSON field renders as [] (the empty state), not [""].
func resolveFranchiseNames(ids []string, names map[string]string) []string {
	if len(ids) == 0 {
		return nil
	}
	out := make([]string, len(ids))
	for i, id := range ids {
		if names == nil {
			out[i] = id
			continue
		}
		if n, ok := names[id]; ok && n != "" {
			out[i] = n
			continue
		}
		out[i] = id
	}
	return out
}

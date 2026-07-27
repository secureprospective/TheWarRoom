package transactions

import (
	"context"
	"fmt"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

// Correction status values — the discriminator the commissioner picks when issuing a correction.
// These mirror state.CorrStatusCorrected / state.CorrStatusReversed verbatim; they are redeclared
// here (unexported) so this package owns the wire vocabulary without reaching across to the store
// package for a constant (the value is the contract; the store validates it).
const (
	corrCorrected = "CORRECTED"
	corrReversed  = "REVERSED"
)

func validCorrectionStatus(s string) bool {
	switch s {
	case corrCorrected, corrReversed:
		return true
	default:
		return false
	}
}

// validCorrectionSource reports whether s is one of the five append-only ledgers the Feed (Session
// 1) unions — the only tables a Correction may reference. A garbage Source would create an orphan
// correction row the reconciled projection can never join back to anything (a review finding:
// unvalidated Source silently accepted a typo). A switch, not a package-level map (gochecknoglobals).
func validCorrectionSource(s string) bool {
	switch s {
	case "trade_notes", "player_status_events", "dead_cap_ledger", "cap_relief_ledger", "contract_year_changes":
		return true
	default:
		return false
	}
}

// validEntryKind reports whether k is one of the known transaction Kind constants — guards against
// a typo'd EntryKind (e.g. "TREDE") writing a garbage kind string into the corrections ledger that
// the reconciled projection can never match to a real op (a review finding).
func validEntryKind(k Kind) bool {
	switch k {
	case KindTrade, KindRosterStatus, KindWaiver, KindRestructure, KindTag, KindExtension, KindBuyout,
		KindAdvancePhase, KindRolloverSeason, KindRetirement, KindDeath, KindCapRelief, KindSign,
		KindSetSigningWindow, KindScheduleEvent, KindRescheduleEvent, KindCancelEvent, KindSetTradeDeadline:
		return true
	case KindCorrect:
		return false // a correction is never the original entry being corrected
	default:
		return false
	}
}

// Correction is the Session-2 retroactive-correction transaction: it appends one row to the
// transaction_corrections ledger tying back to a prior feed entry by its logical tx_id. It is the
// commissioner-facing "Correct this entry" action — NEVER an update/delete of the original ledger
// row (append-only honored). Status CORRECTED amends the original's note/metadata (its effect
// still holds); REVERSED marks the original's effect undone for the reconciled/net-state read
// projection (the original is never deleted — the projection excludes it from net totals and
// renders it struck-through). The commissioner + reason are the required audit trail.
//
// Source + SourceID identify the original feed row (the same (Source, ID) the feed projects); the
// handler composes them into the tx_id the store joins on (Source + ":" + SourceID). EntryKind
// echoes the ORIGINAL entry's transaction Kind (TRADE/SIGN/WAIVER/…) so the correction row carries
// the semantics for projection without a join back to the source table. (It is named EntryKind,
// not Kind, to avoid shadowing the Request interface's Kind() method.)
type Correction struct {
	Source       string // "trade_notes" | "player_status_events" | "dead_cap_ledger" | "cap_relief_ledger" | "contract_year_changes"
	SourceID     string // the original row's id (TEXT or seq-cast-to-text)
	EntryKind    Kind   // the original entry's transaction Kind (echoed onto the correction row)
	Status       string // CORRECTED | REVERSED
	Commissioner string
	Reason       string
	Note         string
}

func (Correction) Kind() Kind { return KindCorrect }
func (Correction) sealed()    {}

// validate enforces the shape a correction must have — a non-empty source + source id (the logical
// entry being corrected), a valid correction status, and a commissioner + reason for the audit
// trail — before a transaction is opened. Whether the original entry actually exists is judged
// inside apply (the store appends regardless; the projection reconciles), so a correction of a
// mistyped id lands as an orphan row the feed renders as "corrects an unknown entry" rather than
// silently no-oping (append-only-honest).
func (c Correction) validate() error {
	if !validCorrectionSource(c.Source) {
		return fmt.Errorf("transactions: correction source %q is not a correctable feed table", c.Source)
	}
	if strings.TrimSpace(c.SourceID) == "" {
		return fmt.Errorf("transactions: correction requires a source id")
	}
	if !validEntryKind(c.EntryKind) {
		return fmt.Errorf("transactions: correction entry kind %q is not a known transaction kind", c.EntryKind)
	}
	if !validCorrectionStatus(c.Status) {
		return fmt.Errorf("transactions: correction status %q is not CORRECTED or REVERSED", c.Status)
	}
	if strings.TrimSpace(c.Commissioner) == "" {
		return fmt.Errorf("transactions: correction requires a commissioner (who issued it)")
	}
	if strings.TrimSpace(c.Reason) == "" {
		return fmt.Errorf("transactions: correction requires a reason (audit trail)")
	}
	return nil
}

func (c Correction) apply(ctx context.Context, w state.TxWriter) (applyResult, error) {
	txID := c.Source + ":" + c.SourceID
	if err := w.AppendCorrection(ctx, state.CorrectionEntry{
		TxID:         txID,
		Kind:         string(c.EntryKind),
		Status:       c.Status,
		Commissioner: c.Commissioner,
		Reason:       c.Reason,
		Note:         c.Note,
	}); err != nil {
		return applyResult{}, fmt.Errorf("correction: %w", err)
	}
	// A correction changes no players; it appends one audit row to transaction_corrections.
	return applyResult{PlayersAffected: 0}, nil
}

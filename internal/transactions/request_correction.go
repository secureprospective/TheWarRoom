package transactions

import (
	"context"
	"fmt"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

// Correction statuses, matching the state package's values. The store validates them.
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

// validCorrectionSource accepts only the five ledgers the feed reads; any other source would
// orphan the correction.
func validCorrectionSource(s string) bool {
	switch s {
	case "trade_notes", "player_status_events", "dead_cap_ledger", "cap_relief_ledger", "contract_year_changes":
		return true
	default:
		return false
	}
}

// validEntryKind rejects an unknown kind (e.g. "TREDE") that could never match a real op.
func validEntryKind(k Kind) bool {
	switch k {
	case KindTrade, KindRosterStatus, KindWaiver, KindRestructure, KindTag, KindExtension, KindBuyout,
		KindAdvancePhase, KindRolloverSeason, KindRetirement, KindDeath, KindCapRelief, KindSign,
		KindSetSigningWindow, KindScheduleEvent, KindRescheduleEvent, KindCancelEvent, KindSetTradeDeadline:
		return true
	case KindCorrect:
		return false
	default:
		return false
	}
}

// Correction appends a correction tying back to a feed entry by tx_id (Source + ":" + SourceID);
// the original row is never changed. CORRECTED amends the note and the effect stands; REVERSED
// marks the effect undone for net totals. EntryKind is the original entry's kind (named so it
// doesn't shadow Kind()).
type Correction struct {
	Source       string // "trade_notes" | "player_status_events" | "dead_cap_ledger" | "cap_relief_ledger" | "contract_year_changes"
	SourceID     string
	EntryKind    Kind
	Status       string // CORRECTED | REVERSED
	Commissioner string
	Reason       string
	Note         string
}

func (Correction) Kind() Kind { return KindCorrect }
func (Correction) sealed()    {}

// validate checks the shape. Whether the entry exists isn't checked: a mistyped id lands as an
// orphan the feed shows as "corrects an unknown entry", rather than silently doing nothing.
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
	return applyResult{PlayersAffected: 0}, nil
}

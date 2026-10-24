package history

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
)

// Retried receipts retain their original recipient token, not the current head.
func viewReceipt(ctx context.Context, receipt commissioner.Receipt, records []commissioner.Record) commissioner.Receipt {
	if receipt.Revision > 0 && receipt.Revision <= int64(len(records)) {
		receipt.Revision = commissioner.ViewRevisionAt(ctx, records[:receipt.Revision], receipt.Revision, receipt.RecordedAt)
	}
	return receipt
}

// A request key owned by another recipient cannot reveal hidden activity through a collision.
func viewRequestKey(ctx context.Context, principal, request string) string {
	if !commissioner.HasRecipient(ctx) {
		return request
	}
	digest := sha256.Sum256([]byte(principal + "\x00" + request))
	return "recipient." + hex.EncodeToString(digest[:])
}

func expectedViewRevision(ctx context.Context, raw commissioner.RawCommand, principal string, records []commissioner.Record, now time.Time) int64 {
	current := commissioner.ViewRevisionAt(ctx, records, int64(len(records)), now)
	if commissioner.OwnerBidAction(raw.Command) != "" && !commissioner.HasRecipient(ctx) {
		// Local adapters may still submit their legacy ledger revision.
		if raw.Expected == int64(len(records)) {
			return raw.Expected
		}
		return commissioner.RecipientRevision(records, principal, now)
	}
	return current
}

// staleVersion refuses a change made against an older league version. It is ErrRevisionConflict, and its
// own message is the reason the owner reads.
type staleVersion struct{ current, expected int64 }

func (e staleVersion) Error() string {
	return fmt.Sprintf("league version is %d; reread before changing it (expected %d)", e.current, e.expected)
}
func (e staleVersion) Is(target error) bool { return target == ErrRevisionConflict }

func checkCommissionerAppend(ctx context.Context, raw commissioner.RawCommand, principal string, records []commissioner.Record, now time.Time) error {
	current := expectedViewRevision(ctx, raw, principal, records, now)
	if raw.Expected != current {
		return staleVersion{current: current, expected: raw.Expected}
	}
	if raw.Command == commissioner.Create {
		if len(records) != 0 {
			return fmt.Errorf("this app league already exists")
		}
		return nil
	}
	if len(records) == 0 {
		return fmt.Errorf("create this app league before changing it")
	}
	if raw.Command != commissioner.Revert {
		return nil
	}
	for _, r := range records {
		if r.Receipt.Entry != raw.Args.Entry {
			continue
		}
		if r.Command.Visibility == "private" && raw.Visibility != "private" {
			return fmt.Errorf("a private recorded change must be reversed privately")
		}
		return nil
	}
	return fmt.Errorf("that recorded change is not in this league")
}

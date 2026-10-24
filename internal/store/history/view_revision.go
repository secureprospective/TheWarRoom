package history

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
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

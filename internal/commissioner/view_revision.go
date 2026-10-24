package commissioner

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"slices"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

type recipientKey struct{}
type recipientView struct {
	principal string
	at        time.Time
}

// WithRecipient selects concurrency tokens for a trusted hosted identity. It grants no access.
func WithRecipient(ctx context.Context, principal string, at time.Time) context.Context {
	return context.WithValue(ctx, recipientKey{}, recipientView{principal, at})
}

// HasRecipient reports whether the trusted adapter selected recipient-scoped concurrency.
func HasRecipient(ctx context.Context) bool {
	_, ok := ctx.Value(recipientKey{}).(recipientView)
	return ok
}

// ViewRevision preserves local ledger revisions unless the trusted adapter selected a recipient.
func ViewRevision(ctx context.Context, records []Record, local int64) int64 {
	view, hosted := ctx.Value(recipientKey{}).(recipientView)
	if !hosted {
		return local
	}
	return RecipientRevision(records, view.principal, view.at)
}

// RecipientRevision is an opaque, JavaScript-safe equality token, not an event count.
// Readings share a recipient's visible-ledger scope so any of them can supply Expected.
// Only visible entry identities enter the digest: canonical bodies contain global revisions.
func RecipientRevision(records []Record, principal string, at time.Time) int64 {
	if len(records) == 0 {
		return 0
	}
	league := records[0].Command.League
	state, err := GovernanceState(records)
	privileged := err == nil && privilegedLedgerReader(league, principal, GovernanceAuthorizer{State: state})
	rules, rulesErr := RulesNow(records, at)
	mode, modeErr := nativecheck.BidMode(rules.Bids, leaguerules.UFAProcess)
	open := rulesErr == nil && modeErr == nil && mode == leaguerules.Open
	roots := reversalRoots(records)
	ids := []string{league, PrincipalFor(records, principal)}
	for _, r := range records {
		target := records[roots[r.Receipt.Entry]]
		own := SamePrincipal(records, target.Principal, principal) || SamePrincipal(records, r.Principal, principal)
		if own || ledgerEntryVisible(r, target, records, principal, privileged) ||
			(open && privateBidRecord(r, records)) {
			ids = append(ids, r.Receipt.Entry)
		}
	}
	// Sorting also makes insertion of an invisible entry independent of ledger offsets.
	slices.Sort(ids[2:])
	h := sha256.New()
	for _, id := range ids {
		_, _ = h.Write([]byte(id))
		_, _ = h.Write([]byte{0})
	}
	return max(2, int64(binary.BigEndian.Uint64(h.Sum(nil)[:8])&((1<<53)-1)))
}

// ViewRevisionAt samples visibility at the store's admission or original receipt instant.
func ViewRevisionAt(ctx context.Context, records []Record, local int64, at time.Time) int64 {
	view, hosted := ctx.Value(recipientKey{}).(recipientView)
	if !hosted {
		return local
	}
	return RecipientRevision(records, view.principal, at)
}

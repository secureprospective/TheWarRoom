// Package freeagency holds the SIGN handler, run only through the Coordinator. v1 records a
// signing outcome (the agreed flat contract); it is not a live auction: bidding, RFA tenders and
// comp picks are deferred.
package freeagency

import (
	"context"
	"fmt"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
	"github.com/secureprospective/TheWarRoom/internal/transactions/deadcap"
)

// signingSource tags the ledger cells a signing lays.
const signingSource = "signing"

const signReason = "free-agency signing §6"

// dollars converts whole dollars to cents; the §6 table is in whole dollars.
func dollars(d int64) domain.Money { return domain.Money(d * 100) }

// MinSalaryFloor is the §6 minimum salary by years of NFL experience (season minus draft year).
// With no real draft year the caller passes 0: the rookie floor, the lenient direction
// (Christopher's ruling).
func MinSalaryFloor(experienceYears int) domain.Money {
	switch {
	case experienceYears <= 0:
		return dollars(330_000)
	case experienceYears == 1:
		return dollars(380_000)
	case experienceYears == 2:
		return dollars(430_000)
	case experienceYears == 3:
		return dollars(480_000)
	case experienceYears <= 6: // 4–6 years
		return dollars(530_000)
	case experienceYears <= 9: // 7–9 years
		return dollars(580_000)
	default: // 10+ years
		return dollars(630_000)
	}
}

// Sign checks the player is a signable free agent, not under a §12 buyout lockout and not below
// the §6 floor, then lays a flat `years` contract at `salary` and rosters him. The cap ceiling is
// not enforced, consistent with every other op. Figures snap to $10k.
func Sign(ctx context.Context, w state.TxWriter, mflID, franchiseID string, salary domain.Money, years, experienceYears int) error {
	status, found, err := w.CurrentStatus(ctx, mflID)
	if err != nil {
		return fmt.Errorf("freeagency: sign %q: status: %w", mflID, err)
	}
	if !found || !status.Signable() {
		return fmt.Errorf("freeagency: sign %q: not a signable free agent (status %q, found=%v)", mflID, status, found)
	}

	locked, err := w.ActiveBuyoutLockout(ctx, mflID, deadcap.BuyoutReason, w.Season())
	if err != nil {
		return fmt.Errorf("freeagency: sign %q: buyout lockout: %w", mflID, err)
	}
	if locked {
		return fmt.Errorf("freeagency: sign %q: bought-out player is locked until the following offseason (§12)", mflID)
	}

	salary = domain.RoundToNearest10k(salary)
	// Checked after the $10k snap: under $5k rounds to $0, which would roster a player for free.
	if salary <= 0 {
		return fmt.Errorf("freeagency: sign %q: salary rounds to $0 (below the $10k grid) — enter at least $10k", mflID)
	}
	if floor := MinSalaryFloor(experienceYears); salary < floor {
		return fmt.Errorf("freeagency: sign %q: salary %s is below the §6 minimum %s for %d years experience", mflID, salary, floor, experienceYears)
	}

	if err := w.SignContract(ctx, mflID, franchiseID, salary, years, signingSource, signReason); err != nil {
		return fmt.Errorf("freeagency: sign %q: %w", mflID, err)
	}
	return nil
}

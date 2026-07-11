// Package freeagency holds the free-agency SIGN handler that runs BEHIND the B7a Coordinator.
// Like the other transaction handlers it is implementation detail: depguard
// (transactions-only-through-coordinator) denies importing it from outside internal/transactions,
// so the ONLY way to sign a free agent is Coordinator.Execute(Sign). It neither opens nor commits
// a transaction — it checks eligibility, resolves the rules, and sequences the store's
// SignContract primitive on the shared TxWriter; the enclosing WriteTx makes the whole signing
// atomic.
//
// v1 RECORDS a signing outcome (a commissioner/GM assigns a free agent a new flat contract); it is
// NOT a live auction (bid points, sniping, RFA tenders, comp picks are deferred as multi-user —
// Free_Agency_Design §1).
package freeagency

import (
	"context"
	"fmt"
	"log"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
	"github.com/secureprospective/TheWarRoom/internal/transactions/deadcap"
)

// signingSource tags every contract_years cell a signing lays (the source facet), distinguishing
// signed cells from "seed" / "op" / "extension" cells in the ledger.
const signingSource = "signing"

// signReason is the audit string on the signing's cell-change log rows.
const signReason = "free-agency signing §6"

// dollars converts a whole-dollar figure to exact cents (domain.Money) — the §6 minimum-salary
// table is stated in whole dollars, all already on the $10k grid.
func dollars(d int64) domain.Money { return domain.Money(d * 100) }

// MinSalaryFloor returns the §6 minimum salary for a player with `experienceYears` years of NFL
// experience. It is the rulebook §6 table encoded once as a pure step function. Exported so a test
// pins the table; enforced by Sign only when experience data is available (Free_Agency_Design R1 —
// experience is not in the player facts yet, so v1 skips the floor with a visible flag).
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

// Sign records a free-agency signing against the shared tx: it verifies the player is a signable
// free agent, that he is not under an active §12 buyout lockout, applies the §6 minimum-salary
// floor when experience data is present, then lays a new flat `years`-year contract at `salary`
// and rosters him on `franchiseID` — all in the Coordinator's one spanning transaction. The cap
// ceiling is NOT enforced (Free_Agency_Design R2 — consistent with every other op; CapUsed simply
// reflects the signing). Every figure is snapped to the $10k grid (§1 universal rule). Fails loud
// if the player is not a free agent (rostered / retired / deceased / unknown), is buyout-locked,
// or (when experience is known) is signed below the §6 floor.
//
// experienceKnown gates the min-salary floor: v1 always passes false (no experience source exists
// yet) so the floor is skipped with a visible WARNING; when an experience source lands, the
// Coordinator resolves it and passes (years, true) and the floor is enforced — the forward seam.
func Sign(ctx context.Context, w state.TxWriter, mflID, franchiseID string, salary domain.Money, years, experienceYears int, experienceKnown bool) error {
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
	// Guard AFTER the $10k snap (GLM L4): validate() rejects a non-positive PRE-rounded salary, but
	// a sub-$5k figure rounds to $0 — and with experience unknown (v1) the §6 floor that would
	// catch it is skipped. A $0 contract would roster a player for free, so reject it here.
	if salary <= 0 {
		return fmt.Errorf("freeagency: sign %q: salary rounds to $0 (below the $10k grid) — enter at least $10k", mflID)
	}
	if experienceKnown {
		if floor := MinSalaryFloor(experienceYears); salary < floor {
			return fmt.Errorf("freeagency: sign %q: salary %s is below the §6 minimum %s for %d years experience", mflID, salary, floor, experienceYears)
		}
	} else {
		log.Printf("freeagency: sign %q: §6 minimum-salary floor NOT enforced — no experience data (skipped, Free_Agency_Design R1)", mflID)
	}

	if err := w.SignContract(ctx, mflID, franchiseID, salary, years, signingSource, signReason); err != nil {
		return fmt.Errorf("freeagency: sign %q: %w", mflID, err)
	}
	return nil
}

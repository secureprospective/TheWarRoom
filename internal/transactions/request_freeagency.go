package transactions

import (
	"context"
	"fmt"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
	"github.com/secureprospective/TheWarRoom/internal/transactions/freeagency"
)

// maxSignYears is §6's "maximum contract length: 4 years" for a free-agency signing (distinct
// from the §10 extension's ≤6 total — a fresh signing tops out at 4).
const maxSignYears = 4

// Sign records a free-agency signing (§6): a free agent is rostered on `FranchiseID` with a NEW
// flat contract of `Years` years at `Salary` per year. Eligibility (must be a signable free
// agent), the §12 buyout lockout, and the §6 minimum-salary floor (when experience data exists)
// are all enforced against authoritative state inside apply. The salary IS a caller field (the
// commissioner records the agreed figure — there is no formula to resolve, like a §13 cap relief),
// carried as exact cents and snapped to the $10k grid by the handler. NO cap-ceiling block (R2).
// The signing window (which phases a SIGN is legal in) is enforced by the Coordinator's phase gate.
type Sign struct {
	MFLID       string
	FranchiseID string
	Salary      domain.Money
	Years       int
}

func (Sign) Kind() Kind { return KindSign }
func (Sign) sealed()    {}

// validate enforces the shape a signing must have — a player, a target franchise, a positive
// salary, and a §6-legal length (1..4 years) — before a transaction is opened. Free-agent
// eligibility, the buyout lockout, and the min-salary floor are resolved against real state in apply.
func (s Sign) validate() error {
	if strings.TrimSpace(s.MFLID) == "" {
		return fmt.Errorf("transactions: sign has an empty player id")
	}
	if strings.TrimSpace(s.FranchiseID) == "" {
		return fmt.Errorf("transactions: sign has an empty franchise id")
	}
	if s.Salary <= 0 {
		return fmt.Errorf("transactions: sign salary must be positive")
	}
	if s.Years < 1 || s.Years > maxSignYears {
		return fmt.Errorf("transactions: sign is %d years, must be 1..%d (§6)", s.Years, maxSignYears)
	}
	return nil
}

func (s Sign) apply(ctx context.Context, w state.TxWriter) (int, error) {
	// v1 passes experienceKnown=false: no experience source exists yet, so the §6 min-salary
	// floor is skipped with a visible flag (Free_Agency_Design R1). When a source lands, resolve
	// it here (or pre-tx in the Coordinator) and pass (years, true).
	if err := freeagency.Sign(ctx, w, s.MFLID, s.FranchiseID, s.Salary, s.Years, 0, false); err != nil {
		return 0, fmt.Errorf("sign: %w", err)
	}
	return 1, nil
}

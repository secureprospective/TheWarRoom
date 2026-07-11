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

// maxPlausibleCareer bounds a believable NFL career in years: a resolved draft year older than
// (season − maxPlausibleCareer) is not a real draft year but MFL's undrafted/unknown placeholder
// ("1970" epoch sentinel) or stale data, so experience is treated as unknown → the rookie floor.
// Generous by design — the §6 table saturates at 10+ years, so the only job of this bound is to
// reject the sentinels (which would otherwise spuriously earn the HIGHEST floor).
const maxPlausibleCareer = 30

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
	// draftYear / hasDraftYear are the player's MFL draft year, resolved by Coordinator.ExecuteSign
	// from the players-DB Directory — NOT caller fields (unexported so no caller supplies them). They
	// feed the §6 min-salary floor: apply derives experience = season − draftYear against the
	// authoritative in-tx season. A Sign constructed directly (hasDraftYear=false) resolves to 0
	// experience → the rookie floor.
	draftYear    int
	hasDraftYear bool
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
	// Derive §6 experience from the resolved draft year against the AUTHORITATIVE in-tx season
	// (w.Season()): experience = season − draftYear (a rookie's draft class = 0). Missing draft
	// data, or a draft year outside a plausible career window (MFL's "1970"/"0" sentinels, or a
	// future year), resolves to 0 → the rookie floor (Christopher's missing → rookie-floor ruling).
	experienceYears := 0
	if s.hasDraftYear {
		if season := w.Season(); s.draftYear >= season-maxPlausibleCareer && s.draftYear <= season {
			experienceYears = season - s.draftYear
		}
	}
	if err := freeagency.Sign(ctx, w, s.MFLID, s.FranchiseID, s.Salary, s.Years, experienceYears); err != nil {
		return 0, fmt.Errorf("sign: %w", err)
	}
	return 1, nil
}

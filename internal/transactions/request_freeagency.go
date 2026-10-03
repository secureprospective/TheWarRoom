package transactions

import (
	"context"
	"fmt"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
	"github.com/secureprospective/TheWarRoom/internal/transactions/freeagency"
)

// maxSignYears is §6's four-year maximum for a signing (an extension's limit is 6 total).
const maxSignYears = 4

// maxPlausibleCareer rejects MFL's placeholder draft years (the "1970" sentinel), which would
// otherwise earn the highest floor. Such a player gets the rookie floor.
const maxPlausibleCareer = 30

// Sign rosters a free agent on FranchiseID with a new flat contract: Years years at Salary. Salary
// is the agreed figure, snapped to $10k by the handler. Eligibility, the §12 buyout lockout and the
// §6 floor are enforced in apply; the window by the phase gate. No cap-ceiling block.
type Sign struct {
	MFLID       string
	FranchiseID string
	Salary      domain.Money
	Years       int
	// draftYear is resolved by the coordinator from the players directory and sets experience
	// for the §6 floor. A player with no real draft year (commissioner-created, or missing
	// data) gets the rookie floor, per Christopher's ruling.
	draftYear    int
	hasDraftYear bool
}

func (Sign) Kind() Kind { return KindSign }
func (Sign) sealed()    {}

// enforceRosterLimits checks roster size and the signee's position cap before he is rostered.
// An axis with no cap, or an unresolved position, is skipped: reject only known violations.
func (s Sign) enforceRosterLimits(ctx context.Context, r state.Reader, p RosterPolicy) error {
	roster, ok := r.Roster(s.FranchiseID)
	if !ok {
		return nil // no roster: nothing to exceed
	}
	if err := checkRosterSize(p, s.FranchiseID, len(roster), 1); err != nil {
		return err
	}
	pos, ok := p.Position(ctx, s.MFLID)
	if !ok {
		return nil // unknown position: only roster size applies
	}
	cur, err := positionCount(ctx, p, roster, pos)
	if err != nil {
		return err
	}
	return checkPositionLimit(p, s.FranchiseID, pos, cur, 1)
}

// A signing needs a player, a franchise, a positive salary and 1-4 years.
func (s Sign) resolve(_ *Coordinator, dir Directory) (Request, error) {
	if facts, ok := dir.Facts(s.MFLID); ok && facts.HasDraftYear {
		s.draftYear, s.hasDraftYear = facts.DraftYear, true
	}
	return s, nil
}

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

func (s Sign) apply(ctx context.Context, w state.TxWriter) (applyResult, error) {
	// Experience is the transaction's season minus the draft year. Missing or implausible draft data
	// gives the rookie floor (Christopher's ruling).
	experienceYears := 0
	if s.hasDraftYear {
		if season := w.Season(); s.draftYear >= season-maxPlausibleCareer && s.draftYear <= season {
			experienceYears = season - s.draftYear
		}
	}
	if err := freeagency.Sign(ctx, w, s.MFLID, s.FranchiseID, s.Salary, s.Years, experienceYears); err != nil {
		return applyResult{}, fmt.Errorf("sign: %w", err)
	}
	// The new cap salary shows after commit.
	return applyResult{PlayersAffected: 1}, nil
}

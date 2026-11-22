package history

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

// olineHandoutAuthorizer is deliberately narrower than any human role: the app itself may only
// hand out an unsold O-Line card to an automatic franchise after the O-Line auction closes.
type olineHandoutAuthorizer struct {
	records   []commissioner.Record
	franchise string
	player    string
	now       time.Time
}

func (a olineHandoutAuthorizer) Allow(league, principal, command string) (string, error) {
	if principal != commissioner.OLineHandoutPrincipal || command != commissioner.RosterAdd {
		return "", errors.New("O-Line handout worker can only record O-Line card assignments")
	}
	if err := commissioner.CheckOLineHandout(league, a.records, a.franchise, a.player,
		a.now); err != nil {
		return "", fmt.Errorf("authorize O-Line handout: %w", err)
	}
	return "automatic", nil
}

func authorizeOLineHandout(raw commissioner.RawCommand, principal string,
	auth commissioner.Authorizer, records []commissioner.Record) (string, error) {
	role, err := authorizeCommandWithAuthorizer(raw, principal, auth)
	if err != nil {
		return "", err
	}
	if raw.Args.Waiver == nil {
		return "", errors.New("O-Line handout requires a franchise and card")
	}
	for _, r := range records {
		if r.Command.Request == raw.Request && r.Principal == principal {
			return role, nil
		}
	}
	if err := commissioner.CheckOLineHandout(raw.League, records, raw.Args.Waiver.Franchise,
		raw.Args.Waiver.Player, time.Time{}); err != nil {
		return "", fmt.Errorf("authorize O-Line handout: %w", err)
	}
	return role, nil
}

// ProcessDueOLineHandouts gives each automatic franchise holding no O-Line card one unsold card
// at the minimum price once the league's startup O-Line auction has closed.
func (s *Store) ProcessDueOLineHandouts(ctx context.Context, league string,
	facts nativecheck.Facts) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("stop O-Line handout worker: %w", err)
	}
	skipped := map[string]bool{}
	for {
		records, err := s.nativeRecords(ctx, league)
		if err != nil {
			return err
		}
		now := s.now()
		cand, unfilled, ok, err := commissioner.DueOLineHandouts(league, records, now, skipped)
		if err != nil {
			return fmt.Errorf("read due O-Line handouts: %w", err)
		}
		if !ok {
			if unfilled > 0 {
				log.Printf("oline handout: league=%s unfilled=%d", league, unfilled)
			}
			return nil
		}
		raw := commissioner.RawCommand{
			League: league, Command: commissioner.RosterAdd,
			Args: commissioner.RawArgs{Waiver: &commissioner.RawWaiverArgs{
				Player: cand.Player, Franchise: cand.Franchise,
			}},
			Expected:   int64(len(records)),
			Request:    olineHandoutRequestKey(league, cand.Side, cand.Franchise, cand.Player),
			Reason:     "Automatic O-Line handout after auction close",
			Effective:  "now",
			Visibility: "league",
		}
		auth := olineHandoutAuthorizer{
			records: records, franchise: cand.Franchise, player: cand.Player, now: now,
		}
		if _, err := s.ExecuteCommissionerWithFacts(ctx, raw, commissioner.OLineHandoutPrincipal,
			auth, facts, time.UTC); err != nil {
			var refusal *commissioner.Refusal
			if errors.As(err, &refusal) {
				skipped[cand.Franchise] = true
				continue
			}
			return fmt.Errorf("record O-Line handout: %w", err)
		}
	}
}

func olineHandoutRequestKey(league, side, franchise, player string) string {
	if side != "" {
		return fmt.Sprintf("oline-handout.%s.%s.%s.%s", league, side, franchise, player)
	}
	return fmt.Sprintf("oline-handout.%s.%s.%s", league, franchise, player)
}

func checkOLinePrincipalAppend(raw commissioner.RawCommand, principal string,
	records []commissioner.Record, now time.Time) error {
	switch principal {
	case commissioner.OLineClockPrincipal:
		return checkPendingOLineAppend(raw, records, now)
	case commissioner.OLineHandoutPrincipal:
		return checkOLineHandoutAppend(raw, records, now)
	default:
		return nil
	}
}

func checkOLineHandoutAppend(raw commissioner.RawCommand, records []commissioner.Record,
	now time.Time) error {
	if raw.Command != commissioner.RosterAdd || raw.Args.Waiver == nil {
		return fmt.Errorf("check O-Line handout: %w",
			commissioner.Refused(errors.New("O-Line handout requires a franchise and card")))
	}
	if err := commissioner.CheckOLineHandout(raw.League, records,
		raw.Args.Waiver.Franchise, raw.Args.Waiver.Player, now); err != nil {
		return fmt.Errorf("check O-Line handout: %w", err)
	}
	return nil
}

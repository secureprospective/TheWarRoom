package commissioner

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	nativeapp "github.com/secureprospective/TheWarRoom/internal/app/commissioner/native"
	"github.com/secureprospective/TheWarRoom/internal/commissioner"
)

// TargetNativeLineupSet sets an owner's lineup through the native action gate and the league ledger. The
// lineup must be legal under the league's lineup check, and no player whose game has started can move.
func (a *API) TargetNativeLineupSet(body string) (commissioner.Receipt, error) {
	in, err := nativeapp.DecodeLineupSet([]byte(body))
	if err != nil {
		return commissioner.Receipt{}, fmt.Errorf("lineup input: %w", err)
	}
	receipt, err := a.host.RunNativeOwnerAction(in.League, in.Franchise, "lineup.set", json.RawMessage(body))
	var refused *commissioner.Refusal
	if errors.As(err, &refused) {
		return commissioner.Receipt{}, errors.New(refused.Reason())
	}
	if err != nil {
		return commissioner.Receipt{}, fmt.Errorf("set lineup: %w", err)
	}
	return receipt, nil
}

// TargetNativeLineup reads a franchise's lineup for a week: starters and bench, who is locked and when it
// locks in the owner's time, and the league's verdict. at is an RFC 3339 instant, or empty for now.
func (a *API) TargetNativeLineup(league, franchise string, week int,
	at string) (commissioner.NativeLineup, error) {
	if !strings.HasPrefix(league, "native.") || league == "native." || strings.TrimSpace(league) != league ||
		strings.TrimSpace(franchise) == "" || week < 1 || week > 22 {
		return commissioner.NativeLineup{}, fmt.Errorf("choose an app league, a franchise and a week")
	}
	instant := a.host.CommissionerNow()
	if at != "" {
		parsed, err := time.Parse(time.RFC3339, at)
		if err != nil {
			return commissioner.NativeLineup{}, fmt.Errorf("choose when to read the lineup: %w", err)
		}
		instant = parsed
	}
	if err := a.host.Ready(); err != nil {
		return commissioner.NativeLineup{}, fmt.Errorf("native lineup startup: %w", err)
	}
	principal, err := a.host.CommissionerPrincipal()
	if err != nil {
		return commissioner.NativeLineup{}, fmt.Errorf("native lineup identity: %w", err)
	}
	ctx, cancel := context.WithTimeout(a.host.CommissionerContext(), refreshTimeout)
	defer cancel()
	reading, err := a.host.NativeLineup(ctx, league, franchise, principal, week, instant)
	if err != nil {
		return commissioner.NativeLineup{}, fmt.Errorf("native lineup: %w", err)
	}
	return reading, nil
}

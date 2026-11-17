package leagueservice

import (
	"errors"
	"fmt"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
)

func TestChangeErrorKeepsPlainReasonAndRefusalIdentity(t *testing.T) {
	t.Parallel()
	for _, denied := range []bool{false, true} {
		leaf := errors.New("this change is refused")
		cause := commissioner.Refused(leaf)
		if denied {
			cause = commissioner.Denied(cause)
		}
		got := ChangeError(fmt.Errorf("check change: %w", cause))
		var refusal *commissioner.Refusal
		if got.Error() != "This change is refused" || !errors.Is(got, leaf) ||
			!errors.As(got, &refusal) || refusal.PermissionDenied() != denied {
			t.Fatalf("denied=%v: lost plain reason, cause or classification: %v", denied, got)
		}
	}
}

func TestDispatchRejectsUntrustedRawBeforeHostAccess(t *testing.T) {
	for _, raw := range []commissioner.RawCommand{
		{},
		{Args: commissioner.RawArgs{TradeOutcome: &commissioner.TradeState{}}},
		{Args: commissioner.RawArgs{BidOutcome: &commissioner.BidOutcome{}}},
		{Args: commissioner.RawArgs{WaiverOutcome: &commissioner.WaiverOutcome{}}},
		{Args: commissioner.RawArgs{DraftOutcome: &commissioner.DraftOutcome{}}},
	} {
		if _, err := New(nil).Execute(t.Context(), raw); err == nil {
			t.Fatal("untrusted input reached the host")
		}
	}
}

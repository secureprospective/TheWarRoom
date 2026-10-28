package history

import (
	"context"
	"errors"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
)

func TestOLineClockAuthorityAndCancellation(t *testing.T) {
	t.Parallel()
	auth := olineClockAuthorizer{}
	for _, tc := range []struct {
		principal, command string
		allowed            bool
	}{
		{commissioner.OLineClockPrincipal, commissioner.OLineAuctionClose, true},
		{"creator", commissioner.OLineAuctionClose, false},
		{commissioner.OLineClockPrincipal, commissioner.BidsProcess, false},
		{commissioner.OLineClockPrincipal, commissioner.RosterCut, false},
		{commissioner.OLineClockPrincipal, commissioner.RulesSet, false},
	} {
		role, err := auth.Allow("native.test", tc.principal, tc.command)
		if (err == nil) != tc.allowed || (tc.allowed && role != "automatic") {
			t.Fatalf("%+v: %q %v", tc, role, err)
		}
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if err := (&Store{}).ProcessDueOLines(ctx, "native.test", nil); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled worker read the store: %v", err)
	}
}

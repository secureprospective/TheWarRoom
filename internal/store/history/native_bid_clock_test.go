package history

import (
	"context"
	"errors"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
)

func TestBidClockAuthorityIsOnlyAutomaticProcessing(t *testing.T) {
	auth := clockBidAuthorizer{}
	for _, c := range []struct {
		principal string
		command   string
		allowed   bool
	}{
		{"native.bid.clock", commissioner.BidsProcess, true},
		{"creator", commissioner.BidsProcess, false},
		{"native.waiver.clock", commissioner.BidsProcess, false},
		{"native.bid.clock", commissioner.BidsOverride, false},
		{"native.bid.clock", commissioner.BidPlace, false},
		{"native.bid.clock", commissioner.BidRaise, false},
		{"native.bid.clock", commissioner.BidWithdraw, false},
		{"native.bid.clock", commissioner.RulesSet, false},
		{"native.bid.clock", commissioner.WaiverProcess, false},
		{"native.bid.clock", commissioner.Revert, false},
	} {
		role, err := auth.Allow("native.test", c.principal, c.command)
		if c.allowed {
			if err != nil {
				t.Fatalf("processing refused: %v", err)
			}
			if role != "automatic" {
				t.Fatalf("processing role: %q", role)
			}
			continue
		}
		if err == nil {
			t.Fatalf("clock authority escaped: %s %s", c.principal, c.command)
		}
		if role != "" {
			t.Fatalf("a refused clock command received role %q", role)
		}
	}
}

func TestBidClockAuthorizerDoesNotDependOnLeagueIdentity(t *testing.T) {
	for _, league := range []string{"native.one", "native.two"} {
		role, err := (clockBidAuthorizer{}).Allow(league, "native.bid.clock", commissioner.BidsProcess)
		if err != nil || role != "automatic" {
			t.Fatalf("%s: %s %v", league, role, err)
		}
	}
}

func TestBidClockCancellationStopsBeforeReading(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if err := (&Store{}).ProcessDueBids(ctx, "native.test", nil); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled clock did not stop: %v", err)
	}
}

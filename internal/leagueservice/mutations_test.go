package leagueservice

import (
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
)

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

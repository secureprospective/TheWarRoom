package envelope

import (
	"testing"
	"time"
)

func TestEveryPlanKindHasLocalWithdrawal(t *testing.T) {
	base, at, _ := testEnvelope(t)
	for _, intent := range []string{"roster.ir", "roster.taxi", BidIntent, CutIntent, BuyoutIntent, "contract.extend"} {
		for _, state := range []State{Draft, Blocked, Ready, HandedOff, NotYetDone, NotVerified, Stale, BidPending} {
			if state == BidPending && (intent == "roster.ir" || intent == "roster.taxi" || IsRelease(intent)) {
				continue
			}
			t.Run(intent+"/"+string(state), func(t *testing.T) {
				e := base
				e.spec.Intent, e.state = intent, state
				var gone Envelope
				var err error
				switch {
				case intent == BidIntent:
					gone, err = e.WithdrawBid(at.Add(time.Minute), "Not processed")
				case IsRelease(intent):
					gone, err = e.WithdrawRelease(at.Add(time.Minute), "Not processed")
				case IsContract(intent):
					gone, err = e.WithdrawContract(at.Add(time.Minute))
				default:
					gone, err = e.WithdrawRoster(at.Add(time.Minute))
				}
				if err != nil || gone.State() != Withdrawn {
					t.Fatalf("no local way out: %s %v", gone.State(), err)
				}
				if e.state != state {
					t.Fatal("withdraw changed original value")
				}
			})
		}
	}
}

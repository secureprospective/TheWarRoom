package commissioner

import (
	"encoding/json"
	"testing"
)

// TestRulesMemoProposedStatMatchesFullResolution: a proposed stat decision resolves exactly the rules a
// full read of the trial ledger does, and every proposal against one held ledger shares one resolution.
func TestRulesMemoProposedStatMatchesFullResolution(t *testing.T) {
	b := seededLedger(t)
	held := WithReplayCache(b.records)
	memo := NewRulesMemo()
	for _, command := range []string{StatApprove, StatReject, StatApprove} {
		for _, item := range []string{"a", "b"} {
			trial := scoreTrial(RawCommand{League: "native.test", Command: command, Request: "queue." + item,
				Args: RawArgs{Stat: &RawStatChange{Item: item}}}, held)
			for _, at := range []int{1, 10, 21} {
				want, wantErr := RulesInForce(trial, 2026, seasonMoment(at))
				got, err := memo.InForce(trial, 2026, seasonMoment(at))
				a, _ := json.Marshal(want)
				z, _ := json.Marshal(got)
				if string(a) != string(z) || (err == nil) != (wantErr == nil) {
					t.Fatalf("%s %s at %d: rules differ", command, item, at)
				}
			}
		}
	}
	if memo.Resolved() != 3 {
		t.Fatalf("proposals resolved the held ledger %d times, want once per instant", memo.Resolved())
	}
	other := scoreTrial(RawCommand{League: "native.test", Command: LineupCorrect, Request: "x"}, held)
	if got := memo.ruleLedger(other); len(got) != len(other) {
		t.Fatal("a non-statistics proposal must be read in full")
	}
}

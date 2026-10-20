package history

import (
	"fmt"
	"os"
	"strconv"
	"testing"
	"time"
)

// TestReplayCost grows one league's ledger through the real append path and times a full ledger
// read at each size: the block 0 replay measurement. Opt-in: TWR_REPLAY_N=5000 go test -run TestReplayCost.
func TestReplayCost(t *testing.T) {
	n, _ := strconv.Atoi(os.Getenv("TWR_REPLAY_N"))
	if n == 0 {
		t.Skip("set TWR_REPLAY_N to measure")
	}
	s, c, _ := newStore(t)
	receipt, err := s.ExecuteCommissioner(t.Context(), createCommand(), "creator", nil)
	if err != nil {
		t.Fatal(err)
	}
	entry, revision := receipt.Entry, receipt.Revision
	next := 1000
	for i := 1; i <= n; i++ {
		c.now = c.now.Add(time.Second)
		start := time.Now()
		r, err := s.ExecuteCommissioner(t.Context(), reverseCommand(entry, revision, fmt.Sprintf("r%d", i)), "creator", nil)
		if err != nil {
			t.Fatalf("append %d: %v", i, err)
		}
		appendCost := time.Since(start)
		entry, revision = r.Entry, r.Revision
		if i == next || i == n {
			start = time.Now()
			if _, err := s.CommissionerLedger(t.Context(), "native.test", "creator", nil); err != nil {
				t.Fatal(err)
			}
			t.Logf("events=%d append=%v ledger_read=%v", i+1, appendCost, time.Since(start))
			next += 1000
		}
	}
}

package serverhttp

import (
	"fmt"
	"testing"
)

func TestRequestTallyBounded(t *testing.T) {
	tally := NewRequestTally()
	for i := range 300 {
		tally.add(fmt.Sprintf("command.%d", i), "read", 0, 0)
	}
	_, counts := tally.Snapshot()
	if len(counts) != 256 || counts["other"].Calls != 45 {
		t.Fatalf("counts not bounded: %d keys, other %+v", len(counts), counts["other"])
	}
}

package researchfeeds

import (
	"runtime"
	"testing"
)

func TestParseDepthBoundsHistoryAllocation(t *testing.T) {
	body := depthHistoryFixture(t)
	runtime.GC()
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	rows, err := ParseDepth(body)
	runtime.ReadMemStats(&after)
	if err != nil || len(rows) != 5_000 || rows[0].DT != "2026-10-20T12:00:00Z" {
		t.Fatalf("latest snapshot: %d rows, %v", len(rows), err)
	}
	allocated := after.TotalAlloc - before.TotalAlloc
	t.Logf("100,000 historical rows: %d bytes allocated", allocated)
	if allocated > 80<<20 {
		t.Fatalf("depth decoding allocated %d bytes; budget is 80 MiB", allocated)
	}
}

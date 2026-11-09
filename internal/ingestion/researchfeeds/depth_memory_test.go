package researchfeeds

import (
	"bytes"
	"fmt"
	"runtime"
	"testing"

	"github.com/parquet-go/parquet-go"
)

func depthHistoryFixture(t testing.TB) []byte {
	t.Helper()
	rows := make([]RawDepth, 100_000)
	for i := range rows {
		rows[i] = RawDepth{DT: fmt.Sprintf("2026-10-%02dT12:00:00Z", 1+i%20),
			Team: "NE", Name: "Player", Group: "OFF", Position: "WR", Slot: 1, Rank: 1}
	}
	var body bytes.Buffer
	if err := parquet.Write(&body, rows); err != nil {
		t.Fatal(err)
	}
	return body.Bytes()
}

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

func BenchmarkParseDepthHistory(b *testing.B) {
	body := depthHistoryFixture(b)
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		if _, err := ParseDepth(body); err != nil {
			b.Fatal(err)
		}
	}
}

package researchfeeds

import (
	"bytes"
	"fmt"
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

package commissioner

import (
	"reflect"
	"testing"
	"time"
)

// TestRecordsThroughMatchesAFilteredCopy: whether the ledger is returned as is or filtered, the result
// holds exactly the records at or before the instant, and appending to it never writes into the ledger.
func TestRecordsThroughMatchesAFilteredCopy(t *testing.T) {
	t.Parallel()
	base := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	records := make([]Record, 4, 8)
	for i := range records {
		records[i] = Record{Receipt: Receipt{Entry: string(rune('a' + i)), Revision: int64(i + 1), RecordedAt: base.Add(time.Duration(i) * time.Hour)}}
	}
	for hours := -1; hours <= 4; hours++ {
		at := base.Add(time.Duration(hours) * time.Hour)
		want := []Record{}
		for _, r := range records {
			if !r.Receipt.RecordedAt.After(at) {
				want = append(want, r)
			}
		}
		got := RecordsThrough(records, at)
		if len(got) != len(want) || (len(want) > 0 && !reflect.DeepEqual(got, want)) {
			t.Fatalf("at +%dh: got %d records, want %d", hours, len(got), len(want))
		}
		_ = append(got, Record{Receipt: Receipt{Entry: "appended"}})
		if spare := records[:cap(records)][len(records)]; spare.Receipt.Entry != "" {
			t.Fatalf("at +%dh: an append to the result wrote into the ledger", hours)
		}
	}
}

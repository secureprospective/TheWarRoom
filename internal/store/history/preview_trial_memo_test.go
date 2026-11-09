package history

import (
	"fmt"
	"reflect"
	"slices"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
)

// TestPreviewTrialMemoMatchesPlainFold: the trial ledger a preview reads carries the replay memo so its
// readings fold it once; every reading must equal the reading of the same ledger without the memo.
func TestPreviewTrialMemoMatchesPlainFold(t *testing.T) {
	t.Parallel()
	f := lineupTest(t)
	f.mustSet(t, f.a, f.legal(f.a))
	seen := map[string]bool{}
	for _, starters := range [][]string{f.legal(f.a), f.legal(f.a)[:20]} {
		raw := f.command(t, commissioner.LineupSet, f.a, 1, starters)
		facts, err := f.s.appendFactsFor(t.Context(), raw)
		if err != nil {
			t.Fatal(err)
		}
		tx, err := f.s.pools.Read().BeginTx(t.Context(), nil)
		if err != nil {
			t.Fatal(err)
		}
		records, err := f.s.transactionRecords(t.Context(), tx, raw.League)
		_ = tx.Rollback()
		if err != nil {
			t.Fatal(err)
		}
		plain := append(slices.Clone(records), commissioner.Record{Command: raw,
			Receipt: commissioner.Receipt{Entry: "proposed." + raw.Request, RecordedAt: f.at}})
		want, err := f.s.previewReadings(t.Context(), raw, f.ownerOfA, plain, facts)
		if err != nil {
			t.Fatal(err)
		}
		seen[fmt.Sprint(want)] = true
		memo := commissioner.WithReplayCache(plain)
		for range 2 { // the second read is answered from the memo the first one filled
			got, err := f.s.previewReadings(t.Context(), raw, f.ownerOfA, memo, facts)
			if err != nil {
				t.Fatal(err)
			}
			if len(want) == 0 || !reflect.DeepEqual(got, want) {
				t.Fatalf("memoised trial readings differ from the plain fold:\n got %v\nwant %v", got, want)
			}
		}
	}
	if len(seen) != 2 {
		t.Fatalf("a legal and an illegal lineup must read differently, or the test proves nothing: %v", seen)
	}
}

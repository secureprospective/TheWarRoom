package offers

import (
	"slices"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/app/report"
)

func TestPoolMarketSortsMissingLast(t *testing.T) {
	t.Parallel()
	for _, key := range []string{"projected", "rank", "adp", "aav"} {
		for _, desc := range []bool{false, true} {
			t.Run(key+map[bool]string{false: " ascending", true: " descending"}[desc], func(t *testing.T) {
				if err := (RawPoolScope{Sort: key, Desc: desc}).Validate(); err != nil {
					t.Fatal(err)
				}
				rows := []PoolRow{{ID: "missing", Name: "A missing"}, {ID: "high", Name: "B high"}, {ID: "zero", Name: "Z zero"}, {ID: "tie", Name: "C tie"}}
				for i := 1; i < len(rows); i++ {
					value := 12.0
					if i == 2 {
						value = 0
					}
					for range 4 {
						rows[i].Market = append(rows[i].Market, report.MarketFact{Value: &value, Text: "Not a number"})
					}
				}
				SortPool(rows, RawPoolScope{Sort: key, Desc: desc})
				want := []string{"zero", "high", "tie", "missing"}
				if desc {
					want = []string{"tie", "high", "zero", "missing"}
				}
				got := []string{}
				for _, r := range rows {
					got = append(got, r.ID)
				}
				if !slices.Equal(got, want) {
					t.Fatalf("%v, want %v", got, want)
				}
			})
		}
	}
}

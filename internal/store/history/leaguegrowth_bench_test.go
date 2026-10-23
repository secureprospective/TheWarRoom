//go:build leaguebench

package history

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
)

// D1 growth curve on the same rule-only ledgers as BenchmarkLeagueWrite: the trend, not the
// realistic absolute (that is BenchmarkLeague/thick3/D1People). Outside bigleague*_test.go so
// adding it does not invalidate the cached thick3 league.
func BenchmarkLeaguePeopleGrowth(b *testing.B) {
	for _, size := range []int{1000, 5000, 10000, 20000, 50000} {
		b.Run(fmt.Sprintf("records=%d", size), func(b *testing.B) {
			s, _ := openBigLeague(b, filepath.Join(b.TempDir(), "people.db"))
			seedLeagueWrite(b, s, size)
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				got, err := s.CommissionerPeople(context.Background(), bigLeagueID, "creator")
				if err != nil {
					b.Fatal(err)
				}
				if got.Revision < int64(size) {
					b.Fatalf("people revision=%d, want at least %d", got.Revision, size)
				}
			}
		})
	}
}

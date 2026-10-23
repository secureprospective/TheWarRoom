//go:build leaguebench

package history

import (
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

// seedClosedSeasons extends the growth workload with real rollover receipts. Every
// season has rule traffic; the authoritative append serializer still builds the bulk.
func seedClosedSeasons(t testing.TB, s *Store, size, firstYear, lastYear int) *bigLeagueBuilder {
	t.Helper()
	g := &bigLeagueBuilder{t: t, s: s, firstYear: firstYear, nextID: 10000000, now: time.Date(firstYear, 4, 1, 12, 0, 0, 0, time.UTC), facts: nativecheck.Facts{}}
	s.SetClock(func() time.Time { return g.now })
	g.start(firstYear, lastYear)
	perSeason := size / (lastYear - firstYear + 1)
	for year := firstYear; year <= lastYear; year++ {
		if year > firstYear {
			g.now = time.Date(year, 4, 1, 12, 0, 0, 0, time.UTC)
			g.command("creator", commissioner.SeasonRollover, commissioner.RawArgs{Rollover: &commissioner.RawRolloverArgs{To: year}}, false)
		}
		seedGrowthRules(t, s, g, perSeason)
	}
	return g
}

func BenchmarkLeagueClosedBooks(b *testing.B) {
	for _, size := range []int{1000, 10000} {
		for _, decision := range []string{"D9", "D1"} {
			b.Run(fmt.Sprintf("%s/records=%d", decision, size), func(b *testing.B) {
				s, _ := openBigLeague(b, filepath.Join(b.TempDir(), "closed.db"))
				g := seedClosedSeasons(b, s, size, 2020, 2023)
				b.ReportAllocs()
				b.ResetTimer()
				for b.Loop() {
					if decision == "D9" {
						g.set("cap.amount", fmt.Sprintf("%d", 100000000000000+g.revision%2))
					} else {
						got, err := s.CommissionerPeople(b.Context(), bigLeagueID, "creator")
						if err != nil {
							b.Fatal(err)
						}
						if got.Revision != g.revision {
							b.Fatalf("revision=%d want %d", got.Revision, g.revision)
						}
					}
				}
			})
		}
	}
}

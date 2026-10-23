//go:build leaguebench

package history

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func benchmarkLeague(b *testing.B, tier string) (*Store, int) {
	b.Helper()
	cache := cachedBigLeague(b, tier)
	path := filepath.Join(os.Getenv("BENCH_LEAGUE_BINARY_DIR"), tier+".db")
	copyLeagueFile(b, cache, path)
	s, _ := openBigLeague(b, path)
	volumes := leagueVolumes(tier == "double")
	season := volumes[len(volumes)-1].year
	assets, err := s.NativeAssets(context.Background(), bigLeagueID)
	if err != nil {
		b.Fatal(err)
	}
	body, err := json.Marshal(assets.Snapshot)
	if err != nil {
		b.Fatal(err)
	}
	if os.Getenv("BENCH_LEAGUE_SCRATCH") != "" {
		root := bigLeagueRoot(b, "BENCH_LEAGUE_SCRATCH")
		if err := root.WriteFile(tier+".json", body, 0600); err != nil {
			b.Fatal(err)
		}
	}
	s.SetClock(func() time.Time { return time.Date(season, 6, 1, 12, 0, 0, 0, time.UTC) })
	installBigLeagueLineups(b, s)
	return s, season
}

func BenchmarkLeague(b *testing.B) {
	for _, tier := range []string{"legacy", "double"} {
		b.Run(tier, func(b *testing.B) {
			s, season := benchmarkLeague(b, tier)
			copies := 9
			if tier == "double" {
				copies = 18
			}
			b.Run("D1People", func(b *testing.B) {
				b.ReportAllocs()
				for b.Loop() {
					got, err := s.CommissionerPeople(context.Background(), bigLeagueID, "creator")
					if err != nil {
						b.Fatal(err)
					}
					if got.Revision == 0 {
						b.Fatal("empty people reading")
					}
				}
			})
			b.Run("D2WeekWarm", func(b *testing.B) {
				checkBigLeagueWeek(b, s, season)
				parsed := len(s.weekly)
				if parsed != copies {
					b.Fatalf("parsed %d bodies, want %d", parsed, copies)
				}
				b.ReportMetric(0, "csv-parses/op")
				b.ReportAllocs()
				for b.Loop() {
					checkBigLeagueWeek(b, s, season)
				}
				if len(s.weekly) != parsed {
					b.Fatal("warm pass parsed additional CSV bodies")
				}
			})
			b.Run("D2WeekCold", func(b *testing.B) {
				b.ReportMetric(float64(copies), "csv-parses/op")
				b.ReportAllocs()
				for b.Loop() {
					// Fresh Store: both ledger and CSV caches are cold; the open pools remain warm.
					cold := New(s.pools, s.reg)
					cold.SetClock(s.now)
					cold.SetNativeLineups(s.lineups)
					checkBigLeagueWeek(b, cold, season)
					if len(cold.weekly) != copies {
						b.Fatal("cold pass did not parse all distinct bodies")
					}
				}
			})
		})
	}
}

func checkBigLeagueWeek(b testing.TB, s *Store, season int) {
	b.Helper()
	got, err := s.NativeResults(context.Background(), bigLeagueID, season, 1)
	if err != nil {
		b.Fatal(err)
	}
	if !got.Published || len(got.Games) != 16 || len(got.Scores) != 32 {
		b.Fatal("week did not score the whole league")
	}
	for _, score := range got.Scores {
		if len(score.Starters) != bigLeagueStarters {
			b.Fatal("week did not score every starter")
		}
		for _, starter := range score.Starters {
			if starter.Position == "" {
				b.Fatal("starter never reached archived statistics")
			}
		}
	}
}

func BenchmarkLeagueArchive(b *testing.B) {
	for _, tier := range []string{"legacy", "double"} {
		b.Run(tier, func(b *testing.B) {
			path := filepath.Join(os.Getenv("BENCH_LEAGUE_BINARY_DIR"), "archive-"+tier+".db")
			root := bigLeagueRoot(b, "BENCH_LEAGUE_BINARY_DIR")
			_, statErr := root.Stat("archive-" + tier + ".db")
			s, _ := openBigLeague(b, path)
			bodies := 2000
			if tier == "double" {
				bodies *= 2
			}
			if os.IsNotExist(statErr) {
				generateBigLeagueArchive(b, s, bodies)
			} else if statErr != nil {
				b.Fatal(statErr)
			}
			bigLeagueQueryPlan(b, s)
			for _, mode := range []string{"NoMatch", "FirstAccept", "RejectAll"} {
				b.Run("D3"+mode, func(b *testing.B) {
					part := "?fixture=match"
					if mode == "NoMatch" {
						part = "never-present"
					}
					b.ReportAllocs()
					for b.Loop() {
						found, err := s.ArchivedBodies(context.Background(), part, func(_ string, _ []byte, _ time.Time) bool { return mode == "FirstAccept" })
						if err != nil {
							b.Fatal(err)
						}
						if found != (mode == "FirstAccept") {
							b.Fatal("wrong archive acceptance")
						}
					}
				})
			}
		})
	}
}

func generateBigLeagueArchive(b *testing.B, s *Store, bodies int) {
	b.Helper()
	feed, ok := s.reg.Feed("stats_player_week")
	if !ok {
		b.Fatal("weekly feed missing")
	}
	for n := range bodies * 5 {
		body := fmt.Sprintf("SYNTHETIC archive body %08d %s", n%bodies, strings.Repeat("x", 512))
		part := "other"
		if n%10 == 0 {
			part = "match"
		}
		f := archiveFetch(feed.URLFor(2026)+"?fixture="+part, body, t0().Add(time.Duration(n)*time.Second))
		if err := s.Record(context.Background(), f); err != nil {
			b.Fatal(err)
		}
	}
	var fetches, distinct int
	if err := s.pools.Read().QueryRow("SELECT (SELECT count(*) FROM fetch_log), (SELECT count(*) FROM raw_archive)").Scan(&fetches, &distinct); err != nil {
		b.Fatal(err)
	}
	if fetches != bodies*5 || distinct != bodies {
		b.Fatal("synthetic archive density differs from its contract")
	}
	b.Logf("SYNTHETIC D3 bodies=%d fetches=%d", distinct, fetches)
}

func bigLeagueQueryPlan(b *testing.B, s *Store) {
	b.Helper()
	rows, err := s.pools.Read().Query("EXPLAIN QUERY PLAN "+archiveContainsQuery, "?fixture=match")
	if err != nil {
		b.Fatal(err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			b.Error(err)
		}
	}()
	for rows.Next() {
		var id, parent, unused int
		var detail string
		if err := rows.Scan(&id, &parent, &unused, &detail); err != nil {
			b.Fatal(err)
		}
		b.Logf("D3 PLAN %d %d %s", id, parent, detail)
	}
	if err := rows.Err(); err != nil {
		b.Fatal(err)
	}
}

func bigLeagueRoot(b *testing.B, variable string) *os.Root {
	b.Helper()
	root, err := os.OpenRoot(os.Getenv(variable))
	if err != nil {
		b.Fatal(err)
	}
	b.Cleanup(func() {
		if err := root.Close(); err != nil {
			b.Error(err)
		}
	})
	return root
}

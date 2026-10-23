//go:build leaguebench

package history

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

const bigLeagueStarters = 20

type bigLeagueLineups map[string][]commissioner.Starter

func (l bigLeagueLineups) WeekLineup(_ context.Context, _ []commissioner.Record, _, franchise string, _, _ int) (commissioner.WeekLineup, error) {
	return commissioner.WeekLineup{Set: true, Starters: l[franchise]}, nil
}

func installBigLeagueLineups(t testing.TB, s *Store) bigLeagueLineups {
	t.Helper()
	assets, err := s.NativeAssets(context.Background(), bigLeagueID)
	if err != nil {
		t.Fatal(err)
	}
	lineups := bigLeagueLineups{}
	for _, p := range assets.Snapshot.Players {
		if len(lineups[p.Franchise]) < bigLeagueStarters {
			lineups[p.Franchise] = append(lineups[p.Franchise], commissioner.Starter{Player: p.ID, Name: p.Name})
		}
	}
	for n := range 32 {
		if len(lineups[bigTeam(n)]) != bigLeagueStarters {
			t.Fatal("not enough starters on every roster")
		}
	}
	s.SetNativeLineups(lineups)
	return lineups
}

func setupBigLeagueScoring(t testing.TB, s *Store, season int) {
	t.Helper()
	records, err := s.nativeRecords(context.Background(), bigLeagueID)
	if err != nil {
		t.Fatal(err)
	}
	g := &bigLeagueBuilder{t: t, s: s, revision: int64(len(records)), now: time.Date(season, 5, 1, 12, 0, 0, 0, time.UTC), facts: nativecheck.Facts{}}
	s.SetClock(func() time.Time { return g.now })
	g.command("creator", commissioner.ScheduleGenerate, commissioner.RawArgs{Schedule: &commissioner.RawScheduleArgs{Weeks: 1}}, false)
	g.command("creator", commissioner.SchedulePublish, commissioner.RawArgs{Schedule: &commissioner.RawScheduleArgs{}}, false)
	lineups := installBigLeagueLineups(t, s)
	links := []PlayerIDLink{}
	for n := range 32 {
		for j, p := range lineups[bigTeam(n)] {
			links = append(links, PlayerIDLink{IDType: "gsis", IDValue: fmt.Sprintf("00-%07d", n*bigLeagueStarters+j+1), PlayerID: p.Player})
		}
	}
	if _, err := s.LinkPlayerIDs(context.Background(), "dynastyprocess", links); err != nil {
		t.Fatal(err)
	}
	fetches, copies := 24, 9
	feed, ok := s.reg.Feed("stats_player_week")
	if !ok {
		t.Fatal("weekly feed missing")
	}
	for n := range fetches {
		body := bigLeagueStatsCSV(t, season, n%copies)
		f := archiveFetch(feed.URLFor(season), body, g.now.Add(time.Duration(n)*time.Second))
		if err := s.Record(context.Background(), f); err != nil {
			t.Fatal(err)
		}
	}
	t.Logf("SYNTHETIC D2 starters=%d fetches=%d distinct=%d rows-per-body=%d", 32*bigLeagueStarters, fetches, copies, 32*bigLeagueStarters*18)
}

func bigLeagueStatsCSV(t testing.TB, season, version int) string {
	t.Helper()
	body, err := os.ReadFile("../../fantasyscore/source/testdata/weeks.csv")
	if err != nil {
		t.Fatal(err)
	}
	rows, err := csv.NewReader(bytes.NewReader(body)).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	header := rows[0]
	columns := map[string]int{}
	for i, name := range header {
		columns[name] = i
	}
	var template []string
	for _, row := range rows[1:] {
		if row[columns["position"]] == "LB" {
			template = row
			break
		}
	}
	if template == nil {
		t.Fatal("LB statistics template missing")
	}
	var out bytes.Buffer
	w := csv.NewWriter(&out)
	if err := w.Write(header); err != nil {
		t.Fatal(err)
	}
	for player := 1; player <= 32*bigLeagueStarters; player++ {
		for week := 1; week <= 18; week++ {
			row := append([]string(nil), template...)
			row[columns["player_id"]] = fmt.Sprintf("00-%07d", player)
			row[columns["season"]], row[columns["week"]] = strconv.Itoa(season), strconv.Itoa(week)
			row[columns["def_tackles_solo"]] = strconv.Itoa(version + 1)
			row[columns["game_id"]] = fmt.Sprintf("%d_%02d_GEN_SYN", season, week)
			if err := w.Write(row); err != nil {
				t.Fatal(err)
			}
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		t.Fatal(err)
	}
	return out.String()
}

func TestBigLeagueScoringSmoke(t *testing.T) {
	s, _ := openBigLeague(t, filepath.Join(t.TempDir(), "scoring.db"))
	g := &bigLeagueBuilder{t: t, s: s, now: time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC), facts: nativecheck.Facts{}}
	s.SetClock(func() time.Time { return g.now })
	g.command("creator", commissioner.Create, commissioner.RawArgs{Name: "Scoring smoke", Preset: commissioner.PresetLegacy, Season: 2026, FranchiseCount: 32}, false)
	seed := heldSeedCommand(t).Args.Native
	seed.Players = nil
	for n := range 32 * bigLeagueStarters {
		seed.Players = append(seed.Players, commissioner.NativePlayer{ID: fmt.Sprintf("%08d", 10000000+n), Franchise: bigTeam(n / bigLeagueStarters),
			Status: domain.RosterActive, Salary: 100000000, ExpirationYear: 2027, ContractStatus: domain.CStatusUFA})
	}
	g.command("creator", commissioner.NativeSeed, commissioner.RawArgs{Native: seed}, false)
	setupBigLeagueScoring(t, s, 2026)
	checkBigLeagueWeek(t, s, 2026)
	if len(s.weekly) != 9 {
		t.Fatalf("cold parsed %d distinct CSVs, want 9", len(s.weekly))
	}
	checkBigLeagueWeek(t, s, 2026)
	if len(s.weekly) != 9 {
		t.Fatal("warm pass reparsed CSVs")
	}
}

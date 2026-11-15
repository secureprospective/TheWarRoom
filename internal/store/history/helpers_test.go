package history

import (
	"bytes"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/fantasyscore"
	"github.com/secureprospective/TheWarRoom/internal/fantasyscore/source"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/leagueweek"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

func footballBody(t *testing.T) ([]byte, string) {
	t.Helper()
	raw := RawFootballEvidence{Version: 1, Season: 2026, Source: "commissioner reference import", AsOf: time.Date(2026, 10, 24, 12, 0, 0, 0, time.UTC),
		Players: map[string]RawFootballPlayer{"0099": {Identity: "reference.player.99", Facts: nativecheck.PlayerFacts{Name: "Player 99", Position: leaguerules.QB, Team: "DAL", DesignationHeld: true}}},
		Weeks:   []leagueweek.Week{{Number: 1, Games: []leagueweek.Game{{Teams: [2]string{"DAL", "NYG"}, Kickoff: time.Date(2026, 10, 25, 17, 0, 0, 0, time.UTC)}}}}}
	body, err := json.Marshal(raw)
	if err != nil {
		t.Fatal(err)
	}
	hash := sha256.Sum256(body)
	return body, hex.EncodeToString(hash[:])
}

func footballRows(t *testing.T, s *Store) [2]int {
	t.Helper()
	var counts [2]int
	if err := s.pools.Read().QueryRowContext(t.Context(), `SELECT (SELECT COUNT(*) FROM raw_archive),(SELECT COUNT(*) FROM fetch_log)`).Scan(&counts[0], &counts[1]); err != nil {
		t.Fatal(err)
	}
	return counts
}

const linePlayHeader = "game_id,play_id,season,week,season_type,passer_player_id,receiver_player_id,rusher_player_id," +
	"passing_yards,receiving_yards,rushing_yards,touchdown,td_player_id,td_team,return_team,interception,punt_attempt," +
	"kickoff_attempt,play_deleted,two_point_attempt,posteam,play_type,rush_attempt,qb_kneel,lateral_rushing_yards," +
	"rush_touchdown,down,ydstogo,third_down_converted,fourth_down_converted,tackled_for_loss,sack,qb_hit_1_player_id,qb_hit_2_player_id,pass_attempt,complete_pass,yardline_100\n"

func lineImport(t *testing.T) RawFootballEvidence {
	t.Helper()
	raw := scoringImport(t)
	for id, p := range map[string]struct {
		identity, name, pfr string
		position            leaguerules.Position
	}{"0101": {"00-0036223", "Running Back", "HarrNa00", leaguerules.RB}, "0102": {"00-0036900", "Receiver", "PickGe00", leaguerules.WR}} {
		player := raw.Players["0099"]
		player.Identity, player.Facts.Name, player.Facts.Position, player.PFR = p.identity, p.name, p.position, p.pfr
		raw.Players[id] = player
	}
	raw.Stats[0].Plays = linePlayHeader +
		"2025_01_PIT_NYJ,1,2026,1,REG,00-0023459,,,40,NA,NA,0,,,,0,0,0,0,0,PIT,pass,0,0,NA,0,1,10,0,0,0,0,NA,NA,1,1,NA\n" +
		"2025_01_PIT_NYJ,2,2026,1,REG,,,00-0036223,NA,NA,12,0,,,,0,0,0,0,0,PIT,run,1,0,NA,0,2,8,0,0,0,0,NA,NA,0,0,NA\n" +
		"2025_01_PIT_NYJ,3,2026,1,REG,,,00-0036223,NA,NA,1,1,00-0036223,PIT,,0,0,0,0,0,PIT,run,1,0,NA,1,1,1,0,0,0,0,NA,NA,0,0,NA\n" +
		"2025_01_PIT_NYJ,4,2026,1,REG,,,,NA,NA,-1,0,,,,0,0,0,0,0,PIT,qb_kneel,1,1,NA,0,1,10,0,0,1,0,NA,NA,0,0,NA\n" +
		"2025_01_PIT_NYJ,5,2026,1,REG,00-0023459,,,NA,NA,NA,0,,,,0,0,0,0,0,PIT,pass,0,0,NA,0,3,7,0,0,0,1,00-0031111,00-0032222,1,0,NA\n" +
		"2025_01_BAL_BUF,1,2026,1,REG,,,,NA,NA,NA,0,,,,0,0,0,0,0,BAL,pass,0,0,NA,0,1,10,0,0,0,0,NA,NA,1,0,NA\n"
	raw.Stats[0].Rushing = "game_id,season,week,game_type,team,pfr_player_id,rushing_yards_before_contact\n" +
		"2025_01_PIT_NYJ,2026,1,REG,PIT,HarrNa00,25\n" +
		"2025_01_PIT_NYJ,2026,1,REG,PIT,PickGe00,5\n" +
		"2025_01_PIT_NYJ,2026,1,REG,PIT,UnknXx00,7\n"
	return raw
}

func heldSeasonLeague(t *testing.T) *bracketFixture {
	t.Helper()
	f := bracketTest(t)
	f.s.evidence = nil
	links := make([]PlayerIDLink, 0, 32)
	for n := 1; n <= 32; n++ {
		links = append(links, PlayerIDLink{IDType: "gsis", IDValue: heldSeasonGSIS(n), PlayerID: fmt.Sprintf("10%02d", n)})
	}
	if _, err := f.s.LinkPlayerIDs(t.Context(), "dynastyprocess", links); err != nil {
		t.Fatal(err)
	}
	putHeldSeasonWeeks(t, f.s, 0, t0())
	return f
}

func heldSeasonGSIS(n int) string { return fmt.Sprintf("00-%07d", 9000000+n) }

func putHeldSeasonWeeks(t *testing.T, s *Store, bump int, at time.Time) {
	t.Helper()
	body, err := os.ReadFile("../../fantasyscore/source/testdata/weeks.csv")
	if err != nil {
		t.Fatal(err)
	}
	rows, err := csv.NewReader(bytes.NewReader(body)).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	col := map[string]int{}
	for i, name := range rows[0] {
		col[name] = i
	}
	out := [][]string{rows[0]}
	for n := 1; n <= 32; n++ {
		for week := 1; week <= 3; week++ {
			row := append([]string(nil), rows[1]...)
			row[col["player_id"]] = heldSeasonGSIS(n)
			row[col["season"]] = "2026"
			row[col["week"]] = fmt.Sprint(week)
			row[col["game_id"]] = fmt.Sprintf("2026_%02d_PIT_NYJ", week)
			row[col["passing_yards"]] = fmt.Sprint(100 + 7*n + 3*week + bump)
			out = append(out, row)
		}
	}
	var buf strings.Builder
	if err := csv.NewWriter(&buf).WriteAll(out); err != nil {
		t.Fatal(err)
	}
	feed, ok := s.reg.Feed("stats_player_week")
	if !ok {
		t.Fatal("missing weekly feed")
	}
	if err := s.Record(t.Context(), archiveFetch(feed.URLFor(2026), buf.String(), at)); err != nil {
		t.Fatal(err)
	}
}

func liveLine(yards leaguerules.Decimal) source.PlayerWeek {
	return source.PlayerWeek{Position: "WR",
		Line: fantasyscore.StatLine{Quantities: map[leaguerules.StatID]leaguerules.Decimal{
			"receiving_yards": yards}}}
}

func nflverseEvidence(yards leaguerules.Decimal) source.NativeEvidence {
	week := liveLine(yards)
	week.Season, week.Week, week.SeasonType = 2026, 8, "REG"
	held := source.HeldWeek{Authority: source.NFLVerse, SHA256: "final", Week: &week, Missing: []string{}}
	return source.NativeEvidence{Held: &held, Lines: source.DistinctLines([]source.HeldWeek{held}), Missing: []string{}}
}

func countedLive(t *testing.T, evidence source.NativeEvidence) source.LineRevision {
	t.Helper()
	selected := commissioner.FeedVersion(evidence.Versions())
	if selected == nil {
		t.Fatal("no revision is held")
	}
	return *selected
}

func setLiveStats(t *testing.T, s *Store, season, week int, at time.Time, authority string, missing []string, lines map[string]source.PlayerWeek) {
	t.Helper()
	if err := s.SetLiveStats(t.Context(), season, week, at, authority, missing, lines); err != nil {
		t.Fatal(err)
	}
}

func liveEvidence(t *testing.T, s *Store, evidence source.NativeEvidence, scope source.RawNativeScope) source.NativeEvidence {
	t.Helper()
	out, err := s.withLiveStats(t.Context(), evidence, scope)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func liveWeeks(t *testing.T, s *Store, weeks []commissioner.StatWeek, season int) []commissioner.StatWeek {
	t.Helper()
	out, err := s.withLiveStatWeeks(t.Context(), weeks, season)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func flightWaiters(f *readingFlights) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	n := 0
	for _, fl := range f.flights {
		n += fl.waiters
	}
	return n
}

func waitFor(t *testing.T, ok func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for !ok() {
		if time.Now().After(deadline) {
			t.Fatal("timed out")
		}
		time.Sleep(time.Millisecond)
	}
}

type fixedLineups struct{ resultLineups }

func (fixedLineups) lineupsFollowLedger() {}

func scoreCacheLeague(t *testing.T, on bool) *bracketFixture {
	t.Helper()
	f := heldSeasonLeague(t)
	f.s.SetNativeLineups(&fixedLineups{})
	f.s.SetSeasonScoreCache(on)
	return f
}

func mustRecords(t *testing.T, s *Store) []commissioner.Record {
	t.Helper()
	records, err := s.nativeRecords(t.Context(), bracketLeague)
	if err != nil {
		t.Fatal(err)
	}
	return records
}

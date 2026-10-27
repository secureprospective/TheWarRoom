package history

import (
	"compress/gzip"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/fantasyscore/source"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/leagueweek"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
)

func weekPart(t *testing.T, week RawFootballWeek) ([]byte, string) {
	t.Helper()
	body, err := json.Marshal(week)
	if err != nil {
		t.Fatal(err)
	}
	return body, footballDigest(body)
}

// splitImport is lineImport with its one revision moved into a week-1 part.
func splitImport(t *testing.T, s *Store) RawFootballEvidence {
	t.Helper()
	raw := lineImport(t)
	inline := raw.Stats[0]
	body, digest := weekPart(t, RawFootballWeek{Version: 1, Season: 2026, Week: 1, Weekly: inline.Weekly, Plays: inline.Plays, Rushing: inline.Rushing})
	if err := s.ImportFootballWeek(t.Context(), body, digest); err != nil {
		t.Fatal(err)
	}
	raw.Stats[0] = RawFootballStats{AsOf: inline.AsOf, Parts: []string{digest}}
	return raw
}

func TestSplitEvidenceServesWhatInlineEvidenceServes(t *testing.T) {
	s, _ := statStore(t, filepath.Join(t.TempDir(), "history.db"))
	inline := installScoringImport(t, s, lineImport(t))
	split := installScoringImport(t, s, splitImport(t, s))
	scope := source.RawStatScope{GSIS: "00-0023459", Season: 2026, Week: 1}
	want, err := inline.PlayerWeekStats(t.Context(), scope)
	if err != nil || len(want) == 0 {
		t.Fatalf("inline weeks: %v %v", want, err)
	}
	got, err := split.PlayerWeekStats(t.Context(), scope)
	if err != nil || len(got) != len(want) || got[0].Week == nil || want[0].Week == nil || !reflect.DeepEqual(*got[0].Week, *want[0].Week) {
		t.Fatalf("split weeks differ: %+v vs %+v (%v)", got, want, err)
	}
	wantLines, _, _ := inline.LineGames(t.Context(), 2026)
	gotLines, ok, err := split.LineGames(t.Context(), 2026)
	if err != nil || !ok || !reflect.DeepEqual(gotLines.Games, wantLines.Games) {
		t.Fatalf("split line games differ: %+v vs %+v (%v)", gotLines.Games, wantLines.Games, err)
	}
	if split.Digest() == inline.Digest() {
		t.Fatal("split and inline evidence share a digest")
	}
}

func TestSplitEvidenceRefusesBadParts(t *testing.T) {
	s, _ := statStore(t, filepath.Join(t.TempDir(), "history.db"))
	good := splitImport(t, s)
	inline := lineImport(t).Stats[0]
	otherWeek, otherDigest := weekPart(t, RawFootballWeek{Version: 1, Season: 2026, Week: 2, Weekly: inline.Weekly, Plays: inline.Plays})
	if err := s.ImportFootballWeek(t.Context(), otherWeek, otherDigest); err == nil {
		t.Fatal("a week-2 part holding week-1 rows was imported")
	}
	relabelled := RawFootballWeek{Version: 1, Season: 2026, Week: 2, Weekly: inline.Weekly, Plays: strings.ReplaceAll(inline.Plays, ",2026,1,REG,", ",2026,2,REG,")}
	if _, _, err := relabelled.weekFiles(); err == nil {
		t.Fatal("a week-2 part read week-1 statistic rows")
	}
	relabelled.Weekly, relabelled.Plays = strings.ReplaceAll(inline.Weekly, ",2026,1,", ",2026,2,"), inline.Plays
	if _, _, err := relabelled.weekFiles(); err == nil {
		t.Fatal("a week-2 part read week-1 plays")
	}
	for name, change := range map[string]func(*RawFootballEvidence){
		"a part never imported":   func(r *RawFootballEvidence) { r.Stats[0].Parts = []string{strings.Repeat("a", 64)} },
		"parts and inline bodies": func(r *RawFootballEvidence) { r.Stats[0].Weekly = inline.Weekly },
		"the same week twice":     func(r *RawFootballEvidence) { r.Stats[0].Parts = append(r.Stats[0].Parts, r.Stats[0].Parts[0]) },
		"a part from another season": func(r *RawFootballEvidence) {
			r.Season = 2025
			for i := range r.Weeks {
				r.Weeks[i].Games = slices.Clone(r.Weeks[i].Games)
			}
		},
	} {
		bad := good
		bad.Stats = []RawFootballStats{{AsOf: good.Stats[0].AsOf, Parts: slices.Clone(good.Stats[0].Parts)}}
		change(&bad)
		body, err := json.Marshal(bad)
		if err != nil {
			t.Fatal(err)
		}
		if err := s.ImportFootball(t.Context(), body, footballDigest(body)); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
	big, bigDigest := weekPart(t, RawFootballWeek{Version: 1, Season: 2026, Week: 1, Weekly: inline.Weekly, Plays: inline.Plays + strings.Repeat(" ", maxFootballItem)})
	if err := s.ImportFootballWeek(t.Context(), big, bigDigest); err == nil {
		t.Fatal("a part over 8 MiB was imported")
	}
	if _, err := TransferFootball(mustJSON(t, good)); err == nil {
		t.Fatal("a transfer read split evidence without its parts")
	}
}

func mustJSON(t *testing.T, v any) []byte {
	t.Helper()
	body, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return body
}

// A full real 2024 regular season (weekly statistics, play-by-play with the line columns and PFR
// advanced rushing) loads as 18 week parts, each under the per-item cap. Opt in with
// TWR_STUDY_CACHE=<config-study p3 cache directory>.
func TestFullRealSeasonLoadsAsWeekParts(t *testing.T) {
	dir := os.Getenv("TWR_STUDY_CACHE")
	if dir == "" {
		t.Skip("set TWR_STUDY_CACHE to the config study's held nflverse cache")
	}
	weekly := studyCSV(t, filepath.Join(dir, "stats_player_week_2024.csv.gz"))
	plays := studyCSV(t, filepath.Join(dir, "play_by_play_2024.csv.gz"))
	rushing := studyCSV(t, filepath.Join(dir, "advstats_week_rush_2024.csv"))
	ids := studyCSV(t, filepath.Join(dir, "db_playerids.csv"))
	playColumns := []string{"game_id", "play_id", "season", "week", "season_type", "passer_player_id",
		"receiver_player_id", "rusher_player_id", "passing_yards", "receiving_yards", "rushing_yards",
		"touchdown", "td_player_id", "td_team", "return_team", "interception", "punt_attempt", "kickoff_attempt",
		"play_deleted", "two_point_attempt", "posteam", "punt_blocked", "field_goal_attempt", "field_goal_result",
		"play_type", "rush_attempt", "qb_kneel", "lateral_rushing_yards", "rush_touchdown", "down", "ydstogo",
		"third_down_converted", "fourth_down_converted", "tackled_for_loss", "sack", "qb_hit_1_player_id", "qb_hit_2_player_id"}

	s, _ := statStore(t, filepath.Join(t.TempDir(), "history.db"))
	raw := RawFootballEvidence{Version: 1, Season: 2024, Source: "nflverse 2024 (config study p3 cache)",
		AsOf: time.Date(2025, 2, 20, 0, 0, 0, 0, time.UTC), Players: studyPlayers(t, weekly, ids)}
	stats := RawFootballStats{AsOf: raw.AsOf}
	rows, total := 0, 0
	for week := 1; week <= 18; week++ {
		w := fmt.Sprint(week)
		reg := func(r map[string]string) bool { return r["week"] == w && r["season_type"] == "REG" }
		weekRows := weekly.filter(func(r map[string]string) bool { return reg(r) && r["player_id"] != "" })
		rows += len(weekRows.rows)
		weekPlays := plays.filter(reg)
		part := RawFootballWeek{Version: 1, Season: 2024, Week: week, Weekly: weekRows.text(t, nil),
			Plays: weekPlays.text(t, playColumns), Rushing: rushing.filter(func(r map[string]string) bool {
				return r["week"] == w && r["game_type"] == "REG"
			}).text(t, nil)}
		body, digest := weekPart(t, part)
		if len(body) > maxFootballItem {
			t.Fatalf("week %d part is %d bytes, over the cap", week, len(body))
		}
		total += len(body)
		if err := s.ImportFootballWeek(t.Context(), body, digest); err != nil {
			t.Fatalf("week %d: %v", week, err)
		}
		stats.Parts = append(stats.Parts, digest)
		raw.Weeks = append(raw.Weeks, studySchedule(t, week, weekPlays))
	}
	raw.Stats = []RawFootballStats{stats}
	body := mustJSON(t, raw)
	if err := s.ImportFootball(t.Context(), body, footballDigest(body)); err != nil {
		t.Fatal(err)
	}
	pinned, err := s.LoadFootball(t.Context(), footballDigest(body))
	if err != nil {
		t.Fatal(err)
	}
	if got := len(pinned.stats[0].weeks); got != rows {
		t.Fatalf("loaded %d player weeks, the season holds %d", got, rows)
	}
	lines, ok, err := pinned.LineGames(t.Context(), 2024)
	if err != nil || !ok || len(lines.Games) != 2*272 {
		t.Fatalf("line games: %d (held %v) %v, want 544", len(lines.Games), ok, err)
	}
	unpublished := 0
	for _, line := range lines.Games {
		if _, held := line.Quantities["line_rb_yards_before_contact"]; !held {
			unpublished++
		}
	}
	t.Logf("2024 REG: %d player weeks, %d players, %d line games (%d without ybc), parts %d bytes, evidence %d bytes",
		rows, len(raw.Players), len(lines.Games), unpublished, total, len(body))
}

type studyTable struct {
	header []string
	rows   []map[string]string
}

func studyCSV(t *testing.T, path string) studyTable {
	t.Helper()
	f, err := os.Open(path) //nolint:gosec // the operator's own study cache, opted in by TWR_STUDY_CACHE
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	var input io.Reader = f
	if strings.HasSuffix(path, ".gz") {
		z, err := gzip.NewReader(f)
		if err != nil {
			t.Fatal(err)
		}
		input = z
	}
	records, err := csv.NewReader(input).ReadAll()
	if err != nil {
		t.Fatalf("%s: %v", path, err)
	}
	out := studyTable{header: records[0]}
	for _, record := range records[1:] {
		row := make(map[string]string, len(record))
		for i, name := range out.header {
			row[name] = record[i]
		}
		out.rows = append(out.rows, row)
	}
	return out
}

func (s studyTable) filter(keep func(map[string]string) bool) studyTable {
	out := studyTable{header: s.header}
	for _, row := range s.rows {
		if keep(row) {
			out.rows = append(out.rows, row)
		}
	}
	return out
}

// text writes the table back as CSV, keeping only columns when given.
func (s studyTable) text(t *testing.T, columns []string) string {
	t.Helper()
	if columns == nil {
		columns = s.header
	}
	var b strings.Builder
	w := csv.NewWriter(&b)
	if err := w.Write(columns); err != nil {
		t.Fatal(err)
	}
	for _, row := range s.rows {
		record := make([]string, len(columns))
		for i, name := range columns {
			record[i] = row[name]
		}
		if err := w.Write(record); err != nil {
			t.Fatal(err)
		}
	}
	w.Flush()
	return b.String()
}

// studyPlayers lists every 2024 regular-season player with a league position and an MFL id.
func studyPlayers(t *testing.T, weekly, ids studyTable) map[string]RawFootballPlayer {
	t.Helper()
	byGSIS := make(map[string]map[string]string)
	for _, row := range ids.rows {
		if row["gsis_id"] != "" && row["mfl_id"] != "" {
			byGSIS[row["gsis_id"]] = row
		}
	}
	positions := map[string]leaguerules.Position{"QB": leaguerules.QB, "RB": leaguerules.RB, "FB": leaguerules.RB,
		"WR": leaguerules.WR, "TE": leaguerules.TE, "K": leaguerules.PK, "DT": leaguerules.DT, "NT": leaguerules.DT,
		"DE": leaguerules.DE, "LB": leaguerules.LB, "OLB": leaguerules.LB, "ILB": leaguerules.LB, "MLB": leaguerules.LB,
		"CB": leaguerules.CB, "DB": leaguerules.CB, "S": leaguerules.S, "FS": leaguerules.S, "SS": leaguerules.S, "SAF": leaguerules.S}
	players := make(map[string]RawFootballPlayer)
	pfrs := make(map[string]bool)
	for _, row := range weekly.rows {
		position, listed := positions[row["position"]]
		ref, known := byGSIS[row["player_id"]]
		if row["season_type"] != "REG" || !listed || !known {
			continue
		}
		id, err := playerid.New(ref["mfl_id"])
		if err != nil || id.String() != ref["mfl_id"] {
			continue
		}
		if _, done := players[id.String()]; done {
			continue
		}
		pfr := ref["pfr_id"]
		if pfrs[pfr] {
			pfr = ""
		}
		pfrs[pfr] = pfr != ""
		players[id.String()] = RawFootballPlayer{Identity: row["player_id"], PFR: pfr,
			Facts: nativecheck.PlayerFacts{Name: row["player_display_name"], Position: position, Team: row["team"]}}
	}
	return players
}

func studySchedule(t *testing.T, week int, plays studyTable) leagueweek.Week {
	t.Helper()
	seen := make(map[string]bool)
	out := leagueweek.Week{Number: week}
	for _, row := range plays.rows {
		if seen[row["game_id"]] {
			continue
		}
		seen[row["game_id"]] = true
		day, err := time.Parse("2006-01-02", row["game_date"])
		if err != nil {
			t.Fatal(err)
		}
		out.Games = append(out.Games, leagueweek.Game{Teams: [2]string{row["away_team"], row["home_team"]}, Kickoff: day.Add(17 * time.Hour)})
	}
	return out
}

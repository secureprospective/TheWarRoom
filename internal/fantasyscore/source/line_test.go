package source

import (
	"os"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/fantasyscore"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

// Two real 2024 games (2024_01_PIT_ATL; 2024_02_BUF_MIA, where BUF allowed no sack): held nflverse
// play-by-play and PFR advanced rushing, with league_rb = the player's 2024 MFL designation is RB.
// Expected components and Set B scores are the config study's part 3 rows for the same team-games.
func TestLineGamesMatchStudyComponents(t *testing.T) {
	playsFile, err := os.Open("testdata/line_plays.csv")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = playsFile.Close() }()
	plays, err := ReadLinePlays(playsFile)
	if err != nil {
		t.Fatal(err)
	}
	rushFile, err := os.Open("testdata/line_rushing.csv")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = rushFile.Close() }()
	rushing, err := ReadLineRushing(rushFile)
	if err != nil {
		t.Fatal(err)
	}
	rbs := map[string]bool{}
	for _, row := range rushing {
		rbs[row.Fields["pfr_player_id"]] = row.Fields["league_rb"] == "1"
	}
	games, err := LineGames(plays, rushing, func(id string) bool { return rbs[id] })
	if err != nil {
		t.Fatal(err)
	}
	want := map[LineGame]struct {
		ybc, r10, rtd, short, tfl, sacks, hits leaguerules.Decimal
		score                                  leaguerules.Decimal
	}{
		{"2024_01_PIT_ATL", "ATL"}: {28, 3, 0, 1, 1, 2, 7, 8700},
		{"2024_01_PIT_ATL", "PIT"}: {27, 2, 0, 4, 5, 2, 5, 13300},
		{"2024_02_BUF_MIA", "BUF"}: {32, 2, 2, 0, 3, 0, 0, 57800},
		{"2024_02_BUF_MIA", "MIA"}: {46, 6, 0, 2, 8, 2, 5, 25900},
	}
	if len(games) != len(want) {
		t.Fatalf("built %d team-games, want %d", len(games), len(want))
	}
	rules := leaguerules.Default32Scoring()
	for game, w := range want {
		q := games[game].Quantities
		for id, n := range map[leaguerules.StatID]leaguerules.Decimal{"line_rb_yards_before_contact": w.ybc,
			"line_rushes_10_plus": w.r10, "line_rushing_tds": w.rtd, "line_short_conversions": w.short,
			"line_tackled_for_loss": w.tfl, "line_sacks": w.sacks, "line_qb_hits": w.hits} {
			if q[id] != n*10000 {
				t.Errorf("%v %s: %d, study %d", game, id, q[id], n*10000)
			}
		}
		points, _, err := fantasyscore.ScoreUnit(rules, games[game])
		if err != nil || points.Value != w.score {
			t.Errorf("%v: %+v %v, study %d", game, points, err, w.score)
		}
	}
	unpublished, err := LineGames(plays, nil, func(string) bool { return true })
	if err != nil {
		t.Fatal(err)
	}
	if points, _, _ := fantasyscore.ScoreUnit(rules, unpublished[LineGame{"2024_01_PIT_ATL", "ATL"}]); points.State != leaguerules.NeedsRuling {
		t.Fatal("unpublished yards before contact scored as zero")
	}
}

func TestLinePlayDefinitions(t *testing.T) {
	run := map[string]string{"game_id": "g", "play_id": "1", "posteam": "BUF", "play_type": "run", "rush_attempt": "1",
		"qb_kneel": "0", "two_point_attempt": "0", "play_deleted": "0", "rushing_yards": "4", "lateral_rushing_yards": "NA",
		"rush_touchdown": "0", "td_team": "NA", "down": "1", "ydstogo": "10", "third_down_converted": "0",
		"fourth_down_converted": "0", "tackled_for_loss": "0", "sack": "0", "qb_hit_1_player_id": "NA", "qb_hit_2_player_id": "NA"}
	with := func(changes map[string]string) RawLinePlay {
		f := make(map[string]string, len(run))
		for k, v := range run {
			f[k] = v
		}
		for k, v := range changes {
			f[k] = v
		}
		return RawLinePlay{Fields: f}
	}
	for _, tc := range []struct {
		name    string
		changes map[string]string
		id      leaguerules.StatID
		want    leaguerules.Decimal
	}{
		{"lateral yards reach 10", map[string]string{"rushing_yards": "6", "lateral_rushing_yards": "4"}, "line_rushes_10_plus", 1},
		{"9 yards is short of 10", map[string]string{"rushing_yards": "9"}, "line_rushes_10_plus", 0},
		{"a kneel is not a rush", map[string]string{"qb_kneel": "1", "tackled_for_loss": "1"}, "line_tackled_for_loss", 0},
		{"a pass is not a rush", map[string]string{"play_type": "pass", "rushing_yards": "15"}, "line_rushes_10_plus", 0},
		{"own rush touchdown", map[string]string{"rush_touchdown": "1", "td_team": "BUF"}, "line_rushing_tds", 1},
		{"defensive return touchdown", map[string]string{"rush_touchdown": "1", "td_team": "MIA"}, "line_rushing_tds", 0},
		{"3rd and 2 converted", map[string]string{"down": "3", "ydstogo": "2", "third_down_converted": "1"}, "line_short_conversions", 1},
		{"4th and 1 scored", map[string]string{"down": "4", "ydstogo": "1", "rush_touchdown": "1", "td_team": "BUF"}, "line_short_conversions", 1},
		{"3rd and 3 is not short", map[string]string{"down": "3", "ydstogo": "3", "third_down_converted": "1"}, "line_short_conversions", 0},
		{"2nd and 1 is not short", map[string]string{"down": "2", "ydstogo": "1", "first_down": "1"}, "line_short_conversions", 0},
		{"two credited hitters", map[string]string{"play_type": "pass", "qb_hit_1_player_id": "a", "qb_hit_2_player_id": "b"}, "line_qb_hits", 2},
		{"a sack on a pass play", map[string]string{"play_type": "pass", "rush_attempt": "0", "sack": "1"}, "line_sacks", 1},
		{"a deleted play counts nothing", map[string]string{"play_deleted": "1", "sack": "1"}, "line_sacks", 0},
		{"a two-point try counts nothing", map[string]string{"two_point_attempt": "1", "rushing_yards": "12"}, "line_rushes_10_plus", 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			games, err := LineGames([]RawLinePlay{with(tc.changes)}, nil, func(string) bool { return true })
			if err != nil {
				t.Fatal(err)
			}
			if got := games[LineGame{"g", "BUF"}].Quantities[tc.id]; got != tc.want*10000 {
				t.Fatalf("%s = %d, want %d", tc.id, got, tc.want*10000)
			}
		})
	}
}

package source

import (
	"fmt"
	"io"
	"slices"

	"github.com/secureprospective/TheWarRoom/internal/fantasyscore"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

// RawLinePlay is one held play-by-play row with the columns the offensive line unit reads.
type RawLinePlay struct{ Fields map[string]string }

// RawLineRushing is one held PFR advanced weekly rushing row.
type RawLineRushing struct{ Fields map[string]string }

// LineGame names one NFL team's offence in one game.
type LineGame struct {
	GameID string
	Team   string
}

func linePlayColumns() []string {
	return []string{"game_id", "play_id", "posteam", "play_type", "rush_attempt", "qb_kneel", "two_point_attempt",
		"play_deleted", "rushing_yards", "lateral_rushing_yards", "rush_touchdown", "td_team", "down", "ydstogo",
		"third_down_converted", "fourth_down_converted", "tackled_for_loss", "sack", "qb_hit_1_player_id",
		"qb_hit_2_player_id"}
}

func ReadLinePlays(input io.Reader) ([]RawLinePlay, error) {
	var result []RawLinePlay
	err := readRecords(input, func(fields map[string]string) error {
		for _, name := range linePlayColumns() {
			if _, exists := fields[name]; !exists {
				return fmt.Errorf("play-by-play is missing %s", name)
			}
		}
		for _, name := range []string{"rush_attempt", "qb_kneel", "two_point_attempt", "play_deleted", "rush_touchdown",
			"third_down_converted", "fourth_down_converted", "tackled_for_loss", "sack"} {
			if !slices.Contains([]string{"0", "1", "NA", ""}, fields[name]) {
				return fmt.Errorf("play flag %s must be 0, 1 or absent", name)
			}
		}
		result = append(result, RawLinePlay{Fields: fields})
		return nil
	})
	if err != nil {
		return nil, err
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("nflverse play-by-play is empty")
	}
	return result, nil
}

func ReadLineRushing(input io.Reader) ([]RawLineRushing, error) {
	var result []RawLineRushing
	err := readRecords(input, func(fields map[string]string) error {
		for _, name := range []string{"game_id", "team", "pfr_player_id", "rushing_yards_before_contact"} {
			if _, exists := fields[name]; !exists {
				return fmt.Errorf("advanced rushing is missing %s", name)
			}
		}
		result = append(result, RawLineRushing{Fields: fields})
		return nil
	})
	return result, err
}

// LineGames builds each team-game's offensive line components (DEFAULT-32 spec §4.6). Yards before
// contact count only players the league lists at RB (isRB); a game with no advanced rushing rows
// yet leaves that component absent, which the scorer reports as not recorded rather than zero.
func LineGames(plays []RawLinePlay, rushing []RawLineRushing, isRB func(pfrID string) bool) (map[LineGame]fantasyscore.UnitLine, error) {
	result := make(map[LineGame]fantasyscore.UnitLine)
	for _, play := range plays {
		f := play.Fields
		if f["posteam"] == "" || f["posteam"] == "NA" {
			continue
		}
		game := LineGame{GameID: f["game_id"], Team: f["posteam"]}
		line, seen := result[game]
		if !seen {
			line = fantasyscore.UnitLine{Played: true, Quantities: map[leaguerules.StatID]leaguerules.Decimal{
				"line_rushes_10_plus": 0, "line_rushing_tds": 0, "line_short_conversions": 0,
				"line_tackled_for_loss": 0, "line_sacks": 0, "line_qb_hits": 0}}
			result[game] = line
		}
		if f["play_deleted"] == "1" || f["two_point_attempt"] == "1" {
			continue
		}
		if err := countLinePlay(line.Quantities, f); err != nil {
			return nil, fmt.Errorf("play %s/%s: %w", f["game_id"], f["play_id"], err)
		}
	}
	return result, attachBeforeContact(result, rushing, isRB)
}

func countLinePlay(q map[leaguerules.StatID]leaguerules.Decimal, f map[string]string) error {
	if f["play_type"] == "run" && f["rush_attempt"] == "1" && f["qb_kneel"] != "1" {
		if err := countRush(q, f); err != nil {
			return err
		}
	}
	if f["sack"] == "1" {
		q["line_sacks"] += 10000
	}
	for _, name := range []string{"qb_hit_1_player_id", "qb_hit_2_player_id"} {
		if f[name] != "" && f[name] != "NA" {
			q["line_qb_hits"] += 10000
		}
	}
	return nil
}

// countRush scores a qualifying rush: any rusher, lateral yards included, offensive touchdowns only.
func countRush(q map[leaguerules.StatID]leaguerules.Decimal, f map[string]string) error {
	gain := leaguerules.Decimal(0)
	for _, name := range []string{"rushing_yards", "lateral_rushing_yards"} {
		yards, err := optionalDecimal(f[name])
		if err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
		gain += yards
	}
	td := f["rush_touchdown"] == "1" && f["td_team"] == f["posteam"]
	toGo, err := optionalDecimal(f["ydstogo"])
	if err != nil {
		return fmt.Errorf("ydstogo: %w", err)
	}
	short := slices.Contains([]string{"3", "4", "3.0", "4.0"}, f["down"]) && toGo <= 20000 &&
		(f["third_down_converted"] == "1" || f["fourth_down_converted"] == "1" || td)
	for _, row := range []struct {
		hit bool
		id  leaguerules.StatID
	}{{gain >= 100000, "line_rushes_10_plus"}, {td, "line_rushing_tds"}, {short, "line_short_conversions"},
		{f["tackled_for_loss"] == "1", "line_tackled_for_loss"}} {
		if row.hit {
			q[row.id] += 10000
		}
	}
	return nil
}

func attachBeforeContact(games map[LineGame]fantasyscore.UnitLine, rushing []RawLineRushing, isRB func(string) bool) error {
	published := make(map[string]bool)
	sums := make(map[LineGame]leaguerules.Decimal)
	for _, row := range rushing {
		f := row.Fields
		game := LineGame{GameID: f["game_id"], Team: f["team"]}
		if _, played := games[game]; !played {
			return fmt.Errorf("advanced rushing for %s %s has no held play-by-play", game.GameID, game.Team)
		}
		published[game.GameID] = true
		if !isRB(f["pfr_player_id"]) {
			continue
		}
		yards, err := optionalDecimal(f["rushing_yards_before_contact"])
		if err != nil {
			return fmt.Errorf("yards before contact: %w", err)
		}
		sums[game] += yards
	}
	for game, line := range games {
		if published[game.GameID] {
			line.Quantities["line_rb_yards_before_contact"] = sums[game]
		}
	}
	return nil
}

func optionalDecimal(raw string) (leaguerules.Decimal, error) {
	if raw == "" || raw == "NA" {
		return 0, nil
	}
	return decimal(raw)
}

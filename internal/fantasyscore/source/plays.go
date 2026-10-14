package source

import (
	"fmt"
	"io"
	"slices"
	"strconv"

	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

// RawPlay retains the source's player credits and distances, not parsed descriptions.
type RawPlay struct {
	GameID       string
	PlayID       string
	Season       string
	Week         string
	SeasonType   string
	Passer       string
	Receiver     string
	Rusher       string
	Passing      string
	Receiving    string
	Rushing      string
	Touchdown    string
	Scorer       string
	ScoringTeam  string
	ReturnTeam   string
	Interception string
	Punt         string
	Kickoff      string
	Deleted      string
	TwoPoint     string
}

func (r RawPlay) Validate() error {
	if r.GameID == "" {
		return fmt.Errorf("play needs its NFL game")
	}
	for _, raw := range []string{r.Season, r.Week, r.PlayID} {
		n, err := strconv.Atoi(raw)
		if err != nil || n <= 0 {
			return fmt.Errorf("play needs a positive season and week")
		}
	}
	if r.SeasonType != "REG" && r.SeasonType != "POST" {
		return fmt.Errorf("play needs a season type")
	}
	for _, raw := range []string{r.Touchdown, r.Interception, r.Punt, r.Kickoff, r.Deleted, r.TwoPoint} {
		if !slices.Contains([]string{"0", "1", "NA", ""}, raw) {
			return fmt.Errorf("play flag must be 0, 1 or absent")
		}
	}
	for _, raw := range []string{r.Passing, r.Receiving, r.Rushing} {
		if raw != "NA" && raw != "" {
			if _, err := decimal(raw); err != nil {
				return fmt.Errorf("play distance: %w", err)
			}
		}
	}
	return nil
}

func ReadPlays(input io.Reader) ([]RawPlay, error) {
	var result []RawPlay
	err := readRecords(input, func(fields map[string]string) error {
		for _, name := range []string{"game_id", "play_id", "season", "week", "season_type", "passer_player_id",
			"receiver_player_id", "rusher_player_id", "passing_yards", "receiving_yards", "rushing_yards",
			"touchdown", "td_player_id", "td_team", "return_team", "interception", "punt_attempt", "kickoff_attempt", "play_deleted", "two_point_attempt"} {
			if _, exists := fields[name]; !exists {
				return fmt.Errorf("play-by-play is missing %s", name)
			}
		}
		play := rawPlay(fields)
		if err := play.Validate(); err != nil {
			return fmt.Errorf("nflverse play: %w", err)
		}
		result = append(result, play)
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

func rawPlay(f map[string]string) RawPlay {
	return RawPlay{GameID: f["game_id"], PlayID: f["play_id"],
		Season: f["season"], Week: f["week"], SeasonType: f["season_type"],
		Passer: f["passer_player_id"], Receiver: f["receiver_player_id"], Rusher: f["rusher_player_id"],
		Passing: f["passing_yards"], Receiving: f["receiving_yards"], Rushing: f["rushing_yards"],
		Touchdown: f["touchdown"], Scorer: f["td_player_id"], ScoringTeam: f["td_team"],
		ReturnTeam: f["return_team"], Interception: f["interception"],
		Punt: f["punt_attempt"], Kickoff: f["kickoff_attempt"], Deleted: f["play_deleted"],
		TwoPoint: f["two_point_attempt"]}
}

// AttachPlays completes the week lines in place using held play-by-play.
func AttachPlays(weeks []PlayerWeek, plays []RawPlay) error {
	if err := validateCoverage(weeks, plays); err != nil {
		return err
	}
	index := make(map[string]int, len(weeks))
	for i, week := range weeks {
		index[playKey(week.GSIS, week.Season, week.Week)] = i
		for _, id := range []leaguerules.StatID{"long_pass", "long_rush", "long_reception"} {
			week.Line.Events[id] = []leaguerules.Decimal{}
		}
		for _, id := range []leaguerules.StatID{"interception_return_td", "punt_return_td", "kickoff_return_td"} {
			week.Line.Quantities[id] = 0
		}
	}
	for _, play := range plays {
		if play.Deleted == "1" || play.TwoPoint == "1" {
			continue
		}
		season, _ := strconv.Atoi(play.Season)
		week, _ := strconv.Atoi(play.Week)
		attachDistances(weeks, index, play, season, week)
		attachTouchdown(weeks, index, play, season, week)
	}
	return nil
}

func validateCoverage(weeks []PlayerWeek, plays []RawPlay) error {
	games := make(map[string]bool)
	seen := make(map[string]bool)
	for _, play := range plays {
		if err := play.Validate(); err != nil {
			return fmt.Errorf("attach play: %w", err)
		}
		key := play.GameID + "/" + play.PlayID
		if seen[key] {
			return fmt.Errorf("duplicate NFL play %s", key)
		}
		seen[key] = true
		games[play.GameID+"/"+play.SeasonType+"/"+play.Season+"/"+play.Week] = true
	}
	seen = make(map[string]bool)
	for _, week := range weeks {
		key := playKey(week.GSIS, week.Season, week.Week)
		if seen[key] || week.Line.Quantities == nil || week.Line.Events == nil {
			return fmt.Errorf("duplicate or incomplete player week %s", key)
		}
		seen[key] = true
		game := fmt.Sprintf("%s/%s/%d/%d", week.GameID, week.SeasonType, week.Season, week.Week)
		if !games[game] {
			return fmt.Errorf("missing held play-by-play for %s", game)
		}
	}
	return nil
}

func attachDistances(weeks []PlayerWeek, index map[string]int, play RawPlay, season, week int) {
	for _, row := range []struct {
		player, distance string
		stat             leaguerules.StatID
	}{{play.Passer, play.Passing, "long_pass"}, {play.Rusher, play.Rushing, "long_rush"},
		{play.Receiver, play.Receiving, "long_reception"}} {
		i, exists := index[playKey(row.player, season, week)]
		if !exists || weeks[i].GameID != play.GameID || weeks[i].SeasonType != play.SeasonType ||
			row.distance == "NA" || row.distance == "" {
			continue
		}
		distance, _ := decimal(row.distance)
		weeks[i].Line.Events[row.stat] = append(weeks[i].Line.Events[row.stat], distance)
	}
}

// A return touchdown belongs to the returning team; a kicking team that recovers its own kick in the
// end zone has not returned anything, so that touchdown is left for the scoring rules to name.
func attachTouchdown(weeks []PlayerWeek, index map[string]int, play RawPlay, season, week int) {
	if play.Touchdown != "1" || play.ScoringTeam == "" || play.ScoringTeam != play.ReturnTeam {
		return
	}
	i, exists := index[playKey(play.Scorer, season, week)]
	if !exists || weeks[i].GameID != play.GameID || weeks[i].SeasonType != play.SeasonType {
		return
	}
	for _, row := range []struct {
		flag string
		stat leaguerules.StatID
	}{{play.Interception, "interception_return_td"}, {play.Punt, "punt_return_td"},
		{play.Kickoff, "kickoff_return_td"}} {
		if row.flag == "1" {
			weeks[i].Line.Quantities[row.stat] += 10000
		}
	}
}

func playKey(player string, season, week int) string {
	return fmt.Sprintf("%s/%d/%d", player, season, week)
}

package commissioner

import (
	"fmt"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/leagueweek"
)

type FranchiseGame struct {
	Week     int        `json:"week"`
	Game     string     `json:"game,omitempty"`
	Opponent string     `json:"opponent,omitempty"`
	Venue    string     `json:"venue"`
	Kickoff  *time.Time `json:"kickoff,omitempty"`
}

type FranchiseSchedule struct {
	League    string          `json:"league"`
	Franchise string          `json:"franchise"`
	Season    int             `json:"season"`
	Revision  int64           `json:"revision"`
	Message   string          `json:"message,omitempty"`
	Games     []FranchiseGame `json:"games"`
}

type NativeWeek struct {
	Schedule FranchiseSchedule `json:"schedule"`
	Current  *FranchiseGame    `json:"current,omitempty"`
	NextWeek int               `json:"nextWeek,omitempty"`
	Message  string            `json:"message,omitempty"`
}

// Opponents are app-owned. Dates describe the held NFL week, not an imported
// fantasy matchup or an inferred weekly Thursday.
func FranchiseScheduleReading(league, franchise string, season int, records []Record,
	weeks []leagueweek.Week, local *time.Location, at time.Time) (FranchiseSchedule, error) {
	if len(records) == 0 || !leagueFranchise(records, franchise) {
		return FranchiseSchedule{}, fmt.Errorf("choose a franchise in this app league")
	}
	if season < 1900 || season > 9999 || local == nil {
		return FranchiseSchedule{}, fmt.Errorf("choose a season and your local time zone")
	}
	published, err := ScheduleSeasonReading(league, records, season, at)
	if err != nil {
		return FranchiseSchedule{}, err
	}
	reading := FranchiseSchedule{League: league, Franchise: franchise, Season: season,
		Revision: published.Revision, Games: []FranchiseGame{}}
	if len(published.Weeks) == 0 {
		reading.Message = "The schedule is not published yet."
		return reading, nil
	}
	for _, week := range published.Weeks {
		game, err := franchiseScheduleGame(week, franchise)
		if err != nil {
			return FranchiseSchedule{}, err
		}
		for _, nfl := range weeks {
			if nfl.Number == week.Week && len(nfl.Games) != 0 {
				kickoff := leagueweek.FirstLock(nfl).In(local)
				game.Kickoff = &kickoff
				break
			}
		}
		reading.Games = append(reading.Games, game)
	}
	return reading, nil
}

func franchiseScheduleGame(week ScheduleWeek, franchise string) (FranchiseGame, error) {
	game := FranchiseGame{Week: week.Week}
	if week.Bye == franchise {
		game.Venue = "Bye"
		return game, nil
	}
	for _, match := range week.Games {
		switch franchise {
		case match.Home:
			game.Game, game.Opponent, game.Venue = match.ID, match.Away, "Home"
			return game, nil
		case match.Away:
			game.Game, game.Opponent, game.Venue = match.ID, match.Home, "Away"
			return game, nil
		}
	}
	return FranchiseGame{}, fmt.Errorf("the published week leaves this franchise without a game or bye")
}

func NativeWeekReading(schedule FranchiseSchedule, at time.Time, weeks []leagueweek.Week) (NativeWeek, error) {
	if at.IsZero() {
		return NativeWeek{}, fmt.Errorf("choose the time to read this week")
	}
	reading := NativeWeek{Schedule: schedule, Message: schedule.Message}
	if len(schedule.Games) == 0 {
		return reading, nil
	}
	var scheduled []leagueweek.Week
	for _, game := range schedule.Games {
		found := false
		for _, week := range weeks {
			if week.Number == game.Week && len(week.Games) != 0 {
				scheduled = append(scheduled, week)
				found = true
				break
			}
		}
		if !found {
			reading.Message = "The NFL week dates are not held yet."
			return reading, nil
		}
	}
	current, found := leagueweek.LineupWeek(at, scheduled...)
	if !found {
		reading.Message = "The regular-season schedule is finished."
		return reading, nil
	}
	for i, game := range schedule.Games {
		if game.Week == current.Number {
			reading.Current = &game
			if i+1 < len(schedule.Games) {
				reading.NextWeek = schedule.Games[i+1].Week
			}
			break
		}
	}
	return reading, nil
}

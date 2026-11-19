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
	if len(records) == 0 || (franchise != "" && !leagueFranchise(records, franchise)) {
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
		reading.Message = "The schedule is not published yet. A commissioner must choose Generate schedule, " +
			"then Publish schedule, in Control Room > Season."
		return reading, nil
	}
	for _, week := range published.Weeks {
		if len(week.Games) == 0 && week.Bye == "" {
			continue
		}
		game, err := franchiseScheduleGame(week, franchise)
		if err != nil {
			return FranchiseSchedule{}, err
		}
		game.Kickoff = scheduleWeekKickoff(week.Week, weeks, local)
		reading.Games = append(reading.Games, game)
	}
	return reading, nil
}

func scheduleWeekKickoff(number int, weeks []leagueweek.Week, local *time.Location) *time.Time {
	for _, week := range weeks {
		if week.Number == number && len(week.Games) != 0 {
			kickoff := leagueweek.FirstLock(week).In(local)
			return &kickoff
		}
	}
	return nil
}

func franchiseScheduleGame(week ScheduleWeek, franchise string) (FranchiseGame, error) {
	game := FranchiseGame{Week: week.Week}
	// League-only weeks carry dates, never a selected franchise or matchup.
	if franchise == "" {
		return game, nil
	}
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
	game.Venue = "Bye"
	return game, nil
}

func NativeWeekReading(schedule FranchiseSchedule, at time.Time, weeks []leagueweek.Week) (NativeWeek, error) {
	if at.IsZero() {
		return NativeWeek{}, fmt.Errorf("choose the time to read this week")
	}
	reading := NativeWeek{Schedule: schedule, Message: schedule.Message}
	if len(schedule.Games) == 0 {
		return reading, nil
	}
	// A missing later date cannot erase a current week already determined by the available prefix.
	var scheduled []leagueweek.Week
	missingDates := false
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
			missingDates = true
			break
		}
	}
	current, found := leagueweek.LineupWeek(at, scheduled...)
	if !found {
		if missingDates {
			reading.Message = "The current NFL week cannot be identified from this server's held schedule data. Ask the server operator to update the NFL schedule; changing the league matchup schedule will not fix Pulse."
			return reading, nil
		}
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

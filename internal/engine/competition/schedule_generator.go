package competition

import (
	"fmt"
	"slices"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

type ScheduleGame struct {
	ID   string `json:"id"`
	Home string `json:"home"`
	Away string `json:"away"`
}

type ScheduleWeek struct {
	Week  int            `json:"week"`
	Games []ScheduleGame `json:"games"`
	Bye   string         `json:"bye,omitempty"`
}

// GenerateSchedule selects complete round-robin rounds, preferring division games
// twice as often as other games and rotating byes before repeating them.
func GenerateSchedule(teams []string, divisions map[string]string,
	rules leaguerules.ScheduleRules) ([]ScheduleWeek, error) {
	if len(teams) < 2 || len(teams) > 32 {
		return nil, fmt.Errorf("choose between 2 and 32 franchises")
	}
	if rules.Pattern.State != leaguerules.IsKnown {
		return nil, fmt.Errorf("%s", "Choose how often teams meet in this smaller league.")
	}
	if rules.Pattern.Value != leaguerules.DivisionWeighted {
		return nil, fmt.Errorf("choose a recognised schedule pattern")
	}
	weeks := 13
	if rules.RegularWeeks.State == leaguerules.IsKnown {
		weeks = rules.RegularWeeks.Value
	}
	if weeks < 1 || weeks > 52 {
		return nil, fmt.Errorf("choose between 1 and 52 regular-season weeks")
	}
	order, err := scheduleOrder(teams, divisions)
	if err != nil {
		return nil, err
	}
	rounds := alignedScheduleRounds(order, divisions)
	uses := make([]int, len(rounds))
	byes := make(map[string]int)
	result := make([]ScheduleWeek, 0, weeks)
	for week := 1; week <= weeks; week++ {
		selected, best := 0, -1.0
		for i, round := range rounds {
			score := scheduleRoundScore(round, divisions, uses[i], byes, len(teams))
			if score > best {
				selected, best = i, score
			}
		}
		round := rounds[selected]
		round.Week = week
		round.Games = slices.Clone(round.Games)
		for i := range round.Games {
			if uses[selected]%2 == 1 {
				round.Games[i].Home, round.Games[i].Away = round.Games[i].Away, round.Games[i].Home
			}
			round.Games[i].ID = fmt.Sprintf("week.%d.game.%d", week, i+1)
		}
		result = append(result, round)
		uses[selected]++
		if round.Bye != "" {
			byes[round.Bye]++
		}
	}
	balanceScheduleHomes(result)
	return result, nil
}

func scheduleOrder(teams []string, divisions map[string]string) ([]string, error) {
	seen := make(map[string]bool)
	for _, team := range teams {
		if strings.TrimSpace(team) == "" || seen[team] {
			return nil, fmt.Errorf("choose distinct franchises")
		}
		if strings.TrimSpace(divisions[team]) == "" {
			return nil, fmt.Errorf("choose a division for every franchise")
		}
		seen[team] = true
	}
	order := slices.Clone(teams)
	slices.SortFunc(order, func(a, b string) int {
		if cmp := strings.Compare(divisions[a], divisions[b]); cmp != 0 {
			return cmp
		}
		return strings.Compare(a, b)
	})
	if len(order)%2 == 1 {
		order = append(order, "")
	}
	return order, nil
}

func scheduleRounds(order []string) []ScheduleWeek {
	rounds := make([]ScheduleWeek, 0, len(order)-1)
	for range len(order) - 1 {
		round := ScheduleWeek{Games: []ScheduleGame{}}
		for i := 0; i < len(order)/2; i++ {
			home, away := order[i], order[len(order)-1-i]
			switch {
			case home == "":
				round.Bye = away
			case away == "":
				round.Bye = home
			default:
				round.Games = append(round.Games, ScheduleGame{Home: home, Away: away})
			}
		}
		rounds = append(rounds, round)
		last := order[len(order)-1]
		copy(order[2:], order[1:len(order)-1])
		order[1] = last
	}
	return rounds
}

func scheduleRoundScore(round ScheduleWeek, divisions map[string]string, uses int, byes map[string]int, teamCount int) float64 {
	if round.Bye != "" {
		for _, count := range byes {
			if byes[round.Bye] > count {
				return -1
			}
		}
		// Include franchises whose first bye has not yet happened.
		if len(byes) < teamCount && byes[round.Bye] > 0 {
			return -1
		}
	}
	weight := 1.0
	for _, game := range round.Games {
		if divisions[game.Home] == divisions[game.Away] {
			weight += 1.0 / float64(len(round.Games))
		}
	}
	return weight / float64(uses+1)
}

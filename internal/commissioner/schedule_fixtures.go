package commissioner

import "fmt"

// validateFixtures checks a complete, manually chosen schedule. Omitted teams rest that week;
// empty weeks retain their season number while playoff pairings are not yet known.
func validateFixtures(weeks []ScheduleWeek, count int) error {
	if count < 1 || count > 52 || len(weeks) != count {
		return fmt.Errorf("choose between 1 and 52 consecutive schedule weeks")
	}
	ids := map[string]bool{}
	for i, week := range weeks {
		if week.Week != i+1 {
			return fmt.Errorf("number schedule weeks consecutively from one")
		}
		if err := validateFrozenGames(week, ids); err != nil {
			return err
		}
	}
	return nil
}

func checkedFixtures(weeks []ScheduleWeek, count int, teams []string) ([]ScheduleWeek, error) {
	if err := validateFixtures(weeks, count); err != nil {
		return nil, err
	}
	known := map[string]bool{}
	for _, team := range teams {
		known[team] = true
	}
	for _, week := range weeks {
		if week.Bye != "" && !known[week.Bye] {
			return nil, fmt.Errorf("choose a franchise in this league for the bye")
		}
		for _, game := range week.Games {
			if !known[game.Home] || !known[game.Away] {
				return nil, fmt.Errorf("choose franchises in this league for every game")
			}
		}
	}
	return cloneSchedule(weeks), nil
}

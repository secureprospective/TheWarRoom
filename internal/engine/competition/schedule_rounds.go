package competition

// Complete division rounds make weighting an opponent rule, not an accidental
// consequence of where franchises happen to sit in one circle.
func alignedScheduleRounds(order []string, divisions map[string]string) []ScheduleWeek {
	groups := scheduleDivisionGroups(order, divisions)
	if len(groups) < 2 || len(groups)%2 != 0 || len(groups[0])%2 != 0 {
		return scheduleRounds(order)
	}
	for _, group := range groups {
		if len(group) != len(groups[0]) {
			return scheduleRounds(order)
		}
	}
	var rounds []ScheduleWeek
	for i, group := range groups {
		inside := scheduleRounds(append([]string{}, group...))
		if i == 0 {
			rounds = inside
			continue
		}
		for j, week := range inside {
			rounds[j].Games = append(rounds[j].Games, week.Games...)
		}
	}
	return append(rounds, crossDivisionRounds(groups)...)
}

func scheduleDivisionGroups(order []string, divisions map[string]string) [][]string {
	var groups [][]string
	previous := ""
	for _, team := range order {
		if team == "" {
			return nil
		}
		division := divisions[team]
		if len(groups) == 0 || division != previous {
			groups = append(groups, []string{})
			previous = division
		}
		last := len(groups) - 1
		groups[last] = append(groups[last], team)
	}
	return groups
}

func crossDivisionRounds(groups [][]string) []ScheduleWeek {
	order := make([]int, len(groups))
	for i := range order {
		order[i] = i
	}
	var rounds []ScheduleWeek
	for range len(groups) - 1 {
		for offset := range len(groups[0]) {
			week := ScheduleWeek{Games: []ScheduleGame{}}
			for i := 0; i < len(order)/2; i++ {
				home, away := groups[order[i]], groups[order[len(order)-1-i]]
				for j, team := range home {
					week.Games = append(week.Games, ScheduleGame{Home: team, Away: away[(j+offset)%len(away)]})
				}
			}
			rounds = append(rounds, week)
		}
		last := order[len(order)-1]
		copy(order[2:], order[1:len(order)-1])
		order[1] = last
	}
	return rounds
}

package competition

import "slices"

type scheduleEdge struct {
	week, game int
	home, away string
}

// Reciprocal meetings cancel first. Orient the remaining graph in trails beginning
// at odd-degree teams, so each team's home and away counts differ by at most one.
func balanceScheduleHomes(weeks []ScheduleWeek) {
	pairs := make(map[[2]string][]scheduleEdge)
	var keys [][2]string
	for w, week := range weeks {
		for g, game := range week.Games {
			pair := [2]string{game.Home, game.Away}
			if pair[0] > pair[1] {
				pair[0], pair[1] = pair[1], pair[0]
			}
			if _, exists := pairs[pair]; !exists {
				keys = append(keys, pair)
			}
			pairs[pair] = append(pairs[pair], scheduleEdge{w, g, pair[0], pair[1]})
		}
	}
	var remaining []scheduleEdge
	for _, key := range keys {
		edges := pairs[key]
		for i := 0; i+1 < len(edges); i += 2 {
			orientScheduleEdge(weeks, edges[i], key[0])
			orientScheduleEdge(weeks, edges[i+1], key[1])
		}
		if len(edges)%2 != 0 {
			remaining = append(remaining, edges[len(edges)-1])
		}
	}
	orientScheduleTrails(weeks, remaining)
}

func orientScheduleTrails(weeks []ScheduleWeek, edges []scheduleEdge) {
	degree := make(map[string]int)
	var teams []string
	for _, edge := range edges {
		for _, team := range []string{edge.home, edge.away} {
			if _, exists := degree[team]; !exists {
				teams = append(teams, team)
			}
			degree[team]++
		}
	}
	slices.Sort(teams)
	used := make([]bool, len(edges))
	for {
		start := scheduleTrailStart(teams, degree)
		if start == "" {
			return
		}
		for {
			next := ""
			for i, edge := range edges {
				if used[i] || (edge.home != start && edge.away != start) {
					continue
				}
				orientScheduleEdge(weeks, edge, start)
				used[i] = true
				degree[edge.home]--
				degree[edge.away]--
				next = edge.away
				if next == start {
					next = edge.home
				}
				break
			}
			if next == "" {
				break
			}
			start = next
		}
	}
}

func scheduleTrailStart(teams []string, degree map[string]int) string {
	for _, team := range teams {
		if degree[team]%2 == 1 {
			return team
		}
	}
	for _, team := range teams {
		if degree[team] != 0 {
			return team
		}
	}
	return ""
}

func orientScheduleEdge(weeks []ScheduleWeek, edge scheduleEdge, home string) {
	game := &weeks[edge.week].Games[edge.game]
	game.Home, game.Away = edge.home, edge.away
	if home == edge.away {
		game.Home, game.Away = game.Away, game.Home
	}
}

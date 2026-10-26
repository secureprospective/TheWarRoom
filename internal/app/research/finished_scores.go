package research

import (
	"strconv"
	"time"
)

type Finished struct {
	Rows   map[string]*Score
	Held   int
	Weeks  int
	Latest time.Time
}

// FinishedScores uses only held weeks before the current week, never YTD or live scores.
func FinishedScores(in Snapshot) Finished {
	result := Finished{Rows: map[string]*Score{}, Weeks: max(in.Week-1, 0)}
	for week := 1; week <= result.Weeks; week++ {
		feed := in.Held.Feeds[Key("playerScores", strconv.Itoa(week))]
		if len(feed.Value) == 0 {
			continue
		}
		result.Held++
		if feed.FetchedAt.After(result.Latest) {
			result.Latest = feed.FetchedAt
		}
		for _, entry := range researchRows(feed) {
			id := entry.ID.String()
			score := result.Rows[id]
			if score == nil {
				score = &Score{Position: Player(in.Lookup, in.Mirror, in.Config, id).Position}
				result.Rows[id] = score
			}
			score.Sum += entry.Value
			score.Games++
		}
	}
	return result
}

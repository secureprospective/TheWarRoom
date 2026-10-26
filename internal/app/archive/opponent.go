package archive

import (
	"github.com/secureprospective/TheWarRoom/internal/app/season"
	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
)

func OpponentFallback(weeks []leaguefeed.WeekResult, week int, team string) string {
	for _, w := range weeks {
		if w.Week != week {
			continue
		}
		if g, ok := season.FindGame(w, team); ok {
			if g.Home.Franchise == team {
				return g.Away.Franchise
			}
			return g.Home.Franchise
		}
	}
	return ""
}

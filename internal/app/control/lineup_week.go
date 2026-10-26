package control

import (
	"fmt"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/ingestion/nflschedule"
	"github.com/secureprospective/TheWarRoom/internal/leagueweek"
)

func LineupWeekFrom(now time.Time, read func(int) (nflschedule.RawWeek, error)) (leagueweek.Week, error) {
	for requested := 0; ; {
		raw, err := read(requested)
		if err != nil {
			return leagueweek.Week{}, fmt.Errorf("week refresh: %w", err)
		}
		w, err := nflschedule.ToWeek(raw, now)
		if err != nil {
			return leagueweek.Week{}, fmt.Errorf("week refresh: convert: %w", err)
		}
		if w.Number > 18 || (requested != 0 && w.Number != requested) {
			return leagueweek.Week{}, fmt.Errorf("week refresh: requested %d, received %d", requested, w.Number)
		}
		if lineup, ok := leagueweek.LineupWeek(now, w); ok {
			return lineup, nil
		}
		if w.Number == 18 {
			return leagueweek.Week{}, nil
		}
		requested = w.Number + 1
	}
}

package native

import (
	"context"
	"fmt"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/nflschedule"
	"github.com/secureprospective/TheWarRoom/internal/leagueweek"
)

// ScheduleDates reads local reference bodies only; its caller bounds the resource lifetime.
func ScheduleDates(ctx context.Context, bodies func(context.Context, string, func(string, []byte, time.Time) bool) (bool, error), season int, at time.Time) ([]leagueweek.Week, error) {
	var weeks []leagueweek.Week
	seen := make(map[int]bool)
	_, err := bodies(ctx, "TYPE="+nflschedule.Export,
		func(src string, body []byte, _ time.Time) bool {
			if !ingestion.SameLeaguelessExport(src, nflschedule.Export, season) {
				return false
			}
			raw, err := nflschedule.Parse(body)
			if err != nil {
				return false
			}
			week, err := nflschedule.ToWeek(raw, at)
			if err == nil && !seen[week.Number] {
				weeks = append(weeks, week)
				seen[week.Number] = true
			}
			return false
		})
	if err != nil {
		return nil, fmt.Errorf("read held NFL week dates: %w", err)
	}
	return weeks, nil
}

package native

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/nflschedule"
	"github.com/secureprospective/TheWarRoom/internal/leagueweek"
)

// ArchiveMark names the held copies of one export: how many there are and the newest one. Any new
// or removed copy changes it.
type ArchiveMark struct{ Count, Newest int64 }

// Bodies walks held copies newest first until accept returns true.
type Bodies func(context.Context, string, func(string, []byte, time.Time) bool) (bool, error)

// ScheduleCache keeps the parsed held NFL schedule copies of each season between reads. A league's
// lineup import checks kickoffs once per franchise-week, hundreds of times in one pass; without the
// cache every check re-reads and re-parses every held copy. Weeks are still graded at the instant
// each caller asks for, so a read gives exactly what ScheduleDates gives.
type ScheduleCache struct {
	mu   sync.Mutex
	held map[int]cachedSchedule
}

type cachedSchedule struct {
	mark ArchiveMark
	raws []nflschedule.RawWeek
}

// Dates reads the season's week dates, re-reading the held copies only when their mark changed.
func (c *ScheduleCache) Dates(ctx context.Context, mark func(context.Context, string) (ArchiveMark, error),
	bodies Bodies, season int, at time.Time) ([]leagueweek.Week, error) {
	part := "TYPE=" + nflschedule.Export
	now, err := mark(ctx, part)
	if err != nil {
		return nil, fmt.Errorf("read held NFL week dates: %w", err)
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	held, ok := c.held[season]
	if !ok || held.mark != now {
		raws, err := heldScheduleCopies(ctx, bodies, part, season)
		if err != nil {
			return nil, err
		}
		if c.held == nil {
			c.held = map[int]cachedSchedule{}
		}
		held = cachedSchedule{mark: now, raws: raws}
		c.held[season] = held
	}
	return gradeWeeks(held.raws, at), nil
}

// heldScheduleCopies parses every held copy of the season, newest first.
func heldScheduleCopies(ctx context.Context, bodies Bodies, part string, season int) ([]nflschedule.RawWeek, error) {
	var raws []nflschedule.RawWeek
	_, err := bodies(ctx, part, func(src string, body []byte, _ time.Time) bool {
		if !ingestion.SameLeaguelessExport(src, nflschedule.Export, season) {
			return false
		}
		if raw, err := nflschedule.Parse(body); err == nil {
			raws = append(raws, raw)
		}
		return false
	})
	if err != nil {
		return nil, fmt.Errorf("read held NFL week dates: %w", err)
	}
	return raws, nil
}

// gradeWeeks keeps the newest copy of each week that grades cleanly at the instant.
func gradeWeeks(raws []nflschedule.RawWeek, at time.Time) []leagueweek.Week {
	var weeks []leagueweek.Week
	seen := make(map[int]bool)
	for _, raw := range raws {
		week, err := nflschedule.ToWeek(raw, at)
		if err == nil && !seen[week.Number] {
			weeks = append(weeks, week)
			seen[week.Number] = true
		}
	}
	return weeks
}

package scoring

import (
	"context"
	"math"
	"strconv"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/leagueschedule"
	"github.com/secureprospective/TheWarRoom/internal/normalize"
)

// RemainingSchedule omits the projected-record column when schedule evidence is unavailable.
func RemainingSchedule(ctx context.Context, read func(context.Context) ([]leagueschedule.RawScheduleWeek, domain.Freshness, error)) []leagueschedule.RawScheduleWeek {
	weeks, _, err := read(ctx)
	if err != nil {
		return nil
	}
	return weeks
}

func PlayerAges(lk normalize.Lookup, err error, today time.Time) func(string) float64 {
	return func(mflID string) float64 {
		if err != nil {
			return math.NaN()
		}
		f, ok := lk.Facts(mflID)
		if !ok || !f.HasBirthdate {
			return math.NaN()
		}
		return today.Sub(time.Unix(f.Birthdate, 0)).Hours() / (24 * 365.2425)
	}
}

func SeasonWeeks(last string) int {
	n, err := strconv.Atoi(last)
	if err != nil {
		return 0
	}
	return n
}

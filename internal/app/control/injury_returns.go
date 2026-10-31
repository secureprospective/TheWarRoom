package control

import (
	"iter"
	"strconv"

	"github.com/secureprospective/TheWarRoom/internal/ingestion/injuries"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/nflschedule"
	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
)

// FlattenScheduleWeeks turns held per-week feeds into one season schedule.
func FlattenScheduleWeeks[T any](feeds iter.Seq[T], value func(T) []nflschedule.RawWeek) []nflschedule.RawWeek {
	var weeks []nflschedule.RawWeek
	for feed := range feeds {
		weeks = append(weeks, value(feed)...)
	}
	return weeks
}

// InjuryReturns copies the held injury feed and resolves each expected return against the
// schedule. A week is only named when every earlier week is held, so a partial schedule
// cannot turn a missing week into a later return.
func InjuryReturns(rows []leaguefeed.Injury, weeks []nflschedule.RawWeek) []leaguefeed.Injury {
	out := append([]leaguefeed.Injury(nil), rows...)
	present := make(map[int]bool, len(weeks))
	for _, week := range weeks {
		if number, err := strconv.Atoi(week.Week); err == nil {
			present[number] = true
		}
	}
	_, complete := nflschedule.SeasonGames(weeks)
	for i := range out {
		season, week, ok, reason := injuries.ReturnWeek(out[i].ExpReturn, weeks)
		if ok && !weeksThrough(present, week) {
			ok, reason = false, "schedule not loaded"
		}
		if !ok && reason == "after this season" && !complete {
			reason = "schedule not loaded"
		}
		if ok {
			out[i].ReturnSeason, out[i].ReturnWeek, out[i].ReturnReason = season, week, ""
			continue
		}
		out[i].ReturnSeason, out[i].ReturnWeek, out[i].ReturnReason = 0, 0, reason
	}
	return out
}

func weeksThrough(present map[int]bool, last int) bool {
	if last < 1 {
		return false
	}
	for week := 1; week <= last; week++ {
		if !present[week] {
			return false
		}
	}
	return true
}

package injuries

import (
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/ingestion/nflschedule"
)

// ReturnWeek maps MFL's expected-return date to the first NFL week with a game on or after
// it. An unreadable date is missing data, not a guess; a date after the held schedule is after
// the regular season. The caller supplies the season's schedule.
func ReturnWeek(expReturn string, weeks []nflschedule.RawWeek) (season, week int, ok bool, reason string) {
	date, ok := parseReturnDate(expReturn)
	if !ok {
		return 0, 0, false, "return date not given"
	}
	type datedWeek struct {
		number int
		starts []time.Time
	}
	dated := make([]datedWeek, 0, len(weeks))
	for _, raw := range weeks {
		number, err := strconv.Atoi(raw.Week)
		if err != nil {
			continue
		}
		starts := make([]time.Time, 0, len(raw.Matchups))
		for _, matchup := range raw.Matchups {
			seconds, err := strconv.ParseInt(matchup.Kickoff, 10, 64)
			if err != nil {
				continue
			}
			starts = append(starts, time.Unix(seconds, 0).UTC())
		}
		if len(starts) > 0 {
			dated = append(dated, datedWeek{number: number, starts: starts})
		}
	}
	sort.SliceStable(dated, func(i, j int) bool { return dated[i].number < dated[j].number })
	for _, candidate := range dated {
		for _, start := range candidate.starts {
			if !start.Before(date) {
				return nflSeason(start), candidate.number, true, ""
			}
		}
	}
	return 0, 0, false, "after this season"
}

func parseReturnDate(text string) (time.Time, bool) {
	for _, layout := range []string{"Jan 2, 2006", "January 2, 2006", "2006-01-02", "01/02/2006"} {
		if date, err := time.Parse(layout, strings.TrimSpace(text)); err == nil {
			return date, true
		}
	}
	return time.Time{}, false
}

func nflSeason(start time.Time) int {
	if start.Month() >= time.July {
		return start.Year()
	}
	return start.Year() - 1
}

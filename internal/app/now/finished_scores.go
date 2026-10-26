package now

import (
	"strconv"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/ingestion/nflschedule"
)

// finishedGameAllowance allows four hours from kickoff when MFL holds no final flag.
const finishedGameAllowance = 4 * time.Hour

// FinishedScoreCutoff reports when every game in the held week finished.
func FinishedScoreCutoff(schedule Feed[[]nflschedule.RawWeek], now time.Time) (time.Time, bool) {
	games := NFLGames(schedule.Value, now)
	if len(games) == 0 || !WeekKickedOff(games, now) {
		return time.Time{}, false
	}
	end := time.Time{}
	for _, raw := range schedule.Value[0].Matchups {
		kickoff, _ := strconv.ParseInt(raw.Kickoff, 10, 64)
		at := time.Unix(kickoff, 0).Add(finishedGameAllowance)
		switch raw.GameSecondsRemaining {
		case "0":
			at = schedule.FetchedAt
		case "":
		default:
			return time.Time{}, false
		}
		if now.Before(at) {
			return time.Time{}, false
		}
		if at.After(end) {
			end = at
		}
	}
	return end, !end.IsZero()
}

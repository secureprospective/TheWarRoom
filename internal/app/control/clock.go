package control

import (
	"fmt"
	"strconv"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/nflschedule"
	"github.com/secureprospective/TheWarRoom/internal/leagueclock"
	"github.com/secureprospective/TheWarRoom/internal/leagueweek"
)

// WeekSnapshot owns its Games slice; the root copies it while holding weekMu.
type WeekSnapshot struct {
	Week         leagueweek.Week
	FetchedAt    time.Time
	RefreshError string
}

func WeekClock(in WeekSnapshot, now time.Time) (*leagueclock.LineupLock, domain.Freshness) {
	fresh := domain.Freshness{State: domain.FreshLive, FetchedAt: now.UTC().Format(time.RFC3339)}
	if !in.FetchedAt.IsZero() {
		fresh = domain.Freshness{State: domain.FreshLive, FetchedAt: in.FetchedAt.UTC().Format(time.RFC3339)}
	}
	if in.RefreshError != "" {
		fresh.State = domain.FreshStale
	}
	fresh.Note = WeekNote(in)
	if lock, ok := leagueweek.NextLock(now, in.Week); ok {
		return &leagueclock.LineupLock{Week: in.Week.Number, At: lock}, fresh
	}
	if in.Week.Number != 0 && len(in.Week.Games) > 0 {
		return &leagueclock.LineupLock{Week: in.Week.Number, At: leagueweek.LastLock(in.Week)}, fresh
	}
	return nil, fresh
}

func WeekNote(in WeekSnapshot) string {
	held := "no lineup week"
	if in.Week.Number != 0 {
		held = "week " + strconv.Itoa(in.Week.Number)
	}
	fetched := in.FetchedAt.Format(time.RFC3339)
	switch {
	case in.RefreshError != "" && in.FetchedAt.IsZero():
		return "NFL schedule refresh failed: " + in.RefreshError
	case in.RefreshError != "":
		return fmt.Sprintf("NFL schedule refresh failed: %s; showing %s fetched %s",
			in.RefreshError, held, fetched)
	case in.FetchedAt.IsZero():
		return "NFL schedule not fetched yet"
	}
	return "NFL schedule: " + held + ", fetched " + fetched
}

func Gameday(in WeekSnapshot, now time.Time) string {
	gameday := "Gameday unavailable"
	if in.RefreshError == "" && len(in.Week.Games) > 0 {
		playing, final := 0, 0
		for _, game := range in.Week.Games {
			started := !game.Kickoff.After(now)
			if started && game.Final {
				final++
			} else if started {
				playing++
			}
		}
		gameday = fmt.Sprintf("Week %d · %s in progress · %d final", in.Week.Number, strconv.Itoa(playing)+" NFL game"+gameSuffix(playing), final)
	}
	return gameday
}
func gameSuffix(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

// WindowSnapshot owns every schedule and matchup slice supplied by the root.
type WindowSnapshot struct {
	LastRegular int
	Schedules   map[int][]nflschedule.RawWeek
}

func WindowInputs(in WindowSnapshot, now time.Time) leagueclock.WindowInputs {
	out := leagueclock.WindowInputs{LastRegular: in.LastRegular, FirstKickoff: map[int]time.Time{}}
	for _, rule := range leagueclock.Rules(in.LastRegular) {
		for _, raw := range in.Schedules[rule.Week] {
			w, err := nflschedule.ToWeek(raw, now)
			if err == nil && w.Number == rule.Week && len(w.Games) > 0 {
				out.FirstKickoff[rule.Week] = leagueweek.FirstLock(w)
			}
		}
	}
	return out
}

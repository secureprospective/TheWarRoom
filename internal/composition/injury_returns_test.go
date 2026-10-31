package composition

import (
	"strconv"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/core"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/nflschedule"
	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
)

func TestInjuryReturnsResolvesHeldSchedule(t *testing.T) {
	t.Parallel()
	app := injuryReturnApp([]leaguefeed.Injury{{ExpReturn: "Oct 15, 2026"}}, []nflschedule.RawWeek{
		injuryReturnWeek(t, 1, "2026-09-10"), injuryReturnWeek(t, 2, "2026-09-17"),
		injuryReturnWeek(t, 3, "2026-09-24"), injuryReturnWeek(t, 4, "2026-10-01"),
		injuryReturnWeek(t, 5, "2026-10-08"), injuryReturnWeek(t, 6, "2026-10-15"),
	})
	got := app.injuryReturns()
	if len(got) != 1 || got[0].ReturnSeason != 2026 || got[0].ReturnWeek != 6 || got[0].ReturnReason != "" {
		t.Fatalf("injuryReturns = %+v", got)
	}
}

func TestInjuryReturnsDoesNotGuessPastAPartialSchedule(t *testing.T) {
	t.Parallel()
	app := injuryReturnApp([]leaguefeed.Injury{{ExpReturn: "Feb 15, 2027"}}, []nflschedule.RawWeek{
		injuryReturnWeek(t, 1, "2026-09-10"), injuryReturnWeek(t, 2, "2026-09-17"),
		injuryReturnWeek(t, 3, "2026-09-24"),
	})
	got := app.injuryReturns()
	if got[0].ReturnReason != "schedule not loaded" || got[0].ReturnWeek != 0 {
		t.Fatalf("injuryReturns = %+v", got)
	}
}

func TestInjuryReturnsDoesNotSkipAMissingEarlyWeek(t *testing.T) {
	t.Parallel()
	app := injuryReturnApp([]leaguefeed.Injury{{ExpReturn: "Oct 15, 2026"}}, []nflschedule.RawWeek{
		injuryReturnWeek(t, 2, "2026-09-17"), injuryReturnWeek(t, 3, "2026-09-24"),
		injuryReturnWeek(t, 4, "2026-10-01"), injuryReturnWeek(t, 5, "2026-10-08"),
		injuryReturnWeek(t, 6, "2026-10-15"),
	})
	got := app.injuryReturns()
	if got[0].ReturnReason != "schedule not loaded" || got[0].ReturnWeek != 0 {
		t.Fatalf("injuryReturns = %+v", got)
	}
}

func TestInjuryReturnsReportsAMissingDate(t *testing.T) {
	t.Parallel()
	app := injuryReturnApp([]leaguefeed.Injury{{}}, nil)
	got := app.injuryReturns()
	if got[0].ReturnReason != "return date not given" || got[0].ReturnWeek != 0 {
		t.Fatalf("injuryReturns = %+v", got)
	}
}

func injuryReturnApp(rows []leaguefeed.Injury, weeks []nflschedule.RawWeek) *App {
	app := &App{core: &core.State{SeasonInjuries: core.HeldFeed[[]leaguefeed.Injury]{Value: rows}}}
	app.now.schedules = make(map[int]heldSeasonFeed[[]nflschedule.RawWeek], len(weeks))
	for _, week := range weeks {
		number, _ := strconv.Atoi(week.Week)
		app.now.schedules[number] = heldSeasonFeed[[]nflschedule.RawWeek]{Value: []nflschedule.RawWeek{week}}
	}
	return app
}

func injuryReturnWeek(t *testing.T, week int, start string) nflschedule.RawWeek {
	t.Helper()
	date, err := time.Parse("2006-01-02", start)
	if err != nil {
		t.Fatal(err)
	}
	return nflschedule.RawWeek{
		Week: strconv.Itoa(week),
		Matchups: []nflschedule.RawMatchup{{
			Teams: [2]string{"AAA", "BBB"}, Kickoff: strconv.FormatInt(date.Unix(), 10),
		}},
	}
}

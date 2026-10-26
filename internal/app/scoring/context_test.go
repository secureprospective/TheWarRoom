package scoring

import (
	"context"
	"errors"
	"math"
	"reflect"
	"strconv"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/leagueschedule"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/players"
	"github.com/secureprospective/TheWarRoom/internal/normalize"
)

func TestPlayerAges(t *testing.T) {
	t.Parallel()
	today := time.Unix(1800000000, 0)
	birth := today.Add(-24 * time.Hour * 365)
	lk, err := normalize.NewLookup([]players.RawPlayer{{ID: "0042", Position: "QB", Birthdate: strconv.FormatInt(birth.Unix(), 10)}, {ID: "0099", Position: "RB"}})
	if err != nil {
		t.Fatal(err)
	}
	ages := PlayerAges(lk, nil, today)
	if got, want := ages("0042"), 365/365.2425; math.Abs(got-want) > 1e-12 {
		t.Fatalf("age %v, want %v", got, want)
	}
	for _, id := range []string{"0099", "missing"} {
		if !math.IsNaN(ages(id)) {
			t.Fatalf("age known for %s", id)
		}
	}
	if !math.IsNaN(PlayerAges(lk, errors.New("directory failed"), today)("0042")) {
		t.Fatal("directory failure retained known age")
	}
}

func TestRemainingSchedule(t *testing.T) {
	t.Parallel()
	rows := []leagueschedule.RawScheduleWeek{{Week: "6"}}
	for _, err := range []error{nil, errors.New("schedule failed")} {
		got := RemainingSchedule(context.Background(), func(context.Context) ([]leagueschedule.RawScheduleWeek, domain.Freshness, error) {
			return rows, domain.Freshness{}, err
		})
		if err != nil {
			if got != nil {
				t.Fatal("failed schedule supplied a column")
			}
		} else if !reflect.DeepEqual(got, rows) {
			t.Fatal(got)
		}
	}
}

func TestSeasonWeeks(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		raw  string
		want int
	}{{"13", 13}, {"", 0}, {"bad", 0}, {"19", 19}, {"-1", -1}} {
		if got := SeasonWeeks(tc.raw); got != tc.want {
			t.Errorf("%q: %d", tc.raw, got)
		}
	}
}

package native

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"
)

func TestScheduleDatesSeasonValidationAndFirstValidWeek(t *testing.T) {
	t.Parallel()
	now := time.Unix(1800000000, 0)
	body := func(week int, kick time.Time) []byte {
		return []byte(fmt.Sprintf(`{"nflSchedule":{"week":"%d","matchup":{"kickoff":"%d","team":[{"id":"DET"},{"id":"NYJ"}]}}}`, week, kick.Unix()))
	}
	rows := []struct {
		src  string
		body []byte
	}{
		{"https://reference.test/2025/export?TYPE=nflSchedule&JSON=1", body(1, now)},
		{"https://reference.test/2026/export?TYPE=nflSchedule&L=14432&JSON=1", body(2, now)},
		{"https://reference.test/2026/export?TYPE=nflSchedule&JSON=1", []byte(`{`)},
		{"https://reference.test/2026/export?TYPE=nflSchedule&JSON=1", body(3, now.Add(time.Hour))},
		{"https://reference.test/2026/export?TYPE=nflSchedule&JSON=1", body(3, now.Add(2*time.Hour))},
		{"https://reference.test/2026/export?TYPE=nflSchedule&JSON=1", body(4, now.Add(3*time.Hour))},
	}
	got, err := ScheduleDates(context.Background(), func(_ context.Context, part string, accept func(string, []byte, time.Time) bool) (bool, error) {
		if part != "TYPE=nflSchedule" {
			t.Fatal(part)
		}
		for _, row := range rows {
			if accept(row.src, row.body, now) {
				t.Fatal("schedule scan stopped before all weeks")
			}
		}
		return false, nil
	}, 2026, now)
	if err != nil || len(got) != 3 {
		t.Fatalf("dates: %+v %v", got, err)
	}
	if !reflect.DeepEqual([]int{got[0].Number, got[1].Number, got[2].Number}, []int{2, 3, 4}) || !got[1].Games[0].Kickoff.Equal(now.Add(time.Hour)) {
		t.Fatalf("first valid per week: %+v", got)
	}
	sentinel := errors.New("store failed")
	got, err = ScheduleDates(context.Background(), func(context.Context, string, func(string, []byte, time.Time) bool) (bool, error) {
		return false, sentinel
	}, 2026, now)
	if got != nil || !errors.Is(err, sentinel) || err.Error() != "read held NFL week dates: store failed" {
		t.Fatalf("failure: %+v %v", got, err)
	}
}

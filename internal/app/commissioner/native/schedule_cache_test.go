package native

import (
	"context"
	"fmt"
	"reflect"
	"testing"
	"time"
)

func TestScheduleCacheMatchesScheduleDatesAndRereadsOnlyOnNewCopies(t *testing.T) {
	now := time.Date(2026, 11, 1, 18, 0, 0, 0, time.UTC)
	body := func(week int, kick time.Time) []byte {
		return []byte(fmt.Sprintf(`{"nflSchedule":{"week":"%d","matchup":{"kickoff":"%d","team":[{"id":"DET"},{"id":"NYJ"}]}}}`, week, kick.Unix()))
	}
	held := [][]byte{body(8, now.Add(-time.Hour)), body(7, now.Add(-7*24*time.Hour)), body(8, now.Add(-2*time.Hour))}
	reads := 0
	bodies := func(_ context.Context, _ string, accept func(string, []byte, time.Time) bool) (bool, error) {
		reads++
		for _, b := range held {
			if accept("https://reference.test/2026/export?TYPE=nflSchedule&JSON=1", b, now) {
				return true, nil
			}
		}
		return false, nil
	}
	mark := func(context.Context, string) (ArchiveMark, error) {
		return ArchiveMark{Count: int64(len(held)), Newest: int64(len(held))}, nil
	}
	want, err := ScheduleDates(context.Background(), bodies, 2026, now)
	if err != nil {
		t.Fatal(err)
	}
	reads = 0
	var cache ScheduleCache
	for i := 0; i < 50; i++ {
		got, err := cache.Dates(context.Background(), mark, bodies, 2026, now)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("cached weeks differ:\n got %+v\nwant %+v", got, want)
		}
	}
	if reads != 1 {
		t.Fatalf("50 reads with an unchanged mark read the held copies %d times, want 1", reads)
	}
	held = append([][]byte{body(9, now.Add(6*24*time.Hour))}, held...)
	got, err := cache.Dates(context.Background(), mark, bodies, 2026, now)
	if err != nil {
		t.Fatal(err)
	}
	if reads != 2 || len(got) != len(want)+1 || got[0].Number != 9 {
		t.Fatalf("a new held copy was not read: reads %d, weeks %+v", reads, got)
	}
}

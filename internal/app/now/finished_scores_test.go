package now

import (
	"fmt"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/ingestion/nflschedule"
)

func TestFinishedScoresDue(t *testing.T) {
	t.Parallel()
	kickoff := time.Now().Add(-5 * time.Hour).Truncate(time.Second)
	end := kickoff.Add(finishedGameAllowance)
	h := Feed[[]nflschedule.RawWeek]{FetchedAt: end, Value: []nflschedule.RawWeek{{Week: "5", Matchups: []nflschedule.RawMatchup{
		{Teams: [2]string{"BUF", "NYJ"}, Kickoff: fmt.Sprint(kickoff.Unix()), Scores: [2]string{"20", "10"}, GameSecondsRemaining: "0"},
	}}}}
	for _, tc := range []struct {
		name         string
		fetched, now time.Time
		seconds      string
		want         bool
	}{
		{"before final", end.Add(-time.Hour), end.Add(time.Minute), "0", true},
		{"after final", end.Add(time.Minute), end.Add(time.Hour), "0", false},
		{"still playing", kickoff, end, "1", false},
		{"no final flag", kickoff, end, "", true},
		{"allowance not elapsed", kickoff, end.Add(-time.Minute), "", false},
		{"not kicked off", kickoff, kickoff.Add(-time.Minute), "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h.Value[0].Matchups[0].GameSecondsRemaining = tc.seconds
			if got := finishedScoresDue(tc.fetched, h, tc.now); got != tc.want {
				t.Fatalf("due %t want %t", got, tc.want)
			}
		})
	}
}

func finishedScoresDue(fetched time.Time, schedule Feed[[]nflschedule.RawWeek], now time.Time) bool {
	end, finished := FinishedScoreCutoff(schedule, now)
	return finished && fetched.Before(end)
}

func TestFinishedScoreCutoff(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 10, 25, 12, 0, 0, 0, time.UTC)
	final := nflschedule.RawMatchup{Teams: [2]string{"BUF", "NYJ"}, Kickoff: fmt.Sprint(now.Add(-6 * time.Hour).Unix()), Scores: [2]string{"20", "10"}, GameSecondsRemaining: "0"}
	unflagged := nflschedule.RawMatchup{Teams: [2]string{"DAL", "NYG"}, Kickoff: fmt.Sprint(now.Add(-5 * time.Hour).Unix()), Scores: [2]string{"20", "10"}}
	for _, tc := range []struct {
		name          string
		rows          []nflschedule.RawWeek
		fetched, want time.Time
		finished      bool
	}{
		{"missing schedule", nil, now, time.Time{}, false},
		{"empty week", []nflschedule.RawWeek{{Week: "5"}}, now, time.Time{}, false},
		{"ambiguous weeks", []nflschedule.RawWeek{{Matchups: []nflschedule.RawMatchup{final}}, {Week: "6"}}, now, time.Time{}, false},
		{"final uses fetch time", []nflschedule.RawWeek{{Matchups: []nflschedule.RawMatchup{final}}}, now.Add(-2 * time.Hour), now.Add(-2 * time.Hour), true},
		{"latest game cutoff", []nflschedule.RawWeek{{Matchups: []nflschedule.RawMatchup{final, unflagged}}}, now.Add(-2 * time.Hour), now.Add(-time.Hour), true},
		{"future final observation", []nflschedule.RawWeek{{Matchups: []nflschedule.RawMatchup{final}}}, now.Add(time.Minute), time.Time{}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, finished := FinishedScoreCutoff(Feed[[]nflschedule.RawWeek]{Value: tc.rows, FetchedAt: tc.fetched}, now)
			if finished != tc.finished || !got.Equal(tc.want) {
				t.Fatalf("cutoff %v finished %t; want %v %t", got, finished, tc.want, tc.finished)
			}
		})
	}
}

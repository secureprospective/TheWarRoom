package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/ingestion/nflschedule"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/researchfeeds"
	"github.com/secureprospective/TheWarRoom/internal/leagueweek"
	"github.com/secureprospective/TheWarRoom/internal/mfl"
)

func TestFinishedScoresDue(t *testing.T) {
	kickoff := time.Now().Add(-5 * time.Hour).Truncate(time.Second)
	end := kickoff.Add(finishedGameAllowance)
	h := heldSeasonFeed[[]nflschedule.RawWeek]{fetchedAt: end, value: []nflschedule.RawWeek{{Week: "5", Matchups: []nflschedule.RawMatchup{
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
			h.value[0].Matchups[0].GameSecondsRemaining = tc.seconds
			if got := finishedScoresDue(tc.fetched, h, tc.now); got != tc.want {
				t.Fatalf("due %t want %t", got, tc.want)
			}
		})
	}
}

type finishedScoresTransport struct {
	t        *testing.T
	requests *[]string
}

func (f finishedScoresTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	kind, week := r.URL.Query().Get("TYPE"), r.URL.Query().Get("W")
	var body string
	switch kind {
	case "league":
		body = `{"league":{"baseURL":"https://www99.myfantasyleague.com/2026/home/14432"}}`
	case "playerScores":
		*f.requests = append(*f.requests, week)
		body = fmt.Sprintf(`{"playerScores":{"week":"%s","playerScore":[{"id":"0042","score":"22","week":"%s"}]}}`, week, week)
	default:
		f.t.Fatalf("unexpected request %s", r.URL)
	}
	return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
}

func TestBackgroundFinishedScoresRereadsOnce(t *testing.T) {
	a := targetTestApp(t)
	a.seasonChanged = func(context.Context) {}
	now := time.Now().Truncate(time.Second)
	a.clockNow = func() time.Time { return now }
	a.week = leagueweek.Week{Number: 6}
	a.now.schedules = map[int]heldSeasonFeed[[]nflschedule.RawWeek]{5: {fetchedAt: now, value: []nflschedule.RawWeek{{Week: "5", Matchups: []nflschedule.RawMatchup{
		{Teams: [2]string{"BUF", "NYJ"}, Kickoff: fmt.Sprint(now.Add(-5 * time.Hour).Unix()), Scores: [2]string{"20", "10"}, GameSecondsRemaining: "0"},
	}}}}}
	a.now.byes.fetchedAt = now
	a.research.feeds = map[string]heldSeasonFeed[[]researchfeeds.Feed]{researchKey("playerScores", "5"): {fetchedAt: now.Add(-3 * time.Hour)}}
	requests, pauses := []string{}, 0
	var err error
	a.mflClient, err = mfl.New("", 1000, mfl.WithTransport(finishedScoresTransport{t, &requests}))
	if err != nil {
		t.Fatal(err)
	}
	a.seasonPause = func(context.Context, time.Duration) error { pauses++; return nil }
	a.refreshFinishedScores(a.ctx)
	next := a.now.schedules[5]
	next.fetchedAt = now.Add(time.Hour)
	a.now.schedules[5] = next
	a.clockNow = func() time.Time { return now.Add(time.Hour) }
	a.refreshFinishedScores(a.ctx)
	if strings.Join(requests, ",") != "5,YTD" || pauses != 2 {
		t.Fatal(requests, pauses)
	}
	held := a.research.feeds[researchKey("playerScores", "5")]
	if len(held.value) != 1 || !held.fetchedAt.Equal(now) || held.refreshError != "" {
		t.Fatal(held)
	}
}

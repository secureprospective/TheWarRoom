package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	nowapp "github.com/secureprospective/TheWarRoom/internal/app/now"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/nflschedule"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/researchfeeds"
	"github.com/secureprospective/TheWarRoom/internal/leagueweek"
	"github.com/secureprospective/TheWarRoom/internal/mfl"
)

func TestFinishedScoresDue(t *testing.T) {
	t.Parallel()
	kickoff := time.Now().Add(-5 * time.Hour).Truncate(time.Second)
	end := kickoff.Add(finishedGameAllowance)
	h := heldSeasonFeed[[]nflschedule.RawWeek]{FetchedAt: end, Value: []nflschedule.RawWeek{{Week: "5", Matchups: []nflschedule.RawMatchup{
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
	t.Parallel()
	a := targetTestApp(t)
	a.core.SeasonChanged = func(context.Context) {}
	now := time.Now().Truncate(time.Second)
	a.core.ClockNow = func() time.Time { return now }
	a.core.Week = leagueweek.Week{Number: 6}
	a.now.schedules = map[int]heldSeasonFeed[[]nflschedule.RawWeek]{5: {FetchedAt: now, Value: []nflschedule.RawWeek{{Week: "5", Matchups: []nflschedule.RawMatchup{
		{Teams: [2]string{"BUF", "NYJ"}, Kickoff: fmt.Sprint(now.Add(-5 * time.Hour).Unix()), Scores: [2]string{"20", "10"}, GameSecondsRemaining: "0"},
	}}}}}
	a.nowHeld.PublishByes(nowapp.Feed[[]nflschedule.RawBye]{FetchedAt: now})
	a.research.feeds = map[string]heldSeasonFeed[[]researchfeeds.Feed]{researchKey("playerScores", "5"): {FetchedAt: now.Add(-3 * time.Hour)}}
	requests, pauses := []string{}, 0
	var err error
	a.core.MFLClient, err = mfl.New("", 1000, mfl.WithTransport(finishedScoresTransport{t, &requests}))
	if err != nil {
		t.Fatal(err)
	}
	a.core.SeasonPause = func(context.Context, time.Duration) error { pauses++; return nil }
	a.refreshFinishedScores(a.core.Ctx)
	next := a.now.schedules[5]
	next.FetchedAt = now.Add(time.Hour)
	a.now.schedules[5] = next
	a.core.ClockNow = func() time.Time { return now.Add(time.Hour) }
	a.refreshFinishedScores(a.core.Ctx)
	if strings.Join(requests, ",") != "5,YTD" || pauses != 2 {
		t.Fatal(requests, pauses)
	}
	held := a.research.feeds[researchKey("playerScores", "5")]
	if len(held.Value) != 1 || !held.FetchedAt.Equal(now) || held.RefreshError != "" {
		t.Fatal(held)
	}
}

func finishedScoresDue(fetched time.Time, schedule heldSeasonFeed[[]nflschedule.RawWeek], now time.Time) bool {
	end, finished := finishedScoreCutoff(schedule, now)
	return finished && fetched.Before(end)
}

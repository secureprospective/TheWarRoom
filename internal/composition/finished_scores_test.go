package composition

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
	a.core.SeasonChanged = func(context.Context) {
		if a.core.RefreshMu.TryLock() {
			a.core.RefreshMu.Unlock()
			t.Error("health event moved outside the existing refresh span")
		}
		if a.researchHeld.Snapshot().Feeds[researchKey("playerScores", "5")].FetchedAt.IsZero() {
			t.Error("health event ran before publication")
		}
	}
	now := time.Now().Truncate(time.Second)
	a.core.ClockNow = func() time.Time { return now }
	a.core.Week = leagueweek.Week{Number: 6}
	a.now.schedules = map[int]heldSeasonFeed[[]nflschedule.RawWeek]{5: {FetchedAt: now, Value: []nflschedule.RawWeek{{Week: "5", Matchups: []nflschedule.RawMatchup{
		{Teams: [2]string{"BUF", "NYJ"}, Kickoff: fmt.Sprint(now.Add(-5 * time.Hour).Unix()), Scores: [2]string{"20", "10"}, GameSecondsRemaining: "0"},
	}}}}}
	a.nowHeld.PublishByes(nowapp.Feed[[]nflschedule.RawBye]{FetchedAt: now})
	editResearch(a, func(value *researchHeld) {
		value.feeds = map[string]heldSeasonFeed[[]researchfeeds.Feed]{researchKey("playerScores", "5"): {FetchedAt: now.Add(-3 * time.Hour)}}
	})
	requests, pauses := []string{}, 0
	var err error
	a.core.MFLClient, err = mfl.New("", 1000, mfl.WithTransport(finishedScoresTransport{t, &requests}))
	if err != nil {
		t.Fatal(err)
	}
	a.core.SeasonPause = func(context.Context, time.Duration) error { pauses++; return nil }
	// Live fetch provenance uses wall time, independently of the app's schedule clock.
	started := time.Now().Truncate(time.Second)
	a.refreshFinishedScores(a.core.Ctx)
	finished := time.Now().Truncate(time.Second)
	next := a.now.schedules[5]
	next.FetchedAt = now.Add(time.Hour)
	a.now.schedules[5] = next
	a.core.ClockNow = func() time.Time { return now.Add(time.Hour) }
	a.refreshFinishedScores(a.core.Ctx)
	if strings.Join(requests, ",") != "5,YTD" || pauses != 2 {
		t.Fatal(requests, pauses)
	}
	held := a.snapshotResearch().feeds[researchKey("playerScores", "5")]
	if len(held.Value) != 1 || held.FetchedAt.Before(started) || held.FetchedAt.After(finished) || held.RefreshError != "" {
		t.Fatal(held)
	}
}

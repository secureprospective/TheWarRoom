package serverhttp_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/ingestion/espnnews"
	"github.com/secureprospective/TheWarRoom/internal/leagueservice"
	"github.com/secureprospective/TheWarRoom/internal/serverhttp"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

type espnTripFunc func(*http.Request) (*http.Response, error)

func (f espnTripFunc) Do(req *http.Request) (*http.Response, error) { return f(req) }

func TestHostedESPNNewsCommandsOverWorkflow(t *testing.T) {
	f := newHostedFixture(t, true, true)
	if _, err := f.history.LinkPlayerIDs(t.Context(), "dynastyprocess", []history.PlayerIDLink{
		{IDType: "espn", IDValue: "3139477", PlayerID: "13116"},
		{IDType: "espn", IDValue: "3919512", PlayerID: "14836"},
	}); err != nil {
		t.Fatal(err)
	}
	readFix := func(name string) []byte {
		b, err := os.ReadFile(filepath.Join("..", "ingestion", "espnnews", "testdata", name))
		if err != nil {
			t.Fatal(err)
		}
		return b
	}
	leagueBody := readFix("league_news.json")
	bioBody := readFix("athlete_bio.json")
	pNewsBody := readFix("player_news_fantasy.json")
	transport := espnTripFunc(func(req *http.Request) (*http.Response, error) {
		body := leagueBody
		if strings.Contains(req.URL.Path, "/athletes/") {
			body = bioBody
		} else if strings.Contains(req.URL.Path, "/fantasy/") {
			body = pNewsBody
		}
		return &http.Response{
			StatusCode: http.StatusOK, Body: io.NopCloser(bytes.NewReader(body)),
		}, nil
	})
	ns := leagueservice.NewNewsService(transport, espnnews.DefaultEndpoints(), nil, f.history)
	defer ns.Close()
	if err := ns.RefreshLeagueNow(t.Context()); err != nil {
		t.Fatal(err)
	}
	if _, _, err := ns.RefreshPlayerNow(t.Context(), "3139477"); err != nil {
		t.Fatal(err)
	}
	// Verify workflow validation + execution via direct service wiring on the fixture store.
	service := leagueservice.New(leagueservice.StoredHost{
		Store: f.history, Context: t.Context, Principal: f.principal,
	}).WithNews(ns)
	pr, err := service.TargetPlayerNews(leagueservice.RawPlayerReadScope{
		League: "native.hosted", Player: "13116",
	})
	if err != nil || pr.ESPNID != "3139477" || len(pr.Items) != 2 || pr.Bio.College != "Texas Tech" {
		t.Fatalf("TargetPlayerNews: %+v, %v", pr, err)
	}
	nf, err := service.TargetNewsFeed("native.hosted")
	if err != nil || len(nf.Items) < 3 {
		t.Fatalf("TargetNewsFeed: %+v, %v", nf, err)
	}
	raw := workflowRequest(t, "player.news.read", "read", `{"league":"native.hosted","player":"13116"}`, "")
	if err := raw.Validate(); err != nil {
		t.Fatalf("player.news.read validate: %v", err)
	}
	rawFeed := workflowRequest(t, "news.feed.read", "read", `{"league":"native.hosted"}`, "")
	if err := rawFeed.Validate(); err != nil {
		t.Fatalf("news.feed.read validate: %v", err)
	}
	member := f.login(t, "owner", "private-password")
	body, err := json.Marshal(serverhttp.RawNativeRequest{Protocol: 1, Workflow: &raw})
	if err != nil {
		t.Fatal(err)
	}
	if status, resp := f.send(t, serverhttp.WorkflowPath, string(body), member, ""); status != 200 {
		t.Fatalf("workflow player.news.read status=%d body=%s", status, resp)
	}
}

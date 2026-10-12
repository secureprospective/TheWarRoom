package researchfeeds

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/parquet-go/parquet-go"
	"github.com/secureprospective/TheWarRoom/internal/mfl"
)

func sample(t *testing.T, name string) []byte {
	t.Helper()
	body, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	return body
}
func TestRealMFLResearchSamples(t *testing.T) {
	for _, kind := range []string{"topAdds", "topDrops", "topStarters", "topOwns", "topTrades", "pointsAllowed", "playerScores"} {
		name, week := kind+".json", ""
		if kind == "playerScores" {
			name, week = "playerScores_WYTD.json", "YTD"
		}
		feed, err := Parse(sample(t, name), kind, week)
		if err != nil {
			t.Fatal(kind, err)
		}
		if kind == "pointsAllowed" {
			if len(feed.Allowed) != 320 {
				t.Fatal(len(feed.Allowed))
			}
			continue
		}
		if len(feed.Players) == 0 {
			t.Fatal(kind, "empty")
		}
		if kind == "topAdds" {
			if feed.Players[0].ID.String() != "17059" || feed.Players[0].Value != 60.76 || feed.Players[3].ID.String() != "0521" {
				t.Fatal(feed.Players[:4])
			}
		}
	}
	for _, tc := range []struct {
		window, id string
		points     float64
	}{{"YTD", "16162", 129}, {"04", "13589", 21.80}} {
		week, name := tc.window, "playerScores_WYTD.json"
		if week != "YTD" {
			name = "playerScores_W" + week + ".json"
			week = "4"
		}
		feed, err := Parse(sample(t, name), "playerScores", week)
		if err != nil {
			t.Fatal(err)
		}
		found := false
		for _, p := range feed.Players {
			if p.ID.String() == tc.id {
				found = true
				if p.Value != tc.points {
					t.Fatalf("%s week %s = %v, want %v", tc.id, week, p.Value, tc.points)
				}
			}
		}
		if !found {
			t.Fatal("missing", tc.id)
		}
	}
	newsByID, err := ParseNews(sample(t, "playerProfile.json"), []string{"16162"})
	if err != nil {
		t.Fatal(err)
	}
	news := newsByID["16162"]
	if len(news) != 1 || news[0].Age != "24" || news[0].Published != "6 days" || news[0].Article != "rotowire640841" {
		t.Fatal(news)
	}
}
func TestResearchRejectsMalformedBoundary(t *testing.T) {
	for _, tc := range []struct{ kind, week, body string }{
		{"topAdds", "", `{"error":{"$t":"down"}}`},
		{"topAdds", "", `{}`},
		{"topAdds", "", `{"topAdds":{"player":{"id":"16162","percent":"NaN"}}}`},
		{"topAdds", "", `{"topAdds":{"player":{"id":"16162","percent":"101"}}}`},
		{"playerScores", "YTD", `{"playerScores":{"playerScore":{"id":"16162","score":"1","week":"6"}}}`},
		{"pointsAllowed", "", `{"pointsAllowed":{"team":{"id":"","position":{"name":"QB","points":"1"}}}}`},
	} {
		if _, err := Parse([]byte(tc.body), tc.kind, tc.week); err == nil {
			t.Fatal(tc)
		}
	}
	if _, err := ParseNews(sample(t, "playerProfile.json"), []string{"16161"}); err == nil {
		t.Fatal("wrong profile accepted")
	}
}
func TestRealDepthNewestSnapshotParquetAndCSV(t *testing.T) {
	rows, err := ParseDepthCSV(sample(t, "depth_charts_2026_latest.csv"))
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2333 || rows[0].Team != "ARI" {
		t.Fatal(len(rows), rows[0])
	}
	var buf bytes.Buffer
	older := rows[0]
	older.DT = "2026-10-10T13:56:31Z"
	older.Name = "Old snapshot"
	if err := parquet.Write(&buf, append(rows, older)); err != nil {
		t.Fatal(err)
	}
	got, err := ParseDepth(buf.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(rows, got) {
		t.Fatal("parquet newest snapshot differs from real CSV")
	}
	for _, r := range got {
		if r.Name == "Old snapshot" {
			t.Fatal("old snapshot leaked")
		}
	}
	bad := strings.Replace(string(sample(t, "depth_charts_2026_latest.csv")), "pos_rank", "missing", 1)
	if _, err := ParseDepthCSV([]byte(bad)); err == nil {
		t.Fatal("missing header accepted")
	}
}

type fixtureTransport struct {
	t      *testing.T
	kind   string
	body   []byte
	league bool
}

func (f fixtureTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	if r.URL.Query().Get("TYPE") == "league" {
		return &http.Response{StatusCode: http.StatusOK, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(`{"league":{"baseURL":"https://www99.myfantasyleague.com/2026/home/14432"}}`)), Request: r}, nil
	}
	if r.URL.Query().Get("TYPE") != f.kind || (r.URL.Query().Get("L") != "") != f.league {
		f.t.Fatal(r.URL)
	}
	return &http.Response{StatusCode: http.StatusOK, Header: http.Header{}, Body: io.NopCloser(bytes.NewReader(f.body)), Request: r}, nil
}
func TestFetchResearchUsesFixtureTransport(t *testing.T) {
	for _, kind := range []string{"topAdds", "pointsAllowed", "playerProfile"} {
		c, err := mfl.New("api", 2, mfl.WithTransport(fixtureTransport{t: t, kind: kind, body: sample(t, kind+".json"), league: kind == "pointsAllowed"}))
		if err != nil {
			t.Fatal(err)
		}
		if kind == "playerProfile" {
			if _, err := FetchNews(context.Background(), c, 2026, []string{"16162"}); err != nil {
				t.Fatal(err)
			}
		} else {

			if _, err := Fetch(context.Background(), c, 2026, kind, ""); err != nil {
				t.Fatal(err)
			}
		}
	}
}

func TestUnplayedWeekPlaceholderIsNotZeroProduction(t *testing.T) {
	for _, week := range []string{"5", "6"} {
		feed, err := Parse(sample(t, "playerScores_W0"+week+".json"), "playerScores", week)
		if err != nil || len(feed.Players) != 0 {
			t.Fatal(week, feed, err)
		}
	}
	if _, err := Parse(sample(t, "playerScores_W06.json"), "playerScores", "5"); err == nil {
		t.Fatal("wrong empty week echo accepted")
	}
}

func TestBatchedNewsRealShapesAndLoss(t *testing.T) {
	for _, ids := range [][]string{{"13116", "15761"}, {"13116", "15761", "16162"}} {
		rows, err := ParseNews(sample(t, "playerProfile-batch2.json"), ids)
		if err != nil || len(rows) != 2 {
			t.Fatal(rows, err)
		}
		if news, ok := rows["13116"]; !ok || len(news) != 0 {
			t.Fatal("Mahomes should be answered without news", rows)
		}
		news := rows["15761"]
		if len(news) != 1 || news[0].Headline != "WR Khalil Shakir: Season-best production in Week 4 loss" || news[0].Published != "6 days" {
			t.Fatal(news)
		}
		if _, ok := rows["16162"]; ok {
			t.Fatal("unanswered player guessed")
		}
	}
	for _, body := range []string{
		`{"playerProfiles":{"playerProfile":[{"id":"13116","name":"Mahomes"},{"id":"13116","name":"Mahomes"}]}}`,
		`{"playerProfiles":{"playerProfile":{"id":"13116","name":"","news":{}}}}`,
		`{"error":{"$t":"down"}}`,
	} {
		if _, err := ParseNews([]byte(body), []string{"13116"}); err == nil {
			t.Fatal(body)
		}
	}
	body := []byte(`{"playerProfiles":{"playerProfile":{"id":"13116","name":"Mahomes","news":{"article":[{"id":"a","headline":"first"},{"id":"b","headline":"second"}]}}}}`)
	rows, err := ParseNews(body, []string{"13116"})
	if err != nil || len(rows["13116"]) != 2 {
		t.Fatal(rows, err)
	}
}

type batchNewsTransport struct {
	t *testing.T
}

func (f batchNewsTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	if r.URL.Query().Get("P") != "13116,15761" || r.URL.Query().Get("L") != "" || r.URL.Query().Get("TYPE") != "playerProfile" {
		f.t.Error("batch query or order changed", r.URL)
	}
	return &http.Response{StatusCode: http.StatusOK, Header: http.Header{}, Body: io.NopCloser(bytes.NewReader(sample(f.t, "playerProfile-batch2.json"))), Request: r}, nil
}

func TestFetchNewsBatchAndRequestLimits(t *testing.T) {
	c, err := mfl.New("api", 10000, mfl.WithTransport(batchNewsTransport{t: t}))
	if err != nil {
		t.Fatal(err)
	}
	rows, err := FetchNews(context.Background(), c, 2026, []string{"13116", "15761"})
	if err != nil || len(rows) != 2 {
		t.Fatal(rows, err)
	}
	for _, ids := range [][]string{nil, {"bad"}, {"13116", "13116"}, make([]string, 101)} {
		if _, err := FetchNews(context.Background(), c, 2026, ids); err == nil {
			t.Fatal("invalid request accepted", ids)
		}
	}
}

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/research/researchtest"

	researchapp "github.com/secureprospective/TheWarRoom/internal/app/research"

	"github.com/parquet-go/parquet-go"
	"github.com/secureprospective/TheWarRoom/internal/archive"
	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
	"github.com/secureprospective/TheWarRoom/internal/leagueweek"

	"github.com/secureprospective/TheWarRoom/internal/ingestion/league"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/researchfeeds"
	"github.com/secureprospective/TheWarRoom/internal/mfl"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

func researchSample(t *testing.T, name string) []byte {
	t.Helper()
	return researchtest.Sample(t, name)
}
func realResearch(t *testing.T) researchInput {
	t.Helper()
	d := researchtest.Load(t)
	in := researchInput{week: 5, zone: time.UTC, now: d.At, lookup: d.Lookup, mirror: d.Mirror,
		held:     researchHeld{feeds: map[string]heldSeasonFeed[[]researchfeeds.Feed]{}, news: map[string]heldSeasonFeed[[]researchfeeds.News]{}, links: map[string]string{}},
		injuries: heldSeasonFeed[[]leaguefeed.Injury]{Value: d.Injuries, FetchedAt: d.At}}
	for key, f := range d.Feeds {
		in.held.feeds[key] = heldSeasonFeed[[]researchfeeds.Feed]{Value: []researchfeeds.Feed{f}, FetchedAt: d.At}
	}
	in.held.depth = heldSeasonFeed[[]researchfeeds.RawDepth]{Value: d.Depth, FetchedAt: d.At}
	for id, n := range d.News {
		in.held.news[id] = heldSeasonFeed[[]researchfeeds.News]{Value: n, FetchedAt: d.At}
	}
	return in
}

func TestResearchWireFixtureMatchesGo(t *testing.T) {
	t.Parallel()
	out := buildResearch(realResearch(t), RawResearchScope{Mine: "0001", Search: "Gibbs"})
	for i := range out.Board.Reports {
		r := &out.Board.Reports[i]
		if len(r.Rows) > 3 {
			r.Rows = r.Rows[:3]
		}
	}
	body, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	body = append(body, '\n')
	path := filepath.Join("frontend", "src", "app", "research", "wire.testdata.json")
	if os.Getenv("TWR_WRITE_WIRE") == "1" {
		if err := os.WriteFile(path, body, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(body, want) {
		t.Fatal("Research wire fixture changed; regenerate from Go")
	}
}

func TestResearchHeldIPCAndArticleHandOff(t *testing.T) {
	t.Parallel()
	in := realResearch(t)
	a := targetTestApp(t)
	if err := a.core.Rulebook.Initialize(a.core.Ctx, lineupRulesSource{cfg: league.RawConfig{Source: "research fixture", Franchises: []league.Franchise{{ID: "0001", Name: "My franchise"}, {ID: "0002", Name: "Other franchise"}}}}); err != nil {
		t.Fatal(err)
	}
	a.research = in.held
	a.core.Lookup = in.lookup
	a.core.HasLookup = true
	if _, err := a.core.League.Replace(a.core.Ctx, in.mirror); err != nil {
		t.Fatal(err)
	}
	out, err := a.TargetResearch(RawResearchScope{Mine: "0001", Search: "Gibbs"})
	if err != nil || len(out.Players) != 1 || out.Players[0].FranchiseID != "0001" {
		t.Fatal(out.Players, err)
	}
	if _, err := a.TargetResearch(RawResearchScope{Mine: "9999"}); err == nil {
		t.Fatal("unknown franchise accepted")
	}
	p, err := a.TargetResearchPlayer("16162")
	if err != nil || p.Name != "Gibbs, Jahmyr" {
		t.Fatal(p, err)
	}
	if _, err := a.TargetResearchPlayer("bad"); err == nil {
		t.Fatal("bad player accepted")
	}
	opened := ""
	a.core.OpenURL = func(_ context.Context, target string) { opened = target }
	if err := a.TargetOpenResearchArticle("16162", "forged"); err == nil {
		t.Fatal("forged article accepted")
	}
	// Host is held by the fixture client, never discovered by a hand-off.
	client, err := mfl.New("www99", 100, mfl.WithTransport(&directoryDown{}))
	if err != nil {
		t.Fatal(err)
	}
	a.core.MFLClient = client
	if err := a.TargetOpenResearchArticle("16162", "rotowire640841"); err != nil {
		t.Fatal(err)
	}
	if opened != "https://www99.myfantasyleague.com/2026/view_news_article?ID=rotowire640841" {
		t.Fatal(opened)
	}
}

type researchFixtureTransport struct {
	t      *testing.T
	events *[]string
	depth  []byte
}

func (f researchFixtureTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	query := request.URL.Query()
	kind := query.Get("TYPE")
	*f.events = append(*f.events, kind+":"+query.Get("W")+":"+query.Get("P"))
	var body []byte
	switch kind {
	case "league":
		body = []byte(`{"league":{"baseURL":"https://www99.myfantasyleague.com/2026/home/14432"}}`)
	case "playerScores":
		window := query.Get("W")
		name := "playerScores_WYTD.json"
		if window != "YTD" {
			name = "playerScores_W0" + window + ".json"
		}
		body = researchSample(f.t, name)
	case "nflSchedule":
		body = researchSample(f.t, "nflSchedule_W06.json")
	case "playerProfile":
		switch query.Get("P") {
		case "16162":
			body = researchSample(f.t, "playerProfile.json")
		case "13589,16162":
			body = []byte(`{"playerProfiles":{"playerProfile":[{"id":"13589","name":"Allen, Josh","news":{}},` + strings.TrimSuffix(strings.TrimPrefix(strings.TrimSpace(string(researchSample(f.t, "playerProfile.json"))), `{"version":"1.0","encoding":"utf-8","playerProfile":`), "}") + `]}}`)
		case "13589":
			body = []byte(`{"playerProfile":{"id":"13589","name":"Allen, Josh","player":{"age":"30"},"news":{}}}`)
		default:
			f.t.Fatal("unrostered profile requested", request.URL)
		}
	case "":
		if request.URL.String() != researchfeeds.DepthURL(2026) {
			f.t.Fatal("unexpected fixture URL", request.URL)
		}
		body = f.depth
	default:
		body = researchSample(f.t, kind+".json")
	}
	return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(bytes.NewReader(body)), Request: request}, nil
}
func TestResearchRefreshArchiveAndRosteredNewsEndToEnd(t *testing.T) {
	t.Parallel()
	in := realResearch(t)
	a := targetTestApp(t)
	if err := a.core.Rulebook.Initialize(a.core.Ctx, lineupRulesSource{cfg: league.RawConfig{Source: "research fixture", Franchises: []league.Franchise{{ID: "0001", Name: "Mine"}, {ID: "0002", Name: "Other"}}}}); err != nil {
		t.Fatal(err)
	}
	if _, err := a.core.League.Replace(a.core.Ctx, in.mirror); err != nil {
		t.Fatal(err)
	}
	a.core.Lookup = in.lookup
	a.core.HasLookup = true
	a.core.Week = leagueweek.Week{Number: 5}
	var parquetBody bytes.Buffer
	if err := parquet.Write(&parquetBody, in.held.depth.Value); err != nil {
		t.Fatal(err)
	}
	events := []string{}
	stages := []string{}
	a.core.Fetches = &archive.Transport{Sink: a.core.History, Base: researchFixtureTransport{t: t, events: &events, depth: parquetBody.Bytes()}}
	client, err := mfl.New("api", 10000, mfl.WithTransport(a.core.Fetches))
	if err != nil {
		t.Fatal(err)
	}
	a.core.MFLClient = client
	a.core.FallbackLog = func(string) {}
	a.core.SeasonPause = func(_ context.Context, d time.Duration) error {
		if d != time.Second {
			t.Fatal(d)
		}
		events = append(events, "pause")
		if status := a.TargetResearchStatus(); status != "" {
			stages = append(stages, status)
		}
		return nil
	}
	if err := a.TargetResearchRefresh(false); err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(stages, "Reading week 3 of 5") || !slices.Contains(stages, "Reading fantasy points") {
		t.Fatal(stages)
	}
	if a.TargetResearchStatus() != "" {
		t.Fatal("refresh status not cleared")
	}
	if len(a.research.depth.Value) != 2333 || len(researchRows(a.research.feeds[researchKey("playerScores", "YTD")])) == 0 {
		t.Fatal("refresh lost real fixtures")
	}
	out, err := a.TargetResearch(RawResearchScope{Mine: "0001"})
	if err != nil {
		t.Fatal(err)
	}
	if len(reportByID(t, out.Board, "M-038").Rows) != 1 {
		t.Fatal("mine report not wired")
	}
	events = nil
	if err := a.TargetResearchNewsRefresh(""); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(events, []string{"pause", "playerProfile::13589,16162"}) {
		t.Fatal("profile scope or spacing", events)
	}
	events = nil
	if err := a.TargetResearchNewsRefresh(""); err != nil {
		t.Fatal(err)
	}
	if len(events) != 0 {
		t.Fatal("same-day news fetched again", events)
	}
	// A cleared in-memory cache reloads today's archived profiles without another request.
	a.research.news = nil
	if err := a.TargetResearchNewsRefresh("0001"); err != nil {
		t.Fatal(err)
	}
	if len(events) != 0 || len(a.research.news["16162"].Value) != 1 {
		t.Fatal("persisted daily cache failed", events)
	}
	a.research = researchHeld{}
	a.core.SeasonInjuries = heldSeasonFeed[[]leaguefeed.Injury]{}
	a.core.RefreshMu.Lock()
	cold, cerr := a.TargetResearch(RawResearchScope{Mine: "0001"})
	a.core.RefreshMu.Unlock()
	if cerr != nil || len(a.research.depth.Value) != 2333 || len(cold.Articles) != 1 || len(events) != 0 {
		t.Fatal("cold read did not serve archived depth/news without fetching", cerr, len(events))
	}
	// The depth fallback reads the exact season URL and preserves stale provenance.
	rows, fresh, err := readResearchDepthArchive(a.core.Ctx, 2026, a.core.History.ArchivedBodies, errors.New("fixture offline"))
	if err != nil || len(rows) != 2333 || fresh.State != FreshStale {
		t.Fatal(len(rows), fresh, err)
	}
}

func TestPlayerFormAndCutUseHeldWeeksAndSharedRule(t *testing.T) {
	t.Parallel()
	in := realResearch(t)
	in.mirror.Season = 2026
	in.mirror.Players = []state.MirrorPlayer{{MFLID: "16162", FranchiseID: "0002", Salary: 100_000_000, ContractYear: 2028}}
	v := researchView{researchInput: in}
	r := v.playerReading("16162", map[string]string{"0002": "Other"}, "Counting")
	text := ""
	for _, row := range r.Report.Rows {
		text += strings.Join(row.Cells, " ") + "\n"
	}
	if !strings.Contains(text, "W2:") || !strings.Contains(text, "W3:") || !strings.Contains(text, "W4:") || !strings.Contains(text, "season ") || !strings.Contains(text, "$700,000") {
		t.Fatal(text)
	}
	a := whatIfTestApp(t)
	cut := whatIfCall(t, a, "cut", "0042", "")
	mirror, _, _ := a.core.League.ContractsView()
	v.mirror = mirror
	probe := v.playerReading("0042", nil, "Counting").Report
	if !strings.Contains(researchRow(probe, "Dead cap if cut today").Cells[1], cut.Value.Moves[0].Charge.Text) {
		t.Fatal(probe, cut)
	}
	delete(v.held.feeds, researchKey("playerScores", "3"))
	delete(v.held.feeds, researchKey("playerScores", "4"))
	probe = v.playerReading("16162", nil, "Counting").Report
	if !strings.Contains(researchRow(probe, "Form").Cells[1], "1 of the last 3") {
		t.Fatal(probe)
	}
}

func TestResearchWireGoldens(t *testing.T) {
	t.Parallel()
	a := targetTestApp(t)
	board, err := a.TargetResearch(RawResearchScope{})
	if err != nil {
		t.Fatal(err)
	}
	checkGolden(t, "research", "unheld", board)
	in := realResearch(t)
	held := buildResearch(in, RawResearchScope{Mine: "0001", Search: "Gibbs"})
	for i := range held.Board.Reports {
		if len(held.Board.Reports[i].Rows) > 3 {
			held.Board.Reports[i].Rows = held.Board.Reports[i].Rows[:3]
		}
	}
	checkGolden(t, "research", "held", held)
	checkGolden(t, "research", "player-held", (researchView{researchInput: in}).playerReading("16162", map[string]string{"0001": "Mine"}, "How this is counted"))
	a.core.Lookup, a.core.HasLookup = in.lookup, true
	player, err := a.TargetResearchPlayer("16162")
	if err != nil {
		t.Fatal(err)
	}
	checkGolden(t, "research", "player-unheld", player)
	a.researchStatus("Reading fantasy points")
	checkGolden(t, "research", "status", a.TargetResearchStatus())
	a.core.SeasonPause = func(context.Context, time.Duration) error { return context.Canceled }
	errors := map[string]string{}
	if err := a.TargetResearchRefresh(false); err != nil {
		errors["refresh"] = err.Error()
	}
	if err := a.TargetResearchNewsRefresh("9999"); err != nil {
		errors["newsRefresh"] = err.Error()
	}
	if err := a.TargetOpenResearchArticle("16162", "forged"); err != nil {
		errors["openArticle"] = err.Error()
	}
	checkGolden(t, "research", "errors", errors)
}

func (v researchView) playerReading(id string, names map[string]string, how string) ResearchPlayerReading {
	return wireResearchPlayer(researchapp.ReadPlayer(v.snapshot(), id, names, how))
}

func researchHeldTime(at time.Time, loss string, zone *time.Location) string {
	return researchapp.HeldTime(at, loss, zone)
}

func researchRow(r Report, label string) ReportRow {
	for _, row := range r.Rows {
		if row.Cells[0] == label {
			return row
		}
	}
	return ReportRow{}
}

func TestResearchWireTwinsMatch(t *testing.T) {
	t.Parallel()
	var board researchapp.Board
	var player researchapp.PlayerReading
	n := 0
	fillWire(reflect.ValueOf(&board).Elem(), &n)
	fillWire(reflect.ValueOf(&player).Elem(), &n)
	for _, tc := range []struct {
		name        string
		built, wire any
	}{
		{"board", board, wireResearchBoard(board)},
		{"player", player, wireResearchPlayer(player)},
		{"nil board", researchapp.Board{}, wireResearchBoard(researchapp.Board{})},
		{"nil player", researchapp.PlayerReading{}, wireResearchPlayer(researchapp.PlayerReading{})},
		{"empty articles", []researchapp.Article{}, wireResearchArticles([]researchapp.Article{})},
	} {
		t.Run(tc.name, func(t *testing.T) {
			want, err := json.Marshal(tc.built)
			if err != nil {
				t.Fatal(err)
			}
			got, err := json.Marshal(tc.wire)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(want, got) {
				t.Fatalf("wire twin differs:\n%s\n%s", want, got)
			}
		})
	}
}

package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/parquet-go/parquet-go"
	"github.com/secureprospective/TheWarRoom/internal/archive"
	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
	"github.com/secureprospective/TheWarRoom/internal/leagueweek"

	"github.com/secureprospective/TheWarRoom/internal/ingestion/injuries"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/league"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/nflschedule"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/players"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/researchfeeds"
	"github.com/secureprospective/TheWarRoom/internal/mfl"
	"github.com/secureprospective/TheWarRoom/internal/normalize"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

func researchSample(t *testing.T, name string) []byte {
	t.Helper()
	root, err := os.OpenRoot(filepath.Join("internal", "ingestion", "researchfeeds", "testdata"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := root.Close(); err != nil {
			t.Error(err)
		}
	}()
	body, err := root.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	return body
}
func realResearch(t *testing.T) researchInput {
	t.Helper()
	at := time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC)
	in := researchInput{week: 5, zone: time.UTC, now: at, held: researchHeld{feeds: map[string]heldSeasonFeed[[]researchfeeds.Feed]{}, news: map[string]heldSeasonFeed[[]researchfeeds.News]{}, links: map[string]string{}}}
	reader, err := gzip.NewReader(bytes.NewReader(researchSample(t, "players.json.gz")))
	if err != nil {
		t.Fatal(err)
	}
	body, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	if err := reader.Close(); err != nil {
		t.Fatal(err)
	}
	raw, err := players.Parse(context.Background(), body)
	if err != nil {
		t.Fatal(err)
	}
	in.lookup, err = normalize.NewLookup(raw)
	if err != nil {
		t.Fatal(err)
	}
	for _, kind := range []string{"playerScores", "topAdds", "topDrops", "topStarters", "topOwns", "topTrades", "pointsAllowed"} {
		name, week := kind+".json", ""
		if kind == "playerScores" {
			name, week = "playerScores_WYTD.json", "YTD"
		}
		feed, err := researchfeeds.Parse(researchSample(t, name), kind, week)
		if err != nil {
			t.Fatal(err)
		}
		in.held.feeds[researchKey(kind, week)] = heldSeasonFeed[[]researchfeeds.Feed]{value: []researchfeeds.Feed{feed}, fetchedAt: at}
	}
	for w := 1; w <= 5; w++ {
		feed, err := researchfeeds.Parse(researchSample(t, "playerScores_W0"+strconv.Itoa(w)+".json"), "playerScores", strconv.Itoa(w))
		if err != nil {
			t.Fatal(err)
		}
		in.held.feeds[researchKey("playerScores", strconv.Itoa(w))] = heldSeasonFeed[[]researchfeeds.Feed]{value: []researchfeeds.Feed{feed}, fetchedAt: at}
	}
	in.held.depth.value, err = researchfeeds.ParseDepthCSV(researchSample(t, "depth_charts_2026_latest.csv"))
	if err != nil {
		t.Fatal(err)
	}
	in.held.depth.fetchedAt = at
	in.injuries.value, err = injuries.ParseToInjuries(context.Background(), researchSample(t, "injuries.json"))
	if err != nil {
		t.Fatal(err)
	}
	in.injuries.fetchedAt = at
	newsByID, err := researchfeeds.ParseNews(researchSample(t, "playerProfile.json"), []string{"16162"})
	if err != nil {
		t.Fatal(err)
	}
	in.held.news["16162"] = heldSeasonFeed[[]researchfeeds.News]{value: newsByID["16162"], fetchedAt: at}
	in.mirror.Season = 2026
	in.mirror.Players = []state.MirrorPlayer{{MFLID: "16162", FranchiseID: "0001"}, {MFLID: "13589", FranchiseID: "0002"}}
	return in
}
func TestResearchRealPointsActivityAndScales(t *testing.T) {
	in := realResearch(t)
	board := buildResearch(in, RawResearchScope{Mine: "0001"})
	stats := reportByID(t, board.Board, "M-037")
	allen := slices.IndexFunc(stats.Rows, func(r ReportRow) bool { return r.Cells[0] == "Allen, Josh BUF QB" || r.Cells[0] == "Allen, Josh" })
	if allen < 0 {
		t.Fatal("Josh Allen missing")
	}
	if row := stats.Rows[allen]; row.Cells[3] != "134.80" || row.Cells[4] != "44.90" || row.Cells[7] != "21.80" || row.Cells[8] != "—" || len(row.Cells) != 9 {
		t.Fatal(row)
	}
	mine := reportByID(t, board.Board, "M-038")
	if len(mine.Rows) != 1 || mine.Rows[0].Cells[3] != "129.00" || !mine.Rows[0].Mine {
		t.Fatal(mine.Rows)
	}
	for _, kind := range []string{"topAdds", "topDrops", "topStarters", "topOwns", "topTrades"} {
		out := buildResearch(in, RawResearchScope{Trend: kind})
		report := reportByID(t, out.Board, "M-052")
		want := researchRows(in.held.feeds[researchKey(kind, "")])
		for i, row := range report.Rows {
			if row.Cells[1] != (researchView{researchInput: in}).player(want[i].ID.String()).Name || row.Cells[4] != strconv.FormatFloat(want[i].Value, 'f', 2, 64)+"%" {
				t.Fatal(kind, i, row)
			}
		}
		if !strings.Contains(report.Source, "MFL-wide, not this league") {
			t.Fatal(report.Source)
		}
	}
	for _, id := range []string{"M-053", "M-054"} {
		r := reportByID(t, board.Board, id)
		if len(r.Rows) == 0 {
			t.Fatal(id)
		}
		axis := r.Rows[0].Range.Max
		for _, row := range r.Rows {
			if row.Range.Max != axis || row.Range.Hi > axis {
				t.Fatal(id, row)
			}
		}
	}
	kick := reportByID(t, buildResearch(in, RawResearchScope{Group: "K"}).Board, "M-040")
	if !strings.Contains(kick.Note, "no K starter slots") || len(kick.Rows) == 0 {
		t.Fatal(kick.Note, len(kick.Rows))
	}
	defense := reportByID(t, buildResearch(in, RawResearchScope{Group: "DST"}).Board, "M-040")
	if !strings.Contains(defense.Note, "no DST starter slots") {
		t.Fatal(defense.Note)
	}
}
func TestResearchDepthInjuriesAndScopedNews(t *testing.T) {
	in := realResearch(t)
	d := in.held.depth.value[0]
	in.held.links["gsis:"+*d.GSIS] = "16162"
	out := buildResearch(in, RawResearchScope{Mine: "0001", Team: "ARI"})
	depth := reportByID(t, out.Board, "M-048")
	if !depth.Rows[0].Mine || !strings.Contains(depth.Note, "no held MFL identity") {
		t.Fatal(depth.Rows[0], depth.Note)
	}
	if !strings.HasPrefix(depth.Rows[0].Group, "ARI · ") {
		t.Fatal(depth.Rows[0].Group)
	}
	grid := reportByID(t, buildResearch(in, RawResearchScope{Mine: "0001", Depth: "grid"}).Board, "M-048")
	if len(grid.Rows) != 32 || len(grid.Columns) < 10 {
		t.Fatal(len(grid.Rows), len(grid.Columns))
	}
	for _, r := range grid.Rows {
		if len(r.Cells) != len(grid.Columns) {
			t.Fatal(r)
		}
	}
	// A missing games feed never reads as nobody injured.
	games := reportByID(t, buildResearch(in, RawResearchScope{Injuries: "games"}).Board, "M-057")
	if games.Status != "failed" || !strings.Contains(games.Note, "without this week’s schedule") {
		t.Fatal(games)
	}
	schedule, err := nflschedule.Parse(researchSample(t, "nflSchedule_W06.json"))
	if err != nil {
		t.Fatal(err)
	}
	in.week = 6
	in.schedules = map[int]heldSeasonFeed[[]nflschedule.RawWeek]{6: {value: []nflschedule.RawWeek{schedule}, fetchedAt: in.injuries.fetchedAt}}
	games = reportByID(t, buildResearch(in, RawResearchScope{Injuries: "games"}).Board, "M-057")
	playing := map[string]bool{}
	for _, m := range schedule.Matchups {
		for _, team := range m.Teams {
			playing[researchTeam(team)] = true
		}
	}
	for _, r := range games.Rows {
		if !playing[r.Cells[2]] {
			t.Fatal("bye in game injury report", r)
		}
	}
	news := reportByID(t, out.Board, "M-060")
	if len(news.Rows) != 1 || news.Rows[0].Cells[1] != "24" || news.Rows[0].Cells[3] != "6 days" {
		t.Fatal(news)
	}
	if out.News.Status != "stale" || len(out.News.Summary) != 1 || !strings.Contains(out.News.Summary[0].Cells[0], "1 profiles need refreshing") {
		t.Fatal(out.News)
	}
	if len(out.Articles) != 1 || out.Articles[0].Article != "rotowire640841" {
		t.Fatal(out.Articles)
	}
}
func TestResearchScopeAndFeedLoss(t *testing.T) {
	for _, raw := range []RawResearchScope{{Mine: "1"}, {Group: "grades"}, {Team: "../../"}, {Depth: "old"}, {Trend: "unknown"}, {Search: strings.Repeat("x", 101)}} {
		if raw.Validate() == nil {
			t.Fatal(raw)
		}
	}
	in := realResearch(t)
	h := in.held.feeds[researchKey("topAdds", "")]
	h.refreshError = "fixture offline"
	in.held.feeds[researchKey("topAdds", "")] = h
	r := reportByID(t, buildResearch(in, RawResearchScope{}).Board, "M-052")
	if r.Status != "stale" || len(r.Rows) == 0 || !strings.Contains(r.Note, "last known") {
		t.Fatal(r)
	}
	out := buildResearch(researchInput{zone: time.UTC}, RawResearchScope{})
	for _, r := range out.Board.Reports {
		if r.Status != "failed" && r.Status != "empty" {
			t.Fatal(r.ID, r.Status)
		}
	}
}
func TestResearchWireFixtureMatchesGo(t *testing.T) {
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
	in := realResearch(t)
	a := targetTestApp(t)
	if err := a.rulebook.Initialize(a.ctx, lineupRulesSource{cfg: league.RawConfig{Source: "research fixture", Franchises: []league.Franchise{{ID: "0001", Name: "My franchise"}, {ID: "0002", Name: "Other franchise"}}}}); err != nil {
		t.Fatal(err)
	}
	a.research = in.held
	a.lookup = in.lookup
	a.hasLookup = true
	if _, err := a.league.Replace(a.ctx, in.mirror); err != nil {
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
	a.openURL = func(_ context.Context, target string) { opened = target }
	if err := a.TargetOpenResearchArticle("16162", "forged"); err == nil {
		t.Fatal("forged article accepted")
	}
	// Host is held by the fixture client, never discovered by a hand-off.
	client, err := mfl.New("www99", 100, mfl.WithTransport(&directoryDown{}))
	if err != nil {
		t.Fatal(err)
	}
	a.mflClient = client
	if err := a.TargetOpenResearchArticle("16162", "rotowire640841"); err != nil {
		t.Fatal(err)
	}
	if opened != "https://www99.myfantasyleague.com/2026/news_articles?P=16162&ARTICLE_ID=rotowire640841" {
		t.Fatal(opened)
	}
}

func TestResearchAliasesAndHiddenFilters(t *testing.T) {
	in := realResearch(t)
	leaders := reportByID(t, buildResearch(in, RawResearchScope{Position: "QB", Group: "IDP", Team: "BUF"}).Board, "M-040")
	if len(leaders.Rows) == 0 {
		t.Fatal("a hidden stats filter emptied the position-group report")
	}
	for _, row := range leaders.Rows {
		if !slices.Contains([]string{"DT", "DE", "LB", "CB", "S"}, row.Cells[1]) {
			t.Fatal(row)
		}
	}
	in.held.depth.value = in.held.depth.value[:1]
	in.held.depth.value[0].Team = "LA"
	for _, mode := range []string{"team", "grid"} {
		report := reportByID(t, buildResearch(in, RawResearchScope{Team: "LAR", Depth: mode}).Board, "M-048")
		if len(report.Rows) != 1 {
			t.Fatal("NFL alias lost depth rows", mode, report)
		}
	}
	if shortDepthNames([]string{"▸ Josh Sweat · MFL identity not linked"}) != "▸ J. Sweat*" {
		t.Fatal("grid name is not short and labelled")
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
	in := realResearch(t)
	a := targetTestApp(t)
	if err := a.rulebook.Initialize(a.ctx, lineupRulesSource{cfg: league.RawConfig{Source: "research fixture", Franchises: []league.Franchise{{ID: "0001", Name: "Mine"}, {ID: "0002", Name: "Other"}}}}); err != nil {
		t.Fatal(err)
	}
	if _, err := a.league.Replace(a.ctx, in.mirror); err != nil {
		t.Fatal(err)
	}
	a.lookup = in.lookup
	a.hasLookup = true
	a.week = leagueweek.Week{Number: 5}
	var parquetBody bytes.Buffer
	if err := parquet.Write(&parquetBody, in.held.depth.value); err != nil {
		t.Fatal(err)
	}
	events := []string{}
	stages := []string{}
	a.fetches = &archive.Transport{Sink: a.history, Base: researchFixtureTransport{t: t, events: &events, depth: parquetBody.Bytes()}}
	client, err := mfl.New("api", 10000, mfl.WithTransport(a.fetches))
	if err != nil {
		t.Fatal(err)
	}
	a.mflClient = client
	a.fallbackLog = func(string) {}
	a.seasonPause = func(_ context.Context, d time.Duration) error {
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
	if len(a.research.depth.value) != 2333 || len(researchRows(a.research.feeds[researchKey("playerScores", "YTD")])) == 0 {
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
	if len(events) != 0 || len(a.research.news["16162"].value) != 1 {
		t.Fatal("persisted daily cache failed", events)
	}
	a.research = researchHeld{}
	a.seasonInjuries = heldSeasonFeed[[]leaguefeed.Injury]{}
	a.refreshMu.Lock()
	cold, cerr := a.TargetResearch(RawResearchScope{Mine: "0001"})
	a.refreshMu.Unlock()
	if cerr != nil || len(a.research.depth.value) != 2333 || len(cold.Articles) != 1 || len(events) != 0 {
		t.Fatal("cold read did not serve archived depth/news without fetching", cerr, len(events))
	}
	// The depth fallback reads the exact season URL and preserves stale provenance.
	rows, fresh, err := readResearchDepthArchive(a.ctx, 2026, a.history.ArchivedBodies, errors.New("fixture offline"))
	if err != nil || len(rows) != 2333 || fresh.State != FreshStale {
		t.Fatal(len(rows), fresh, err)
	}
}

package research

import (
	"bytes"
	"compress/gzip"
	"context"
	"io"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/report"
	"github.com/secureprospective/TheWarRoom/internal/app/research/researchtest"

	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"

	"github.com/secureprospective/TheWarRoom/internal/ingestion/crosswalk"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/nflschedule"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/researchfeeds"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

func TestResearchRealPointsActivityAndScales(t *testing.T) {
	t.Parallel()
	in := realResearch(t)
	board := Build(in, Scope{Mine: "0001"})
	stats := reportByID(t, board.Board, "M-037")
	allen := slices.IndexFunc(stats.Rows, func(r report.Row) bool { return r.Cells[0] == "Allen, Josh BUF QB" || r.Cells[0] == "Allen, Josh" })
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
		out := Build(in, Scope{Trend: kind})
		report := reportByID(t, out.Board, "M-052")
		want := researchRows(in.Held.Feeds[Key(kind, "")])
		for i, row := range report.Rows {
			if row.Cells[1] != (researchView{Snapshot: in}).player(want[i].ID.String()).Name || row.Cells[4] != strconv.FormatFloat(want[i].Value, 'f', 2, 64)+"%" {
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
	kick := reportByID(t, Build(in, Scope{Group: "K"}).Board, "M-040")
	if !strings.Contains(kick.Note, "no K starter slots") || len(kick.Rows) == 0 {
		t.Fatal(kick.Note, len(kick.Rows))
	}
	defense := reportByID(t, Build(in, Scope{Group: "DST"}).Board, "M-040")
	if !strings.Contains(defense.Note, "no DST starter slots") {
		t.Fatal(defense.Note)
	}
}
func TestResearchDepthInjuriesAndScopedNews(t *testing.T) {
	t.Parallel()
	in := realResearch(t)
	d := in.Held.Depth.Value[0]
	in.Held.Links["gsis:"+*d.GSIS] = "16162"
	out := Build(in, Scope{Mine: "0001", Team: "ARI"})
	depth := reportByID(t, out.Board, "M-048")
	if !depth.Rows[0].Mine || !strings.Contains(depth.Note, "no held MFL identity") {
		t.Fatal(depth.Rows[0], depth.Note)
	}
	if !strings.HasPrefix(depth.Rows[0].Group, "ARI · ") {
		t.Fatal(depth.Rows[0].Group)
	}
	grid := reportByID(t, Build(in, Scope{Mine: "0001", Depth: "grid"}).Board, "M-048")
	if len(grid.Rows) != 32 || len(grid.Columns) < 10 {
		t.Fatal(len(grid.Rows), len(grid.Columns))
	}
	for _, r := range grid.Rows {
		if len(r.Cells) != len(grid.Columns) {
			t.Fatal(r)
		}
	}
	// A missing games feed never reads as nobody injured.
	games := reportByID(t, Build(in, Scope{Injuries: "games"}).Board, "M-057")
	if games.Status != "failed" || !strings.Contains(games.Note, "without this week’s schedule") {
		t.Fatal(games)
	}
	schedule, err := nflschedule.Parse(researchSample(t, "nflSchedule_W06.json"))
	if err != nil {
		t.Fatal(err)
	}
	in.Week = 6
	in.Schedules = map[int]Feed[[]nflschedule.RawWeek]{6: {Value: []nflschedule.RawWeek{schedule}, FetchedAt: in.Injuries.FetchedAt}}
	games = reportByID(t, Build(in, Scope{Injuries: "games"}).Board, "M-057")
	playing := map[string]bool{}
	for _, m := range schedule.Matchups {
		for _, team := range m.Teams {
			playing[Team(team)] = true
		}
	}
	for _, r := range games.Rows {
		if !playing[r.Cells[2]] {
			t.Fatal("bye in game injury report", r)
		}
	}
	news := reportByID(t, out.Board, "M-060")
	if len(news.Rows) != 1 || news.Rows[0].Cells[1] != "24" || news.Rows[0].Cells[3] != "Oct 5, 2026" {
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
	t.Parallel()
	for _, raw := range []Scope{{Mine: "1"}, {Group: "grades"}, {Team: "../../"}, {Depth: "old"}, {Trend: "unknown"}, {Search: strings.Repeat("x", 101)}} {
		if raw.Validate() == nil {
			t.Fatal(raw)
		}
	}
	in := realResearch(t)
	h := in.Held.Feeds[Key("topAdds", "")]
	h.RefreshError = "fixture offline"
	in.Held.Feeds[Key("topAdds", "")] = h
	r := reportByID(t, Build(in, Scope{}).Board, "M-052")
	if r.Status != "stale" || len(r.Rows) == 0 || !strings.Contains(r.Note, "last known") {
		t.Fatal(r)
	}
	out := Build(Snapshot{Zone: time.UTC}, Scope{})
	for _, r := range out.Board.Reports {
		if r.Status != "failed" && r.Status != "empty" {
			t.Fatal(r.ID, r.Status)
		}
	}
}
func TestResearchAliasesAndHiddenFilters(t *testing.T) {
	t.Parallel()
	in := realResearch(t)
	leaders := reportByID(t, Build(in, Scope{Position: "QB", Group: "IDP", Team: "BUF"}).Board, "M-040")
	if len(leaders.Rows) == 0 {
		t.Fatal("a hidden stats filter emptied the position-group report")
	}
	for _, row := range leaders.Rows {
		if !slices.Contains([]string{"DT", "DE", "LB", "CB", "S"}, row.Cells[1]) {
			t.Fatal(row)
		}
	}
	in.Held.Depth.Value = in.Held.Depth.Value[:1]
	in.Held.Depth.Value[0].Team = "LA"
	for _, mode := range []string{"team", "grid"} {
		report := reportByID(t, Build(in, Scope{Team: "LAR", Depth: mode}).Board, "M-048")
		if len(report.Rows) != 1 {
			t.Fatal("NFL alias lost depth rows", mode, report)
		}
	}
}

func TestResearchNewsNewestFirstAndUnknownLastStable(t *testing.T) {
	t.Parallel()
	in := realResearch(t)
	rows := []researchfeeds.News{
		{Article: "unknown1", Headline: "first unknown", Published: "recently"},
		{Article: "old", Headline: "six days", Published: "6 days"},
		{Article: "new", Headline: "fifteen hours", Published: "15 hours"},
		{Article: "middle", Headline: "one day", Published: "1 day"},
		{Article: "unknown2", Headline: "second unknown", Published: "yesterday"},
	}
	in.Held.News["16162"] = Feed[[]researchfeeds.News]{Value: rows, FetchedAt: time.Now()}
	out := Build(in, Scope{Mine: "0001"})
	got := []string{}
	for _, r := range reportByID(t, out.Board, "M-060").Rows {
		got = append(got, r.Cells[2])
	}
	if !slices.Equal(got, []string{"fifteen hours", "one day", "six days", "first unknown", "second unknown"}) {
		t.Fatal(got)
	}
	if out.Articles[0].Article != "new" {
		t.Fatal("article hand-offs lost report order", out.Articles)
	}
}

func TestResearchPlayerHeldFactsAndMissingPieces(t *testing.T) {
	t.Parallel()
	in := realResearch(t)
	in.Injuries.Value = []leaguefeed.Injury{{ID: in.Held.News["16162"].Value[0].ID, Status: "Questionable", Details: "Knee", ExpReturn: "Week 6"}}
	v := researchView{Snapshot: in}
	p := v.playerReading("16162", map[string]string{"0001": "My franchise"}, "How this is counted")
	if p.Owner != "My franchise" || p.FranchiseID != "0001" || len(p.Articles) != 1 {
		t.Fatal(p)
	}
	text := ""
	for _, r := range p.Report.Rows {
		text += strings.Join(r.Cells, " ") + "\n"
	}
	for _, want := range []string{"Questionable · Knee · Week 6", "Season fantasy points 129.00", "position rank", "Last finished week · Week 4", "Points per game", "not held", "as of Oct 11"} {
		if !strings.Contains(text, want) {
			t.Fatal(want, text)
		}
	}
	in.Mirror.Players = slices.DeleteFunc(slices.Clone(in.Mirror.Players), func(p state.MirrorPlayer) bool { return p.MFLID == "16162" })
	free := (researchView{Snapshot: in}).playerReading("16162", nil, "How this is counted")
	if free.Owner != "Free agent" || free.FranchiseID != "" {
		t.Fatal(free)
	}
	empty := (researchView{Snapshot: Snapshot{Lookup: in.Lookup, Zone: time.UTC}}).playerReading("16162", nil, "How this is counted")
	for _, r := range empty.Report.Rows[1:] {
		if !strings.HasPrefix(r.Cells[1], "not held") || r.Cells[2] != "No held copy available" {
			t.Fatal(r)
		}
	}
	read := researchView{Snapshot: in}
	read.Injuries.FetchedAt = time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC)
	read.Injuries.Value = nil
	quiet := read.Held.News["16162"]
	quiet.Value = nil
	read.Held.News = map[string]Feed[[]researchfeeds.News]{"16162": quiet}
	healthy := read.playerReading("16162", nil, "How this is counted").Report.Rows
	if healthy[1].Cells[1] != "Not on MFL’s injury report" || healthy[len(healthy)-1].Cells[1] != "No MFL news for this player" {
		t.Fatal(healthy)
	}
	d := in.Held.Depth.Value[0]
	in.Held.Links["gsis:"+*d.GSIS] = "16162"
	linked := (researchView{Snapshot: in}).playerReading("16162", nil, "How this is counted")
	if !strings.Contains(linked.Report.Rows[2].Cells[1], d.Group+" · "+d.Position+" · depth") {
		t.Fatal(linked.Report.Rows)
	}
}

func TestResearchSourcesFoldWeeksAndUseWords(t *testing.T) {
	t.Parallel()
	in := realResearch(t)
	out := Build(in, Scope{})
	r := reportByID(t, out.Board, "M-038")
	if len(r.Summary) != 0 || !strings.Contains(r.Source, "weeks 1–5 as of") {
		t.Fatal(r)
	}
	for _, r := range append(out.Board.Reports, out.News) {
		labels := r.Source + r.Title + r.Note + strings.Join(r.Columns, " ")
		for _, raw := range append(Kinds(), "playerProfile") {
			if strings.Contains(labels, raw) {
				t.Fatal(raw, labels)
			}
		}
	}
	h2 := in.Held.Feeds[Key("playerScores", "2")]
	h2.FetchedAt = h2.FetchedAt.Add(time.Second)
	in.Held.Feeds[Key("playerScores", "2")] = h2
	sameMinute := reportByID(t, Build(in, Scope{}).Board, "M-037")
	if strings.Contains(sameMinute.Note, "Week 2 ·") {
		t.Fatal("identical displayed times repeated", sameMinute.Note)
	}
	h := in.Held.Feeds[Key("playerScores", "3")]
	h.RefreshError = "fixture offline"
	in.Held.Feeds[Key("playerScores", "3")] = h
	r = reportByID(t, Build(in, Scope{}).Board, "M-037")
	if !strings.Contains(r.Note, "Week 3") || strings.Contains(r.Note, "Week 2 ·") || len(r.Summary) != 0 {
		t.Fatal(r)
	}
	if len(out.Players) != 0 {
		t.Fatal("empty search yielded a count")
	}
}

func TestResearchDepthMissingIdentityBreakdown(t *testing.T) {
	t.Parallel()
	in := realResearch(t)
	gz, err := gzip.NewReader(bytes.NewReader(researchSample(t, "crosswalk.csv.gz")))
	if err != nil {
		t.Fatal(err)
	}
	body, err := io.ReadAll(gz)
	if err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	cw, err := crosswalk.Fetch(context.Background(), &http.Client{Transport: researchCrosswalkFixture{body: body}}, "https://fixture.invalid/crosswalk")
	if err != nil {
		t.Fatal(err)
	}
	links, _ := cw.Links(in.Lookup.Name)
	full := map[string]string{}
	allowed := map[string]bool{}
	for _, id := range in.Lookup.IDs() {
		allowed[id] = true
	}
	for _, l := range links {
		if l.IDType != "gsis" && l.IDType != "espn" {
			continue
		}
		key := l.IDType + ":" + l.IDValue
		full[key] = l.MFLID
		if allowed[l.MFLID] {
			in.Held.Links[key] = l.MFLID
		}
	}
	missing, excluded, absent, noLink := 0, 0, 0, 0
	for _, d := range in.Held.Depth.Value {
		if DepthID(d, in.Held.Links) != "" {
			continue
		}
		missing++
		id := DepthID(d, full)
		switch {
		case id == "":
			noLink++
		case allowed[id]:
			t.Fatal("valid directory link missed", d)
		default:
			_, _, ok := in.Lookup.DisplayFacts(id)
			if ok {
				excluded++
			} else {
				absent++
			}
		}
	}
	if missing != 576 || noLink != 472 || excluded != 66 || absent != 38 {
		t.Fatal(missing, noLink, excluded, absent)
	}
	note := reportByID(t, Build(in, Scope{}).Board, "M-048").Note
	if !strings.Contains(note, "401 offensive line, 97 punter/holder/long snapper, 78 other") {
		t.Fatal(note)
	}
	t.Logf("2333 slots: 1757 linked; %d missing = %d no accepted link + %d excluded punter slots + %d absent directory slots; %s", missing, noLink, excluded, absent, note)
}

func TestInspectorNewsPostedLocalMinute(t *testing.T) {
	t.Parallel()
	in := realResearch(t)
	in.Zone = time.FixedZone("CDT", -5*60*60)
	at := time.Date(2026, 10, 13, 5, 48, 37, 0, time.UTC)
	in.Held.News["16162"] = Feed[[]researchfeeds.News]{FetchedAt: at, Value: []researchfeeds.News{
		{Article: "new", Headline: "newest", Published: "26 minutes"},
		{Article: "unknown", Headline: "unknown", Published: "recently"},
	}}
	got := (researchView{Snapshot: in}).playerReading("16162", nil, "How")
	rows := []report.Row{}
	for _, row := range got.Report.Rows {
		if strings.HasPrefix(row.Cells[0], "News · ") {
			rows = append(rows, row)
		}
	}
	if len(rows) != 2 || rows[0].Cells[0] != "News · Oct 13, 2026, 12:22 AM" ||
		rows[1].Cells[0] != "News · Posted time unknown: MFL wording not recognized" {
		t.Fatal(rows)
	}
	if !strings.Contains(rows[0].Cells[2], "Oct 13, 2026, 12:48 AM") {
		t.Fatal(rows[0])
	}
	if got.Articles[0].Article != "new" || !strings.Contains(got.Report.How, "estimated") {
		t.Fatal(got)
	}
}

func TestHeldNewsPostedSameOnEverySurface(t *testing.T) {
	t.Parallel()
	in := realResearch(t)
	in.Zone = time.FixedZone("CDT", -5*60*60)
	out := Build(in, Scope{Mine: "0001"})
	v := researchView{Snapshot: in}
	player := v.playerReading("16162", nil, "How")
	posted := ""
	for _, row := range player.Report.Rows {
		if strings.HasPrefix(row.Cells[0], "News · ") {
			posted = strings.TrimPrefix(row.Cells[0], "News · ")
		}
	}
	if posted != "Oct 5, 2026" {
		t.Fatal("held six-day article should have date precision", posted)
	}
	for _, report := range []report.Report{out.News, reportByID(t, out.Board, "M-060")} {
		if report.Columns[3] != "Posted" || len(report.Rows) != 1 || report.Rows[0].Cells[3] != posted {
			t.Fatal("news surfaces disagree", report)
		}
		if !strings.Contains(report.How, NewsHow) {
			t.Fatal("missing elapsed-time disclosure", report.How)
		}
	}
}

func TestNewsNewestFirstAcrossOwners(t *testing.T) {
	t.Parallel()
	in := realResearch(t)
	at := in.Held.News["16162"].FetchedAt
	in.Held.News["13589"] = Feed[[]researchfeeds.News]{FetchedAt: at, Value: []researchfeeds.News{
		{Article: "new", Headline: "Newest other franchise", Published: "1 minute"},
	}}
	out := Build(in, Scope{Mine: "0001"})
	if out.News.Rows[0].Cells[2] != "Newest other franchise" || out.Articles[0].Article != "new" {
		t.Fatal("ownership must not override newest first", out.News.Rows, out.Articles)
	}
}

type researchCrosswalkFixture struct{ body []byte }

func (f researchCrosswalkFixture) RoundTrip(r *http.Request) (*http.Response, error) {
	return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(bytes.NewReader(f.body)), Request: r}, nil
}

func researchSample(t *testing.T, name string) []byte {
	t.Helper()
	return researchtest.Sample(t, name)
}
func realResearch(t *testing.T) Snapshot {
	t.Helper()
	d := researchtest.Load(t)
	in := Snapshot{Week: 5, Zone: time.UTC, Now: d.At, Lookup: d.Lookup, Mirror: d.Mirror,
		Held:     Values{Feeds: map[string]Feed[[]researchfeeds.Feed]{}, News: map[string]Feed[[]researchfeeds.News]{}, Links: map[string]string{}},
		Injuries: Feed[[]leaguefeed.Injury]{Value: d.Injuries, FetchedAt: d.At}, Finished: map[string]*Score{}, Latest: d.At}
	for key, f := range d.Feeds {
		in.Held.Feeds[key] = Feed[[]researchfeeds.Feed]{Value: []researchfeeds.Feed{f}, FetchedAt: d.At}
	}
	in.Held.Depth = Feed[[]researchfeeds.RawDepth]{Value: d.Depth, FetchedAt: d.At}
	for id, n := range d.News {
		in.Held.News[id] = Feed[[]researchfeeds.News]{Value: n, FetchedAt: d.At}
	}
	for w := 1; w < in.Week; w++ {
		for _, entry := range d.Feeds[Key("playerScores", strconv.Itoa(w))].Players {
			id := entry.ID.String()
			row := in.Finished[id]
			if row == nil {
				row = &Score{Position: Player(in.Lookup, in.Mirror, in.Config, id).Position}
				in.Finished[id] = row
			}
			row.Sum += entry.Value
			row.Games++
		}
	}
	return in
}
func reportByID(t *testing.T, board report.Board, id string) report.Report {
	t.Helper()
	for _, r := range board.Reports {
		if r.ID == id {
			return r
		}
	}
	t.Fatalf("report %s missing", id)
	return report.Report{}
}

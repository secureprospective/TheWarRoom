package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"io"
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/ingestion/crosswalk"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/researchfeeds"
	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

func TestResearchColdOpenReadsArchiveBeforeFetch(t *testing.T) {
	in := realResearch(t)
	a := targetTestApp(t)
	a.lookup, a.hasLookup = in.lookup, true
	if _, err := a.league.Replace(a.ctx, in.mirror); err != nil {
		t.Fatal(err)
	}
	at := time.Date(2026, 10, 11, 10, 0, 0, 0, time.UTC)
	recordSeasonArchive(t, a, "playerScores", "YTD", string(researchSample(t, "playerScores_WYTD.json")), at)
	recordSeasonArchive(t, a, "playerScores", "1", string(researchSample(t, "playerScores_W01.json")), at)
	recordSeasonBareArchive(t, a, "injuries", string(researchSample(t, "injuries.json")), at)
	recordSeasonArchiveForm(t, a, "playerProfile", "", string(researchSample(t, "playerProfile.json")), at, false)
	// Profiles require the exact P member; this helper's URL has no P, so it must not answer.
	a.refreshMu.Lock()
	defer a.refreshMu.Unlock()
	type answer struct {
		board ResearchBoard
		err   error
	}
	read := make(chan answer, 1)
	go func() { board, err := a.TargetResearch(RawResearchScope{}); read <- answer{board, err} }()
	var out ResearchBoard
	select {
	case result := <-read:
		if result.err != nil {
			t.Fatal(result.err)
		}
		out = result.board
	case <-time.After(15 * time.Second):
		t.Fatal("opening blocked behind launch refresh")
	}
	r := reportByID(t, out.Board, "M-037")
	if len(r.Rows) == 0 || !strings.Contains(r.Note, "Archived copy") || !strings.Contains(r.AsOf, reportTime(at, time.Local)) {
		t.Fatal(r.Note, r.AsOf, len(r.Rows))
	}
	if len(a.seasonInjuries.value) == 0 {
		t.Fatal("cold injury archive not served")
	}
	if !a.research.feeds[researchKey("playerScores", "YTD")].fetchedAt.Equal(at) {
		t.Fatal("archive time replaced")
	}
	if len(a.research.news["16162"].value) != 0 {
		t.Fatal("profile URL without exact player accepted")
	}
	recordSeasonArchive(t, a, "playerProfile&P=16162", "", string(researchSample(t, "playerProfile.json")), at)
	// An archive lookup that found nothing is not repeated on every read: the refresh fills it.
	if _, err := a.TargetResearch(RawResearchScope{}); err != nil {
		t.Fatal(err)
	}
	if len(a.research.news["16162"].value) != 0 || !a.research.archiveRead["news:16162"] {
		t.Fatal("archive looked up again on a second read")
	}
	a.research.archiveRead = nil // the next launch looks again
	if _, err := a.TargetResearch(RawResearchScope{}); err != nil {
		t.Fatal(err)
	}
	if len(a.research.news["16162"].value) != 1 {
		t.Fatal("cold archived news not served")
	}
	in.mirror.Players = slices.DeleteFunc(slices.Clone(in.mirror.Players), func(p state.MirrorPlayer) bool { return p.MFLID == "16162" })
	if _, err := a.league.Replace(a.ctx, in.mirror); err != nil {
		t.Fatal(err)
	}
	a.research.news, a.research.archiveRead = nil, nil
	player, err := a.TargetResearchPlayer("16162")
	if err != nil || player.Owner != "Free agent" || len(player.Articles) != 1 {
		t.Fatal("free agent’s archived news was not served", player.Owner, err)
	}
}

func TestResearchNewsNewestFirstAndUnknownLastStable(t *testing.T) {
	in := realResearch(t)
	rows := []researchfeeds.News{
		{Article: "unknown1", Headline: "first unknown", Published: "recently"},
		{Article: "old", Headline: "six days", Published: "6 days"},
		{Article: "new", Headline: "fifteen hours", Published: "15 hours"},
		{Article: "middle", Headline: "one day", Published: "1 day"},
		{Article: "unknown2", Headline: "second unknown", Published: "yesterday"},
	}
	in.held.news["16162"] = heldSeasonFeed[[]researchfeeds.News]{value: rows, fetchedAt: time.Now()}
	out := buildResearch(in, RawResearchScope{Mine: "0001"})
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
	for _, word := range []string{"6 days", "15 hours", "1 day", "1 hour", "2 minutes"} {
		if _, ok := researchNewsAge(word); !ok {
			t.Fatal(word)
		}
	}
	for _, word := range []string{"", "-1 day", "1 day ago", "recently", "1.5 days"} {
		if _, ok := researchNewsAge(word); ok {
			t.Fatal(word)
		}
	}
}

func TestResearchPlayerHeldFactsAndMissingPieces(t *testing.T) {
	in := realResearch(t)
	in.injuries.value = []leaguefeed.Injury{{ID: in.held.news["16162"].value[0].ID, Status: "Questionable", Details: "Knee", ExpReturn: "Week 6"}}
	v := researchView{researchInput: in}
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
	in.mirror.Players = slices.DeleteFunc(slices.Clone(in.mirror.Players), func(p state.MirrorPlayer) bool { return p.MFLID == "16162" })
	free := (researchView{researchInput: in}).playerReading("16162", nil, "How this is counted")
	if free.Owner != "Free agent" || free.FranchiseID != "" {
		t.Fatal(free)
	}
	empty := (researchView{researchInput: researchInput{lookup: in.lookup, zone: time.UTC}}).playerReading("16162", nil, "How this is counted")
	for _, r := range empty.Report.Rows[1:] {
		if !strings.HasPrefix(r.Cells[1], "not held") || r.Cells[2] != "not read yet" {
			t.Fatal(r)
		}
	}
	read := researchView{researchInput: in}
	read.injuries.fetchedAt = time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC)
	read.injuries.value = nil
	quiet := read.held.news["16162"]
	quiet.value = nil
	read.held.news = map[string]heldSeasonFeed[[]researchfeeds.News]{"16162": quiet}
	healthy := read.playerReading("16162", nil, "How this is counted").Report.Rows
	if healthy[1].Cells[1] != "Not on MFL’s injury report" || healthy[len(healthy)-1].Cells[1] != "No MFL news for this player" {
		t.Fatal(healthy)
	}
	d := in.held.depth.value[0]
	in.held.links["gsis:"+*d.GSIS] = "16162"
	linked := (researchView{researchInput: in}).playerReading("16162", nil, "How this is counted")
	if !strings.Contains(linked.Report.Rows[2].Cells[1], d.Group+" · "+d.Position+" · depth") {
		t.Fatal(linked.Report.Rows)
	}
}

func TestResearchSourcesFoldWeeksAndUseWords(t *testing.T) {
	in := realResearch(t)
	out := buildResearch(in, RawResearchScope{})
	r := reportByID(t, out.Board, "M-038")
	if len(r.Summary) != 0 || !strings.Contains(r.Source, "weeks 1–5 as of") {
		t.Fatal(r)
	}
	for _, r := range append(out.Board.Reports, out.News) {
		labels := r.Source + r.Title + r.Note + strings.Join(r.Columns, " ")
		for _, raw := range append(researchKinds(), "playerProfile") {
			if strings.Contains(labels, raw) {
				t.Fatal(raw, labels)
			}
		}
	}
	h2 := in.held.feeds[researchKey("playerScores", "2")]
	h2.fetchedAt = h2.fetchedAt.Add(time.Second)
	in.held.feeds[researchKey("playerScores", "2")] = h2
	sameMinute := reportByID(t, buildResearch(in, RawResearchScope{}).Board, "M-037")
	if strings.Contains(sameMinute.Note, "Week 2 ·") {
		t.Fatal("identical displayed times repeated", sameMinute.Note)
	}
	h := in.held.feeds[researchKey("playerScores", "3")]
	h.refreshError = "fixture offline"
	in.held.feeds[researchKey("playerScores", "3")] = h
	r = reportByID(t, buildResearch(in, RawResearchScope{}).Board, "M-037")
	if !strings.Contains(r.Note, "Week 3") || strings.Contains(r.Note, "Week 2 ·") || len(r.Summary) != 0 {
		t.Fatal(r)
	}
	if len(out.Players) != 0 {
		t.Fatal("empty search yielded a count")
	}
}

type researchCrosswalkFixture struct{ body []byte }

func (f researchCrosswalkFixture) RoundTrip(r *http.Request) (*http.Response, error) {
	return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(bytes.NewReader(f.body)), Request: r}, nil
}
func TestResearchDepthMissingIdentityBreakdown(t *testing.T) {
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
	links, _ := cw.Links(in.lookup.Name)
	full := map[string]string{}
	allowed := map[string]bool{}
	for _, id := range in.lookup.IDs() {
		allowed[id] = true
	}
	for _, l := range links {
		if l.IDType != "gsis" && l.IDType != "espn" {
			continue
		}
		key := l.IDType + ":" + l.IDValue
		full[key] = l.MFLID
		if allowed[l.MFLID] {
			in.held.links[key] = l.MFLID
		}
	}
	missing, excluded, absent, noLink := 0, 0, 0, 0
	for _, d := range in.held.depth.value {
		if depthID(d, in.held.links) != "" {
			continue
		}
		missing++
		id := depthID(d, full)
		switch {
		case id == "":
			noLink++
		case allowed[id]:
			t.Fatal("valid directory link missed", d)
		default:
			_, _, ok := in.lookup.DisplayFacts(id)
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
	note := (researchView{researchInput: in}).depthMissNote(false)
	if !strings.Contains(note, "401 offensive line, 97 punter/holder/long snapper, 78 other") {
		t.Fatal(note)
	}
	t.Logf("2333 slots: 1757 linked; %d missing = %d no accepted link + %d excluded punter slots + %d absent directory slots; %s", missing, noLink, excluded, absent, note)
}

func TestResearchStatusWhileWaitingForLaunchRefresh(t *testing.T) {
	a := targetTestApp(t)
	a.seasonPause = func(context.Context, time.Duration) error { return context.Canceled }
	a.refreshMu.Lock()
	done := make(chan error, 1)
	go func() { done <- a.TargetResearchRefresh(false) }()
	deadline := time.Now().Add(time.Second)
	for a.TargetResearchStatus() != "Waiting for the MFL refresh to finish" && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	status := a.TargetResearchStatus()
	a.refreshMu.Unlock()
	if err := <-done; err == nil {
		t.Fatal("cancelled fixture refresh passed")
	}
	if status != "Waiting for the MFL refresh to finish" {
		t.Fatal(status)
	}
	if a.TargetResearchStatus() != "" {
		t.Fatal("status retained after failure")
	}
}

func TestResearchWeekRunsNameEachDifferingRunOnce(t *testing.T) {
	got := weekRuns([]string{"", "as of A · Archived copy", "as of A · Archived copy", "as of A · Archived copy", "as of B"})
	if got != " Weeks 2–4 · as of A · Archived copy. Week 5 · as of B." {
		t.Fatal(got)
	}
	if weekRuns([]string{"", ""}) != "" {
		t.Fatal("matching weeks named")
	}
}

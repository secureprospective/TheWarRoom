package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/facts"

	"github.com/secureprospective/TheWarRoom/internal/manifest"
	"github.com/secureprospective/TheWarRoom/internal/measures"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
	"github.com/secureprospective/TheWarRoom/internal/store/params"
)

// Two bodies byte for byte as the frozen Lab read them (2013/aav.json, 2013/tradeBait.json).
const (
	labAAV2013   = `{"aav":{"timestamp":"","totalPlayers":"0","totalAuctions":"0"},"version":"1.0","encoding":"utf-8"}`
	labTrade2013 = `{"version":"1.0","tradeBaits":{},"encoding":"utf-8"}`
)

// archiveTree writes a three-season archive: the Lab's own 2013 aav and trade bait, one body
// saved twice, and an MFL error body.
func archiveTree(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	league := func(id, host string) string {
		return `{"league":{"id":"` + id + `","baseURL":"https://` + host + `.myfantasyleague.com"}}`
	}
	files := map[string]string{
		"2013/league.json":             league("51719", "www48"),
		"2013/aav.json":                labAAV2013,
		"2013/tradeBait.json":          labTrade2013,
		"2014/league.json":             league("47710", "www46"),
		"2014/tradeBait.json":          labTrade2013,
		"2014/polls.json":              `{"error":{"$t":"API requires logged in user"}}`,
		"2026/league.json":             league("14432", "www47"),
		"2026/weekly/rosters_W01.json": `{"rosters":{}}`,
	}
	captured := time.Date(2026, 10, 5, 2, 0, 0, 0, time.UTC)
	for rel, body := range files {
		path := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.Chtimes(path, captured, captured); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestArchiveImportIsIdempotentAndPinsTheLabsInputs(t *testing.T) {
	a := targetTestApp(t)
	a.dataDir = t.TempDir()
	dir := archiveTree(t)
	if r := a.SetLeagueArchiveDir(dir); !r.OK {
		t.Fatal(r.Error)
	}
	first := a.TargetDataRefresh("archive")
	if !first.OK || !strings.Contains(first.Summary, "Looked at 8 files: 8 new, 0 already held. MFL error bodies, kept as errors: 1.") ||
		!strings.Contains(first.Summary, "1,482 files pinned, 1,480 not in the archive") {
		t.Fatalf("first import = %+v", first)
	}
	again := a.TargetDataRefresh("archive")
	if !again.OK || !strings.Contains(again.Summary, "8 files: 0 new, 8 already held") {
		t.Fatalf("second import = %+v, want nothing new", again)
	}
	ctx := context.Background()
	fetches, err := a.history.SourceFetches(ctx, leagueArchiveSource)
	if err != nil || len(fetches) != 8 {
		t.Fatalf("archive fetches = %d, %v; want 8 after two runs", len(fetches), err)
	}
	m, published, err := a.publishLabManifest(ctx)
	if err != nil || published {
		t.Fatalf("republish = %v, %v; want the same manifest already held", published, err)
	}
	var held []manifest.Entry
	for _, e := range m.Entries {
		if e.Held() {
			held = append(held, e)
		}
	}
	if len(held) != 2 || held[0].File != "2013/aav.json" || held[0].League != "" || held[0].Scope != "site" ||
		held[1].File != "2013/tradeBait.json" || held[1].League != "51719" || held[1].CapturedAt != "2026-10-05T02:00:00Z" {
		t.Fatalf("held entries = %+v, want the Lab's 2013 aav and trade bait", held)
	}
	if _, err := m.Resolve(ctx, a.history); err == nil || !strings.Contains(err.Error(), "2013/assets.json") {
		t.Errorf("resolve err = %v, want the missing frozen files named", err)
	}
	ds, err := a.TargetDataSources()
	if err != nil {
		t.Fatal(err)
	}
	if got := cell(ds.Facts, "In the archive"); got != "2 of 1,482" {
		t.Errorf("In the archive = %q", got)
	}
	if got := cell(ds.Facts, "Compiled snapshot"); !strings.HasPrefix(got, "None yet. Nothing has been compiled") {
		t.Errorf("Compiled snapshot = %q", got)
	}
	if ds.ArchiveDir != dir || len(ds.Scopes) != 4 || len(ds.Checks.Rows) < 3 {
		t.Errorf("data sources = dir %q, %d scopes, %d checks", ds.ArchiveDir, len(ds.Scopes), len(ds.Checks.Rows))
	}
}

func TestArchiveImportRefusesAMalformedFileAndRecordsTheFailure(t *testing.T) {
	a := targetTestApp(t)
	a.dataDir = t.TempDir()
	dir := archiveTree(t)
	if err := os.WriteFile(filepath.Join(dir, "2014/rules.json"), []byte(`{"rules":`), 0o600); err != nil {
		t.Fatal(err)
	}
	if r := a.SetLeagueArchiveDir(dir); !r.OK {
		t.Fatal(r.Error)
	}
	r := a.TargetDataRefresh("archive")
	if r.OK || !strings.Contains(r.Error, "2014/rules.json: not an MFL export") {
		t.Fatalf("import = %+v, want the malformed file named", r)
	}
	ctx := context.Background()
	if fetches, _ := a.history.SourceFetches(ctx, leagueArchiveSource); len(fetches) != 0 {
		t.Errorf("%d fetches recorded from a malformed archive", len(fetches))
	}
	health, err := a.history.SourceHealth(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, h := range health {
		if h.Source == leagueArchiveSource && !strings.Contains(h.LastError, "2014/rules.json") {
			t.Errorf("archive health = %+v, want the failure recorded", h)
		}
	}
	for _, bad := range []string{"relative/raw", t.TempDir()} {
		if r := a.SetLeagueArchiveDir(bad); r.OK {
			t.Errorf("archive folder %q accepted", bad)
		}
	}
	if r := a.TargetDataRefresh("everything"); r.OK || !strings.Contains(r.Error, `unknown data scope "everything"`) {
		t.Errorf("unknown scope = %+v", r)
	}
}

func cell(r Report, key string) string {
	for _, row := range r.Rows {
		if row.Cells[0] == key {
			return row.Cells[1]
		}
	}
	return ""
}

func TestHealthReportSaysWhatFailedAndWhetherTheBoardRunsReduced(t *testing.T) {
	reg, err := measures.Embedded()
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 10, 11, 18, 0, 0, 0, time.UTC)
	var nflverseMeasure string
	for _, f := range reg.Fields {
		if f.Source == "nflverse" {
			nflverseMeasure = f.Measure
			break
		}
	}
	in := healthInput{reg: reg, now: now, zone: time.UTC, boardRun: true, missing: []string{nflverseMeasure},
		sources: []history.SourceHealth{
			{Source: "mfl", State: history.SourceActive, LastSuccess: now.Add(-3 * time.Hour)},
			{Source: "nflverse", State: history.SourceLost, LastSuccess: now.Add(-20 * 24 * time.Hour),
				LastFailure: now.Add(-time.Hour), LastError: "HTTP 404"},
			{Source: "cfbd", State: history.SourceActive, LastSuccess: now.Add(-time.Minute), LastFailure: now.Add(-time.Hour), LastError: "timeout"},
			{Source: "madden", State: history.SourceRetired},
			{Source: leagueArchiveSource, State: history.SourceActive},
		},
		mfl: []history.URLHealth{
			{URL: "https://www47.myfantasyleague.com/2026/export?TYPE=rosters&L=14432", LastSuccess: now.Add(-2 * time.Hour)},
			{URL: "https://www47.myfantasyleague.com/2026/export?TYPE=rosters&L=14432&W=3", LastFailure: now.Add(-time.Hour), LastError: "HTTP 503"},
		}}
	r := healthReport(in)
	rows := map[string][]string{}
	for _, row := range r.Rows {
		rows[strings.Split(row.Cells[0], "\n")[0]] = row.Cells
	}
	want := map[string][]string{
		"MFL · rosters": {"MFL · rosters", "Oct 11, 2026, 4:00 PM\n2 hours", "Oct 11, 2026, 5:00 PM: HTTP 503"},
		"nflverse data releases": {"nflverse data releases\nRunning reduced: 1 of its measures not flowing (" + nflverseMeasure + ")",
			"Sep 21, 2026, 6:00 PM\nLost · 20 days", "Oct 11, 2026, 5:00 PM: HTTP 404"},
		"CollegeFootballData API": {"CollegeFootballData API\nFeeding the board", "Oct 11, 2026, 5:59 PM\nUnder an hour",
			"Oct 11, 2026, 5:00 PM (recovered since)"},
		"EA Madden ratings":             {"EA Madden ratings", "Never\nRetired", "None"},
		"MFL league archive, 2013–2026": {"MFL league archive, 2013–2026", "Never\nNo good read yet", "None"},
	}
	// MFL's own row stands for its exports: the newest export read and failure, never "Never".
	mfl, _ := reg.Source("mfl")
	want[mfl.Name] = []string{mfl.Name + "\nFeeding the board", "Oct 11, 2026, 4:00 PM\n2 hours",
		"Oct 11, 2026, 5:00 PM: HTTP 503"}
	for name, cells := range want {
		if strings.Join(rows[name], " | ") != strings.Join(cells, " | ") {
			t.Errorf("%s:\n got %q\nwant %q", name, rows[name], cells)
		}
	}
	in.boardRun = false
	if got := boardText("cfbd", in); got != "No board run yet" {
		t.Errorf("board text without a run = %q", got)
	}
}

func TestFactsReportIsHonestBeforeAndAfterTheManifest(t *testing.T) {
	in := factsInput{zone: time.UTC, labBuilt: "2026-10-06T15:57:20+00:00", labThrough: "2026-10-06T01:42:57+00:00"}
	r := factsReport(in)
	if r.Status != "empty" || !strings.Contains(r.Note, "import the league archive") ||
		cell(r, "Lab data") != "Captured through Oct 6, 2026, 1:42 AM; built Oct 6, 2026, 3:57 PM" {
		t.Errorf("before the manifest: %+v", r)
	}
	in.published, in.publishedAt = true, time.Date(2026, 10, 11, 20, 0, 0, 0, time.UTC)
	in.manifest = manifest.Manifest{SHA256: strings.Repeat("ab", 32), Entries: []manifest.Entry{
		{File: "2013/polls.json", Export: "polls", Season: 2013, SHA256: "p", MFLError: "MFL error: login", FetchID: 1},
		{File: "2013/assets.json", Export: "assets", Season: 2013, SHA256: "a"},
		{File: "2026/rules.json", Export: "rules", Season: 2026, SHA256: "r", FetchID: 2},
	}}
	in.held = map[string]bool{"p": true, "r": true}
	built := time.Date(2026, 10, 12, 1, 0, 0, 0, time.UTC)
	in.status = manifest.Status{State: manifest.StateStale, Good: manifest.Attempt{BuiltAt: built},
		Reason: "The inputs changed after it was built."}
	r = factsReport(in)
	for k, v := range map[string]string{
		"Manifest":           "abababababab, published Oct 11, 2026, 8:00 PM",
		"Files pinned":       "3 archive files, over seasons 2013–2026",
		"Exports":            "assets 1, polls 1, rules 1",
		"In the archive":     "2 of 3",
		"Not in the archive": "1: 2013/assets.json. No replacement bytes were used; approved non-data gaps are named below.",
		"MFL error bodies":   "1 pinned files are MFL errors; they are kept as errors and never read as data",
		"Compiled snapshot":  "Stale, built Oct 12, 2026, 1:00 AM. The inputs changed after it was built.",
	} {
		if got := cell(r, k); got != v {
			t.Errorf("%s = %q, want %q", k, got, v)
		}
	}
}

func TestRefreshSummariesSayWhatHappened(t *testing.T) {
	r := SignalsReport{Loads: []SignalLoad{{Status: loadDone, Added: 1200}, {Status: loadCurrent}, {Status: loadFailed},
		{Status: loadSkipped}, {Status: loadDone, Added: 34}}}
	if got := signalsSummary(r); got != "Looked at 5 files: 2 loaded, 1 already current, 1 skipped, 1 failed; 1,234 new values." {
		t.Errorf("signals summary = %q", got)
	}
	if got := leagueRefreshSummary(RefreshResult{Season: 2026, Changed: true}); got !=
		"Season 2026: rosters and salary adjustments updated; rules unchanged." {
		t.Errorf("league summary = %q", got)
	}
	for n, want := range map[int]string{0: "0", 999: "999", 1000: "1,000", 1234567: "1,234,567", -4200: "-4,200"} {
		if got := thousands(n); got != want {
			t.Errorf("thousands(%d) = %q, want %q", n, got, want)
		}
	}
	for d, want := range map[time.Duration]string{59 * time.Minute: "Under an hour", 47 * time.Hour: "47 hours", 49 * time.Hour: "2 days"} {
		if got := plainAge(d); got != want {
			t.Errorf("plainAge(%s) = %q, want %q", d, got, want)
		}
	}
}

func TestCrosswalkReportKeepsLeadingZeroIDsAndGoWording(t *testing.T) {
	a := targetTestApp(t)
	if r := a.TargetCrosswalk(); r.Status != "empty" || !strings.Contains(r.Note, "No crosswalk loaded") {
		t.Fatalf("empty crosswalk = %+v", r)
	}
	a.crosswalk = CrosswalkReport{OK: true, LoadedAt: "2026-10-11T18:00:00Z", Links: 1000, Promoted: 12,
		Rates:     []CrosswalkRate{{Position: "QB", Rostered: 4, RosteredMatched: 3, FreeAgents: 0}},
		Total:     CrosswalkRate{Position: "ALL", Rostered: 4, RosteredMatched: 3},
		Unmatched: []CrosswalkMiss{{MFLID: "0042", Name: "Waiting", Team: "BUF", Position: "QB", Franchise: "0001", Reason: "no NFL id yet"}}}
	r := a.TargetCrosswalk()
	if r.Note != "1,000 links; 12 waiting facts resolved." || r.Rows[0].Cells[1] != "3 of 4 · 75.0%" ||
		r.Rows[0].Cells[2] != "None" || r.Rows[2].Cells[0] != "Waiting" || r.Rows[2].Cells[1] != "QB · BUF" {
		t.Fatalf("crosswalk = %+v", r)
	}
}

func TestDataSourcesWireMatchesGo(t *testing.T) {
	reg, err := measures.Embedded()
	if err != nil {
		t.Fatal(err)
	}
	at := time.Date(2026, 10, 11, 18, 0, 0, 0, time.UTC)
	a := targetTestApp(t)
	a.crosswalk = CrosswalkReport{OK: true, LoadedAt: "2026-10-11T18:00:00Z", Links: 1000,
		Rates: []CrosswalkRate{{Position: "WR", Rostered: 4, RosteredMatched: 3}},
		Total: CrosswalkRate{Position: "ALL", Rostered: 4, RosteredMatched: 3},
		Unmatched: []CrosswalkMiss{{MFLID: "0816", Name: "Gosnell, Stephen", Team: "MIN", Position: "WR",
			Franchise: "Cincinnati Bengals", Reason: "no NFL id yet"}}}
	c := a.TargetCrosswalk()
	wire := struct {
		Data      DataSources   `json:"data"`
		Crosswalk Report        `json:"crosswalk"`
		Params    ParamsResult  `json:"params"`
		Result    DataActResult `json:"result"`
	}{
		Data: DataSources{Health: healthReport(healthInput{reg: reg, now: at, zone: time.UTC,
			sources: []history.SourceHealth{{Source: leagueArchiveSource, State: history.SourceActive, LastSuccess: at}}}),
			Signals: signalFilesReport(SignalsView{Feeds: []FeedView{{Feed: "stats_player_week", Source: "nflverse",
				LastLoaded: "2026-10-11T18:00:00Z", Fresh: true, CoverageSeason: 2026,
				Seasons:  []SeasonCount{{Season: 2026, Players: 1300, Waiting: 37}},
				Coverage: []PositionShare{{Position: "WR", Rostered: 1454, WithData: 1105}}}}}, SignalsReport{}, time.UTC),
			Facts: factsReport(factsInput{labBuilt: "2026-10-06T15:57:20+00:00", labThrough: "2026-10-06T01:42:57+00:00", zone: time.UTC}),
			Checks: checksReport(factsInput{zone: time.UTC, latest: &history.SnapshotRecord{BuiltAt: at}, results: []facts.CheckResult{
				{Name: "Win-loss records match MFL standings", Detail: "MFL counted playoff games in the 2013 standings.",
					Checked: 448, Agreed: 446, Explained: 1, Review: 1, Passed: true, Items: []facts.CheckItem{
						{Key: "2013/0001", Words: "2013 Buffalo Bills: MFL standings 11-3-0 count its playoff games (regular season 11-2-0).",
							Class: facts.ClassExplained, Reason: "In 2013 MFL counted playoff games in the standings."},
						{Key: "2014/0009", Words: "2014 Houston Texans: MFL standings 9-4-0, weekly games 9-3-0.",
							Class: facts.ClassReview, Reason: "Possibly a commissioner adjustment."}}},
				{Name: "Lineup scores equal MFL player scores", Checked: 202954, Agreed: 202954, Passed: true}}}),
			ArchiveDir: "/fixture/raw", PhotosOn: true, Scopes: dataScopes(), Words: dataWords()},
		Crosswalk: c, Params: ParamsResult{OK: true, Params: []ParamView{{Key: "test.weight", Label: "Test weight", Group: "Other settings", Position: "WR", Value: 0.5, Default: 0.5, Min: 0, Max: 1, Description: "Test weight", Overridden: true}}},
		Result: DataActResult{OK: true, Summary: "Looked at 8 files: 0 new, 8 already held."},
	}
	body, err := json.MarshalIndent(wire, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	body = append(body, '\n')
	path := filepath.Join("frontend", "src", "app", "datasources", "wire.testdata.json")
	if os.Getenv("TWR_WRITE_WIRE") == "1" {
		if err := os.WriteFile(path, body, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	held, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != string(held) {
		t.Fatal("data sources wire stale; regenerate with TWR_WRITE_WIRE=1")
	}
}

func TestCheckedFactsNamesOnlyExactApprovedGaps(t *testing.T) {
	a := targetTestApp(t)
	a.dataDir = t.TempDir()
	if r := a.SetLeagueArchiveDir(archiveTree(t)); !r.OK {
		t.Fatal(r.Error)
	}
	if r := a.TargetDataRefresh("archive"); !r.OK {
		t.Fatal(r.Error)
	}
	in, err := a.factsInput(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	r := factsReport(in)
	for _, gap := range in.gaps {
		if got := cell(r, "Named gap: "+gap.File); got != gap.Reason || !strings.Contains(got, "fed no fact") {
			t.Errorf("gap %s = %q", gap.File, got)
		}
	}
	for i := range in.manifest.Entries {
		if in.manifest.Entries[i].File == in.gaps[0].File {
			in.manifest.Entries[i].SHA256 = strings.Repeat("0", 64)
		}
	}
	if got := cell(factsReport(in), "Named gap: "+in.gaps[0].File); got != "" {
		t.Errorf("changed pin incorrectly received exception: %q", got)
	}
}

func TestEveryEngineParameterHasGoPlainWordsAndOrderedAreas(t *testing.T) {
	a := targetTestApp(t)
	a.params = params.New(a.pools)
	if err := a.params.Initialize(a.ctx); err != nil {
		t.Fatal(err)
	}
	result := a.GetParams()
	if !result.OK || len(result.Params) < 100 {
		t.Fatalf("params = %+v", result)
	}
	previous := ""
	for _, p := range result.Params {
		if p.Label == "" || p.Group == "" || p.Group == "Other settings" ||
			strings.ContainsAny(p.Label, "._") || strings.Contains(p.Label, "model") {
			t.Errorf("parameter %s has no plain label and area: %q / %q", p.Key, p.Label, p.Group)
		}
		if p.Group < previous {
			t.Errorf("areas not grouped: %q after %q", p.Group, previous)
		}
		previous = p.Group
		if p.Key == "captier.cold_ceiling_pct" && p.Label != "Cold salary tier ceiling (% of cap)" {
			t.Errorf("cold ceiling = %q", p.Label)
		}
		if p.Key == "l4.breakout.athletic_lift" && p.Label != "how far athletic rating lifts breakout age and age trajectory" {
			t.Errorf("breakout label = %q", p.Label)
		}
	}
}

func TestSignalTableKeepsAllFactsInFourColumns(t *testing.T) {
	r := signalFilesReport(SignalsView{Feeds: []FeedView{{Feed: "stats_player_week", Source: "nflverse",
		LastLoaded: "2026-10-11T18:00:00Z", Fresh: true, CoverageSeason: 2026,
		Seasons:  []SeasonCount{{Season: 2026, Players: 1300, Waiting: 37}},
		Coverage: []PositionShare{{Rostered: 1454, WithData: 1105}}}}}, SignalsReport{}, time.UTC)
	want := []string{"stats_player_week\nnflverse", "Oct 11, 2026, 6:00 PM\nFresh: Yes",
		"1,300 in 2026\nWaiting: 37", "75% (1,105 of 1,454)"}
	if len(r.Columns) != 4 || strings.Join(r.Rows[0].Cells, "|") != strings.Join(want, "|") {
		t.Fatalf("signal files = %+v", r)
	}
}

func TestSignalResultExplainsCrosswalkTimeAndRelinkFailure(t *testing.T) {
	for _, note := range []string{
		"Player crosswalk reloaded and relinked before signals; its Loaded time records that relink.",
		"Player crosswalk relink failed; its last Loaded time is kept: held failure",
	} {
		got := signalsSummary(SignalsReport{CrosswalkNote: note})
		if !strings.HasSuffix(got, note) {
			t.Errorf("summary = %q, want %q", got, note)
		}
	}
}

func TestPhotoHealthRowSaysWhatIsHeldAndWhenOff(t *testing.T) {
	at := time.Date(2026, 10, 11, 21, 0, 0, 0, time.UTC)
	got := photoCells(photoHealth{held: 412, bytes: 3 << 20, fetched: 40, missing: 2, lastFail: at,
		lastError: "headshot: a.espncdn.com answered HTTP 503"}, time.UTC)
	want := []string{"Player photos (ESPN, then Sleeper)\nPersonal use only",
		"412 photos held (3.0 MB)\nSince launch: 40 fetched, 2 with no photo",
		"Oct 11, 2026, 9:00 PM: headshot: a.espncdn.com answered HTTP 503"}
	if strings.Join(got, " | ") != strings.Join(want, " | ") {
		t.Fatalf("photo row\n got %q\nwant %q", got, want)
	}
	off := photoCells(photoHealth{off: true, held: 1}, time.UTC)
	if !strings.HasSuffix(off[0], "Off: no photo is requested") || !strings.HasPrefix(off[1], "1 photo held") ||
		off[2] != "None" {
		t.Fatalf("photos off: %q", off)
	}
}

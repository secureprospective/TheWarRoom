package sources

import (
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/normalize"

	"github.com/secureprospective/TheWarRoom/internal/app/report"

	"github.com/secureprospective/TheWarRoom/internal/facts"

	"github.com/secureprospective/TheWarRoom/internal/manifest"
	"github.com/secureprospective/TheWarRoom/internal/measures"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

func TestHealthReportSaysWhatFailedAndWhenTheEngineRan(t *testing.T) {
	t.Parallel()
	reg, err := measures.Embedded()
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 10, 11, 18, 0, 0, 0, time.UTC)
	in := HealthInput{Reg: reg, Now: now, Zone: time.UTC,
		Sources: []history.SourceHealth{
			{Source: "mfl", State: history.SourceActive, LastSuccess: now.Add(-3 * time.Hour)},
			{Source: "nflverse", State: history.SourceLost, LastSuccess: now.Add(-20 * 24 * time.Hour),
				LastFailure: now.Add(-time.Hour), LastError: "HTTP 404"},
			{Source: "cfbd", State: history.SourceActive, LastSuccess: now.Add(-time.Minute), LastFailure: now.Add(-time.Hour), LastError: "timeout"},
			{Source: "madden", State: history.SourceRetired},
			{Source: LeagueArchiveSource, State: history.SourceActive},
		},
		MFL: []history.URLHealth{
			{URL: "https://www47.myfantasyleague.com/2026/export?TYPE=rosters&L=14432", LastSuccess: now.Add(-2 * time.Hour)},
			{URL: "https://www47.myfantasyleague.com/2026/export?TYPE=rosters&L=14432&W=3", LastFailure: now.Add(-time.Hour), LastError: "HTTP 503"},
		}}
	r := HealthReport(in)
	rows := map[string][]string{}
	for _, row := range r.Rows {
		rows[strings.Split(row.Cells[0], "\n")[0]] = row.Cells
	}
	want := map[string][]string{
		"MFL · rosters": {"MFL · rosters", "Oct 11, 2026, 4:00 PM\n2 hours", "Oct 11, 2026, 5:00 PM: HTTP 503"},
		"nflverse data releases": {"nflverse data releases\nEngine run not yet available",
			"Sep 21, 2026, 6:00 PM\nLost · 20 days", "Oct 11, 2026, 5:00 PM: HTTP 404"},
		"CollegeFootballData API": {"CollegeFootballData API\nEngine run not yet available", "Oct 11, 2026, 5:59 PM\nUnder an hour",
			"Oct 11, 2026, 5:00 PM (recovered since)"},
		"EA Madden ratings":             {"EA Madden ratings", "Never\nRetired", "None"},
		"MFL league archive, 2013–2026": {"MFL league archive, 2013–2026", "Never\nNo good read yet", "None"},
	}
	// MFL's own row stands for its exports: the newest export read and failure, never "Never".
	mfl, _ := reg.Source("mfl")
	want[mfl.Name] = []string{mfl.Name + "\nEngine run not yet available", "Oct 11, 2026, 4:00 PM\n2 hours",
		"Oct 11, 2026, 5:00 PM: HTTP 503"}
	for name, cells := range want {
		if strings.Join(rows[name], " | ") != strings.Join(cells, " | ") {
			t.Errorf("%s:\n got %q\nwant %q", name, rows[name], cells)
		}
	}
	in.EngineAt = now.Add(-time.Hour)
	if got := boardText("cfbd", in); got != "Engine run: "+in.EngineAt.In(in.Zone).Format("January 2, 2006, 3:04 PM") {
		t.Errorf("board text after an engine run = %q", got)
	}
	if got := boardText(LeagueArchiveSource, in); got != "Not read by the board" {
		t.Errorf("board text for a reference source = %q", got)
	}
}

func TestFactsReportIsHonestBeforeAndAfterTheManifest(t *testing.T) {
	t.Parallel()
	in := FactsInput{Zone: time.UTC, LabBuilt: "2026-10-06T15:57:20+00:00", LabThrough: "2026-10-06T01:42:57+00:00"}
	r := FactsReport(in)
	if r.Status != "empty" || !strings.Contains(r.Note, "import the league archive") ||
		cell(r, "Lab data") != "Captured through Oct 6, 2026, 1:42 AM; built Oct 6, 2026, 3:57 PM" {
		t.Errorf("before the Manifest: %+v", r)
	}
	in.Published, in.PublishedAt = true, time.Date(2026, 10, 11, 20, 0, 0, 0, time.UTC)
	in.Manifest = manifest.Manifest{SHA256: strings.Repeat("ab", 32), Entries: []manifest.Entry{
		{File: "2013/polls.json", Export: "polls", Season: 2013, SHA256: "p", MFLError: "MFL error: login", FetchID: 1},
		{File: "2013/assets.json", Export: "assets", Season: 2013, SHA256: "a"},
		{File: "2026/rules.json", Export: "rules", Season: 2026, SHA256: "r", FetchID: 2},
	}}
	in.Held = map[string]bool{"p": true, "r": true}
	built := time.Date(2026, 10, 12, 1, 0, 0, 0, time.UTC)
	in.Status = manifest.Status{State: manifest.StateStale, Good: manifest.Attempt{BuiltAt: built},
		Reason: "The inputs changed after it was built."}
	r = FactsReport(in)
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

func TestSignalTableKeepsAllFactsInFourColumns(t *testing.T) {
	t.Parallel()
	r := SignalFilesReport(SignalsView{Feeds: []FeedView{{Feed: "stats_player_week", Source: "nflverse",
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
	t.Parallel()
	for _, note := range []string{
		"Player crosswalk reloaded and relinked before signals; its Loaded time records that relink.",
		"Player crosswalk relink failed; its last Loaded time is kept: held failure",
	} {
		got := SignalsSummary(SignalsReport{CrosswalkNote: note})
		if !strings.HasSuffix(got, note) {
			t.Errorf("summary = %q, want %q", got, note)
		}
	}
}

func TestPhotoHealthRowSaysWhatIsHeldAndWhenOff(t *testing.T) {
	t.Parallel()
	at := time.Date(2026, 10, 11, 21, 0, 0, 0, time.UTC)
	got := photoCells(PhotoHealth{Held: 412, Bytes: 3 << 20, Fetched: 40, Missing: 2, LastFail: at,
		LastError: "headshot: a.espncdn.com answered HTTP 503"}, time.UTC)
	want := []string{"Player photos (ESPN, then Sleeper)\nPersonal use only",
		"412 photos held (3.0 MB)\nSince launch: 40 fetched, 2 with no photo",
		"Oct 11, 2026, 9:00 PM: headshot: a.espncdn.com answered HTTP 503"}
	if strings.Join(got, " | ") != strings.Join(want, " | ") {
		t.Fatalf("photo row\n got %q\nwant %q", got, want)
	}
	off := photoCells(PhotoHealth{Off: true, Held: 1}, time.UTC)
	if !strings.HasSuffix(off[0], "Off: no photo is requested") || !strings.HasPrefix(off[1], "1 photo held") ||
		off[2] != "None" {
		t.Fatalf("photos Off: %q", off)
	}
}

func TestSourceStandingAlarmsOnlyForTheBoard(t *testing.T) {
	t.Parallel()
	reg, err := measures.Embedded()
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 10, 12, 9, 0, 0, 0, time.UTC)
	good := func(src string) history.SourceHealth {
		return history.SourceHealth{Source: src, State: history.SourceActive, LastSuccess: now.Add(-time.Hour)}
	}
	base := func() HealthInput {
		return HealthInput{Reg: reg, Now: now, Zone: time.UTC, Sources: []history.SourceHealth{
			good("mfl"), good("nflverse"), good("cfbd"), {Source: "madden", State: history.SourceRetired},
			{Source: LeagueArchiveSource, State: history.SourceActive}, // never read: not a board source
		}}
	}
	for name, test := range map[string]struct {
		change func(*HealthInput)
		state  string
	}{
		"all flowing, archive never read": {func(*HealthInput) {}, "live"},
		"board source never read":         {func(in *HealthInput) { in.Sources[1].LastSuccess = time.Time{} }, "fail"},
		"board source lost":               {func(in *HealthInput) { in.Sources[2].State = history.SourceLost }, "fail"},
		"failure not recovered": {func(in *HealthInput) {
			in.Sources[2].LastFailure, in.Sources[2].LastError = now, "timeout"
		}, "stale"},
		"failure recovered": {func(in *HealthInput) {
			in.Sources[2].LastFailure, in.Sources[2].LastError = now.Add(-2*time.Hour), "timeout"
		}, "live"},
		"MFL export failing": {func(in *HealthInput) {
			in.MFL = []history.URLHealth{{URL: "https://www47.myfantasyleague.com/2026/export?TYPE=rosters&L=14432",
				LastFailure: now, LastError: "HTTP 503"}}
		}, "stale"},
	} {
		in := base()
		test.change(&in)
		if state, label := SourceStanding(in); state != test.state {
			t.Errorf("%s: %s (%s), want %s", name, state, label, test.state)
		}
	}
	in := base()
	in.Sources[2].LastSuccess = time.Time{}
	if _, label := SourceStanding(in); label != "No data from CollegeFootballData API · board will run reduced" {
		t.Errorf("the alarm names the missing source: %q", label)
	}
	// A key this install never set up is named, not the alarm; once read, the source is held to its window.
	in.Unset = map[string]string{"cfbd": "needs a key"}
	if state, label := SourceStanding(in); state != "live" || label != "Sources flowing · CollegeFootballData API not set up" {
		t.Errorf("an unset source: %s %q", state, label)
	}
	if age := sourceAge(healthRows(in)[2], now); age != "Not set up · needs a key" {
		t.Errorf("an unset source's age: %q", age)
	}
	in.Sources[2].LastSuccess = now.Add(-40 * 24 * time.Hour)
	in.Sources[2].State = history.SourceLost
	if state, _ := SourceStanding(in); state != "fail" {
		t.Errorf("a once-read source that is lost still alarms: %s", state)
	}
}

func TestCheckDifferencesSharingAnExplanationAreListedOnce(t *testing.T) {
	t.Parallel()
	why := "MFL counted playoff games in the 2013 standings."
	c := facts.CheckResult{Name: "Win-loss records match MFL standings", Items: []facts.CheckItem{
		{Words: "2013 Buffalo Bills: standings 11-3-0, games 11-2-0", Reason: why, Class: facts.ClassExplained},
		{Words: "2014 Houston Texans: standings 9-4-0, games 9-3-0", Class: facts.ClassReview,
			Reason: "No game accounts for the difference."},
		{Words: "2013 Miami Dolphins: standings 8-6-0, games 8-5-0", Reason: why, Class: facts.ClassExplained},
	}}
	rows := checkItemRows(c)
	if len(rows) != 2 {
		t.Fatalf("rows = %d, want one per explanation: %+v", len(rows), rows)
	}
	if got := rows[0].Cells[0]; got != "2 differences\n2013 Buffalo Bills: standings 11-3-0, games 11-2-0\n"+
		"2013 Miami Dolphins: standings 8-6-0, games 8-5-0" || rows[0].Cells[1] != why || rows[0].Cells[2] != "Explained" {
		t.Fatal(rows[0].Cells)
	}
	if rows[1].Cells[0] != "2014 Houston Texans: standings 9-4-0, games 9-3-0" || rows[1].Cells[2] != "For review" ||
		rows[1].Group != c.Name {
		t.Fatal(rows[1])
	}
}

func cell(r report.Report, key string) string {
	for _, row := range r.Rows {
		if row.Cells[0] == key {
			return row.Cells[1]
		}
	}
	return ""
}

func TestPositionsWithoutLookupDoesNotReadRoster(t *testing.T) {
	got := Positions(normalize.Lookup{}, false, func() map[string]string { t.Fatal("read absent lookup"); return nil })
	if got == nil || len(got) != 0 {
		t.Fatal(got)
	}
}

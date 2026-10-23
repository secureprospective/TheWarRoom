package sources_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/report"

	sourcesapp "github.com/secureprospective/TheWarRoom/internal/app/sources"

	"github.com/secureprospective/TheWarRoom/internal/manifest"
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
	t.Parallel()
	a := testHost(t)

	dir := archiveTree(t)
	if r := a.api().SetLeagueArchiveDir(dir); !r.OK {
		t.Fatal(r.Error)
	}
	first := a.api().TargetDataRefresh("archive")
	if !first.OK || !strings.Contains(first.Summary, "Looked at 8 files: 8 new, 0 already held. MFL error bodies, kept as errors: 1.") ||
		!strings.Contains(first.Summary, "1,482 files pinned, 1,480 not in the archive") {
		t.Fatalf("first import = %+v", first)
	}
	again := a.api().TargetDataRefresh("archive")
	if !again.OK || !strings.Contains(again.Summary, "8 files: 0 new, 8 already held") {
		t.Fatalf("second import = %+v, want nothing new", again)
	}
	ctx := context.Background()
	fetches, err := a.hist.SourceFetches(ctx, sourcesapp.LeagueArchiveSource)
	if err != nil || len(fetches) != 8 {
		t.Fatalf("archive fetches = %d, %v; want 8 after two runs", len(fetches), err)
	}
	m, published, err := sourcesapp.ServiceFor(a).PublishLabManifest(ctx)
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
	if _, _, err := m.ResolveWithGaps(ctx, a.hist, nil); err == nil || !strings.Contains(err.Error(), "2013/assets.json") {
		t.Errorf("resolve err = %v, want the missing frozen files named", err)
	}
	ds, err := a.api().TargetDataSources()
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
	t.Parallel()
	a := testHost(t)

	dir := archiveTree(t)
	if err := os.WriteFile(filepath.Join(dir, "2014/rules.json"), []byte(`{"rules":`), 0o600); err != nil {
		t.Fatal(err)
	}
	if r := a.api().SetLeagueArchiveDir(dir); !r.OK {
		t.Fatal(r.Error)
	}
	r := a.api().TargetDataRefresh("archive")
	if r.OK || !strings.Contains(r.Error, "An archive file is incomplete or is not a valid MFL download.") {
		t.Fatalf("import = %+v, want a plain explanation and recovery step", r)
	}
	ctx := context.Background()
	if fetches, _ := a.hist.SourceFetches(ctx, sourcesapp.LeagueArchiveSource); len(fetches) != 0 {
		t.Errorf("%d fetches recorded from a malformed archive", len(fetches))
	}
	health, err := a.hist.SourceHealth(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, h := range health {
		if h.Source == sourcesapp.LeagueArchiveSource && h.LastError != r.Error {
			t.Errorf("archive health = %+v, want the failure recorded", h)
		}
	}
	for _, bad := range []string{"relative/raw", t.TempDir()} {
		if r := a.api().SetLeagueArchiveDir(bad); r.OK {
			t.Errorf("archive folder %q accepted", bad)
		}
	}
	if r := a.api().TargetDataRefresh("everything"); r.OK || !strings.Contains(r.Error, `unknown data scope "everything"`) {
		t.Errorf("unknown scope = %+v", r)
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

func TestRefreshSummariesSayWhatHappened(t *testing.T) {
	t.Parallel()
	r := sourcesapp.SignalsReport{Loads: []sourcesapp.SignalLoad{{Status: sourcesapp.LoadDone, Added: 1200}, {Status: sourcesapp.LoadCurrent}, {Status: sourcesapp.LoadFailed},
		{Status: sourcesapp.LoadSkipped}, {Status: sourcesapp.LoadDone, Added: 34}}}
	if got := sourcesapp.SignalsSummary(r); got != "Looked at 5 files: 2 loaded, 1 already current, 1 skipped, 1 failed; 1,234 new values." {
		t.Errorf("signals summary = %q", got)
	}
	if got := sourcesapp.LeagueRefreshSummary(2026, true, false); got !=
		"Season 2026: rosters and salary adjustments updated; rules unchanged." {
		t.Errorf("league summary = %q", got)
	}
	for n, want := range map[int]string{0: "0", 999: "999", 1000: "1,000", 1234567: "1,234,567", -4200: "-4,200"} {
		if got := sourcesapp.Thousands(n); got != want {
			t.Errorf("sourcesapp.Thousands(%d) = %q, want %q", n, got, want)
		}
	}
	for d, want := range map[time.Duration]string{59 * time.Minute: "Under an hour", 47 * time.Hour: "47 hours", 49 * time.Hour: "2 days"} {
		if got := sourcesapp.PlainAge(d); got != want {
			t.Errorf("sourcesapp.PlainAge(%s) = %q, want %q", d, got, want)
		}
	}
}

func TestCheckedFactsNamesOnlyExactApprovedGaps(t *testing.T) {
	t.Parallel()
	a := testHost(t)

	if r := a.api().SetLeagueArchiveDir(archiveTree(t)); !r.OK {
		t.Fatal(r.Error)
	}
	if r := a.api().TargetDataRefresh("archive"); !r.OK {
		t.Fatal(r.Error)
	}
	in, err := sourcesapp.ServiceFor(a).FactsInput(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	r := sourcesapp.FactsReport(in)
	for _, gap := range in.Gaps {
		if got := cell(r, "Named gap: "+gap.File); got != gap.Reason || !strings.Contains(got, "fed no fact") {
			t.Errorf("gap %s = %q", gap.File, got)
		}
	}
	for i := range in.Manifest.Entries {
		if in.Manifest.Entries[i].File == in.Gaps[0].File {
			in.Manifest.Entries[i].SHA256 = strings.Repeat("0", 64)
		}
	}
	if got := cell(sourcesapp.FactsReport(in), "Named gap: "+in.Gaps[0].File); got != "" {
		t.Errorf("changed pin incorrectly received exception: %q", got)
	}
}

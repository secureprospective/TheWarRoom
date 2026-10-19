package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/lab"
	"github.com/secureprospective/TheWarRoom/internal/manifest"
)

// publishTinyManifest archives one season's league, standings and weekly results and publishes them, with the
// curated readings, as the Lab's manifest.
func publishTinyManifest(t *testing.T, a *App, bills string, readings lab.Curated) {
	t.Helper()
	ctx := context.Background()
	dir := t.TempDir()
	files := map[string]string{
		"2020/league.json": `{"league":{"id":"14432","baseURL":"https://www47.myfantasyleague.com",` +
			`"franchises":{"franchise":[{"id":"0001","name":"Buffalo Bills"},{"id":"0002","name":"Miami Dolphins"}]}}}`,
		"2020/weeklyResults_WYTD.json": weekResults(bills),
		"2020/leagueStandings.json": `{"leagueStandings":{"franchise":[{"id":"0001","h2hw":"1","h2hl":"0","h2ht":"0","pf":"` +
			bills + `"},{"id":"0002","h2hw":"0","h2hl":"1","h2ht":"0","pf":"50"}]}}`,
	}
	var pins []manifest.Pin
	for rel, body := range files {
		path := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256([]byte(body))
		pins = append(pins, manifest.Pin{File: rel, SHA256: hex.EncodeToString(sum[:])})
	}
	if _, err := a.importArchiveFrom(ctx, dir); err != nil {
		t.Fatal(err)
	}
	fetches, err := a.history.SourceFetches(ctx, leagueArchiveSource)
	if err != nil {
		t.Fatal(err)
	}
	held := make([]manifest.Fetch, len(fetches))
	for i, f := range fetches {
		held[i] = manifest.Fetch{ID: f.ID, URL: f.URL, SHA256: f.SHA256, FetchedAt: f.FetchedAt}
	}
	m, err := manifest.Build(labFrozenInputs, pins, held, 2020, time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if m, err = m.WithCurated(manifest.Pin{File: readings.File, SHA256: readings.SHA256}); err != nil {
		t.Fatal(err)
	}
	enc, err := m.Encode()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := a.history.PublishManifest(ctx, historyManifest(m, enc)); err != nil {
		t.Fatal(err)
	}
}

func weekResults(bills string) string {
	return `{"allWeeklyResults":{"weeklyResults":{"week":"1","matchup":{"franchise":[
		{"id":"0001","score":"` + bills + `","result":"W","starters":"10","player":{"id":"10","status":"starter","score":"100"}},
		{"id":"0002","score":"50","result":"L","starters":"20","player":{"id":"20","status":"starter","score":"50"}}]}}}}`
}

// TestCheckedFactsCompileFailKeepAndCache: an unexplained difference fails the compile and is shown
// in words; the same inputs are not compiled twice; fixed inputs compile, publish and pass; a later
// failure keeps the good snapshot, marked stale.
func TestCheckedFactsCompileFailKeepAndCache(t *testing.T) {
	t.Parallel()
	a := targetTestApp(t)
	a.dataDir = t.TempDir()
	ctx := context.Background()
	// No trade notes in a one-week archive: the curated readings are the header alone.
	readings := lab.Curated{File: "note_readings.csv",
		Body: []byte("txn_id,from_fid,to_fid,kind,pick_year,pick_round,pick_slot,pick_orig,cap_m,cap_seasons,player,clarity,reading\n")}
	sum := sha256.Sum256(readings.Body)
	readings.SHA256 = hex.EncodeToString(sum[:])
	compile := func() (string, error) { return a.compileFactsWith(ctx, readings) }
	publishTinyManifest(t, a, "117", readings)
	msg, err := compile()
	if err != nil || !strings.Contains(msg, "did not compile") || !strings.Contains(msg, "Buffalo Bills: team score 117.00") {
		t.Fatalf("unexplained difference: %q, %v", msg, err)
	}
	if msg, _ := compile(); !strings.Contains(msg, "already compiled") {
		t.Fatalf("the same inputs compiled twice: %q", msg)
	}
	ds, err := a.TargetDataSources()
	if err != nil {
		t.Fatal(err)
	}
	if ds.Checks.Status != "failed" || !strings.Contains(checkResult(ds.Checks, "Team scores equal their starters"), "Failed: 2 compared") {
		t.Fatalf("checks = %s %q", ds.Checks.Status, checkResult(ds.Checks, "Team scores equal their starters"))
	}
	publishTinyManifest(t, a, "100", readings)
	if msg, err := compile(); err != nil || msg != "The checked facts compiled and every check passed." {
		t.Fatalf("good compile: %q, %v", msg, err)
	}
	ds, _ = a.TargetDataSources()
	if got := cell(ds.Facts, "Compiled snapshot"); !strings.HasPrefix(got, "Current, built") || ds.Checks.Status != "ok" ||
		!strings.HasPrefix(checkResult(ds.Checks, "Team scores equal their starters"), "Passed: 2 compared, 2 agree") {
		t.Fatalf("snapshot %q, checks %s", got, ds.Checks.Status)
	}
	publishTinyManifest(t, a, "90", readings)
	if _, err := compile(); err != nil {
		t.Fatal(err)
	}
	ds, _ = a.TargetDataSources()
	if got := cell(ds.Facts, "Compiled snapshot"); !strings.HasPrefix(got, "Stale, built") || !strings.Contains(got, "latest build failed") {
		t.Fatalf("a failed build must keep the good snapshot, stale: %q", got)
	}
	if got := cell(ds.Facts, "Curated inputs"); !strings.HasPrefix(got, "note_readings.csv (") {
		t.Fatalf("curated inputs = %q", got)
	}
}

// checkResult is a check's Result cell in the checks report.
func checkResult(r Report, name string) string {
	for _, row := range r.Rows {
		if row.Group == "" && len(row.Cells) == 3 && row.Cells[0] == name {
			return row.Cells[2]
		}
	}
	return ""
}

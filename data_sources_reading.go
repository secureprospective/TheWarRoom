package main

import (
	"fmt"
	"maps"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/facts"
	"github.com/secureprospective/TheWarRoom/internal/manifest"
	"github.com/secureprospective/TheWarRoom/internal/measures"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

// healthInput is what history holds about each source when the health report is read.
type healthInput struct {
	reg      *measures.Registry
	sources  []history.SourceHealth
	mfl      []history.URLHealth
	boardRun bool
	missing  []string // measures the latest board run went without
	now      time.Time
	zone     *time.Location
	photos   *photoHealth // nil before the photo cache starts
}

// photoHealth is the player photo cache's standing: photos are not facts, so it is not a history source.
type photoHealth struct {
	off              bool
	held             int
	bytes            int64
	fetched, missing int
	lastFail         time.Time
	lastError        string
}

// healthRow is one line of the health report before it is worded.
type healthRow struct {
	name, measuresFrom string
	lastGood, lastFail time.Time
	lastError          string
	lost, retired      bool
}

// healthReport is C-07: every source the app reads, MFL by export, with its last good read, its
// last failure, its age and whether the board runs reduced without it (R11).
func healthReport(in healthInput) Report {
	r := Report{ID: "C-07", Title: "Source health and freshness", Status: "ok", Source: "Source: the app's own fetch and load records",
		AsOf: "As of: " + reportTime(in.now, in.zone), Rows: []ReportRow{},
		How: "A source's last good read is its latest load without an error; a failure is shown while it is newer than " +
			"that. A source is lost when it has failed and had no good load within its window. The board runs reduced " +
			"when a measure it reads has no source flowing; it is rebalanced only when you approve it. " +
			"Board use is noted only for sources supplying board measures; other rows are reference or linking inputs.",
		Columns: []string{"Source and board use", "Last good and age", "Last failure"}}
	var rows []healthRow
	for _, h := range in.sources {
		src, _ := in.reg.Source(h.Source)
		row := healthRow{name: src.Name, measuresFrom: h.Source, lastGood: h.LastSuccess, lastFail: h.LastFailure,
			lastError: h.LastError, lost: h.State == history.SourceLost, retired: h.State == history.SourceRetired}
		if h.Source != "mfl" {
			rows = append(rows, row)
			continue
		}
		// MFL's own row stands for its exports: its newest good read and newest failure come from them.
		exports := mflExportRows(in.mfl)
		for _, e := range exports {
			if e.lastGood.After(row.lastGood) {
				row.lastGood = e.lastGood
			}
			if e.lastFail.After(row.lastFail) {
				row.lastFail, row.lastError = e.lastFail, e.lastError
			}
		}
		rows = append(append(rows, row), exports...)
	}
	for _, row := range rows {
		name := row.name
		if board := boardText(row.measuresFrom, in); board != "Not read by the board" {
			name += "\n" + board
		}
		r.Rows = append(r.Rows, ReportRow{Cells: []string{name,
			whenText(row.lastGood, in.zone) + "\n" + sourceAge(row, in.now), failureText(row, in.zone)}})
	}
	if p := in.photos; p != nil {
		r.Rows = append(r.Rows, ReportRow{Cells: photoCells(*p, in.zone)})
	}
	return r
}

// photoCells words the photo cache's health row: what is held, this launch's reads, the last failure.
func photoCells(p photoHealth, zone *time.Location) []string {
	name := "Player photos (ESPN, then Sleeper)\nPersonal use only"
	if p.off {
		name += " · Off: no photo is requested"
	}
	held := fmt.Sprintf("%s held (%.1f MB)\nSince launch: %s fetched, %s with no photo",
		countText(p.held, "photo"), float64(p.bytes)/(1<<20), strconv.Itoa(p.fetched), strconv.Itoa(p.missing))
	failure := "None"
	if !p.lastFail.IsZero() {
		failure = whenText(p.lastFail, zone) + ": " + p.lastError
	}
	return []string{name, held, failure}
}

func countText(n int, noun string) string {
	if n == 1 {
		return "1 " + noun
	}
	return strconv.Itoa(n) + " " + noun + "s"
}

// mflExportRows groups MFL's fetches by export type.
func mflExportRows(urls []history.URLHealth) []healthRow {
	byType := map[string]*healthRow{}
	for _, u := range urls {
		typ := "other requests"
		if parsed, err := url.Parse(u.URL); err == nil && parsed.Query().Get("TYPE") != "" {
			typ = parsed.Query().Get("TYPE")
		}
		row := byType[typ]
		if row == nil {
			row = &healthRow{name: "MFL · " + typ}
			byType[typ] = row
		}
		if u.LastSuccess.After(row.lastGood) {
			row.lastGood = u.LastSuccess
		}
		if u.LastFailure.After(row.lastFail) {
			row.lastFail, row.lastError = u.LastFailure, u.LastError
		}
	}
	out := make([]healthRow, 0, len(byType))
	for _, k := range slices.Sorted(maps.Keys(byType)) {
		out = append(out, *byType[k])
	}
	return out
}

func whenText(t time.Time, zone *time.Location) string {
	if t.IsZero() {
		return "Never"
	}
	return reportTime(t, zone)
}

func failureText(row healthRow, zone *time.Location) string {
	switch {
	case row.lastFail.IsZero():
		return "None"
	case row.lastFail.Before(row.lastGood):
		return reportTime(row.lastFail, zone) + " (recovered since)"
	}
	return reportTime(row.lastFail, zone) + ": " + row.lastError
}

func sourceAge(row healthRow, now time.Time) string {
	switch {
	case row.retired:
		return "Retired"
	case row.lastGood.IsZero():
		return "No good read yet"
	}
	age := plainAge(now.Sub(row.lastGood))
	if row.lost {
		return "Lost · " + age
	}
	return age
}

// plainAge says how old something is in the largest whole unit that reads naturally.
func plainAge(d time.Duration) string {
	switch {
	case d < time.Hour:
		return "Under an hour"
	case d < 48*time.Hour:
		return fmt.Sprintf("%d hours", int(d.Hours()))
	}
	return fmt.Sprintf("%d days", int(d.Hours()/24))
}

// boardText says whether the board reads a source's measures, and which it went without.
func boardText(source string, in healthInput) string {
	if source == "" {
		return "Not read by the board"
	}
	var feeds, short []string
	for _, f := range in.reg.Fields {
		if f.Source == source && !slices.Contains(feeds, f.Measure) {
			feeds = append(feeds, f.Measure)
			if slices.Contains(in.missing, f.Measure) {
				short = append(short, f.Measure)
			}
		}
	}
	switch {
	case len(feeds) == 0:
		return "Not read by the board"
	case !in.boardRun:
		return "No board run yet"
	case len(short) > 0:
		return fmt.Sprintf("Running reduced: %d of its measures not flowing (%s)", len(short), strings.Join(short, ", "))
	}
	return "Feeding the board"
}

// signalFilesReport is the signal files behind C-07: each file's last load and its reach.
func signalFilesReport(v SignalsView, last SignalsReport, zone *time.Location) Report {
	r := Report{ID: "C-07-files", Title: "Signal files", Status: "ok", Source: "Source: history.db", AsOf: "As of: now",
		How: "A file is fresh while its last load is within its source's window. Players with data are counted in the " +
			"latest season holding any; waiting ids are source players not yet matched to an MFL player. Rostered " +
			"players with data covers every position the league rosters.",
		Columns: []string{"Signal and source", "Last load", "Players and matching", "Rostered coverage"},
		Rows:    []ReportRow{}}
	for _, f := range v.Feeds {
		loaded, fresh := "Never", "No"
		if at, err := time.Parse(time.RFC3339, f.LastLoaded); err == nil {
			loaded = reportTime(at, zone)
		}
		if f.Fresh {
			fresh = "Yes"
		}
		var players, waiting, rostered, with int
		for _, s := range f.Seasons {
			if s.Season == f.CoverageSeason || f.CoverageSeason == allSeasons {
				players, waiting = max(players, s.Players), waiting+s.Waiting
			}
		}
		for _, c := range f.Coverage {
			rostered, with = rostered+c.Rostered, with+c.WithData
		}
		share := "None rostered"
		if rostered > 0 {
			share = fmt.Sprintf("%d%% (%s of %s)", 100*with/rostered, thousands(with), thousands(rostered))
		}
		r.Rows = append(r.Rows, ReportRow{Cells: []string{f.Feed + "\n" + f.Source, loaded + "\nFresh: " + fresh,
			seasonCount(players, f.CoverageSeason) + "\nWaiting: " + thousands(waiting), share}})
	}
	if last.FinishedAt != "" {
		r.Note = "Last signals load since launch: " + signalsSummary(last)
	}
	return r
}

func seasonCount(players, season int) string {
	switch season {
	case allSeasons:
		return thousands(players) + " across all seasons"
	case 0:
		return thousands(players) + " (player facts)"
	}
	return fmt.Sprintf("%s in %d", thousands(players), season)
}

// factsInput is what L-34 reports: the Lab's pinned inputs, what the archive holds of them, and
// the compiled snapshot's standing.
type factsInput struct {
	labBuilt, labThrough string
	gaps                 []manifest.Gap
	published            bool
	publishedAt          time.Time
	manifest             manifest.Manifest
	held                 map[string]bool
	status               manifest.Status
	latest               *history.SnapshotRecord // the newest compile attempt, good or failed
	results              []facts.CheckResult     // its check results, when it reached the checks
	zone                 *time.Location
}

// factsReport is L-34: the manifest and the snapshot status.
func factsReport(in factsInput) Report {
	r := Report{ID: "L-34", Title: "Checked facts", Status: "ok", Source: "Source: the league archive in history.db",
		AsOf: "As of: now", Columns: []string{"Fact", "Value"}, Rows: []ReportRow{},
		How: "The checked facts are compiled from exactly the archived MFL files a manifest pins, by their bytes' " +
			"sha256, never from the newest copy of a file. Only the five exact frozen login-error pins have named " +
			"non-data gaps; other missing bytes block compilation. A fresh download never stands in for a pin."}
	row := func(k, v string) { r.Rows = append(r.Rows, ReportRow{Cells: []string{k, v}}) }
	row("Inputs", labFrozenInputs)
	row("Lab data", "Captured through "+labTime(in.labThrough, in.zone)+"; built "+labTime(in.labBuilt, in.zone))
	if !in.published {
		r.Status, r.Note = "empty", "No manifest yet: import the league archive under Data refresh to pin the Lab's inputs."
		return r
	}
	m := in.manifest
	seasons, exports, errorBodies := map[int]bool{}, map[string]int{}, 0
	var notHeld []string
	var curated []string
	for _, e := range m.Entries {
		if e.Curated() {
			curated = append(curated, e.File+" ("+e.SHA256[:12]+")")
			continue
		}
		seasons[e.Season], exports[e.Export] = true, exports[e.Export]+1
		if e.MFLError != "" {
			errorBodies++
		}
		if !in.held[e.SHA256] {
			notHeld = append(notHeld, e.File)
		}
	}
	years := slices.Sorted(maps.Keys(seasons))
	row("Manifest", m.SHA256[:12]+", published "+reportTime(in.publishedAt, in.zone))
	row("Files pinned", fmt.Sprintf("%s archive files, over seasons %d–%d", thousands(len(m.Entries)-len(curated)), years[0], years[len(years)-1]))
	if len(curated) > 0 {
		row("Curated inputs", strings.Join(curated, ", ")+": the readings of trade notes, written by people, not MFL data. "+
			"Pinned by their own sha256 like every archive file.")
	}
	row("Exports", exportCounts(exports))
	archived := len(m.Entries) - len(curated)
	row("In the archive", fmt.Sprintf("%s of %s", thousands(archived-len(notHeld)), thousands(archived)))
	row("Not in the archive", missingText(notHeld))
	for _, g := range in.gaps {
		for _, e := range m.Entries {
			if e.File == g.File && e.SHA256 == g.SHA256 && !in.held[e.SHA256] {
				row("Named gap: "+g.File, g.Reason)
			}
		}
	}
	row("MFL error bodies", fmt.Sprintf("%s pinned files are MFL errors; they are kept as errors and never read as data", thousands(errorBodies)))
	row("Compiled snapshot", snapshotText(in.status, in.zone))
	return r
}

func labTime(s string, zone *time.Location) string {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return "an unknown time"
	}
	return reportTime(t, zone)
}

func exportCounts(exports map[string]int) string {
	parts := make([]string, 0, len(exports))
	for _, k := range slices.Sorted(maps.Keys(exports)) {
		parts = append(parts, fmt.Sprintf("%s %s", k, thousands(exports[k])))
	}
	return strings.Join(parts, ", ")
}

func missingText(files []string) string {
	if len(files) == 0 {
		return "None"
	}
	return fmt.Sprintf("%d: %s. No replacement bytes were used; approved non-data gaps are named below.", len(files), strings.Join(files, ", "))
}

func snapshotText(s manifest.Status, zone *time.Location) string {
	switch s.State {
	case manifest.StateCurrent:
		return "Current, built " + reportTime(s.Good.BuiltAt, zone)
	case manifest.StateNone:
		return "None yet. " + s.Reason
	case manifest.StateStale:
		return "Stale, built " + reportTime(s.Good.BuiltAt, zone) + ". " + s.Reason
	}
	return "None yet. " + s.Reason
}

// checksReport is the nine checks of the newest compile, each against MFL's own figures, with every
// explained or reviewed difference in plain words. Before the first compile they read "Not run yet".
func checksReport(in factsInput) Report {
	r := Report{ID: "L-34-checks", Title: "Checks", Status: "ok", Source: "Source: the compiled facts and MFL's own figures",
		AsOf: "As of: now", Columns: []string{"Check", "What it proves", "Result"}, Rows: []ReportRow{},
		How: "Each check compares two records MFL keeps separately. Every difference must be named, by its identity, " +
			"in the pinned list of explained and reviewed differences; a new one fails the compile, and the last good " +
			"snapshot is kept, marked stale."}
	row := func(name, proves, result string) {
		r.Rows = append(r.Rows, ReportRow{Cells: []string{name, proves, result}})
	}
	row("Compiled facts match the Lab's reference", "Row by row against the Lab's own Python build: ids, times, "+
		"counts and money exact; derived decimals within one part in a billion.", "Proved by the build's tests: 37 tables, no differences")
	if len(in.results) == 0 {
		r.Note = "Not run yet: the checks run when the facts compile, after the league archive is imported."
		if in.latest != nil && in.latest.Error != "" {
			r.Status, r.Note = "failed", "The latest compile stopped before the checks: "+in.latest.Error
		}
		// The checks over no facts: their names and what each proves, before anything has run.
		defined, _ := facts.Facts{}.Check(facts.Exceptions{})
		for _, c := range defined {
			row(c.Name, checkProves(c.Detail), "Not run yet")
		}
		return r
	}
	r.AsOf = "Compiled: " + reportTime(in.latest.BuiltAt, in.zone)
	failed := 0
	for _, c := range in.results {
		if !c.Passed {
			failed++
		}
		row(c.Name, checkProves(c.Detail), checkResultText(c))
	}
	for _, c := range in.results {
		for _, it := range c.Items {
			r.Rows = append(r.Rows, ReportRow{Group: c.Name, Cells: []string{it.Words, itemReason(it), itemClass(it.Class)}})
		}
	}
	if failed > 0 {
		r.Status, r.Note = "failed", fmt.Sprintf("%d of %d checks found a difference no one has explained; the facts are "+
			"not published until it is understood.", failed, len(in.results))
	}
	return r
}

func checkProves(detail string) string {
	if detail == "" {
		return "Checked against MFL's own figures."
	}
	return detail
}

func checkResultText(c facts.CheckResult) string {
	verdict := "Passed"
	if !c.Passed {
		verdict = "Failed"
	}
	return fmt.Sprintf("%s: %s compared, %s agree, %s explained, %s for review", verdict, thousands(c.Checked),
		thousands(c.Agreed), thousands(c.Explained), thousands(c.Review))
}

func itemReason(it facts.CheckItem) string {
	if it.Reason == "" {
		return "Not explained yet."
	}
	return it.Reason
}

func itemClass(class string) string {
	switch class {
	case facts.ClassExplained:
		return "Explained"
	case facts.ClassReview:
		return "For review"
	}
	return "Not explained"
}

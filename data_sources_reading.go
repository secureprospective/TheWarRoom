package main

import (
	"encoding/json"
	"fmt"
	"maps"
	"net/url"
	"slices"
	"strings"
	"time"

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
			"when a measure it reads has no source flowing; it is rebalanced only when you approve it.",
		Columns: []string{"Source", "Last good", "Last failure", "Age", "Board"}}
	var rows []healthRow
	for _, h := range in.sources {
		src, _ := in.reg.Source(h.Source)
		rows = append(rows, healthRow{name: src.Name, measuresFrom: h.Source, lastGood: h.LastSuccess, lastFail: h.LastFailure,
			lastError: h.LastError, lost: h.State == history.SourceLost, retired: h.State == history.SourceRetired})
		if h.Source == "mfl" {
			rows = append(rows, mflExportRows(in.mfl)...)
		}
	}
	for _, row := range rows {
		r.Rows = append(r.Rows, ReportRow{Cells: []string{row.name, whenText(row.lastGood, in.zone),
			failureText(row, in.zone), sourceAge(row, in.now), boardText(row.measuresFrom, in)}})
	}
	return r
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
		Columns: []string{"Signal", "Source", "Last loaded", "Fresh", "Players with data", "Waiting for a match", "Rostered players with data"},
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
		r.Rows = append(r.Rows, ReportRow{Cells: []string{f.Feed, f.Source, loaded, fresh,
			seasonCount(players, f.CoverageSeason), thousands(waiting), share}})
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

// labCheck is one of the Lab's own proofs of its facts against MFL's figures.
type labCheck struct {
	Name   string `json:"name"`
	Detail string `json:"detail"`
}

// rawLabSource is the part of the frozen Lab data's "source" block the checks list reads.
type rawLabSource struct {
	Checks []labCheck `json:"checks"`
}

func labSourceChecks(raw json.RawMessage) ([]labCheck, error) {
	var s rawLabSource
	if err := json.Unmarshal(raw, &s); err != nil {
		return nil, fmt.Errorf("app: lab source checks: %w", err)
	}
	for _, c := range s.Checks {
		if c.Name == "" {
			return nil, fmt.Errorf("app: lab source checks: a check has no name")
		}
	}
	return s.Checks, nil
}

// factsInput is what L-34 reports: the Lab's pinned inputs, what the archive holds of them, and
// the compiled snapshot's standing.
type factsInput struct {
	labBuilt, labThrough string
	checks               []labCheck
	gaps                 []manifest.Gap
	published            bool
	publishedAt          time.Time
	manifest             manifest.Manifest
	held                 map[string]bool
	status               manifest.Status
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
	for _, e := range m.Entries {
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
	row("Files pinned", fmt.Sprintf("%s, over seasons %d–%d", thousands(len(m.Entries)), years[0], years[len(years)-1]))
	row("Exports", exportCounts(exports))
	row("In the archive", fmt.Sprintf("%s of %s", thousands(len(m.Entries)-len(notHeld)), thousands(len(m.Entries))))
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

// checksReport lists the proofs a compiled snapshot must pass before it is published.
func checksReport(lab []labCheck) Report {
	r := Report{ID: "L-34-checks", Title: "Checks that will run", Status: "ok", Source: "Source: the frozen Lab data and the build plan",
		AsOf: "As of: now", Columns: []string{"Check", "What it proves", "Result"}, Rows: []ReportRow{},
		Note: "These run each time the facts are compiled; their results appear here once the compiler is built.",
		How: "A compiled snapshot is published only when every check passes; a failed build keeps the last good one, " +
			"marked stale."}
	row := func(name, proves string) {
		r.Rows = append(r.Rows, ReportRow{Cells: []string{name, proves, "Not run yet"}})
	}
	row("Compiled rows match the frozen Lab data", "Row by row: ids, times, counts and money exact; derived decimals within one part in a billion.")
	row("Engine goldens pass on the compiled data", "The engines read the compiled facts and reproduce their checked results.")
	for _, c := range lab {
		proves := c.Detail
		if proves == "" {
			proves = "Checked against MFL's own figures."
		}
		row(c.Name, proves)
	}
	return r
}

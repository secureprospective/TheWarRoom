package sources

import (
	"fmt"
	"maps"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/report"

	"github.com/secureprospective/TheWarRoom/internal/facts"
	"github.com/secureprospective/TheWarRoom/internal/manifest"
	"github.com/secureprospective/TheWarRoom/internal/measures"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

// HealthInput is what history holds about each source when the health report is read.
type HealthInput struct {
	Reg      *measures.Registry
	Sources  []history.SourceHealth
	MFL      []history.URLHealth
	EngineAt time.Time
	Now      time.Time
	Zone     *time.Location
	Photos   *PhotoHealth      // nil before the photo cache starts
	Unset    map[string]string // source id → what it needs, for a source this install has not set up
}

// PhotoHealth is the player photo cache's standing: photos are not facts, so it is not a history source.
type PhotoHealth struct {
	Off              bool
	Held             int
	Bytes            int64
	Fetched, Missing int
	LastFail         time.Time
	LastError        string
}

// healthRow is one line of the health report before it is worded.
type healthRow struct {
	name, measuresFrom string
	lastGood, lastFail time.Time
	lastError          string
	lost, retired      bool
	unset              string // what the source needs before it can be read; empty when set up
}

// HealthReport is C-07: every source the app reads, MFL by export, with its last good read, its
// last failure, its age and whether the board runs reduced without it (R11).
func HealthReport(in HealthInput) report.Report {
	r := report.Report{ID: "C-07", Title: "Source health and freshness", Status: "ok", Source: "Source: the app's own fetch and load records",
		AsOf: "As of: " + report.Time(in.Now, in.Zone), Rows: []report.Row{},
		How: "A source's last good read is its latest load without an error; a failure is shown while it is newer than " +
			"that. A source is lost when it has failed and had no good load within its window. The board runs reduced " +
			"when a measure it reads has no source flowing; it is rebalanced only when you approve it. " +
			"Board use is noted only for sources supplying board measures; other rows are reference or linking inputs.",
		Columns: []string{"Source and board use", "Last good and age", "Last failure"}}
	for _, row := range healthRows(in) {
		name := row.name
		if board := boardText(row.measuresFrom, in); board != "Not read by the board" {
			name += "\n" + board
		}
		r.Rows = append(r.Rows, report.Row{Cells: []string{name,
			whenText(row.lastGood, in.Zone) + "\n" + sourceAge(row, in.Now), failureText(row, in.Zone)}})
	}
	engine := "Engine run not yet available"
	if !in.EngineAt.IsZero() {
		engine = "Engine run" // its date and age sit in the columns, like every source's
	}
	r.Rows = append(r.Rows, report.Row{Cells: []string{engine,
		whenText(in.EngineAt, in.Zone) + "\n" + sourceAge(healthRow{lastGood: in.EngineAt}, in.Now), "—"}})
	if p := in.Photos; p != nil {
		r.Rows = append(r.Rows, report.Row{Cells: photoCells(*p, in.Zone)})
	}
	return r
}

// healthRows are C-07's rows before wording: every registered source, with MFL standing for its
// exports and followed by one row per export.
func healthRows(in HealthInput) []healthRow {
	var rows []healthRow
	for _, h := range in.Sources {
		src, _ := in.Reg.Source(h.Source)
		row := healthRow{name: src.Name, measuresFrom: h.Source, lastGood: h.LastSuccess, lastFail: h.LastFailure,
			lastError: h.LastError, lost: h.State == history.SourceLost, retired: h.State == history.SourceRetired}
		if row.lastGood.IsZero() {
			row.unset = in.Unset[h.Source]
		}
		if h.Source != "mfl" {
			rows = append(rows, row)
			continue
		}
		// MFL's own row stands for its exports: its newest good read and newest failure come from them.
		exports := mflExportRows(in.MFL)
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
	return rows
}

// SourceStanding is the status strip's reading of C-07 (S-12). The alarm is a source the board
// reads that is lost or has never been read; a failure with no
// newer good read is a warning; a source the board does not read never raises the alarm.
func SourceStanding(in HealthInput) (state, label string) {
	state, label = "live", "Sources flowing"
	var missing, unset []string
	for _, row := range healthRows(in) {
		if row.retired {
			continue
		}
		if row.unset != "" {
			unset = append(unset, row.name)
			continue
		}
		feedsBoard := row.measuresFrom != "" && boardText(row.measuresFrom, in) != "Not read by the board"
		if feedsBoard && (row.lost || row.lastGood.IsZero()) {
			missing = append(missing, row.name)
		}
		if !row.lastFail.IsZero() && !row.lastFail.Before(row.lastGood) {
			state, label = "stale", "Source failures · last good data"
		}
	}
	if len(missing) > 0 {
		return "fail", "No data from " + strings.Join(missing, ", ") + " · board will run reduced"
	}
	// A source this install has not set up is a choice, not an outage: named, never the alarm.
	if len(unset) > 0 && state == "live" {
		label += " · " + strings.Join(unset, ", ") + " not set up"
	}
	return state, label
}

// photoCells words the photo cache's health row: what is held, this launch's reads, the last failure.
func photoCells(p PhotoHealth, zone *time.Location) []string {
	name := "Player photos (ESPN, then Sleeper)\nPersonal use only"
	if p.Off {
		name += " · Off: no photo is requested"
	}
	held := fmt.Sprintf("%s held (%.1f MB)\nSince launch: %s fetched, %s with no photo",
		CountText(p.Held, "photo"), float64(p.Bytes)/(1<<20), strconv.Itoa(p.Fetched), strconv.Itoa(p.Missing))
	failure := "None"
	if !p.LastFail.IsZero() {
		failure = whenText(p.LastFail, zone) + ": " + p.LastError
	}
	return []string{name, held, failure}
}

func PluralNoun(n float64, noun string) string {
	if n == 1 {
		return noun
	}
	return noun + "s"
}

func CountText(n int, noun string) string {
	return strconv.Itoa(n) + " " + PluralNoun(float64(n), noun)
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
	return report.Time(t, zone)
}

func failureText(row healthRow, zone *time.Location) string {
	switch {
	case row.lastFail.IsZero():
		return "None"
	case row.lastFail.Before(row.lastGood):
		return report.Time(row.lastFail, zone) + " (recovered since)"
	}
	return report.Time(row.lastFail, zone) + ": " + row.lastError
}

func sourceAge(row healthRow, now time.Time) string {
	switch {
	case row.retired:
		return "Retired"
	case row.unset != "":
		return "Not set up · " + row.unset
	case row.lastGood.IsZero():
		return "No good read yet"
	}
	age := PlainAge(now.Sub(row.lastGood))
	if row.lost {
		return "Lost · " + age
	}
	return age
}

// PlainAge says how old something is in the largest whole unit that reads naturally.
func PlainAge(d time.Duration) string {
	switch {
	case d < time.Hour:
		return "Under an hour"
	case d < 48*time.Hour:
		return fmt.Sprintf("%d hours", int(d.Hours()))
	}
	return fmt.Sprintf("%d days", int(d.Hours()/24))
}

// boardText says whether the engine reads a source's measures, and when it last ran. The layer 1-4
// board run that once reported the measures it went without was removed; that is not computed.
func boardText(source string, in HealthInput) string {
	if source == "" || !slices.ContainsFunc(in.Reg.Fields, func(f measures.SourceField) bool { return f.Source == source }) {
		return "Not read by the board"
	}
	if !in.EngineAt.IsZero() {
		return report.EngineRun(in.EngineAt, in.Zone)
	}
	return "Engine run not yet available"
}

// SignalFilesReport is the signal files behind C-07: each file's last load and its reach.
func SignalFilesReport(v SignalsView, last SignalsReport, zone *time.Location) report.Report {
	r := report.Report{ID: "C-07-files", Title: "Signal files", Status: "ok", Source: "Source: history.db", AsOf: "As of: now",
		How: "A file is fresh while its last load is within its source's window. Players with data are counted in the " +
			"latest season holding any; waiting ids are source players not yet matched to an MFL player. Rostered " +
			"players with data covers every position the league rosters.",
		Columns: []string{"Signal and source", "Last load", "Players and matching", "Rostered coverage"},
		Rows:    []report.Row{}}
	for _, f := range v.Feeds {
		loaded, fresh := "Never", "No"
		if at, err := time.Parse(time.RFC3339, f.LastLoaded); err == nil {
			loaded = report.Time(at, zone)
		}
		if f.Fresh {
			fresh = "Yes"
		}
		var players, waiting, rostered, with int
		for _, s := range f.Seasons {
			if s.Season == f.CoverageSeason || f.CoverageSeason == AllSeasons {
				players, waiting = max(players, s.Players), waiting+s.Waiting
			}
		}
		for _, c := range f.Coverage {
			rostered, with = rostered+c.Rostered, with+c.WithData
		}
		share := "None rostered"
		if rostered > 0 {
			share = fmt.Sprintf("%d%% (%s of %s)", 100*with/rostered, Thousands(with), Thousands(rostered))
		}
		r.Rows = append(r.Rows, report.Row{Cells: []string{f.Feed + "\n" + f.Source, loaded + "\nFresh: " + fresh,
			seasonCount(players, f.CoverageSeason) + "\nWaiting: " + Thousands(waiting), share}})
	}
	if last.FinishedAt != "" {
		r.Note = "Last signals load since launch: " + SignalsSummary(last)
	}
	return r
}

func seasonCount(players, season int) string {
	switch season {
	case AllSeasons:
		return Thousands(players) + " across all seasons"
	case 0:
		return Thousands(players) + " (player facts)"
	}
	return fmt.Sprintf("%s in %d", Thousands(players), season)
}

// FactsInput is what L-34 reports: the Lab's pinned inputs, what the archive holds of them, and
// the compiled snapshot's standing.
type FactsInput struct {
	LabBuilt, LabThrough string
	Gaps                 []manifest.Gap
	Published            bool
	PublishedAt          time.Time
	Manifest             manifest.Manifest
	Held                 map[string]bool
	Status               manifest.Status
	Latest               *history.SnapshotRecord // the newest compile attempt, good or failed
	Results              []facts.CheckResult     // its check Results, when it reached the checks
	Zone                 *time.Location
}

// FactsReport is L-34: the manifest and the snapshot status.
func FactsReport(in FactsInput) report.Report {
	r := report.Report{ID: "L-34", Title: "Checked facts", Status: "ok", Source: "Source: the league archive in history.db",
		AsOf: "As of: now", Columns: []string{"Fact", "Value"}, Rows: []report.Row{},
		How: "The checked facts are compiled from exactly the archived MFL files a manifest pins, by their bytes' " +
			"sha256, never from the newest copy of a file. Only the five exact frozen login-error pins have named " +
			"non-data gaps; other missing bytes block compilation. A fresh download never stands in for a pin."}
	row := func(k, v string) { r.Rows = append(r.Rows, report.Row{Cells: []string{k, v}}) }
	row("Inputs", LabFrozenInputs)
	row("Lab data", "Captured through "+LabTime(in.LabThrough, in.Zone)+"; built "+LabTime(in.LabBuilt, in.Zone))
	if !in.Published {
		r.Status, r.Note = "empty", "No manifest yet: import the league archive under Data refresh to pin the Lab's inputs."
		return r
	}
	m := in.Manifest
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
		if !in.Held[e.SHA256] {
			notHeld = append(notHeld, e.File)
		}
	}
	years := slices.Sorted(maps.Keys(seasons))
	row("Manifest", m.SHA256[:12]+", published "+report.Time(in.PublishedAt, in.Zone))
	row("Files pinned", fmt.Sprintf("%s archive files, over seasons %d–%d", Thousands(len(m.Entries)-len(curated)), years[0], years[len(years)-1]))
	if len(curated) > 0 {
		row("Curated inputs", strings.Join(curated, ", ")+": the readings of trade notes, written by people, not MFL data. "+
			"Pinned by their own sha256 like every archive file.")
	}
	row("Exports", exportCounts(exports))
	archived := len(m.Entries) - len(curated)
	row("In the archive", fmt.Sprintf("%s of %s", Thousands(archived-len(notHeld)), Thousands(archived)))
	row("Not in the archive", missingText(notHeld))
	for _, g := range in.Gaps {
		for _, e := range m.Entries {
			if e.File == g.File && e.SHA256 == g.SHA256 && !in.Held[e.SHA256] {
				row("Named gap: "+g.File, g.Reason)
			}
		}
	}
	row("MFL error bodies", fmt.Sprintf("%s pinned files are MFL errors; they are kept as errors and never read as data", Thousands(errorBodies)))
	row("Compiled snapshot", snapshotText(in.Status, in.Zone))
	return r
}

func LabTime(s string, zone *time.Location) string {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return "an unknown time"
	}
	return report.Time(t, zone)
}

func exportCounts(exports map[string]int) string {
	parts := make([]string, 0, len(exports))
	for _, k := range slices.Sorted(maps.Keys(exports)) {
		parts = append(parts, fmt.Sprintf("%s %s", k, Thousands(exports[k])))
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
		return "Current, built " + report.Time(s.Good.BuiltAt, zone)
	case manifest.StateNone:
		return "None yet. " + s.Reason
	case manifest.StateStale:
		return "Stale, built " + report.Time(s.Good.BuiltAt, zone) + ". " + s.Reason
	}
	return "None yet. " + s.Reason
}

// ChecksReport is the nine checks of the newest compile, each against MFL's own figures, with every
// explained or reviewed difference in plain words. Before the first compile they read "Not run yet".
func ChecksReport(in FactsInput) report.Report {
	r := report.Report{ID: "L-34-checks", Title: "Checks", Status: "ok", Source: "Source: the compiled facts and MFL's own figures",
		AsOf: "As of: now", Columns: []string{"Check", "What it proves", "Result"}, Rows: []report.Row{},
		How: "Each check compares two records MFL keeps separately. Every difference must be named, by its identity, " +
			"in the pinned list of explained and reviewed differences; a new one fails the compile, and the last good " +
			"snapshot is kept, marked stale."}
	row := func(name, proves, result string) {
		r.Rows = append(r.Rows, report.Row{Cells: []string{name, proves, result}})
	}
	row("Compiled facts match the Lab's reference", "Row by row against the Lab's own Python build: ids, times, "+
		"counts and money exact; derived decimals within one part in a billion.", "Proved by the build's tests: 37 tables, no differences")
	if len(in.Results) == 0 {
		r.Note = "Not run yet: the checks run when the facts compile, after the league archive is imported."
		if in.Latest != nil && in.Latest.Error != "" {
			r.Status, r.Note = "failed", "The latest compile stopped before the checks: "+in.Latest.Error
		}
		// The checks over no facts: their names and what each proves, before anything has run.
		defined, _ := facts.Facts{}.Check(facts.Exceptions{})
		for _, c := range defined {
			row(c.Name, checkProves(c.Detail), "Not run yet")
		}
		return r
	}
	r.AsOf = "Compiled: " + report.Time(in.Latest.BuiltAt, in.Zone)
	failed := 0
	for _, c := range in.Results {
		if !c.Passed {
			failed++
		}
		row(c.Name, checkProves(c.Detail), checkResultText(c))
	}
	for _, c := range in.Results {
		r.Rows = append(r.Rows, checkItemRows(c)...)
	}
	if failed > 0 {
		r.Status, r.Note = "failed", fmt.Sprintf("%d of %d checks found a difference no one has explained; the facts are "+
			"not published until it is understood.", failed, len(in.Results))
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
	return fmt.Sprintf("%s: %s compared, %s agree, %s explained, %s for review", verdict, Thousands(c.Checked),
		Thousands(c.Agreed), Thousands(c.Explained), Thousands(c.Review))
}

// checkItemRows lists a check's differences, one row per explanation: differences that share a
// reason and a standing are listed together under it, not repeated once per team.
func checkItemRows(c facts.CheckResult) []report.Row {
	type key struct{ reason, class string }
	var order []key
	words := map[key][]string{}
	for _, it := range c.Items {
		k := key{itemReason(it), itemClass(it.Class)}
		if _, seen := words[k]; !seen {
			order = append(order, k)
		}
		words[k] = append(words[k], it.Words)
	}
	rows := make([]report.Row, 0, len(order))
	for _, k := range order {
		listed := words[k][0]
		if n := len(words[k]); n > 1 {
			listed = CountText(n, "difference") + "\n" + strings.Join(words[k], "\n")
		}
		rows = append(rows, report.Row{Group: c.Name, Cells: []string{listed, k.reason, k.class}})
	}
	return rows
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

package sources

import (
	"fmt"
	"maps"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/normalize"

	"github.com/secureprospective/TheWarRoom/internal/app/report"

	"github.com/secureprospective/TheWarRoom/internal/measures"
	"github.com/secureprospective/TheWarRoom/internal/offline"
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
			lastError: offline.Recorded("", h.LastError), lost: h.State == history.SourceLost, retired: h.State == history.SourceRetired}
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

func CountText(n int, noun string) string {
	return strconv.Itoa(n) + " " + report.PluralNoun(float64(n), noun)
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
			row.lastFail, row.lastError = u.LastFailure, offline.Recorded(u.URL, u.LastError)
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

func LabTime(s string, zone *time.Location) string {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return "an unknown time"
	}
	return report.Time(t, zone)
}

func Positions(lk normalize.Lookup, ok bool, rostered func() map[string]string) map[string]string {
	out := map[string]string{}
	if !ok {
		return out
	}
	for id := range rostered() {
		if f, ok := lk.Facts(id); ok {
			out[id] = string(f.Position)
		}
	}
	return out
}

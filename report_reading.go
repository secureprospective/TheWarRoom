package main

import (
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/report"
)

type ReportRow struct {
	Sort   []float64    `json:"sort,omitempty"`
	Titles []string     `json:"titles,omitempty"`
	Cells  []string     `json:"cells"`
	Range  *MarketRange `json:"range"`
	// Mine marks the GM's own franchise; Bars are the row's chart values on the report's BarMax.
	Mine bool      `json:"mine,omitempty"`
	Bars []float64 `json:"bars,omitempty"`
	// Group heads a run of rows (a division in the standings); rows of one group stay together.
	Group string `json:"group,omitempty"`
}

type Report struct {
	Entities       []ReportEntity `json:"entities,omitempty"`
	SortAscending  string         `json:"sortAscending,omitempty"`
	SortDescending string         `json:"sortDescending,omitempty"`
	ID             string         `json:"id"`
	Title          string         `json:"title"`
	Note           string         `json:"note"`
	Status         string         `json:"status"`
	Source         string         `json:"source"`
	AsOf           string         `json:"asOf"`
	How            string         `json:"how"`
	Columns        []string       `json:"columns"`
	Rows           []ReportRow    `json:"rows"`
	Summary        []ReportRow    `json:"summary"`
	Link           *ReportLink    `json:"link"`
	BarMax         float64        `json:"barMax,omitempty"`
	BarLabels      []string       `json:"barLabels,omitempty"`
}

// ReportLink opens another report by its registry id, e.g. the full list behind a summary.
type ReportLink struct {
	Label    string `json:"label"`
	Endpoint string `json:"endpoint"`
}

type ReportBoard struct {
	Title    string   `json:"title"`
	Loading  string   `json:"loading"`
	Refresh  string   `json:"refresh"`
	HowLabel string   `json:"howLabel"`
	MFLLabel string   `json:"mflLabel"`
	MFLURL   string   `json:"mflUrl"`
	Reports  []Report `json:"reports"`
}

func reportShell[T any](id, title, how string, feed heldSeasonFeed[[]T], zone *time.Location, columns ...string) Report {
	r := Report{ID: id, Title: title, Source: "Source: MFL", How: how,
		Columns: columns, Rows: []ReportRow{}, Status: "ok", AsOf: "As of: not fetched"}
	switch {
	case feed.fetchedAt.IsZero():
		r.Status, r.Note = "failed", "Report data unavailable; refresh from MFL to fill this report."
	case len(feed.value) == 0:
		r.Status, r.Note = "empty", "MFL lists no records for this report."
	}
	if !feed.fetchedAt.IsZero() {
		r.AsOf = "As of: " + reportTime(feed.fetchedAt, zone)
	}
	if feed.readPending {
		r.Status, r.Note = "stale", pendingShellRead(feed.fetchedAt, zone)
	}
	if feed.refreshError != "" {
		if !feed.fetchedAt.IsZero() {
			r.Note += " Refresh failed; showing the last known data. Try refreshing from MFL again."
			r.Status = "stale"
		} else {
			r.Note += " Refresh failed. Try refreshing from MFL again."
		}
	}
	return r
}

func reportTime(at time.Time, zone *time.Location) string {
	if at.IsZero() {
		return "Time not supplied"
	}
	return at.In(zone).Format("Jan 2, 2006, 3:04 PM")
}

func pendingShellRead(at time.Time, zone *time.Location) string {
	return "Last read " + reportTime(at, zone) + "; today's read pending"
}

// wireBoard converts a board built by a feature service under internal/app to its wire twin.
// The two type sets are kept identical field for field; TestReportWireTwinsMatch proves it.
func wireBoard(b report.Board) ReportBoard {
	reports := make([]Report, len(b.Reports))
	for i, r := range b.Reports {
		reports[i] = wireReport(r)
	}
	return ReportBoard{Title: b.Title, Loading: b.Loading, Refresh: b.Refresh, HowLabel: b.HowLabel,
		MFLLabel: b.MFLLabel, MFLURL: b.MFLURL, Reports: wireSlice(b.Reports, reports)}
}

func wireReport(r report.Report) Report {
	out := Report{SortAscending: r.SortAscending, SortDescending: r.SortDescending, ID: r.ID, Title: r.Title,
		Note: r.Note, Status: r.Status, Source: r.Source, AsOf: r.AsOf, How: r.How, Columns: r.Columns,
		Rows: wireRows(r.Rows), Summary: wireRows(r.Summary), BarMax: r.BarMax, BarLabels: r.BarLabels}
	if r.Entities != nil {
		out.Entities = make([]ReportEntity, len(r.Entities))
		for i, e := range r.Entities {
			out.Entities[i] = ReportEntity(e)
		}
	}
	if r.Link != nil {
		link := ReportLink(*r.Link)
		out.Link = &link
	}
	return out
}

func wireRows(rows []report.Row) []ReportRow {
	out := make([]ReportRow, len(rows))
	for i, r := range rows {
		out[i] = ReportRow{Sort: r.Sort, Titles: r.Titles, Cells: r.Cells, Mine: r.Mine, Bars: r.Bars, Group: r.Group}
		if r.Range != nil {
			rng := MarketRange(*r.Range)
			out[i].Range = &rng
		}
	}
	return wireSlice(rows, out)
}

func wireChoices(choices []report.Choice) []RaceChoice {
	out := make([]RaceChoice, len(choices))
	for i, c := range choices {
		out[i] = RaceChoice(c)
	}
	return wireSlice(choices, out)
}

// wireSlice keeps a nil slice nil, so it still encodes as null rather than [].
func wireSlice[S, T any](in []S, out []T) []T {
	if in == nil {
		return nil
	}
	return out
}

// Package report is the shared report contract the feature services under internal/app build:
// a board of reports, each a table of rows, with the same fields and JSON tags as the wire types
// in package main (ReportBoard, Report, ReportRow and their parts).
//
// The wire types stay in package main. Wails names a TypeScript model after the Go package that
// declares it, and an alias does not change that, so main.Report must remain a type declared in
// main for frontend/wailsjs to keep its namespace. The root adapters convert these values to the
// wire twins (report_reading.go); a root test fails if the two sets drift apart.
package report

import (
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/lab"
	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
)

// Row is one table row; it is main.ReportRow.
type Row struct {
	Sort   []float64 `json:"sort,omitempty"`
	Titles []string  `json:"titles,omitempty"`
	Cells  []string  `json:"cells"`
	Range  *Range    `json:"range"`
	// Mine marks the GM's own franchise; Bars are the row's chart values on the report's BarMax.
	Mine bool      `json:"mine,omitempty"`
	Bars []float64 `json:"bars,omitempty"`
	// Group heads a run of rows (a division in the standings); rows of one group stay together.
	Group string `json:"group,omitempty"`
}

// Range is a row's value interval; it is main.MarketRange.
type Range struct {
	Max float64 `json:"max,omitempty"`
	lab.Interval
	Text string `json:"text"`
}

// Report is one report; it is main.Report.
type Report struct {
	Entities       []Entity `json:"entities,omitempty"`
	SortAscending  string   `json:"sortAscending,omitempty"`
	SortDescending string   `json:"sortDescending,omitempty"`
	ID             string   `json:"id"`
	Title          string   `json:"title"`
	Note           string   `json:"note"`
	Status         string   `json:"status"`
	Source         string   `json:"source"`
	AsOf           string   `json:"asOf"`
	How            string   `json:"how"`
	Columns        []string `json:"columns"`
	Rows           []Row    `json:"rows"`
	Summary        []Row    `json:"summary"`
	Link           *Link    `json:"link"`
	BarMax         float64  `json:"barMax,omitempty"`
	BarLabels      []string `json:"barLabels,omitempty"`
}

// Entity is a player, team or franchise named in a report's cells; it is main.ReportEntity.
type Entity struct {
	Text string `json:"text"`
	Kind string `json:"kind"`
	ID   string `json:"id"`
}

// Link opens another report by its registry id, e.g. the full list behind a summary; it is
// main.ReportLink.
type Link struct {
	Label    string `json:"label"`
	Endpoint string `json:"endpoint"`
}

// Board is a screen of reports; it is main.ReportBoard.
type Board struct {
	Title    string   `json:"title"`
	Loading  string   `json:"loading"`
	Refresh  string   `json:"refresh"`
	HowLabel string   `json:"howLabel"`
	MFLLabel string   `json:"mflLabel"`
	MFLURL   string   `json:"mflUrl"`
	Reports  []Report `json:"reports"`
}

// Choice is one option of a picker (a season, a franchise); it is main.RaceChoice.
type Choice struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

// Cells is a row of plain text cells.
func Cells(cells ...string) Row { return Row{Cells: cells} }

// ScoreText writes points with thousands separators and two decimals: 51,484.67.
func ScoreText(p leaguefeed.Points) string {
	sign, n := "", int64(p)
	if n < 0 {
		sign, n = "-", -n
	}
	whole := strconv.FormatInt(n/100, 10)
	for i := len(whole) - 3; i > 0; i -= 3 {
		whole = whole[:i] + "," + whole[i:]
	}
	return fmt.Sprintf("%s%s.%02d", sign, whole, n%100)
}

// Hundredths rounds a decimal score to exact hundredths of a point.
func Hundredths(points float64) leaguefeed.Points {
	return leaguefeed.Points(math.Round(points * 100))
}

// Feed is the held state of one MFL feed a report reads: when it was last read, whether today's
// read is still pending, and the last refresh's error.
type Feed struct {
	FetchedAt    time.Time
	RefreshError string
	ReadPending  bool
}

// Shell is a report with no rows yet, its status and note set from the feed: failed when never
// read, empty when the feed held no records, stale when a read is pending or a refresh failed.
// held is the number of records the feed holds.
func Shell(id, title, how string, feed Feed, held int, zone *time.Location, columns ...string) Report {
	r := Report{ID: id, Title: title, Source: "Source: MFL", How: how,
		Columns: columns, Rows: []Row{}, Status: "ok", AsOf: "As of: not fetched"}
	switch {
	case feed.FetchedAt.IsZero():
		r.Status, r.Note = "failed", "Report data unavailable; refresh from MFL to fill this report."
	case held == 0:
		r.Status, r.Note = "empty", "MFL lists no records for this report."
	}
	if !feed.FetchedAt.IsZero() {
		r.AsOf = "As of: " + Time(feed.FetchedAt, zone)
	}
	if feed.ReadPending {
		r.Status, r.Note = "stale", PendingRead(feed.FetchedAt, zone)
	}
	if feed.RefreshError != "" {
		if !feed.FetchedAt.IsZero() {
			r.Note += " Refresh failed; showing the last known data. Try refreshing from MFL again."
			r.Status = "stale"
		} else {
			r.Note += " Refresh failed. Try refreshing from MFL again."
		}
	}
	return r
}

// Time is the one wording of a report time: Jan 2, 2006, 3:04 PM in zone.
func Time(at time.Time, zone *time.Location) string {
	if at.IsZero() {
		return "Time not supplied"
	}
	return at.In(zone).Format("Jan 2, 2006, 3:04 PM")
}

// PendingRead notes a feed whose daily read has not happened yet.
func PendingRead(at time.Time, zone *time.Location) string {
	return "Last read " + Time(at, zone) + "; today's read pending"
}

// EngineRun is the one wording for when the engine last valued players, wherever it shows.
func EngineRun(at time.Time, zone *time.Location) string {
	return "Engine run: " + at.In(zone).Format("January 2, 2006, 3:04 PM")
}

type ReadingStatus string

const (
	ReadingOK       ReadingStatus = "ok"
	ReadingThin     ReadingStatus = "thin"
	ReadingFailed   ReadingStatus = "failed"
	ReadingNotWired ReadingStatus = "not_wired"
)

type Reading[T any] struct {
	snapshot.Sourced[T]
	Status        ReadingStatus `json:"status"`
	AbsenceReason *string       `json:"absenceReason"`
	Count         int           `json:"count"`
	Threshold     int           `json:"threshold"`
	Baseline      string        `json:"baseline"`
	Cutoff        string        `json:"cutoff"`
	Method        string        `json:"method"`
	InputSHA      string        `json:"inputSHA"`
	BuildStamp    string        `json:"buildStamp"`
	HowCounted    string        `json:"howCounted"`
}

// Count words a count with its noun: "1 trade", "1,204 trades".
func Count(n int, noun string) string {
	suffix := "s"
	if n == 1 {
		suffix = ""
	}
	return fmt.Sprintf("%s %s%s", Number(n), noun, suffix)
}

// Number writes a whole number with thousands separators: 1,204.
func Number(n int) string {
	text := strconv.Itoa(n)
	start := 0
	if n < 0 {
		start = 1
	}
	for i := len(text) - 3; i > start; i -= 3 {
		text = text[:i] + "," + text[i:]
	}
	return text

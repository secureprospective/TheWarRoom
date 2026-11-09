package composition

import (
	"github.com/secureprospective/TheWarRoom/internal/app/report"
)

type RaceChoice struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

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

// wireRange converts a service range to its wire twin, MarketRange; nil stays nil.
func wireRange(r *report.Range) *MarketRange {
	if r == nil {
		return nil
	}
	rng := MarketRange{Max: r.Max, Interval: r.Interval, Text: r.Text}
	return &rng
}

// reportHeld copies a held season feed into the shape feature services read.

// ReportEntity ties a literal display name to its held identity, including names in prose.
type ReportEntity struct {
	Text string `json:"text"`
	Kind string `json:"kind"`
	ID   string `json:"id"`
}

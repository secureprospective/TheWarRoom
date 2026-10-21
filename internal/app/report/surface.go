package report

import (
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/lab"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
)

// Line is one row of a Lab surface; it is main.MarketLine.
type Line struct {
	Key    string `json:"key"`
	Label  string `json:"label"`
	Detail string `json:"detail"`
	Count  int    `json:"count"`
	Range  *Range `json:"range"`
	Mark   string `json:"mark"`
}

// Breakdown is a titled run of lines inside a section; it is main.MarketBreakdown.
type Breakdown struct {
	Title string `json:"title"`
	Count int    `json:"count"`
	Rows  []Line `json:"rows"`
}

// Section is one panel of a Lab surface; it is main.MarketSection.
type Section struct {
	Breakdowns []Breakdown `json:"breakdowns,omitempty"`
	ID         string      `json:"id"`
	Title      string      `json:"title"`
	Note       string      `json:"note"`
	Rows       []Line      `json:"rows"`
}

// Surface is a Lab screen's value: a summary and its sections; it is main.MarketSurface.
type Surface struct {
	Summary  string    `json:"summary"`
	Sections []Section `json:"sections"`
}

// Assets are the traded kinds, with the eight positions first.
func Assets() []string {
	return []string{"QB", "RB", "WR", "TE", "PK", "DL", "LB", "DB", "pick1:next", "pick2:next", "pick3:next", "pick1:later", "pick2:later", "pick3:later"}
}

// RangeOf words a ratio's interval; nil remains nil.
func RangeOf(r *lab.Interval) *Range {
	if r == nil {
		return nil
	}
	return &Range{Interval: *r, Text: Percent(r.Value) + "; likely " + Percent(r.Lo) + " to " + Percent(r.Hi)}
}

// LabProvenance identifies frozen historical evidence, never current league truth.
func LabProvenance(built string) snapshot.Provenance {
	return snapshot.Provenance{Source: "Lab compiled facts sha256:" + lab.DataSHA, Kind: "archive", Freshness: domain.Freshness{State: domain.FreshStale, FetchedAt: built, Note: "Frozen historical input; not current league truth"}}
}

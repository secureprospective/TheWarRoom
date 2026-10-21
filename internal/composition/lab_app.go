package composition

import (
	marketapp "github.com/secureprospective/TheWarRoom/internal/app/market"
	"github.com/secureprospective/TheWarRoom/internal/lab"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
)

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

// MarketRange is the wire twin of report.Range, on every report row and pool market fact.
type MarketRange struct {
	Max float64 `json:"max,omitempty"`
	lab.Interval
	Text string `json:"text"`
}

// MarketLine, MarketBreakdown, MarketSection and MarketSurface are Market's wire types; the Draft
// Lab still built here words its surface in them.
type (
	MarketLine      = marketapp.MarketLine
	MarketBreakdown = marketapp.MarketBreakdown
	MarketSection   = marketapp.MarketSection
	MarketSurface   = marketapp.MarketSurface
)

// lab returns the shared decode, preserving the zero-state fallback.
func (a *App) lab() (*lab.Data, error) {
	return a.core.LabFacts() //nolint:wrapcheck // Preserve the frozen Lab loader error text.
}

// labStamp is this build's stamp on every Lab reading.
func labStamp() string { return buildLabel() + " " + buildDate }

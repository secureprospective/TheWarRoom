package composition

import (
	"github.com/secureprospective/TheWarRoom/internal/app/report"
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
	report.Interval
	Text string `json:"text"`
}

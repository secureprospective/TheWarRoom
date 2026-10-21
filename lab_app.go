package main

import (
	"fmt"
	"slices"

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

type RawMarketScope struct {
	From   int      `json:"from"`
	To     int      `json:"to"`
	Phases []string `json:"phases"`
	Assets []string `json:"assets"`
	ByAge  bool     `json:"byAge"`
}

// Validate checks a scope that came over IPC, in the service.
func (r RawMarketScope) Validate() error {
	return marketapp.Scope(r).Validate() //nolint:wrapcheck // the service's errors already start "market scope:"
}

type RawMarketAsset struct {
	Scope RawMarketScope `json:"scope"`
	Kind  string         `json:"kind"`
}

// Validate checks an asset profile request that came over IPC, in the service.
func (r RawMarketAsset) Validate() error {
	return marketapp.ValidateAsset(marketapp.Scope(r.Scope), r.Kind) //nolint:wrapcheck // the service's errors already start "asset profile:"
}

// MarketRange, MarketLine, MarketBreakdown, MarketSection and MarketSurface are the wire twins
// of report.Range and the marketapp surface types; wireSurface converts.
type MarketRange struct {
	Max float64 `json:"max,omitempty"`
	lab.Interval
	Text string `json:"text"`
}
type MarketLine struct {
	Key    string       `json:"key"`
	Label  string       `json:"label"`
	Detail string       `json:"detail"`
	Count  int          `json:"count"`
	Range  *MarketRange `json:"range"`
	Mark   string       `json:"mark"`
}
type MarketBreakdown struct {
	Title string       `json:"title"`
	Count int          `json:"count"`
	Rows  []MarketLine `json:"rows"`
}

type MarketSection struct {
	Breakdowns []MarketBreakdown `json:"breakdowns,omitempty"`
	ID         string            `json:"id"`
	Title      string            `json:"title"`
	Note       string            `json:"note"`
	Rows       []MarketLine      `json:"rows"`
}
type MarketSurface struct {
	Summary  string          `json:"summary"`
	Sections []MarketSection `json:"sections"`
}

func (a *App) TargetMarket(raw RawMarketScope) Reading[lab.Market] {
	if err := raw.Validate(); err != nil {
		return failedMarket(err)
	}
	d, err := a.lab()
	if err != nil {
		return failedMarket(err)
	}
	return marketReading(d, lab.Scope{From: raw.From, To: raw.To, Phases: slices.Clone(raw.Phases), ByAge: raw.ByAge})
}

func (a *App) TargetMarketLists(raw RawMarketScope) Reading[MarketSurface] {
	l, err := a.marketInput(raw)
	if err != nil {
		return failedSurface(err)
	}
	return wireReading(l.Lists(marketapp.Scope(raw)), wireSurface)
}

func (a *App) TargetPickClock(raw RawMarketScope) Reading[MarketSurface] {
	l, err := a.marketInput(raw)
	if err != nil {
		return failedSurface(err)
	}
	return wireReading(l.PickClock(marketapp.Scope(raw)), wireSurface)
}

func (a *App) TargetMarketStretches(raw RawMarketScope) Reading[MarketSurface] {
	l, err := a.marketInput(raw)
	if err != nil {
		return failedSurface(err)
	}
	return wireReading(l.Stretches(marketapp.Scope(raw)), wireSurface)
}

func (a *App) TargetAssetProfile(raw RawMarketAsset) Reading[MarketSurface] {
	if err := raw.Validate(); err != nil {
		return failedSurface(err)
	}
	d, err := a.lab()
	if err != nil {
		return failedSurface(err)
	}
	return wireReading(marketLab(d).Profile(marketapp.Scope(raw.Scope), raw.Kind), wireSurface)
}

// lab returns the shared decode, preserving the zero-state fallback.
func (a *App) lab() (*lab.Data, error) {
	return a.core.Lab() //nolint:wrapcheck // Preserve the frozen Lab loader error text.
}

// marketInput checks the scope, then reads the Lab the surface is built on.
func (a *App) marketInput(raw RawMarketScope) (marketapp.Lab, error) {
	if err := raw.Validate(); err != nil {
		return marketapp.Lab{}, fmt.Errorf("read Market: %w", err)
	}
	d, err := a.lab()
	return marketLab(d), err
}

// marketLab is the Lab decode with this build's stamp, as the Market service reads it.
func marketLab(d *lab.Data) marketapp.Lab {
	return marketapp.Lab{Data: d, Stamp: buildLabel() + " " + buildDate}
}

func failedMarket(err error) Reading[lab.Market] {
	return wireReading(marketapp.FailedMarket(err), sameMarket)
}
func marketReading(d *lab.Data, s lab.Scope) Reading[lab.Market] {
	return wireReading(marketLab(d).Market(s), sameMarket)
}
func sameMarket(m lab.Market) lab.Market { return m }
func failedSurface(err error) Reading[MarketSurface] {
	return wireReading(marketapp.FailedSurface(err), wireSurface)
}

// wireSurface converts a surface built by the Market service to its wire twin. The two type
// sets are kept identical field for field; TestMarketWireTwinsMatch proves it.
func wireSurface(s marketapp.Surface) MarketSurface {
	sections := make([]MarketSection, len(s.Sections))
	for i, section := range s.Sections {
		sections[i] = MarketSection{ID: section.ID, Title: section.Title, Note: section.Note, Rows: wireLines(section.Rows)}
		if section.Breakdowns != nil {
			breakdowns := make([]MarketBreakdown, len(section.Breakdowns))
			for j, b := range section.Breakdowns {
				breakdowns[j] = MarketBreakdown{Title: b.Title, Count: b.Count, Rows: wireLines(b.Rows)}
			}
			sections[i].Breakdowns = breakdowns
		}
	}
	return MarketSurface{Summary: s.Summary, Sections: wireSlice(s.Sections, sections)}
}

func wireLines(lines []marketapp.Line) []MarketLine {
	out := make([]MarketLine, len(lines))
	for i, l := range lines {
		out[i] = MarketLine{Key: l.Key, Label: l.Label, Detail: l.Detail, Count: l.Count, Range: wireRange(l.Range), Mark: l.Mark}
	}
	return wireSlice(lines, out)
}

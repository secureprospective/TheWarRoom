package labcompile

import (
	"context"
	"fmt"
	"time"
)

// Evidence carries audit inputs that are not facts about league activity.
type Evidence struct {
	Built, ArchiveThrough time.Time
	Pages                 PageExam
}
type PageExam struct {
	Asked   int `json:"asked"`
	Matched int `json:"matched"`
}

// Analysis indexes trades in the original transaction-id order, never timestamp order.
type Analysis struct {
	Market      Market             `json:"market"`
	Checks2     AnalysisChecks     `json:"checks2"`
	Legs        []AnalysisLeg      `json:"legs"`
	Deals       []AnalysisDeal     `json:"deals"`
	Delivered   []AnalysisDelivery `json:"delivered"`
	AllPlay     []AnalysisTeam     `json:"allPlay"`
	PickSlots   []AnalysisPick     `json:"pickSlots"`
	PickSeasons []AnalysisPick     `json:"pickSeasons"`
	Weeks       map[int64]int64    `json:"weeks"`
}
type AnalysisLeg struct {
	Trade       int      `json:"trade"`
	Side        int      `json:"side"`
	Asset       int      `json:"asset"`
	Value       *float64 `json:"value"`
	Class       *string  `json:"class"`
	Price       *float64 `json:"price"`
	Real        *float64 `json:"real"`
	RealSeasons *int64   `json:"realSeasons"`
}
type AnalysisDeal struct {
	Trade       int      `json:"trade"`
	ANet        *float64 `json:"aNet"`
	ARealNet    *float64 `json:"aRealNet"`
	RealSeasons *int64   `json:"realSeasons"`
	Window      *string  `json:"window"`
	Season      *int64   `json:"season"`
}
type AnalysisDelivery struct {
	Trade     int     `json:"trade"`
	Class     string  `json:"class"`
	Expected  float64 `json:"expected"`
	Delivered float64 `json:"delivered"`
	Seasons   int64   `json:"seasons"`
}
type AnalysisTeam struct {
	Year         int64    `json:"year"`
	Fid          string   `json:"fid"`
	AllPlay      *float64 `json:"allPlay"`
	Wins         float64  `json:"wins"`
	Games        int64    `json:"games"`
	ExpectedWins *float64 `json:"expectedWins"`
	Luck         *float64 `json:"luck"`
}
type AnalysisPick struct {
	Year   int64      `json:"year"`
	Round  int64      `json:"round"`
	Slot   int64      `json:"slot"`
	Values []*float64 `json:"values"`
}
type Analyse func(context.Context, *Lab, []Trade, []Pick, map[teamYear]int64, map[teamYear]int64) (Analysis, error)

type legKey struct{ Trade, Side, Asset int }
type analysisIndex struct {
	Legs                   map[legKey]AnalysisLeg
	Deals                  map[int]AnalysisDeal
	AllPlay                map[teamYear]AnalysisTeam
	PickSlots, PickSeasons map[pickKey][]*float64
	Weeks                  map[int64]int64
}

func indexAnalysis(a Analysis) analysisIndex {
	out := analysisIndex{Legs: map[legKey]AnalysisLeg{}, Deals: map[int]AnalysisDeal{}, AllPlay: map[teamYear]AnalysisTeam{}, PickSlots: map[pickKey][]*float64{}, PickSeasons: map[pickKey][]*float64{}, Weeks: a.Weeks}
	for _, v := range a.Legs {
		out.Legs[legKey{v.Trade, v.Side, v.Asset}] = v
	}
	for _, v := range a.Deals {
		out.Deals[v.Trade] = v
	}
	for _, v := range a.AllPlay {
		out.AllPlay[teamYear{v.Year, v.Fid}] = v
	}
	for _, v := range a.PickSlots {
		out.PickSlots[pickKey{v.Year, v.Round, v.Slot}] = v.Values
	}
	for _, v := range a.PickSeasons {
		out.PickSeasons[pickKey{v.Year, v.Round, v.Slot}] = v.Values
	}
	return out
}

func (e Evidence) Validate() error {
	if e.Built.IsZero() || e.ArchiveThrough.IsZero() {
		return fmt.Errorf("lab evidence needs built and archive timestamps")
	}
	if e.Pages.Asked < 0 || e.Pages.Matched < 0 || e.Pages.Matched > e.Pages.Asked {
		return fmt.Errorf("lab page exam counts are invalid")
	}
	return nil
}

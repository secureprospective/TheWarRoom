// Package leagueclock reads only supplied league facts; missing windows remain unknown.
//
// Held facts today are the season, the phase log and the commissioner calendar. MFL's league
// schedule carries week numbers but no dates, so it cannot date a deadline or name the current
// week; both arrive with a dated source (ring 1), never by inference.
package leagueclock

import (
	"cmp"
	"slices"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
)

type Urgency string

const (
	U0 Urgency = "U0"
	U1 Urgency = "U1"
	U2 Urgency = "U2"
	U3 Urgency = "U3"
)

// Ordering rule (spec §3): within PinWithin a deadline is pinned to Home's alert tray and the
// seasonal card; within PromoteWithin it moves up its lane.
const (
	urgentWithin  = time.Hour
	PinWithin     = 48 * time.Hour
	PromoteWithin = 7 * 24 * time.Hour
)

type WindowKind string

const (
	ContractOptions WindowKind = "contract_options"
	RFATender       WindowKind = "rfa_tender"
	UFABidding      WindowKind = "ufa_bidding"
	Resign          WindowKind = "re_sign"
	CutDay          WindowKind = "cut_day"
	TradeDeadline   WindowKind = "trade_deadline"
	Draft           WindowKind = "draft"
)

type WindowStatus string

const WindowUnknown WindowStatus = "unknown"

type Window struct {
	Kind   WindowKind   `json:"kind"`
	Status WindowStatus `json:"status"`
	Reason string       `json:"reason"`
}

// Event is a commissioner calendar entry; only PLANNED events are deadlines.
type Event struct {
	ID, Kind, Status string
	At               time.Time
}

type Inputs struct {
	Season int
	Phase  domain.Phase
	Events []Event
}

type Deadline struct {
	ID       string     `json:"id"`
	Label    string     `json:"label"`
	At       *time.Time `json:"at,omitempty"`
	Urgency  Urgency    `json:"urgency"`
	Pinned   bool       `json:"pinned"`
	Promoted bool       `json:"promoted"`
}

type Reading struct {
	Season    int          `json:"season"`
	Phase     domain.Phase `json:"phase"`
	Deadlines []Deadline   `json:"deadlines"`
	Windows   []Window     `json:"windows"`
}

// Clock reads the league at a supplied instant: deadlines soonest first, undated last.
func Clock(at time.Time, in Inputs) Reading {
	out := Reading{Season: in.Season, Phase: in.Phase, Deadlines: []Deadline{}, Windows: unknownWindows()}
	for _, e := range in.Events {
		if e.Status == "PLANNED" {
			out.Deadlines = append(out.Deadlines, deadline(at, e))
		}
	}
	slices.SortFunc(out.Deadlines, func(a, b Deadline) int {
		switch {
		case a.At == nil && b.At != nil:
			return 1
		case a.At != nil && b.At == nil:
			return -1
		case a.At != nil && !a.At.Equal(*b.At):
			return a.At.Compare(*b.At)
		}
		return cmp.Compare(a.ID, b.ID)
	})
	return out
}

// deadline grades urgency against the instant: U3 under an hour or passed while still planned,
// U2 within the pin window, U1 dated, U0 undated.
func deadline(at time.Time, e Event) Deadline {
	d := Deadline{ID: e.ID, Label: e.Kind, Urgency: U0}
	if e.At.IsZero() {
		return d
	}
	utc := e.At.UTC()
	d.At = &utc
	remaining := utc.Sub(at)
	d.Pinned, d.Promoted = remaining <= PinWithin, remaining <= PromoteWithin
	switch {
	case remaining < urgentWithin:
		d.Urgency = U3
	case d.Pinned:
		d.Urgency = U2
	default:
		d.Urgency = U1
	}
	return d
}

func unknownWindows() []Window {
	kinds := []WindowKind{ContractOptions, RFATender, UFABidding, Resign, CutDay, TradeDeadline, Draft}
	out := make([]Window, 0, len(kinds))
	for _, k := range kinds {
		out = append(out, Window{Kind: k, Status: WindowUnknown,
			Reason: "league rules not yet captured — gap closure"})
	}
	return out
}

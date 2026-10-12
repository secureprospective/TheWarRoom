package leagueclock

import (
	"fmt"
	"time"
)

// Window statuses: dated when the rulebook or the commissioner gives an instant, unknown otherwise.
const (
	WindowDated WindowStatus = "dated"
	// PlayoffBidsOnly and MultiYearBidsClose are the §6 in-season bidding cutoffs.
	PlayoffBidsOnly    WindowKind = "playoff_bids_only"
	MultiYearBidsClose WindowKind = "multiyear_bids_close"
)

// TradeDeadlineWeek is §14 of the rulebook: trades close at the first kickoff of Week 9. The
// 2025 GB/WAS trade was voided for missing it by minutes on Thursday 2025-10-30, Week 9's opener.
const TradeDeadlineWeek = 9

// WindowRule is one league window's base date: the first kickoff of an NFL week, or no week when
// the rulebook gives no date and only the commissioner can set one.
type WindowRule struct {
	Kind  WindowKind
	Week  int
	Basis string
}

// Rules lists every league window in display order. The §6 cutoffs follow the league's last
// regular-season week: "after Week 12" and "end of Week 13" each end at the next week's first
// kickoff. Without that setting they stay undated rather than assume the rulebook's 13.
func Rules(lastRegular int) []WindowRule {
	none := "the rulebook gives no date; the commissioner sets it"
	narrow, closing := "Rulebook §6: after the second-to-last regular week only playoff teams bid, 1-year only",
		"Rulebook §6: multi-year bids close at the end of the regular season"
	bids := 0
	if lastRegular >= 2 {
		bids = lastRegular
		narrow = fmt.Sprintf("Rulebook §6: after Week %d only playoff teams bid, 1-year bids only", lastRegular-1)
		closing = fmt.Sprintf("Rulebook §6: multi-year bids close at the end of Week %d", lastRegular)
	}
	return []WindowRule{
		{Kind: TradeDeadline, Week: TradeDeadlineWeek,
			Basis: fmt.Sprintf("Rulebook §14: trade deadline, Week %d's first kickoff", TradeDeadlineWeek)},
		{Kind: PlayoffBidsOnly, Week: bids,
			Basis: narrow},
		{Kind: MultiYearBidsClose, Week: bids + min(bids, 1),
			Basis: closing},
		{Kind: ContractOptions, Basis: none}, {Kind: RFATender, Basis: none}, {Kind: UFABidding, Basis: none},
		{Kind: Resign, Basis: none}, {Kind: CutDay, Basis: none}, {Kind: Draft, Basis: none},
	}
}

// WindowInputs are the facts that date the windows: the league's last regular-season week, each
// rule week's first kickoff from the NFL schedule, and the commissioner's moves, which always win.
type WindowInputs struct {
	LastRegular  int
	FirstKickoff map[int]time.Time
	Moved        map[WindowKind]time.Time
}

// windows dates every rule and returns the upcoming dated ones as deadlines too, so the seasonal
// card, the strip and the alert tray grade them like any other deadline.
func windows(at time.Time, in WindowInputs) ([]Window, []Deadline) {
	rules := Rules(in.LastRegular)
	out := make([]Window, 0, len(rules))
	var due []Deadline
	for _, r := range rules {
		w := Window{Kind: r.Kind, Status: WindowUnknown, Reason: r.Basis}
		kickoff, scheduled := in.FirstKickoff[r.Week]
		switch moved, ok := in.Moved[r.Kind]; {
		case ok:
			w.At, w.Moved = &moved, true
			w.Reason = "Set by the commissioner"
			if r.Week != 0 {
				w.Reason = "Moved by the commissioner · " + r.Basis
			}
		case r.Week != 0 && scheduled:
			w.At = &kickoff
		case r.Week != 0:
			w.Reason = fmt.Sprintf("%s · Week %d schedule not held yet", r.Basis, r.Week)
		}
		if w.At != nil {
			utc := w.At.UTC()
			w.At, w.Status = &utc, WindowDated
			if utc.After(at) {
				due = append(due, deadline(at, Event{ID: WindowEventPrefix + string(r.Kind), Kind: string(r.Kind), At: utc}))
			}
		}
		out = append(out, w)
	}
	return out, due
}

// WindowEventPrefix marks a deadline that is a league window rather than a calendar entry.
const WindowEventPrefix = "window:"

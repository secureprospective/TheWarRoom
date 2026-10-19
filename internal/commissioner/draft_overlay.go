package commissioner

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/leagueclock"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

// errNeedsDraftStart says the calendar has no window that opens the draft's picks.
var errNeedsDraftStart = errors.New("needs a ruling: the rookie draft start")

// draftRulesFor are the league's rules at an instant, with the order the commissioner recorded for one
// draft laid over them. A reversed order command is not in force, so it lays nothing.
func draftRulesFor(records []Record, season int, at time.Time) (leaguerules.RuleSet, error) {
	rules, err := RulesNow(records, at)
	if err != nil {
		return leaguerules.RuleSet{}, err
	}
	active := Replay(records).Active
	for _, r := range records {
		d := r.Command.Args.Draft
		if !active[r.Receipt.Entry] || d == nil || d.Season != season {
			continue
		}
		switch r.Command.Command {
		case DraftOrderAuto:
			rules.Draft.OrderSource = leaguerules.Known(d.Source)
			rules.Draft.Order = leaguerules.Known(map[int][]string{})
		case DraftOrderSet:
			rules.Draft.Order = leaguerules.Known(withOrder(rules.Draft.Order, d.Round, d.Order,
				rules.Draft.Rounds))
		}
	}
	return rules, nil
}

// withOrder copies the recorded orders and sets one round, or every round when round is 0.
func withOrder(now leaguerules.Value[map[int][]string], round int, order []string,
	rounds leaguerules.Value[int]) map[int][]string {
	orders := map[int][]string{}
	if now.State == leaguerules.IsKnown {
		for r, teams := range now.Value {
			orders[r] = slices.Clone(teams)
		}
	}
	if round != 0 {
		orders[round] = slices.Clone(order)
		return orders
	}
	for r := 1; rounds.State == leaguerules.IsKnown && r <= rounds.Value; r++ {
		orders[r] = slices.Clone(order)
	}
	return orders
}

// draftStartFor is the start recorded for one draft, if the commissioner has set one.
func draftStartFor(records []Record, season int) (time.Time, bool) {
	active := Replay(records).Active
	var start time.Time
	found := false
	for _, r := range records {
		if r.Command.Command == DraftStart && active[r.Receipt.Entry] && r.Command.Args.Draft.Season == season {
			start, found = r.Command.Args.Draft.At, true
		}
	}
	return start, found
}

// draftStartSet says whether any draft has a start recorded.
func draftStartSet(records []Record) bool {
	active := Replay(records).Active
	return slices.ContainsFunc(records, func(r Record) bool {
		return r.Command.Command == DraftStart && active[r.Receipt.Entry]
	})
}

// projectDraftCalendar gives the calendar the draft's own start. The calendar and the draft both read the
// start through here, so a draft start can never disagree with the window that opens its picks.
func projectDraftCalendar(clock *leagueclock.ActionCalendar, records []Record, season int, at time.Time) {
	start, found := draftStartFor(records, season)
	if !found {
		return
	}
	window := leagueclock.ActionWindow{ID: rookieDraftWindow, Opens: []leaguerules.Action{draftPickAction},
		Start: start.UTC()}
	for i, w := range clock.Windows {
		if w.ID == rookieDraftWindow && w.Close != nil && w.Close.After(start) {
			window.Close = w.Close
		}
		clock.Windows[i].Opens = slices.DeleteFunc(slices.Clone(w.Opens), func(a leaguerules.Action) bool {
			return a == draftPickAction
		})
	}
	clock.Windows = slices.DeleteFunc(clock.Windows, func(w leagueclock.ActionWindow) bool {
		return w.ID == rookieDraftWindow
	})
	window.Live = !at.Before(window.Start) && (window.Close == nil || at.Before(*window.Close))
	if window.Live && window.Close != nil {
		seconds := int64(window.Close.Sub(at) / time.Second)
		window.Seconds = &seconds
	}
	clock.Windows = append(clock.Windows, window)
	slices.SortFunc(clock.Windows, func(a, b leagueclock.ActionWindow) int { return strings.Compare(a.ID, b.ID) })
}

// draftCalendarStart is when the draft's picks open, read from the same calendar the gate reads.
func draftCalendarStart(records []Record, rules leaguerules.RuleSet, season int, at time.Time) (time.Time, error) {
	clock, err := leagueclock.Calendar(at, season, rules.Calendar, nil)
	if err != nil {
		return time.Time{}, fmt.Errorf("date the rookie draft: %w", err)
	}
	projectDraftCalendar(&clock, records, season, at)
	for _, w := range clock.Windows {
		if slices.Contains(w.Opens, draftPickAction) {
			return w.Start, nil
		}
	}
	return time.Time{}, errNeedsDraftStart
}

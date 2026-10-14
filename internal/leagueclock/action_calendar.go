package leagueclock

import (
	"fmt"
	"slices"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

type ActionWindow struct {
	ID      string               `json:"id"`
	Opens   []leaguerules.Action `json:"opens"`
	Start   time.Time            `json:"start"`
	Close   *time.Time           `json:"close"`
	Live    bool                 `json:"live"`
	Seconds *int64               `json:"secondsToClose"`
}

type WindowException struct {
	Action    leaguerules.Action `json:"action"`
	Franchise string             `json:"franchise"`
	Start     time.Time          `json:"start"`
	Until     time.Time          `json:"until"`
	Reason    string             `json:"reason"`
}

type ActionDecision struct {
	Open   bool       `json:"open"`
	Reason string     `json:"reason"`
	Window string     `json:"window"`
	Next   *time.Time `json:"next"`
}

type ActionCalendar struct {
	Windows       []ActionWindow    `json:"windows"`
	Exceptions    []WindowException `json:"exceptions"`
	TradeDeadline *time.Time        `json:"tradeDeadline"`
	TradeSeconds  *int64            `json:"secondsToTradeDeadline"`
}

// Calendar uses civil dates, not 24-hour durations: a league day can be 23 or 25 hours.
func Calendar(at time.Time, year int, rules leaguerules.CalendarRules,
	exceptions []WindowException) (ActionCalendar, error) {
	out := ActionCalendar{Windows: []ActionWindow{}, Exceptions: slices.Clone(exceptions)}
	if rules.Timezone.State != leaguerules.IsKnown || rules.Windows.State != leaguerules.IsKnown {
		return out, fmt.Errorf("the league time zone and action windows need a ruling")
	}
	zone, err := time.LoadLocation(rules.Timezone.Value)
	if err != nil {
		return out, fmt.Errorf("read league time zone: %w", err)
	}
	ids := make([]string, 0, len(rules.Windows.Value))
	for id := range rules.Windows.Value {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	for _, id := range ids {
		window, err := dateWindow(id, year, zone, rules)
		if err != nil {
			return out, err
		}
		window.Live = !at.Before(window.Start) && (window.Close == nil || at.Before(*window.Close))
		if window.Live && window.Close != nil {
			seconds := int64(window.Close.Sub(at) / time.Second)
			window.Seconds = &seconds
		}
		out.Windows = append(out.Windows, window)
	}
	if rules.TradeDeadline.State == leaguerules.IsKnown {
		deadline := rules.TradeDeadline.Value.UTC()
		seconds := int64(deadline.Sub(at) / time.Second)
		out.TradeDeadline, out.TradeSeconds = &deadline, &seconds
	}
	return out, nil
}

func dateWindow(id string, year int, zone *time.Location,
	rules leaguerules.CalendarRules) (ActionWindow, error) {
	w := rules.Windows.Value[id]
	out := ActionWindow{ID: id, Opens: []leaguerules.Action{}}
	if w.Opens.State == leaguerules.IsKnown {
		out.Opens = slices.Clone(w.Opens.Value)
	}
	start, err := windowDate(w.Start, w.DatedStart, year, zone)
	if err != nil {
		return out, fmt.Errorf("window %s opens: %w", id, err)
	}
	if sharedStart(id, w.Start, rules.Windows.Value) {
		if rules.Boundary.State != leaguerules.IsKnown {
			return out, fmt.Errorf("choose which side of the shared date opens window %s", id)
		}
		if rules.Boundary.Value == leaguerules.EndOfDay && w.DatedStart.State != leaguerules.IsKnown {
			start = start.AddDate(0, 0, 1)
		}
	}
	out.Start = start.UTC()
	if w.End.State != leaguerules.IsKnown && w.DatedEnd.State != leaguerules.IsKnown {
		return out, nil
	}
	closeAt, err := windowDate(w.End, w.DatedEnd, year, zone)
	if err != nil {
		return out, fmt.Errorf("window %s closes: %w", id, err)
	}
	if w.DatedEnd.State != leaguerules.IsKnown {
		if !sharedEnd(id, w.End, rules.Windows.Value) || rules.Boundary.Value == leaguerules.EndOfDay {
			closeAt = closeAt.AddDate(0, 0, 1)
		}
	}
	if !closeAt.After(start) {
		return out, fmt.Errorf("window %s must close after it opens", id)
	}
	utc := closeAt.UTC()
	out.Close = &utc
	return out, nil
}

func windowDate(date leaguerules.Value[leaguerules.MonthDay], dated leaguerules.Value[time.Time],
	year int, zone *time.Location) (time.Time, error) {
	if dated.State == leaguerules.IsKnown {
		local := dated.Value.In(zone)
		if local.Hour() != 0 || local.Minute() != 0 || local.Second() != 0 || local.Nanosecond() != 0 {
			return time.Time{}, fmt.Errorf("choose midnight in the league time zone")
		}
		return local, nil
	}
	if date.State != leaguerules.IsKnown || date.Value.Month.State != leaguerules.IsKnown ||
		date.Value.Day.State != leaguerules.IsKnown {
		return time.Time{}, fmt.Errorf("choose an opening date")
	}
	month, day := date.Value.Month.Value, date.Value.Day.Value
	at := time.Date(year, time.Month(month), day, 0, 0, 0, 0, zone)
	if int(at.Month()) != month || at.Day() != day {
		return time.Time{}, fmt.Errorf("choose a valid date for this season")
	}
	return at, nil
}

func sharedStart(id string, start leaguerules.Value[leaguerules.MonthDay],
	windows map[string]leaguerules.Window) bool {
	for other, w := range windows {
		if other != id && sameDate(start, w.End) {
			return true
		}
	}
	return false
}

func sharedEnd(id string, end leaguerules.Value[leaguerules.MonthDay],
	windows map[string]leaguerules.Window) bool {
	for other, w := range windows {
		if other != id && sameDate(end, w.Start) {
			return true
		}
	}
	return false
}

func sameDate(a, b leaguerules.Value[leaguerules.MonthDay]) bool {
	return a.State == leaguerules.IsKnown && b.State == leaguerules.IsKnown &&
		a.Value.Month.Value == b.Value.Month.Value && a.Value.Day.Value == b.Value.Day.Value
}

// Decide is default-deny, with franchise-scoped exceptions and an exact trade cutoff.
func (c ActionCalendar) Decide(at time.Time, action leaguerules.Action, franchise string) ActionDecision {
	if exception, ok := c.exception(at, action, franchise); ok {
		return ActionDecision{Open: true, Reason: exception.Reason}
	}
	if action == "trade" {
		if c.TradeDeadline == nil {
			return ActionDecision{Reason: "The trade deadline needs a ruling."}
		}
		if !at.Before(*c.TradeDeadline) {
			return ActionDecision{Reason: "The trade deadline has passed."}
		}
	}
	for _, w := range c.Windows {
		if slices.Contains(w.Opens, action) && !at.Before(w.Start) && (w.Close == nil || at.Before(*w.Close)) {
			return ActionDecision{Open: true}
		}
	}
	return c.nextWindow(at, action)
}

func (c ActionCalendar) exception(at time.Time, action leaguerules.Action, franchise string) (WindowException, bool) {
	for _, exception := range c.Exceptions {
		if exception.Action == action && exception.Franchise == franchise && !at.Before(exception.Start) &&
			at.Before(exception.Until) {
			return exception, true
		}
	}
	return WindowException{}, false
}

func (c ActionCalendar) nextWindow(at time.Time, action leaguerules.Action) ActionDecision {
	decision := ActionDecision{Reason: "No league window opens this action now."}
	for _, w := range c.Windows {
		if slices.Contains(w.Opens, action) && w.Start.After(at) &&
			(decision.Next == nil || w.Start.Before(*decision.Next)) {
			next := w.Start
			decision.Window, decision.Next = w.ID, &next
		}
	}
	return decision
}

type BidCountdown struct {
	SnipeMark time.Time `json:"snipeMark"`
	Close     time.Time `json:"close"`
}

func Countdown(start, closeAt time.Time, snipeAfter time.Duration) BidCountdown {
	return BidCountdown{SnipeMark: start.Add(snipeAfter).UTC(), Close: closeAt.UTC()}
}

func (c *ActionCalendar) Localize(local *time.Location) {
	for i := range c.Windows {
		w := &c.Windows[i]
		w.Start = w.Start.In(local)
		if w.Close != nil {
			closeAt := w.Close.In(local)
			w.Close = &closeAt
		}
	}
	for i := range c.Exceptions {
		c.Exceptions[i].Start = c.Exceptions[i].Start.In(local)
		c.Exceptions[i].Until = c.Exceptions[i].Until.In(local)
	}
	if c.TradeDeadline != nil {
		deadline := c.TradeDeadline.In(local)
		c.TradeDeadline = &deadline
	}
}

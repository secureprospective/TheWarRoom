package commissioner

import (
	"fmt"
	"slices"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/leagueclock"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

func rolloverRules(state RolloverState, rules leaguerules.RuleSet, to int) (leaguerules.RuleSet, error) {
	changes := leaguerules.SeedChanges(rules, rules.LeagueID, to, state.At)
	waiting := slices.Clone(state.Waiting)
	slices.SortStableFunc(waiting, func(a, b leaguerules.Change) int {
		return a.RecordedAt.Compare(b.RecordedAt)
	})
	latest := map[leaguerules.FieldPath]leaguerules.Change{}
	order := []leaguerules.FieldPath{}
	for _, change := range waiting {
		if change.LeagueID != rules.LeagueID || change.Season != rules.Season ||
			change.Effective.Kind != leaguerules.NextRollover || !change.RecordedAt.Before(state.At) {
			return leaguerules.RuleSet{}, fmt.Errorf("waiting changes must belong to this league's next rollover")
		}
		change.Effective.At = state.At
		if err := change.Validate(); err != nil {
			return leaguerules.RuleSet{}, fmt.Errorf("preview waiting rule: %w", err)
		}
		if _, seen := latest[change.Field]; !seen {
			order = append(order, change.Field)
		}
		latest[change.Field] = change
	}
	for _, field := range order {
		change := latest[field]
		change.Season = to
		changes = append(changes, change)
	}
	resolved, err := leaguerules.Resolve(changes, to, state.At)
	if err != nil {
		return leaguerules.RuleSet{}, fmt.Errorf("preview next season rules: %w", err)
	}
	return resolved, nil
}

func (p *RolloverPlan) dateCalendar(at time.Time) {
	calendar := &p.Rules.Calendar
	if calendar.Timezone.State != leaguerules.IsKnown {
		p.block("League timezone needs a ruling: choose where the new season's dates begin.")
		return
	}
	zone, err := time.LoadLocation(calendar.Timezone.Value)
	if err != nil {
		p.block("League timezone needs a ruling: " + err.Error())
		return
	}
	windows := make(map[string]leaguerules.Window, len(calendar.Windows.Value))
	for id, window := range calendar.Windows.Value {
		window.DatedStart = p.redate(window.DatedStart, zone, "Window "+id+" opening")
		window.DatedEnd = p.redate(window.DatedEnd, zone, "Window "+id+" closing")
		windows[id] = window
	}
	calendar.Windows.Value = windows
	calendar.TradeDeadline = p.redate(calendar.TradeDeadline, zone, "Trade deadline")
	calendar.Rollover = p.redate(calendar.Rollover, zone, "Next rollover")
	reading, err := leagueclock.Calendar(at, p.To, *calendar, nil)
	if err != nil {
		p.block("League action windows need a ruling: " + err.Error())
		return
	}
	p.Calendar = reading
	for _, window := range reading.Windows {
		words := fmt.Sprintf("Window %s opens %s", window.ID, window.Start.In(zone).Format("2 Jan 2006 at 3:04 PM MST"))
		if window.Close != nil {
			words += " and closes " + window.Close.In(zone).Format("2 Jan 2006 at 3:04 PM MST")
		}
		p.Lines = append(p.Lines, words+".")
	}
}

// Newly waiting dates in the destination season are already dated; old dates advance by civil year.
func (p *RolloverPlan) redate(value leaguerules.Value[time.Time], zone *time.Location,
	label string) leaguerules.Value[time.Time] {
	if value.State != leaguerules.IsKnown {
		return value
	}
	local := value.Value.In(zone)
	if local.Year() == p.To {
		return value
	}
	if local.Year() != p.From {
		p.block(label + " needs a ruling: the recorded date belongs to neither season.")
		return value
	}
	moved := local.AddDate(1, 0, 0)
	if moved.Month() != local.Month() || moved.Day() != local.Day() {
		p.block(label + " needs a ruling: this day does not exist in the new season.")
		return value
	}
	value.Value = moved.UTC()
	return value
}

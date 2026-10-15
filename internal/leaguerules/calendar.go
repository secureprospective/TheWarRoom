package leaguerules

import "time"

type CalendarRules struct {
	Windows           Value[map[string]Window]
	Timezone          Value[string]
	Boundary          Value[BoundarySide]
	Rollover          Value[time.Time]
	RolloverDate      Value[MonthDay]
	TradeDeadline     Value[time.Time]
	TradeDeadlineWeek Value[int]
}

func legacyCalendar() CalendarRules {
	return CalendarRules{
		Windows:           cited[map[string]Window](legacyWindows(), "Census CM-023"),
		Timezone:          cited[string]("America/New_York", "fact 12a"),
		Boundary:          Needs[BoundarySide]("fact 12b: commissioner chooses"),
		Rollover:          Needs[time.Time]("fact 9b: March 1 requires a dated season"),
		RolloverDate:      cited[MonthDay](MonthDay{Month: cited(3, "fact 9b"), Day: cited(1, "fact 9b")}, "fact 9b"),
		TradeDeadline:     Needs[time.Time]("fact 12c: exact instant chosen by commissioner"),
		TradeDeadlineWeek: cited[int](9, "fact 12c"),
	}
}

func calendarFields() []ruleField {
	return []ruleField{
		field[map[string]Window]{path: "calendar.windows", label: "League action windows", access: func(r *RuleSet) *Value[map[string]Window] { return &r.Calendar.Windows }},
		field[string]{path: "calendar.timezone", label: "League timezone", access: func(r *RuleSet) *Value[string] { return &r.Calendar.Timezone }},
		field[BoundarySide]{path: "calendar.boundary", label: "Shared date boundary", access: func(r *RuleSet) *Value[BoundarySide] { return &r.Calendar.Boundary }},
		field[time.Time]{path: "calendar.rollover", label: "Season rollover instant", access: func(r *RuleSet) *Value[time.Time] { return &r.Calendar.Rollover }},
		field[MonthDay]{path: "calendar.rolloverDate", label: "Annual rollover date", access: func(r *RuleSet) *Value[MonthDay] { return &r.Calendar.RolloverDate }},
		field[time.Time]{path: "calendar.tradeDeadline", label: "Trade deadline", access: func(r *RuleSet) *Value[time.Time] { return &r.Calendar.TradeDeadline }},
		field[int]{path: "calendar.tradeDeadlineWeek", label: "Trade deadline week", access: func(r *RuleSet) *Value[int] { return &r.Calendar.TradeDeadlineWeek }},
	}
}

type MonthDay struct {
	Month Value[int]
	Day   Value[int]
}
type Window struct {
	Start      Value[MonthDay]
	End        Value[MonthDay]
	Opens      Value[[]Action]
	DatedStart Value[time.Time]
	DatedEnd   Value[time.Time]
}

func monthDay(month, day int, source string) MonthDay {
	return MonthDay{Month: cited(month, source), Day: cited(day, source)}
}

// lineupSource cites the lineups window: lineups stay open all league year, and each player locks at
// his own kickoff (the native lineup check), so no window closes them.
const lineupSource = "Ring 4 lineup ruling: open all year, each player locks at kickoff"

func legacyWindows() map[string]Window {
	rows := []struct {
		id                                     string
		startMonth, startDay, endMonth, endDay int
		actions                                []Action
	}{
		{"contracts", 3, 1, 4, 30, nil},
		{"rfa_tenders", 5, 1, 5, 3, []Action{"rfa_tender"}},
		{"ufa", 5, 3, 5, 10, []Action{"ufa_bid"}},
		{"resign", 5, 10, 5, 15, []Action{"resign"}},
		{"open_fa", 5, 19, 0, 0, []Action{"ufa_bid"}},
		{"rookie_draft", 7, 14, 0, 0, []Action{"draft_pick"}},
		{"cut_buyout", 8, 25, 9, 1, []Action{"cut", "buyout"}},
		{"lineups", 3, 1, 0, 0, []Action{"lineup"}},
	}
	sourceOf := func(id string) string {
		if id == "lineups" {
			return lineupSource
		}
		return "Census CM-023"
	}
	result := make(map[string]Window, len(rows))
	for _, row := range rows {
		source := sourceOf(row.id)
		window := Window{Start: cited(monthDay(row.startMonth, row.startDay, source), source),
			End:        Needs[MonthDay]("Census CM-023: no closing date given"),
			Opens:      Needs[[]Action]("Census CM-023: actions chosen by commissioner"),
			DatedStart: Needs[time.Time]("Census CM-023: date for season required"),
			DatedEnd:   Needs[time.Time]("Census CM-023: date for season required")}
		if row.endMonth != 0 {
			window.End = cited(monthDay(row.endMonth, row.endDay, source), source)
		}
		if row.actions != nil {
			window.Opens = cited(row.actions, source)
		}
		result[row.id] = window
	}
	return result
}

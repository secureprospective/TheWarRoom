package leaguerules

import (
	"fmt"
	"reflect"
	"strings"
	"time"
)

func validateValue(v reflect.Value) error {
	if !v.IsValid() {
		return nil
	}
	if v.Kind() == reflect.Struct && strings.HasPrefix(v.Type().Name(), "Value[") {
		state := v.FieldByName("State").Uint()
		payload := v.FieldByName("Value")
		if state == uint64(NeedsRuling) {
			if !payload.IsZero() {
				return fmt.Errorf("an undecided rule must not carry a value")
			}
			return nil
		}
		if state != uint64(IsKnown) {
			return fmt.Errorf("mark this rule as set or Needs a ruling")
		}
		return validateValue(payload)
	}
	if v.CanInterface() {
		if err := validateSpecial(v.Interface()); err != nil {
			return err
		}
	}
	return validateChildren(v)
}
func validateChildren(v reflect.Value) error {
	switch v.Kind() {
	case reflect.Struct:
		for i := range v.NumField() {
			if v.Field(i).CanInterface() {
				if err := validateValue(v.Field(i)); err != nil {
					return err
				}
			}
		}
	case reflect.Map:
		return validateMap(v)
	case reflect.Slice:
		for i := range v.Len() {
			if err := validateValue(v.Index(i)); err != nil {
				return err
			}
		}
	case reflect.Int:
		if v.Int() < 0 {
			return fmt.Errorf("a count cannot be negative")
		}
	case reflect.String:
		return validateMenu(v.Type().Name(), v.String())
	case reflect.Invalid, reflect.Bool, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr, reflect.Float32, reflect.Float64, reflect.Complex64, reflect.Complex128, reflect.Array, reflect.Chan, reflect.Func, reflect.Interface, reflect.Pointer, reflect.UnsafePointer:
		return nil
	}
	return nil
}
func validateSpecial(v any) error {
	switch value := v.(type) {
	case Bounds:
		return validateBounds(value.Min, value.Max)
	case BoundsMoney:
		return validateBounds(value.Min, value.Max)
	case Money:
		if value < 0 {
			return fmt.Errorf("money cannot be negative")
		}
	case time.Duration:
		if value < 0 {
			return fmt.Errorf("a time period cannot be negative")
		}
	case time.Time:
		if value.IsZero() {
			return fmt.Errorf("choose a date or mark it Needs a ruling")
		}
	case MonthDay:
		return validateMonthDay(value)
	case TimeoutPolicy:
		return validateTimeouts(value)

	}
	return nil
}
func validateMonthDay(v MonthDay) error {
	if v.Month.State != IsKnown || v.Day.State != IsKnown {
		return nil
	}
	month, day := v.Month.Value, v.Day.Value
	if month < 1 || month > 12 || day < 1 {
		return fmt.Errorf("choose a valid calendar date")
	}
	maxDays := time.Date(2000, time.Month(month)+1, 0, 0, 0, 0, 0, time.UTC).Day()
	if day > maxDays {
		return fmt.Errorf("choose a valid calendar date")
	}
	return nil
}
func validateRules(r RuleSet) error {
	if r.Trades.DOTSeats.State == IsKnown && (r.Trades.DOTSeats.Value < 4 || r.Trades.DOTSeats.Value > 10) {
		return fmt.Errorf("trade review needs between 4 and 10 seats")
	}
	for _, share := range []Value[Decimal]{r.Cap.IRShare, r.Cap.TaxiShare} {
		if share.State == IsKnown && (share.Value < 0 || share.Value > 10000) {
			return fmt.Errorf("reserve salary shares must be between 0 and 100 percent")
		}
	}
	if r.Cap.Quantum.State == IsKnown && r.Cap.Quantum.Value <= 0 {
		return fmt.Errorf("money rounding increment must be greater than zero")
	}
	if r.Playoffs.StartWeek.State == IsKnown && r.Playoffs.FinalWeek.State == IsKnown &&
		r.Playoffs.StartWeek.Value > r.Playoffs.FinalWeek.Value {
		return fmt.Errorf("first playoff week cannot follow the final week")
	}
	return validateScoringKinds(r.Scoring)
}
func validateMenu(kind, value string) error {
	menus := map[string][]string{
		"Format": {"head_to_head"}, "InitialRosters": {"draft"}, "CarryPolicy": {"all", "keepers"},
		"Penalty": {"zero_top_starter", "zero_slot", "none"}, "CapMode": {"hard", "soft"},
		"NFLStatus":       {"IR", "PUP", "NFI", "Out", "suspended"},
		"ExperienceBasis": {"calendar_years", "career_games"}, "WaiverTrigger": {"cut"},
		"WaiverMethod": {"reverse_standings", "rolling"}, "BidMode": {"blind", "open"},
		"ReviewRoute": {"dot", "commissioner", "none"}, "BoundarySide": {"start_of_day", "end_of_day"},
		"DraftMode": {"slow", "live"}, "ExpiryAction": {"skip", "autopick"},
		"OrderSource": {"standings"}, "OrderPattern": {"linear", "snake"},
		"Criterion": {"win_percentage", "head_to_head", "division_percentage", "conference_percentage",
			"all_play_percentage", "points", "power_rank"},
		"Position":    {"QB", "RB", "WR", "TE", "PK", "DT", "DE", "LB", "CB", "S"},
		"LineupGroup": {"offense", "defense", "total"},
	}
	choices, found := menus[kind]
	if !found {
		return nil
	}
	for _, choice := range choices {
		if value == choice {
			return nil
		}
	}
	return fmt.Errorf("choose an available option")
}

func validateBounds[T int | Money](minimum, maximum Value[T]) error {
	if minimum.State == IsKnown && maximum.State == IsKnown && minimum.Value > maximum.Value {
		return fmt.Errorf("minimum cannot be greater than maximum")
	}
	return nil
}
func validateTimeouts(value TimeoutPolicy) error {
	if value.Unlimited.State == IsKnown && value.Unlimited.Value && value.Count.State == IsKnown {
		return fmt.Errorf("choose unlimited timeouts or a count, not both")
	}
	return nil
}

func validateMap(v reflect.Value) error {
	iter := v.MapRange()
	for iter.Next() {
		if err := validateValue(iter.Key()); err != nil {
			return err
		}
		if err := validateValue(iter.Value()); err != nil {
			return err
		}
	}
	return nil
}

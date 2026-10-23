package leaguerules

import (
	"fmt"
	"reflect"
	"testing"
	"time"
)

func TestResolveMatchesPerChangeFields(t *testing.T) {
	at := testTime()
	seed := SeedChanges(LegacyNFL(), "test-league", 2026, at)
	traffic := append([]Change(nil), seed...)
	for i := range 1000 {
		c := testChange("cap.amount", Known(Money(13000000000+i)))
		c.RecordedAt = at.Add(time.Duration(i+1) * time.Second)
		traffic = append(traffic, c)
	}
	invalid := testChange("cap.amount", Known("wrong type"))
	future := testChange("cap.amount", Known(Money(-1)))
	future.RecordedAt = at.Add(24 * time.Hour)
	other := testChange("cap.amount", Known(Money(1)))
	other.LeagueID = "other-league"
	cases := []struct {
		name    string
		changes []Change
		season  int
		at      time.Time
	}{
		{"all fields", seed, 2026, at},
		{"traffic", traffic, 2026, at.Add(time.Hour)},
		{"earlier instant", traffic, 2026, at.Add(time.Minute)},
		{"invalid type", append(append([]Change(nil), seed...), invalid), 2026, at},
		{"invalid future", []Change{future}, 2026, at},
		{"mixed leagues", append(append([]Change(nil), seed...), other), 2026, at},
		{"other season", []Change{invalid}, 2025, at},
		{"invalid season", seed, 0, at},
		{"invalid instant", seed, 2026, time.Time{}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			before := cloneValue(tc.changes)
			want, wantErr := resolvePerChangeFields(tc.changes, tc.season, tc.at)
			got, gotErr := Resolve(tc.changes, tc.season, tc.at)
			if !reflect.DeepEqual(got, want) || fmt.Sprint(gotErr) != fmt.Sprint(wantErr) {
				t.Fatalf("Resolve differs: got %+v, %v; want %+v, %v", got, gotErr, want, wantErr)
			}
			if !reflect.DeepEqual(tc.changes, before) {
				t.Fatal("resolution changed the input")
			}
		})
	}
}

// resolvePerChangeFields retains the original resolution path as a differential oracle.
func resolvePerChangeFields(changes []Change, season int, at time.Time) (RuleSet, error) {
	result := RuleSet{Season: season}
	if season <= 0 || at.IsZero() {
		return RuleSet{}, fmt.Errorf("choose a season and a reading time")
	}
	for _, c := range changes {
		if result.LeagueID == "" {
			result.LeagueID = c.LeagueID
		}
		if result.LeagueID != c.LeagueID {
			return RuleSet{}, fmt.Errorf("read rules for one league at a time")
		}
		if c.Season != season {
			continue
		}
		if err := c.Validate(); err != nil {
			return RuleSet{}, fmt.Errorf("rule change: %w", err)
		}
		start, err := c.starts()
		if err != nil {
			return RuleSet{}, fmt.Errorf("rule start: %w", err)
		}
		if at.Before(start) || at.Before(c.RecordedAt) {
			continue
		}
		for _, f := range ruleFields() {
			if f.pathName() == c.Field {
				if err := f.apply(&result, c.New); err != nil {
					return RuleSet{}, fmt.Errorf("rule value: %w", err)
				}
				break
			}
		}
	}
	if err := validateRules(result); err != nil {
		return RuleSet{}, fmt.Errorf("league rules: %w", err)
	}
	return result, nil
}

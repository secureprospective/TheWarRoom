package leaguerules

import (
	"reflect"
	"testing"
)

// Every 32-Team Default value is cited and valid as a commissioner setting.
func TestDefault32PresetIsCitedAndValid(t *testing.T) {
	preset := Default32()
	assertCitations(t, reflect.ValueOf(preset))
	for _, c := range SeedChanges(preset, "test-league", 2026, testTime()) {
		if err := c.Validate(); err != nil {
			t.Fatalf("preset %s: %v", c.Field, err)
		}
	}
}

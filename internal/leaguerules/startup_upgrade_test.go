package leaguerules

import (
	"reflect"
	"slices"
	"testing"
)

// A league seeded before the startup draft settings reads them as a new seed would have them, its rounds
// filling its own recorded roster, so it has nothing new to rule on; a seed that recorded them keeps its own.
func TestSeedBeforeStartupSettingsRecordsThem(t *testing.T) {
	recorded := Default32()
	recorded.Draft.StartupRounds = cited(9, "commissioner")
	for _, row := range []struct {
		name   string
		preset RuleSet
		strip  bool
		rounds int
	}{
		{"legacy seeded before", LegacyNFL(), true, 80},
		{"default32 seeded before", Default32(), true, 43},
		{"seeded with them", recorded, false, 9},
	} {
		seed := SeedChanges(row.preset, "old", 2026, testTime())
		if row.strip {
			seed = slices.DeleteFunc(seed, func(c Change) bool { return slices.Contains(startupPaths(), c.Field) })
		}
		before := slices.Clone(seed)
		got := UpgradeSeed(seed)
		if !reflect.DeepEqual(before, seed) || !reflect.DeepEqual(UpgradeSeed(got), got) {
			t.Fatalf("%s: seed mutated or upgrade not idempotent", row.name)
		}
		rules, err := Resolve(got, 2026, testTime())
		if err != nil {
			t.Fatal(err)
		}
		d := rules.Draft
		if d.StartupRounds.State != IsKnown || d.StartupRounds.Value != row.rounds ||
			d.StartupSalary.Value != 33000000 || d.StartupYears.Value != 1 || d.StartupPattern.Value != Snake ||
			d.StartupOrder.Value != RandomOrder || d.StartupList.State != IsKnown {
			t.Fatalf("%s: %+v", row.name, d)
		}
	}
}

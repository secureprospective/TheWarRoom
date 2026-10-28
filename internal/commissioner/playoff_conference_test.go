package commissioner

import (
	"fmt"
	"maps"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

// sizedLedger is a native league of count franchises with every rule waiting for a ruling.
func sizedLedger(t *testing.T, count int) *ledgerBuilder {
	t.Helper()
	create, err := RawCommand{League: rolloverLeague, Command: Create, Args: RawArgs{Name: "Ours", Preset: PresetEmpty,
		Season: 2026, FranchiseCount: count}, Request: "create", Effective: "now", Visibility: "league"}.Prepared()
	if err != nil {
		t.Fatal(err)
	}
	b := &ledgerBuilder{t: t}
	b.add(create, seasonMoment(1))
	return b
}

// heldSnapshot is the imported start with each held division's conference: div.0–3 AFC, div.4–7 NFC.
func heldSnapshot() *RawNativeSnapshot {
	snap := ledgerSnapshot()
	snap.Conferences = map[string]string{}
	for i := range 8 {
		snap.Conferences[fmt.Sprintf("div.%d", i)] = map[bool]string{true: "AFC", false: "NFC"}[i < 4]
	}
	return snap
}

func seedHeld(b *ledgerBuilder, day int) {
	b.add(RawCommand{League: rolloverLeague, Command: NativeSeed, Args: RawArgs{Native: heldSnapshot()},
		Request: "seed", Effective: "now", Visibility: "league"}, seasonMoment(day))
}

// splitLayout puts the first half of the franchises in one conference and the rest in the other.
func splitLayout(count int, first, second string) map[string]string {
	of := map[string]string{}
	for i := 1; i <= count; i++ {
		of[fmt.Sprintf("franchise.%02d", i)] = map[bool]string{true: first, false: second}[i <= count/2]
	}
	return of
}

func TestConferencesAreALeagueSetting(t *testing.T) {
	for _, tc := range []struct {
		name  string
		build func(t *testing.T) *ledgerBuilder
		want  map[string]string
		words string
	}{
		{"native two conferences settle from the setting", func(t *testing.T) *ledgerBuilder {
			b := sizedLedger(t, 12)
			b.set("league.conferences", `2`, "now", seasonMoment(2))
			b.set("league.alignment", alignmentJSON(12, "East", "West"), "now", seasonMoment(2))
			b.set("league.divisionConferences", `{"East":"Eastern","West":"Western"}`, "now", seasonMoment(2))
			return b
		}, splitLayout(12, "Eastern", "Western"), ""},
		{"native without the setting still needs a ruling", func(t *testing.T) *ledgerBuilder {
			b := sizedLedger(t, 12)
			b.set("league.conferences", `2`, "now", seasonMoment(2))
			b.set("league.alignment", alignmentJSON(12, "East", "West"), "now", seasonMoment(2))
			return b
		}, nil, `Needs a ruling: which conference the division "East" is in`},
		{"imported league reads its held conferences", func(t *testing.T) *ledgerBuilder {
			b := newLedger(t, PresetLegacy)
			seedHeld(b, 2)
			return b
		}, splitLayout(32, "AFC", "NFC"), ""},
		{"a later setting overrides the held conferences", func(t *testing.T) *ledgerBuilder {
			b := newLedger(t, PresetLegacy)
			seedHeld(b, 2)
			b.set("league.divisionConferences", conferencesJSON("NFC", "AFC"), "now", seasonMoment(3))
			return b
		}, splitLayout(32, "NFC", "AFC"), ""},
		{"a setting made before the start is kept over the held conferences", func(t *testing.T) *ledgerBuilder {
			b := newLedger(t, PresetLegacy)
			b.set("league.divisionConferences", conferencesJSON("NFC", "AFC"), "now", seasonMoment(2))
			seedHeld(b, 3)
			return b
		}, splitLayout(32, "NFC", "AFC"), ""},
		{"a fresh 32-Team Default league carries the NFL layout", func(t *testing.T) *ledgerBuilder {
			return newLedger(t, PresetDefault32)
		}, splitLayout(32, "AFC", "NFC"), ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			b := tc.build(t)
			rules, err := RulesNow(b.records, seasonMoment(9))
			if err != nil {
				t.Fatal(err)
			}
			got := conferencesFor(b.records, rules, b.records[0].Command.Args.Franchises)
			if got.Words != tc.words || !maps.Equal(got.Of, tc.want) && tc.want != nil {
				t.Fatalf("got %q %v, want %q %v", got.Words, got.Of, tc.words, tc.want)
			}
		})
	}
}

// TestImportedConferencesReadAsBefore pins an imported league to the layout it read from its held
// conferences alone, before the setting existed.
func TestImportedConferencesReadAsBefore(t *testing.T) {
	b := newLedger(t, PresetLegacy)
	seedHeld(b, 2)
	rules, err := RulesNow(b.records, seasonMoment(9))
	if err != nil {
		t.Fatal(err)
	}
	carried := rules.League.DivisionConferences
	if carried.State != leaguerules.IsKnown || !maps.Equal(carried.Value, heldSnapshot().Conferences) ||
		carried.Citation != carriedConferences {
		t.Fatalf("the imported conferences are not carried as a setting: %+v", carried)
	}
	franchises := b.records[0].Command.Args.Franchises
	withSetting := conferencesFor(b.records, rules, franchises)
	rules.League.DivisionConferences = leaguerules.Needs[map[string]string]("before the setting")
	heldOnly := conferencesFor(b.records, rules, franchises)
	if !reflect.DeepEqual(withSetting, heldOnly) || heldOnly.Words != "" {
		t.Fatalf("setting %+v, held only %+v", withSetting, heldOnly)
	}
}

func TestNoDivisionIsNamedNeedsARuling(t *testing.T) {
	for _, tc := range []struct {
		name      string
		alignment string
		want      []string
	}{
		{"no alignment copies no division", "", slices.Repeat([]string{""}, 4)},
		{"a settled alignment is copied", alignmentJSON(4, "East", "West"), []string{"East", "East", "West", "West"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			b := sizedLedger(t, 4)
			if tc.alignment != "" {
				b.set("league.alignment", tc.alignment, "now", seasonMoment(2))
			}
			seed, err := NativeStart(rolloverLeague, b.records, seasonMoment(3))
			if err != nil {
				t.Fatal(err)
			}
			got := []string{}
			for _, f := range seed.Franchises {
				got = append(got, f.Division)
			}
			if !slices.Equal(got, tc.want) {
				t.Fatalf("divisions %q, want %q", got, tc.want)
			}
		})
	}
	// An older start stored the words themselves; they are never carried into the alignment.
	b := sizedLedger(t, 4)
	seed, err := NativeStart(rolloverLeague, b.records, seasonMoment(3))
	if err != nil {
		t.Fatal(err)
	}
	for i := range seed.Franchises {
		seed.Franchises[i].Division = unsettledDivision
	}
	b.add(RawCommand{League: rolloverLeague, Command: NativeSeed, Args: RawArgs{Native: &seed},
		Request: "seed", Effective: "now", Visibility: "league"}, seasonMoment(3))
	rules, err := RulesNow(b.records, seasonMoment(4))
	if err != nil {
		t.Fatal(err)
	}
	if rules.League.Alignment.State == leaguerules.IsKnown {
		t.Fatalf("an unsettled division became an alignment: %v", rules.League.Alignment.Value)
	}
}

// alignmentJSON splits count franchises into two named divisions, first half and second half.
func alignmentJSON(count int, first, second string) string {
	parts := []string{}
	for id, division := range splitLayout(count, first, second) {
		parts = append(parts, fmt.Sprintf("%q:%q", id, division))
	}
	slices.Sort(parts)
	return "{" + strings.Join(parts, ",") + "}"
}

// conferencesJSON names the conference of the held divisions div.0–3 and div.4–7.
func conferencesJSON(first, second string) string {
	parts := []string{}
	for i := range 8 {
		parts = append(parts, fmt.Sprintf(`"div.%d":%q`, i, map[bool]string{true: first, false: second}[i < 4]))
	}
	return "{" + strings.Join(parts, ",") + "}"
}

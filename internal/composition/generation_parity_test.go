package composition

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	catalog "github.com/secureprospective/TheWarRoom/internal/commandcatalog"
)

// TestGenerationRegistryMatchesDesktopRegistry keeps catalog.GenerationRegistry (what nativegen reads, T710 PR-A)
// in step with the live desktop registry until this package is removed (PR-D): same entries, same desktop
// exposures, same argument names, types and optionality. It goes when composition goes.
func TestGenerationRegistryMatchesDesktopRegistry(t *testing.T) {
	live, err := CommandRegistry()
	if err != nil {
		t.Fatal(err)
	}
	generated, err := catalog.GenerationRegistry()
	if err != nil {
		t.Fatal(err)
	}
	want, got := registrySurface(live), registrySurface(generated)
	for _, line := range want {
		if !slices.Contains(got, line) {
			t.Errorf("generation registry is missing %s", line)
		}
	}
	for _, line := range got {
		if !slices.Contains(want, line) {
			t.Errorf("generation registry has extra %s", line)
		}
	}
}

func registrySurface(registry *catalog.Registry) []string {
	var out []string
	for _, entry := range registry.Entries() {
		if assistOnly(entry) {
			continue // the desktop MFL import assistant (league.migrate.*): no generator reads the Assist door
		}
		out = append(out, fmt.Sprintf("entry %s cli=%t native=%d mfl=%d browser=%t", entry.ID, entry.CLI,
			len(entry.Native), len(entry.MFL), entry.Browser != nil))
		for _, door := range entry.Desktop {
			out = append(out, fmt.Sprintf("desktop %s %s.%s.%s(%s)", entry.ID, door.Feature, door.Command.ID, door.Command.Method,
				fieldSurface(door.Binding.Arguments.Fields)))
		}
	}
	slices.Sort(out)
	return out
}

func assistOnly(entry catalog.Entry) bool {
	return entry.Assist != nil && !entry.CLI && entry.Native == nil && entry.MFL == nil && entry.Console == nil &&
		len(entry.Desktop) == 0 && entry.Browser == nil && entry.Workflow == nil
}

func fieldSurface(fields []catalog.Field) string {
	parts := make([]string, 0, len(fields))
	for _, field := range fields {
		parts = append(parts, fmt.Sprintf("%s:%s:%t", field.Name, field.Schema.Type, field.Optional))
	}
	return strings.Join(parts, " ")
}

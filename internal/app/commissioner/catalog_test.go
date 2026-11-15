package commissioner

import (
	"reflect"
	"slices"
	"testing"

	catalog "github.com/secureprospective/TheWarRoom/internal/commandcatalog"
	"github.com/secureprospective/TheWarRoom/internal/commissioner"
)

func TestDraftPickCatalogMatchesOwnerInput(t *testing.T) {
	for _, command := range NativeCommands() {
		if command.ID != commissioner.DraftPick {
			continue
		}
		want := catalog.Shape(reflect.TypeFor[commissioner.DraftPickInput](), "DraftPickInput.Validate")
		if !reflect.DeepEqual(command.Arguments, want) || !slices.Equal(command.Modes, []catalog.Mode{catalog.Execute}) {
			t.Fatal("draft pick must accept an owner choice, not a computed contract")
		}
	}
}

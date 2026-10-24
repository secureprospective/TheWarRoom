package commissioner

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/app/catalog"
	"github.com/secureprospective/TheWarRoom/internal/commissioner"
)

func TestNativeCatalogPublication(t *testing.T) {
	commands, err := catalog.Publication(NativeCommands())
	if err != nil {
		t.Fatal(err)
	}
	methods := map[string]bool{}
	mutations := map[string]bool{}
	for _, command := range commands {
		methods[command.Method] = true
		if slices.Contains(command.Modes, catalog.Read) {
			if command.Gravity != "G0" || command.Undo != "none" {
				t.Fatalf("read can change state: %+v", command)
			}
			continue
		}
		mutations[command.ID] = true
		if command.Permission != "command:"+command.ID || !slices.Equal(command.Modes, []catalog.Mode{catalog.Preview, catalog.Execute}) {
			t.Fatalf("mutation lost domain admission or modes: %+v", command)
		}
		if command.ID != commissioner.NativeSeed {
			assertMutationShape(t, command)
		}
	}
	// Generic dispatch/preview doors cannot be published as commands with one blanket role.
	// The desktop's local sign-in links an identity; it is not a league command a server could publish.
	generic := map[string]bool{"TargetCommissionerExecute": true, "TargetCommissionerPreview": true, "TargetNativeScorePreview": true,
		"TargetCommissionerSeat": true, "TargetCommissionerSeatSignIn": true}
	typ := reflect.TypeFor[*API]()
	for i := range typ.NumMethod() {
		name := typ.Method(i).Name
		if !methods[name] && !generic[name] {
			t.Errorf("native method %s has no logical catalog entry", name)
		}
	}
	domainIDs := domainCommandIDs(t)
	for id := range domainIDs {
		if !mutations[id] {
			t.Errorf("native mutation %s is missing", id)
		}
	}
	for id := range mutations {
		if !domainIDs[id] {
			t.Errorf("unrecognized native mutation %s", id)
		}
	}
	if len(mutations) != 99 {
		t.Fatalf("mutation inventory changed: got %d; review the new family's metadata", len(mutations))
	}
}

func assertMutationShape(t *testing.T, command catalog.Command) {
	t.Helper()
	foundCommand, foundArgs := false, false
	for _, field := range command.Arguments.Fields {
		if field.Name == "command" {
			foundCommand = true
			if !slices.Equal(field.Schema.Enum, []string{command.ID}) {
				t.Errorf("%s permits another command ID", command.ID)
			}
		}
		if field.Name != "args" {
			continue
		}
		foundArgs = true
		for _, arg := range field.Schema.Fields {
			if strings.HasSuffix(arg.Name, "Outcome") || arg.Name == "native" || arg.Name == "seed" || arg.Name == "franchises" {
				t.Errorf("%s advertises server-owned %s", command.ID, arg.Name)
			}
		}
	}
	if !foundCommand || !foundArgs {
		t.Errorf("%s is missing its command envelope", command.ID)
	}
}

// Independent inventory: every domain-recognized string literal must have a declaration. This
// catches newly implemented families without teaching the catalog test their names in advance.
func domainCommandIDs(t *testing.T) map[string]bool {
	t.Helper()
	ids := map[string]bool{commissioner.RulesSet: true, commissioner.ContractActionSet: true}
	entries, err := os.ReadDir("../../commissioner")
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		file, err := parser.ParseFile(token.NewFileSet(), filepath.Join("../../commissioner", entry.Name()), nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		ast.Inspect(file, func(node ast.Node) bool {
			literal, ok := node.(*ast.BasicLit)
			if !ok || literal.Kind != token.STRING {
				return true
			}
			value, err := strconv.Unquote(literal.Value)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := commissioner.DefinitionFor(value); err == nil {
				ids[value] = true
			}
			return true
		})
	}
	return ids
}

func TestDesktopGenericDoorsStayUnpublishable(t *testing.T) {
	for _, command := range Commands() {
		if slices.Contains([]string{"TargetCommissionerExecute", "TargetCommissionerPreview", "TargetNativeScorePreview"}, command.Method) {
			if err := command.ValidatePublication(); err == nil {
				t.Errorf("generic door %s was publishable", command.Method)
			}
		}
	}
	if published, err := catalog.Publication(Commands()); err == nil || published != nil {
		t.Fatal("mixed desktop inventory was published")
	}
}

func TestNativeCatalogRetainsSpecialSafeguards(t *testing.T) {
	commands := map[string]catalog.Command{}
	for _, command := range NativeCommands() {
		commands[command.ID] = command
	}
	for _, id := range []string{commissioner.RulesSet, commissioner.ContractActionSet} {
		if commands[id].Gravity != "argument-dependent" {
			t.Fatalf("%s lost field-dependent safeguards", id)
		}
	}
	seed := commands[commissioner.NativeSeed]
	if seed.Availability != "desktop" || seed.Undo != "none" || seed.Facts != "held-mirror" {
		t.Fatalf("held seed was presented as a hosted import: %+v", seed)
	}
	for _, id := range []string{commissioner.BidPlace, commissioner.LineupSet, commissioner.DraftPick, commissioner.TradeAccept} {
		if commands[id].Safeguards != "RawCommand.EffectiveDefinition+MissingReason" {
			t.Errorf("%s lost argument-dependent reason/confirmation checks", id)
		}
	}
}

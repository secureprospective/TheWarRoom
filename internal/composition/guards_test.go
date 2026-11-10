package composition

import (
	"bytes"
	"fmt"
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/app/commissioner"
	controlapp "github.com/secureprospective/TheWarRoom/internal/app/control"
	"github.com/secureprospective/TheWarRoom/internal/app/draft"
	marketapp "github.com/secureprospective/TheWarRoom/internal/app/market"
	contractsapp "github.com/secureprospective/TheWarRoom/internal/app/mflref/contracts"
	"github.com/secureprospective/TheWarRoom/internal/app/mflref/trades"
	nowapp "github.com/secureprospective/TheWarRoom/internal/app/now"
	"github.com/secureprospective/TheWarRoom/internal/app/offers"
	platformapp "github.com/secureprospective/TheWarRoom/internal/app/platform"
	raceapp "github.com/secureprospective/TheWarRoom/internal/app/race"
	"github.com/secureprospective/TheWarRoom/internal/app/research"
	"github.com/secureprospective/TheWarRoom/internal/app/sources"
	catalog "github.com/secureprospective/TheWarRoom/internal/commandcatalog"
)

func TestRunFlag_UnknownArgumentNeverOpensAWindow(t *testing.T) {
	t.Parallel()
	for _, args := range [][]string{{"-version", "x"}, {"--version"}, {"-probe", "-version"}, {"run"}} {
		var out, errOut bytes.Buffer
		if code := runFlag(args, &out, &errOut); code != 2 || !strings.Contains(errOut.String(), "usage") {
			t.Errorf("runFlag(%q) = %d, stderr %q; want 2 with usage", args, code, errOut.String())
		}
	}
	var out, errOut bytes.Buffer
	if code := runFlag([]string{"-version"}, &out, &errOut); code != 0 || strings.TrimSpace(out.String()) != buildLabel() {
		t.Errorf("-version = %d, stdout %q; want 0 and %q", code, out.String(), buildLabel())
	}
}

func TestBindingManifest(t *testing.T) {
	t.Parallel()
	got, problems := bindingManifest(bindings(NewApp()))
	if len(problems) > 0 {
		t.Fatal(strings.Join(problems, "\n"))
	}
	slices.Sort(got)
	want, err := os.ReadFile("testdata/bindings/manifest.txt")
	if err != nil {
		t.Fatal(err)
	}
	if actual := strings.Join(got, "\n") + "\n"; actual != string(want) {
		t.Fatalf("binding manifest changed; review the surface before updating the golden\ngot:\n%s", actual)
	}
}

func bindingManifest(bound []interface{}) ([]string, []string) {
	catalogs := boundCatalogs()
	seen := map[reflect.Type]bool{}
	var got, problems []string
	for _, value := range bound {
		typ := reflect.TypeOf(value)
		commands, feature := catalogs[typ]
		if !feature {
			problems = append(problems, fmt.Sprintf("unexpected bound type %v", typ))
			continue
		}
		if seen[typ] {
			problems = append(problems, fmt.Sprintf("duplicate bound type %v", typ))
		}
		seen[typ] = true
		methods := map[string]bool{}
		for _, command := range commands {
			if command.Role == catalog.Unset || command.Gravity == catalog.Unset {
				problems = append(problems, "catalog method "+command.Method+" has unset role or gravity")
			}
			if methods[command.Method] {
				problems = append(problems, "duplicate catalog method "+command.Method)
			}
			methods[command.Method] = true
		}
		prefix := typ.Elem().String() + "."
		for i := range typ.NumMethod() {
			name := typ.Method(i).Name
			got = append(got, prefix+name)
			if feature {
				if !methods[name] {
					problems = append(problems, "bound method "+prefix+name+" has no catalog entry")
				}
				delete(methods, name)
			}
		}
		for name := range methods {
			problems = append(problems, "catalog method "+name+" is not bound")
		}
	}
	for typ := range catalogs {
		if !seen[typ] {
			problems = append(problems, fmt.Sprintf("missing bound type %v", typ))
		}
	}
	return got, problems
}

func TestBindingManifestRejectsHeld(t *testing.T) {
	for _, held := range []interface{}{&research.Held{}, &sources.Held{}} {
		bound := append(bindings(NewApp()), held)
		_, problems := bindingManifest(bound)
		want := fmt.Sprintf("unexpected bound type %T", held)
		if len(problems) != 1 || problems[0] != want {
			t.Fatalf("directly bound holder accepted: %v", problems)
		}
	}
}

func boundCatalogs() map[reflect.Type][]catalog.Command {
	return map[reflect.Type][]catalog.Command{
		reflect.TypeFor[*commissioner.API](): commissioner.Commands(),
		reflect.TypeFor[*controlapp.API]():   controlapp.Commands(),
		reflect.TypeFor[*contractsapp.API](): contractsapp.Commands(),
		reflect.TypeFor[*draft.API]():        draft.Commands(),
		reflect.TypeFor[*marketapp.API]():    marketapp.Commands(),
		reflect.TypeFor[*nowapp.API]():       nowapp.Commands(),
		reflect.TypeFor[*offers.API]():       offers.Commands(),
		reflect.TypeFor[*platformapp.API]():  platformapp.Commands(),
		reflect.TypeFor[*raceapp.API]():      raceapp.Commands(),
		reflect.TypeFor[*research.API]():     research.Commands(),
		reflect.TypeFor[*sources.API]():      sources.Commands(),
		reflect.TypeFor[*trades.API]():       trades.Commands(),
	}
}

package composition

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"sort"
	"testing"

	archive "github.com/secureprospective/TheWarRoom/internal/app/archive"
	commissioner "github.com/secureprospective/TheWarRoom/internal/app/commissioner"
	comms "github.com/secureprospective/TheWarRoom/internal/app/comms"
	contracts "github.com/secureprospective/TheWarRoom/internal/app/contracts"
	control "github.com/secureprospective/TheWarRoom/internal/app/control"
	draft "github.com/secureprospective/TheWarRoom/internal/app/draft"
	market "github.com/secureprospective/TheWarRoom/internal/app/market"
	now "github.com/secureprospective/TheWarRoom/internal/app/now"
	offers "github.com/secureprospective/TheWarRoom/internal/app/offers"
	platform "github.com/secureprospective/TheWarRoom/internal/app/platform"
	race "github.com/secureprospective/TheWarRoom/internal/app/race"
	research "github.com/secureprospective/TheWarRoom/internal/app/research"
	sources "github.com/secureprospective/TheWarRoom/internal/app/sources"
	trades "github.com/secureprospective/TheWarRoom/internal/app/trades"
	catalog "github.com/secureprospective/TheWarRoom/internal/commandcatalog"
)

// Frozen before moving declarations; schema hashes also protect the Raw JSON contract.
type doorBefore struct {
	Door, Feature, ID, Method, Role, Gravity, Availability, Permission, Schema string
	Modes                                                                      []catalog.Mode
}

func legacyDoors() []doorBefore {
	out := []doorBefore{}
	add := func(door, feature string, cs []catalog.Command) {
		for _, c := range cs {
			b, _ := json.Marshal(c.Arguments)
			out = append(out, doorBefore{door, feature, c.ID, c.Method, c.Role, c.Gravity, c.Availability, c.Permission, fmt.Sprintf("%x", sha256.Sum256(b)), c.Modes})
		}
	}
	add("native", "commissioner", commissioner.NativeCommands())
	add("mfl", "offers", offers.MFLCommands())
	add("wails", "archive", archive.Commands())
	add("wails", "commissioner", commissioner.Commands())
	add("wails", "comms", comms.Commands())
	add("wails", "contracts", contracts.Commands())
	add("wails", "control", control.Commands())
	add("wails", "draft", draft.Commands())
	add("wails", "market", market.Commands())
	add("wails", "now", now.Commands())
	add("wails", "offers", offers.Commands())
	add("wails", "platform", platform.Commands())
	add("wails", "race", race.Commands())
	add("wails", "research", research.Commands())
	add("wails", "sources", sources.Commands())
	add("wails", "trades", trades.Commands())
	return out
}
func registryDoors(registry *catalog.Registry) []doorBefore {
	out := []doorBefore{}
	add := func(door, feature string, c catalog.Command) {
		body, _ := json.Marshal(c.Arguments)
		out = append(out, doorBefore{door, feature, c.ID, c.Method, c.Role, c.Gravity, c.Availability, c.Permission, fmt.Sprintf("%x", sha256.Sum256(body)), c.Modes})
	}
	for _, entry := range registry.Entries() {
		if entry.Native != nil {
			add("native", "commissioner", entry.Definition)
		}
		if entry.MFL != nil {
			add("mfl", "offers", entry.Definition)
		}
		for _, desktop := range entry.Desktop {
			add("wails", desktop.Feature, desktop.Command)
		}
	}
	return out
}

func TestFrozenDoorMetadata(t *testing.T) {
	body, err := os.ReadFile("internal/commandcatalog/testdata/doors-before.json")
	if err != nil {
		t.Fatal(err)
	}
	var before []doorBefore
	if err := json.Unmarshal(body, &before); err != nil {
		t.Fatal(err)
	}
	app := NewApp()
	apis := newCommandAPIs(app)
	registry, err := catalog.NewRegistry(apis.registryInputs(app))
	if err != nil {
		t.Fatal(err)
	}
	sortDoors(before)
	for name, after := range map[string][]doorBefore{"feature views": legacyDoors(), "executable registry": registryDoors(registry)} {
		sortDoors(after)
		if len(before) != 275 || len(after) != len(before) {
			t.Fatalf("%s inventory changed: %d -> %d", name, len(before), len(after))
		}
		for i, want := range before {
			if !reflect.DeepEqual(want, after[i]) {
				t.Errorf("%s %s/%s/%s changed: before=%+v after=%+v", name, want.Door, want.Feature, want.Method, want, after[i])
			}
		}
	}
	counts := map[string]int{}
	for _, e := range registry.Entries() {
		if e.Workflow != nil {
			counts["workflow"]++
		}
		if e.Browser != nil {
			counts["browser"]++
		}
	}
	if counts["workflow"] != 42 || counts["browser"] != 61 {
		t.Fatalf("other exposure changed: %v", counts)
	}
}
func sortDoors(rows []doorBefore) {
	sort.Slice(rows, func(i, j int) bool {
		a, b := rows[i], rows[j]
		return a.Door+a.Feature+a.Method+a.ID < b.Door+b.Feature+b.Method+b.ID
	})
}

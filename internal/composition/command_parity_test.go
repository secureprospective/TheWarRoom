package composition

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"sort"
	"testing"

	commissioner "github.com/secureprospective/TheWarRoom/internal/app/commissioner"
	control "github.com/secureprospective/TheWarRoom/internal/app/control"
	draft "github.com/secureprospective/TheWarRoom/internal/app/draft"
	market "github.com/secureprospective/TheWarRoom/internal/app/market"
	contracts "github.com/secureprospective/TheWarRoom/internal/app/mflref/contracts"
	trades "github.com/secureprospective/TheWarRoom/internal/app/mflref/trades"
	now "github.com/secureprospective/TheWarRoom/internal/app/now"
	offers "github.com/secureprospective/TheWarRoom/internal/app/offers"
	platform "github.com/secureprospective/TheWarRoom/internal/app/platform"
	race "github.com/secureprospective/TheWarRoom/internal/app/race"
	research "github.com/secureprospective/TheWarRoom/internal/app/research"
	sources "github.com/secureprospective/TheWarRoom/internal/app/sources"
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
	add("wails", "commissioner", commissioner.Commands())
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
		if entry.MFL != nil && entry.Definition.Method != "TargetDraftLineup" && entry.Definition.Method != "TargetDraftTradeAccept" {
			add("mfl", "offers", entry.Definition)
		}
		for _, desktop := range entry.Desktop {
			add("wails", desktop.Feature, desktop.Command)
		}
	}
	return out
}

func TestFrozenDoorMetadata(t *testing.T) {
	path := "internal/commandcatalog/testdata/doors-before.json"
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var before []doorBefore
	if err := json.Unmarshal(body, &before); err != nil {
		t.Fatal(err)
	}
	if os.Getenv("TWR_WRITE_WIRE") == "1" {
		writeDoorSchemas(t, before)
	}
	for i := range before {
		if before[i].Door != "wails" {
			continue
		}
		switch before[i].Feature + "." + before[i].Method {
		case "contracts.TargetDraftLineup":
			before[i].ID = "mfl.lineup.set"
		case "trades.TargetDraftTradeAccept":
			before[i].ID = "mfl.trade.accept"
		}
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
		if len(before) != 294 || len(after) != len(before) {
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
	if counts["workflow"] != 93 || counts["browser"] != 58 {
		t.Fatalf("other exposure changed: %v", counts)
	}
}
func writeDoorSchemas(t *testing.T, before []doorBefore) {
	t.Helper()
	const path = "internal/commandcatalog/testdata/doors-before.json"
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, current := range legacyDoors() {
		for i, held := range before {
			if held.Door == current.Door && held.Feature == current.Feature && held.ID == current.ID && held.Method == current.Method {
				body = bytes.ReplaceAll(body, []byte(held.Schema), []byte(current.Schema))
				before[i].Schema = current.Schema
			}
		}
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			t.Error(err)
		}
	}()
	if _, err := file.Write(body); err != nil {
		t.Fatal(err)
	}
}

func sortDoors(rows []doorBefore) {
	sort.Slice(rows, func(i, j int) bool {
		a, b := rows[i], rows[j]
		return a.Door+a.Feature+a.Method+a.ID < b.Door+b.Feature+b.Method+b.ID
	})
}

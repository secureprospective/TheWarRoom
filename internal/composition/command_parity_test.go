package composition

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"testing"

	archive "github.com/secureprospective/TheWarRoom/internal/app/archive"
	"github.com/secureprospective/TheWarRoom/internal/app/catalog"
	commissioner "github.com/secureprospective/TheWarRoom/internal/app/commissioner"
	comms "github.com/secureprospective/TheWarRoom/internal/app/comms"
	contracts "github.com/secureprospective/TheWarRoom/internal/app/contracts"
	control "github.com/secureprospective/TheWarRoom/internal/app/control"
	counterparties "github.com/secureprospective/TheWarRoom/internal/app/counterparties"
	draft "github.com/secureprospective/TheWarRoom/internal/app/draft"
	market "github.com/secureprospective/TheWarRoom/internal/app/market"
	now "github.com/secureprospective/TheWarRoom/internal/app/now"
	offers "github.com/secureprospective/TheWarRoom/internal/app/offers"
	platform "github.com/secureprospective/TheWarRoom/internal/app/platform"
	race "github.com/secureprospective/TheWarRoom/internal/app/race"
	research "github.com/secureprospective/TheWarRoom/internal/app/research"
	sources "github.com/secureprospective/TheWarRoom/internal/app/sources"
	trades "github.com/secureprospective/TheWarRoom/internal/app/trades"
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
	add("wails", "counterparties", counterparties.Commands())
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
func TestFrozenDoorMetadata(t *testing.T) {
	body, err := os.ReadFile("internal/app/catalog/testdata/doors-before.json")
	if err != nil {
		t.Fatal(err)
	}
	var before []doorBefore
	if err := json.Unmarshal(body, &before); err != nil {
		t.Fatal(err)
	}
	after := legacyDoors()
	if len(before) != 281 || len(after) != len(before) {
		t.Fatalf("inventory changed: %d -> %d", len(before), len(after))
	}
	for i, want := range before {
		if !reflect.DeepEqual(want, after[i]) {
			t.Errorf("%s/%s/%s changed: before=%+v after=%+v", want.Door, want.Feature, want.Method, want, after[i])
		}
	}
}

package catalog

import (
	"reflect"
	"testing"
)

func completeCommand() Command {
	return Command{ID: "test.read", Method: "TestRead", Version: "1", Role: "gm", Gravity: "G0",
		Availability: "both", Arguments: Shape(reflect.TypeFor[struct {
			League string `json:"league"`
		}](), "RawScope.Validate"),
		Permission: "league-view", Modes: []Mode{Read}, Undo: "none", Facts: "ledger", Safeguards: "none"}
}

func TestPublicationRejectsIncompleteEntriesAtomically(t *testing.T) {
	tests := map[string]func(*Command){
		"id":                   func(c *Command) { c.ID = Unset },
		"method":               func(c *Command) { c.Method = "" },
		"version":              func(c *Command) { c.Version = Unset },
		"zero version":         func(c *Command) { c.Version = "0" },
		"noncanonical version": func(c *Command) { c.Version = "01" },
		"role":                 func(c *Command) { c.Role = Unset },
		"gravity":              func(c *Command) { c.Gravity = Unset },
		"availability":         func(c *Command) { c.Availability = Unset },
		"permission":           func(c *Command) { c.Permission = Unset },
		"undo":                 func(c *Command) { c.Undo = Unset },
		"facts":                func(c *Command) { c.Facts = Unset },
		"safeguards":           func(c *Command) { c.Safeguards = Unset },
		"schema":               func(c *Command) { c.Arguments.Type = Unset },
		"validator":            func(c *Command) { c.Arguments.Validator = "" },
		"nested schema":        func(c *Command) { c.Arguments.Fields[0].Schema.Type = Unset },
		"no modes":             func(c *Command) { c.Modes = nil },
		"unknown mode":         func(c *Command) { c.Modes = []Mode{"admin"} },
		"duplicate mode":       func(c *Command) { c.Modes = []Mode{Read, Read} },
		"read execute":         func(c *Command) { c.Modes = []Mode{Read, Execute} },
	}
	for name, breakEntry := range tests {
		t.Run(name, func(t *testing.T) {
			invalid := completeCommand()
			breakEntry(&invalid)
			if published, err := Publication([]Command{completeCommand(), invalid}); err == nil || published != nil {
				t.Fatalf("incomplete catalog published: %v, %v", published, err)
			}
		})
	}
	if published, err := Publication([]Command{{ID: Unset}}); err == nil || published != nil {
		t.Fatal("deliberately unset entry was published")
	}
}

func TestPublicationAcceptsCompleteAndRejectsDuplicateIDs(t *testing.T) {
	command := completeCommand()
	if published, err := Publication([]Command{command}); err != nil || len(published) != 1 {
		t.Fatalf("complete entry refused: %v", err)
	}
	if published, err := Publication([]Command{command, command}); err == nil || published != nil {
		t.Fatal("duplicate ID was published")
	}
}

func TestShapePreservesJSONStructure(t *testing.T) {
	type embedded struct {
		League string `json:"league"`
	}
	type raw struct {
		embedded
		Names   []string        `json:"names,omitempty"`
		Count   *int            `json:"count"`
		Groups  map[string]bool `json:"groups"`
		Ignored string          `json:"-"`
	}
	// Exported embedded fields are flattened just as encoding/json flattens them.
	schema := Shape(reflect.TypeFor[raw](), "Raw.Validate")
	if err := schema.Validate(); err != nil {
		t.Fatal(err)
	}
	fields := map[string]Field{}
	for _, field := range schema.Fields {
		fields[field.Name] = field
	}
	if len(fields) != 4 || fields["league"].Schema.Type != "string" || !fields["names"].Optional || fields["names"].Schema.Items.Type != "string" ||
		!fields["count"].Schema.Nullable || fields["groups"].Schema.Items.Type != "boolean" {
		t.Fatalf("wrong JSON shape: %+v", fields)
	}
	for _, typ := range []reflect.Type{reflect.TypeFor[chan int](), reflect.TypeFor[map[int]string]()} {
		if err := Shape(typ, "Raw.Validate").Validate(); err == nil {
			t.Fatalf("unsupported shape accepted: %v", typ)
		}
	}
}

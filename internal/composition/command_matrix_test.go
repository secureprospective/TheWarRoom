package composition

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"

	catalog "github.com/secureprospective/TheWarRoom/internal/commandcatalog"
)

type parityCell struct {
	Role, Permission string
	Modes            []catalog.Mode
	Allowed          bool
}

type parityMatrix map[string]parityCell

func (m parityMatrix) add(id, door, role, permission string, modes []catalog.Mode) {
	for _, held := range []string{"gm", "commish", "admin", "operator"} {
		// Domain admission is argument-aware, not a display-role grant.
		allowed := role == "domain" || slices.Contains(strings.Split(role, ","), held)
		m[id+"/"+door+"/"+held] = parityCell{role, permission, modes, allowed}
	}
}

func readParityFixture[T any](t *testing.T, path string) T {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var value T
	if err := json.Unmarshal(body, &value); err != nil {
		t.Fatal(err)
	}
	return value
}

func frozenCommandMatrix(t *testing.T) parityMatrix {
	t.Helper()
	rows := readParityFixture[[]doorBefore](t, "internal/commandcatalog/testdata/doors-before.json")
	browser := readParityFixture[[]catalog.BrowserCommand](t, "frontend/src/app/commands/browser-before.json")
	workflow := readParityFixture[[]struct {
		Command string
		Modes   []catalog.Mode
	}](t, "internal/serverhttp/testdata/workflow-before.json")
	m := parityMatrix{}
	for _, row := range readParityFixture[[]doorBefore](t, "internal/commandcatalog/testdata/operator-before.json") {
		m.add(row.ID, row.Door, row.Role, row.Permission, row.Modes)
	}
	shared := map[string]string{}
	native := map[string]doorBefore{}
	for _, row := range rows {
		if row.Door != "wails" {
			shared[row.ID] = row.Schema
			if row.Door == "native" {
				native[row.ID] = row
			}
		}
	}
	for _, row := range rows {
		id := row.ID
		if row.Door == "wails" && shared[id] != row.Schema {
			id = "desktop." + row.Feature + "." + row.Method
		}
		m.add(id, row.Door, row.Role, row.Permission, row.Modes)
		if row.Door == "native" && slices.Contains(row.Modes, catalog.Preview) && row.Availability != "desktop" {
			m.add(id, "cli", row.Role, row.Permission, row.Modes)
		}
	}
	for _, row := range workflow {
		c := native[row.Command]
		m.add(row.Command, "workflow", c.Role, c.Permission, row.Modes)
	}
	for _, row := range browser {
		role := strings.Join(row.Roles, ",")
		m.add(row.ID, "browser", role, "local", nil)
		if len(row.Args) == 0 && !slices.Contains([]string{"commandbar.open", "commandbar.close", "overlays.close", "surface.close"}, row.ID) {
			m.add(row.ID, "commandbar", role, "local", nil)
		}
	}
	return m
}

func executableCommandMatrix(registry *catalog.Registry) parityMatrix {
	m := parityMatrix{}
	for _, e := range registry.Entries() {
		c := e.Definition
		if e.Console != nil {
			m.add(e.ID, "operator-console", c.Role, c.Permission, c.Modes)
		}
		if e.Native != nil {
			m.add(e.ID, "native", c.Role, c.Permission, c.Modes)
		}
		if e.MFL != nil {
			m.add(e.ID, "mfl", c.Role, c.Permission, c.Modes)
		}
		if e.CLI {
			m.add(e.ID, "cli", c.Role, c.Permission, c.Modes)
		}
		for _, d := range e.Desktop {
			m.add(e.ID, "wails", d.Command.Role, d.Command.Permission, d.Command.Modes)
		}
		if e.Workflow != nil {
			modes := []catalog.Mode{}
			for _, mode := range []catalog.Mode{catalog.Read, catalog.Preview, catalog.Execute} {
				err := e.Workflow.Validate(catalog.RawWorkflow{Command: e.ID, Version: "1", Mode: string(mode), Arguments: json.RawMessage(`{}`)})
				wrong := err != nil && slices.Contains([]string{"this is a reading only", "check or confirm the change", "confirm your rookie pick"}, err.Error())
				if !wrong {
					modes = append(modes, mode)
				}
			}
			m.add(e.ID, "workflow", c.Role, c.Permission, modes)
		}
		if b := e.Browser; b != nil {
			role := strings.Join(b.Roles, ",")
			m.add(e.ID, "browser", role, "local", nil)
			if b.Searchable && len(b.Args) == 0 {
				m.add(e.ID, "commandbar", role, "local", nil)
			}
		}
	}
	return m
}

func compareCommandMatrix(before, after parityMatrix) error {
	ids := map[string]bool{}
	for key := range before {
		ids[strings.Split(key, "/")[0]] = true
	}
	for key := range after {
		ids[strings.Split(key, "/")[0]] = true
	}
	for id := range ids {
		for _, door := range []string{"native", "mfl", "wails", "workflow", "browser", "cli", "commandbar", "operator-console"} {
			for _, role := range []string{"gm", "commish", "admin", "operator"} {
				key := id + "/" + door + "/" + role
				want, declared := before[key]
				got, exposed := after[key]
				if declared != exposed || !reflect.DeepEqual(want, got) {
					return fmt.Errorf("command door parity %s: declared=%t exposed=%t before=%+v after=%+v", key, declared, exposed, want, got)
				}
			}
		}
	}
	return nil
}

func TestEveryCommandDoorRoleParity(t *testing.T) {
	registry, err := CommandRegistry()
	if err != nil {
		t.Fatal(err)
	}
	before, after := frozenCommandMatrix(t), executableCommandMatrix(registry)
	if err := compareCommandMatrix(before, after); err != nil {
		t.Fatal(err)
	}
	t.Logf("%d entries × 8 doors × 4 roles; %d exposed role cells", len(registry.Entries()), len(after))
	for _, defect := range []string{"door without entry", "entry missing declared door"} {
		t.Run(defect, func(t *testing.T) {
			bad := executableCommandMatrix(registry)
			if defect == "door without entry" {
				bad.add("unregistered", "wails", "admin", "local", nil)
			} else {
				delete(bad, "lineup.read/workflow/gm")
			}
			if err := compareCommandMatrix(before, bad); err == nil || !strings.Contains(err.Error(), "command door parity") {
				t.Fatalf("drift gate missed %s: %v", defect, err)
			}
		})
	}
}

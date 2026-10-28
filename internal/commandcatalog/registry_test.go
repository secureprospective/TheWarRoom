package catalog

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestExecutableRegistryRejectsBrokenDeclarations(t *testing.T) {
	handler := func(context.Context, json.RawMessage) (json.RawMessage, error) { return json.RawMessage(`null`), nil }
	valid := Entry{ID: "test.read", Label: "Read the test", Definition: completeCommand(), Native: map[Mode]Handler{Read: handler}}
	tests := map[string]func(*Registry){
		"duplicate ID":                  func(r *Registry) { r.entries = append(r.entries, r.entries[0]) },
		"missing native handler":        func(r *Registry) { r.entries[0].Native[Read] = nil },
		"unsupported mode":              func(r *Registry) { r.entries[0].Native[Execute] = handler },
		"missing desktop handler":       func(r *Registry) { r.entries[0].Desktop = []DesktopExposure{{}} },
		"missing workflow handler":      func(r *Registry) { r.entries[0].Workflow = &WorkflowCommand{} },
		"empty label":                   func(r *Registry) { r.entries[0].Label = "" },
		"unsupported CLI":               func(r *Registry) { r.entries[0].CLI = true },
		"missing browser argument type": func(r *Registry) { r.entries[0].Browser = &BrowserCommand{ID: "test.read"} },
		"no door":                       func(r *Registry) { r.entries[0].Native = nil },
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			entry := valid
			entry.Native = map[Mode]Handler{Read: handler}
			registry := &Registry{entries: []Entry{entry}}
			if err := registry.Validate(); err != nil {
				t.Fatal(err)
			}
			mutate(registry)
			if err := registry.Validate(); err == nil {
				t.Fatal("broken declaration accepted")
			}
		})
	}
}

func TestCommandLabelsDoNotExposeMachineSyntax(t *testing.T) {
	for id, want := range map[string]string{"mfl.bid.revise": "revise local MFL bid", "commish.rules.set": "set league rules", "TargetMFLKeyStatus": "MFL Key Status", "GetParams": "Get Params"} {
		if got := commandLabel(id); got != want || strings.Contains(got, ".") {
			t.Errorf("%s label=%q, want %q", id, got, want)
		}
	}
}

func TestOperatorRegistryKeepsOfflineAuthorityConsoleOnly(t *testing.T) {
	registry, err := NewOperatorRegistry()
	if err != nil {
		t.Fatal(err)
	}
	for _, door := range []Door{NativeDoor, MFLDoor, WailsDoor, WorkflowDoor, BrowserDoor, "unknown"} {
		if _, err := registry.Invoke(t.Context(), door, "league.stand_up", Execute, json.RawMessage(`{}`)); err == nil || !strings.Contains(err.Error(), "not available") {
			t.Fatalf("offline command leaked into %s: %v", door, err)
		}
	}
	for _, mode := range []Mode{Read, Preview, "unknown"} {
		if _, err := registry.Invoke(t.Context(), ConsoleDoor, "league.stand_up", mode, json.RawMessage(`{}`)); err == nil || !strings.Contains(err.Error(), "not available") {
			t.Fatalf("offline command accepted %s: %v", mode, err)
		}
	}
	if _, err := registry.Invoke(t.Context(), ConsoleDoor, "league.stand_up", Execute, json.RawMessage(`{"unexpected":true}`)); err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("offline command bypassed Raw decoding: %v", err)
	}
	registry.entries[0].Console[Execute] = nil
	if err := registry.Validate(); err == nil || !strings.Contains(err.Error(), "missing execute handler") {
		t.Fatalf("offline entry accepted a missing handler: %v", err)
	}
}

func TestOLineCloseRegistryMetadata(t *testing.T) {
	commands := NativeCommands(Schema{})
	for _, c := range commands {
		if c.ID != "oline.auction.close" {
			continue
		}
		if c.Role != "commish" || c.Undo != "none" || c.Method != "TargetCommissionerExecute" || commandLabel(c.ID) != "Close O-Line auction" {
			t.Fatalf("close metadata: %+v", c)
		}
		for _, field := range c.Arguments.Fields {
			if field.Name == "args" && len(field.Schema.Fields) != 0 {
				t.Fatal("close advertises app-owned outcomes")
			}
		}
		return
	}
	t.Fatal("close command missing")
}

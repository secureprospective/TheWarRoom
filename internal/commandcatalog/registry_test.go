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

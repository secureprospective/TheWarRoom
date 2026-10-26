package catalog

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"unicode"

	"github.com/secureprospective/TheWarRoom/internal/leagueservice"
)

type Door string

const (
	NativeDoor   Door = "native"
	MFLDoor      Door = "mfl"
	WailsDoor    Door = "wails"
	WorkflowDoor Door = "workflow"
	BrowserDoor  Door = "browser"
	ConsoleDoor  Door = "operator-console"
)

// Entry owns one operation and its explicit transport exposure. Display roles never replace admission.
// A local desktop adapter with no logical ID has a desktop.* ID, not invented remote authority.
type Entry struct {
	ID, Label  string
	Definition Command
	CLI        bool
	Native     map[Mode]Handler
	MFL        map[Mode]Handler
	Console    map[Mode]Handler
	Desktop    []DesktopExposure
	Browser    *BrowserCommand
	Workflow   *WorkflowCommand
}

type DesktopExposure struct {
	Feature string
	Command Command
	Binding Binding
}

type Registry struct {
	entries []Entry
	service *leagueservice.Service
}

type RegistryInputs struct {
	Seed    Schema
	MFL     MFLShapes
	Desktop map[string]map[string]Binding
	Service *leagueservice.Service
}

// NewRegistry binds application-lifetime ports without globals or reflective invocation.
func NewRegistry(in RegistryInputs) (*Registry, error) {
	if in.Service == nil {
		return nil, fmt.Errorf("command registry needs the league service")
	}
	r := &Registry{entries: operatorEntries(), service: in.Service}
	native := NativeCommands(in.Seed)
	mfl := MFLCommands(in.MFL)
	for _, c := range native {
		handlers, err := nativeHandlers(c, in)
		if err != nil {
			return nil, err
		}
		r.entries = append(r.entries, Entry{ID: c.ID, Label: commandLabel(c.ID), Definition: c, CLI: strings.HasPrefix(c.Arguments.Validator, "RawCommand.Validate/ValidatePreview:"), Native: handlers})
	}
	for _, c := range mfl {
		b, ok := in.Desktop["offers"][c.Method]
		if !ok || b.Handler == nil {
			return nil, fmt.Errorf("MFL command %s has no handler", c.ID)
		}
		r.entries = append(r.entries, Entry{ID: c.ID, Label: commandLabel(c.ID), Definition: c, MFL: map[Mode]Handler{c.Modes[0]: mflHandler(c, b)}})
	}
	if err := r.addDesktop(in.Desktop, native, mfl); err != nil {
		return nil, err
	}
	for id, w := range WorkflowCommands() {
		entry := r.find(id)
		if entry == nil || entry.Native == nil {
			return nil, fmt.Errorf("workflow %s has no native entry", id)
		}
		entry.Workflow = &w
	}
	for _, c := range BrowserCommands() {
		r.entries = append(r.entries, Entry{ID: c.ID, Label: c.Label, Browser: &c})
	}
	if err := r.Validate(); err != nil {
		return nil, err
	}
	return r, nil
}

func (r *Registry) addDesktop(bindings map[string]map[string]Binding, native, mfl []Command) error {
	features := strings.Fields("archive commissioner comms contracts control counterparties draft market now offers platform race research sources trades")
	for _, feature := range features {
		related := native
		if feature == "offers" {
			related = mfl
		}
		declarations := DesktopCommands(feature, related)
		if len(bindings[feature]) != len(declarations) {
			return fmt.Errorf("%s desktop declaration/handler count differs", feature)
		}
		for _, c := range declarations {
			b, ok := bindings[feature][c.Method]
			if !ok || b.Handler == nil {
				return fmt.Errorf("desktop %s.%s has no typed handler", feature, c.Method)
			}
			r.addDesktopEntry(feature, c, b)
		}
	}
	if len(bindings) != len(features) {
		return fmt.Errorf("undeclared desktop feature")
	}
	return nil
}

func (r *Registry) addDesktopEntry(feature string, c Command, b Binding) {
	entry := r.find(c.ID)
	// Only complete shared definitions identify a native/MFL operation. Old display IDs alone do not.
	if c.Version != "1" || entry == nil {
		r.entries = append(r.entries, Entry{ID: "desktop." + feature + "." + c.Method, Label: commandLabel(c.Method), Definition: c})
		entry = &r.entries[len(r.entries)-1]
	}
	entry.Desktop = append(entry.Desktop, DesktopExposure{Feature: feature, Command: c, Binding: b})
}

func (r *Registry) find(id string) *Entry {
	for i := range r.entries {
		if r.entries[i].ID == id {
			return &r.entries[i]
		}
	}
	return nil
}

func (r *Registry) Validate() error {
	seen := map[string]bool{}
	desktop := map[string]bool{}
	for _, entry := range r.entries {
		if entry.ID == "" || entry.Label == "" || seen[entry.ID] {
			return fmt.Errorf("missing or duplicate registry entry %s", entry.ID)
		}
		seen[entry.ID] = true
		if err := entry.validate(); err != nil {
			return fmt.Errorf("registry %s: %w", entry.ID, err)
		}
		for _, d := range entry.Desktop {
			key := d.Feature + "." + d.Command.Method
			if desktop[key] {
				return fmt.Errorf("duplicate desktop handler %s", key)
			}
			desktop[key] = true
		}
	}
	return nil
}

func (e Entry) validate() error {
	if err := e.validateGeneratedDoors(); err != nil {
		return err
	}
	for _, handlers := range []map[Mode]Handler{e.Native, e.MFL, e.Console} {
		if handlers != nil {
			if err := validateHandlers(e.Definition, handlers); err != nil {
				return err
			}
		}
	}
	for _, d := range e.Desktop {
		if d.Binding.Handler == nil {
			return fmt.Errorf("missing desktop handler")
		}
		if err := d.Binding.Arguments.Validate(); err != nil {
			return err
		}
	}
	if e.Workflow != nil && (e.Workflow.Run == nil || e.Workflow.Validate == nil) {
		return fmt.Errorf("missing workflow handler")
	}
	if !e.hasDoor() {
		return fmt.Errorf("no executable door")
	}
	return nil
}

func (r *Registry) Entries() []Entry { return slices.Clone(r.entries) }

func (r *Registry) Invoke(ctx context.Context, door Door, id string, mode Mode, body json.RawMessage) (json.RawMessage, error) {
	if door == WailsDoor && mode == "" {
		for _, e := range r.entries {
			for _, d := range e.Desktop {
				if d.Feature+"."+d.Command.Method == id {
					return d.Binding.Handler(ctx, body)
				}
			}
		}
	} else if entry := r.find(id); entry != nil {
		if door == WorkflowDoor && entry.Workflow != nil {
			raw := RawWorkflow{Command: id, Version: "1", Mode: string(mode), Arguments: body}
			if err := entry.Workflow.Validate(raw); err != nil {
				return nil, err
			}
			return entry.Workflow.Run(ctx, r.service, raw)
		}
		var handlers map[Mode]Handler
		switch door {
		case ConsoleDoor:
			handlers = entry.Console
		case MFLDoor:
			handlers = entry.MFL
		case NativeDoor:
			handlers = entry.Native
		case WailsDoor, WorkflowDoor, BrowserDoor:
			handlers = nil
		}
		if handler := handlers[mode]; handler != nil {
			return handler(ctx, body)
		}
	}
	return nil, fmt.Errorf("command %s is not available in %s mode through %s", id, mode, door)
}

func commandLabel(id string) string {
	if strings.Contains(id, ".") {
		words := strings.Split(strings.ReplaceAll(id, "_", " "), ".")
		for i, word := range words {
			switch word {
			case "commish", "native":
				words[i] = "league"
			case "mfl":
				words[i] = "local MFL"
			case "cap":
				words[i] = "salary cap"
			case "stat", "stats":
				words[i] = "statistics"
			case "auto":
				words[i] = "automatically"
			}
		}
		if len(words) > 1 {
			words = append([]string{words[len(words)-1]}, words[:len(words)-1]...)
		}
		return strings.Join(words, " ")
	}
	return desktopLabel(strings.TrimPrefix(id, "Target"))
}

func desktopLabel(id string) string {
	letters := []rune(id)
	var out strings.Builder
	for i, c := range letters {
		boundary := i > 0 && unicode.IsUpper(c) && (unicode.IsLower(letters[i-1]) || i+1 < len(letters) && unicode.IsLower(letters[i+1]))
		if boundary {
			out.WriteByte(' ')
		}
		out.WriteRune(c)
	}
	return out.String()
}

func validateHandlers(c Command, handlers map[Mode]Handler) error {
	if err := c.ValidatePublication(); err != nil {
		return err
	}
	if len(handlers) != len(c.Modes) {
		return fmt.Errorf("mode/handler count differs")
	}
	for _, mode := range c.Modes {
		if handlers[mode] == nil {
			return fmt.Errorf("missing %s handler", mode)
		}
	}
	return nil
}

func (e Entry) hasDoor() bool {
	return e.Native != nil || e.MFL != nil || e.Console != nil || len(e.Desktop) > 0 || e.Browser != nil || e.Workflow != nil
}

func (e Entry) validateGeneratedDoors() error {
	if e.CLI && (e.Native == nil || !strings.HasPrefix(e.Definition.Arguments.Validator, "RawCommand.Validate/ValidatePreview:")) {
		return fmt.Errorf("CLI requires the generic native change handler")
	}
	if e.Browser != nil && e.Browser.Arguments.ClientType == "" {
		return fmt.Errorf("missing browser argument type")
	}
	return nil
}

// NewOperatorRegistry binds only offline operator commands, without application services.
func NewOperatorRegistry() (*Registry, error) {
	registry := &Registry{entries: operatorEntries()}
	if err := registry.Validate(); err != nil {
		return nil, err
	}
	return registry, nil
}

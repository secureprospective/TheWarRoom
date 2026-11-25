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
	WorkflowDoor Door = "workflow"
	BrowserDoor  Door = "browser"
	ConsoleDoor  Door = "operator-console"
)

// Entry owns one operation and its explicit transport exposure. Display roles never replace admission.
type Entry struct {
	ID, Label  string
	Definition Command
	CLI        bool
	Native     map[Mode]Handler
	MFL        map[Mode]Handler
	Console    map[Mode]Handler
	Assist     map[Mode]Handler
	Browser    *BrowserCommand
	Workflow   *WorkflowCommand
}

type Registry struct {
	entries []Entry
	service *leagueservice.Service
}

type RegistryInputs struct {
	Seed    Schema
	MFL     MFLShapes
	Service *leagueservice.Service
	// Migration adds the move from MFL; nil leaves it out.
	Migration *MigrationInputs
}

// NewRegistry binds application-lifetime ports without globals or reflective invocation.
func NewRegistry(in RegistryInputs) (*Registry, error) {
	if in.Service == nil {
		return nil, fmt.Errorf("command registry needs the league service")
	}
	r := &Registry{entries: operatorEntries(), service: in.Service}
	native := NativeCommands(in.Seed)
	mfl := append(MFLCommands(in.MFL), MFLPlanCommands()...)
	r.addNativeCommands(in, native)
	r.entries = append(r.entries, serverWorkflowEntries()...)
	r.addMFLCommands(mfl)
	for id, w := range WorkflowCommands() {
		entry := r.find(id)
		if entry == nil || (entry.Native == nil && entry.Definition.Availability != "server") {
			return nil, fmt.Errorf("workflow %s has no executable entry", id)
		}
		entry.Workflow = &w
	}
	if in.Migration != nil {
		r.entries = append(r.entries, MigrationEntries(*in.Migration)...)
	}
	for _, c := range BrowserCommands() {
		if err := r.addBrowser(c); err != nil {
			return nil, err
		}
	}
	if err := r.Validate(); err != nil {
		return nil, err
	}
	return r, nil
}

func (r *Registry) addNativeCommands(in RegistryInputs, commands []Command) {
	for _, command := range commands {
		handlers := nativeHandlers(command, in)
		r.entries = append(r.entries, Entry{ID: command.ID, Label: commandLabel(command.ID), Definition: command,
			CLI: strings.HasPrefix(command.Arguments.Validator, "RawCommand.Validate/ValidatePreview:"), Native: handlers})
	}
}

func (r *Registry) addMFLCommands(commands []Command) {
	for _, command := range commands {
		r.entries = append(r.entries, Entry{ID: command.ID, Label: commandLabel(command.ID), Definition: command,
			MFL: map[Mode]Handler{command.Modes[0]: nonExecutableHandler}})
	}
}

func (r *Registry) addBrowser(command BrowserCommand) error {
	entry := r.find(command.ID)
	if entry == nil {
		r.entries = append(r.entries, Entry{ID: command.ID, Label: command.Label, Browser: &command})
		return nil
	}
	// backup.now shares its identity across the local browser action and hosted workflow transport.
	if command.ID != leagueservice.BackupNowCommand || entry.Workflow == nil {
		return fmt.Errorf("browser command %s duplicates a registered operation", command.ID)
	}
	entry.Browser = &command
	return nil
}

func serverWorkflowEntries() []Entry {
	commands := append(nativeDraftQueueCommands(), tradeDraftCommands()...)
	commands = append(commands, accountPreferenceCommands()...)
	commands = append(commands, serverBackupCommands()...)
	entries := make([]Entry, 0, len(commands))
	for _, command := range commands {
		entries = append(entries, Entry{ID: command.ID, Label: commandLabel(command.ID), Definition: command})
	}
	return entries
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
	for _, entry := range r.entries {
		if entry.ID == "" || entry.Label == "" || seen[entry.ID] {
			return fmt.Errorf("missing or duplicate registry entry %s", entry.ID)
		}
		seen[entry.ID] = true
		if entry.MFL != nil && (entry.Native != nil || !strings.HasPrefix(entry.ID, "mfl.")) {
			return fmt.Errorf("MFL command %s must be namespaced and separate from native", entry.ID)
		}
		if err := entry.validate(); err != nil {
			return fmt.Errorf("registry %s: %w", entry.ID, err)
		}
	}
	return nil
}

func (e Entry) validate() error {
	if err := e.validateGeneratedDoors(); err != nil {
		return err
	}
	for _, handlers := range []map[Mode]Handler{e.Native, e.MFL, e.Console, e.Assist} {
		if handlers != nil {
			if err := validateHandlers(e.Definition, handlers); err != nil {
				return err
			}
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
	if entry := r.find(id); entry != nil {
		if door == WorkflowDoor && entry.Workflow != nil {
			raw := RawWorkflow{Command: id, Version: "1", Mode: string(mode), Arguments: body}
			if err := entry.Workflow.Validate(raw); err != nil {
				return nil, err
			}
			return entry.Workflow.Run(ctx, r.service, raw)
		}
		if handler := entry.handlers(door)[mode]; handler != nil {
			return handler(ctx, body)
		}
	}
	return nil, fmt.Errorf("command %s is not available in %s mode through %s", id, mode, door)
}

// handlers are the entry's handlers on a door that dispatches by mode; nil for the other doors.
func (e *Entry) handlers(door Door) map[Mode]Handler {
	switch door {
	case ConsoleDoor:
		return e.Console
	case AssistDoor:
		return e.Assist
	case MFLDoor:
		return e.MFL
	case NativeDoor:
		return e.Native
	case WorkflowDoor, BrowserDoor:
		return nil
	}
	return nil
}

func commandLabel(id string) string {
	switch id {
	case "oline.auction.close":
		return "Close O-Line auction"
	case "waivers.claim.order.set":
		return "Set your waiver claim order"
	case "waivers.claim.cancel":
		return "Cancel your waiver claim"
	}
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
	return camelLabel(strings.TrimPrefix(id, "Target"))
}

func camelLabel(id string) string {
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
	return e.Native != nil || e.MFL != nil || e.Console != nil || e.Assist != nil || e.Browser != nil || e.Workflow != nil
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

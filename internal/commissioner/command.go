// Package commissioner defines the app-owned command boundary and its replayable receipts.
package commissioner

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

const (
	Create = "commish.league.create"
	Revert = "commish.correction.revert"
	Stream = "commissioner"
)

// Refusal marks a league change the league's own rules turned down, as distinct from a failure to read
// or record it. Each layer wraps the cause with its own context for the log; the owner reads Reason.
type Refusal struct{ err error }

// Refused marks err as a refusal. A nil err stays nil, and an error already marked is returned as is.
func Refused(err error) error {
	var r *Refusal
	if err == nil || errors.As(err, &r) {
		return err
	}
	return &Refusal{err: err}
}

func (r *Refusal) Error() string { return r.err.Error() }
func (r *Refusal) Unwrap() error { return r.err }

// Reason is the innermost message, the sentence that says why, starting with a capital letter.
func (r *Refusal) Reason() string {
	cause := r.err
	for next := errors.Unwrap(cause); next != nil; next = errors.Unwrap(cause) {
		cause = next
	}
	text := cause.Error()
	if text == "" {
		return text
	}
	first, size := utf8.DecodeRuneInString(text)
	return string(unicode.ToUpper(first)) + text[size:]
}

// Session supplies identity independently of IPC arguments and franchise selection.
type Session interface{ Principal() (string, error) }

// Authorizer runs inside the append transaction. Implementations must not write or
// reenter the ledger. Command is its canonical id, not a storage record.
type Authorizer interface {
	Allow(league, principal, command string) (string, error)
}

type Definition struct{ Gravity string }

func DefinitionFor(command string) (Definition, error) {
	switch command {
	case Create, Revert:
		return Definition{Gravity: "G3"}, nil
	case CalendarException:
		return Definition{Gravity: "G3"}, nil
	case WindowSet, WindowMove, WindowRemove:
		return Definition{Gravity: "G2"}, nil
	case ScheduleGenerate, ScheduleEdit, SchedulePublish:
		return Definition{Gravity: "G2"}, nil
	default:
		if d, ok := peopleDefinitionFor(command); ok {
			return Definition{Gravity: d.gravity}, nil
		}
		return Definition{}, fmt.Errorf("this commissioner command is not available yet")
	}
}

type RawArgs struct {
	Name           string              `json:"name,omitempty"`
	Entry          string              `json:"entry,omitempty"`
	Franchises     []string            `json:"franchises,omitempty"`
	People         *RawPeople          `json:"people,omitempty"`
	Field          string              `json:"field,omitempty"`
	Value          json.RawMessage     `json:"value,omitempty"`
	Preset         string              `json:"preset,omitempty"`
	FranchiseCount int                 `json:"franchiseCount,omitempty"`
	Seed           []SeedValue         `json:"seed,omitempty"`
	Exception      *RawWindowException `json:"exception,omitempty"`
	Schedule       *RawScheduleArgs    `json:"schedule,omitempty"`
	Window         *RawWindowArgs      `json:"window,omitempty"`
}

func (a RawArgs) Validate(command string) error {
	if scheduleCommand(command) {
		return a.validateSchedule(command)
	}
	if windowCommand(command) {
		return a.validateWindow(command)
	}
	if a.Schedule != nil || a.Window != nil {
		return fmt.Errorf("send only the details of this league action")
	}
	if command == CalendarException {
		return a.validateException()
	}
	if a.Exception != nil {
		return fmt.Errorf("send only the details of this league action")
	}
	if a.foreignRuleArgs(command) {
		return fmt.Errorf("send only the details of this league action")
	}
	switch command {
	case RulesSet:
		return a.validateRule()
	case Create:
		return a.validateCreation()
	case Revert:
		if strings.TrimSpace(a.Entry) == "" || a.Name != "" || a.People != nil {
			return fmt.Errorf("choose the recorded change to reverse")
		}
	default:
		return a.validatePeople(command)
	}
	return nil
}

func (a RawArgs) foreignRuleArgs(command string) bool {
	return (command != RulesSet && (a.Field != "" || len(a.Value) != 0)) || (command != Create &&
		(a.Preset != "" || a.FranchiseCount != 0 || len(a.Seed) != 0 || len(a.Franchises) != 0))
}

func (a RawArgs) validatePeople(command string) error {
	if _, ok := peopleDefinitionFor(command); !ok {
		return fmt.Errorf("this commissioner command is not available yet")
	}
	if a.People == nil || a.Name != "" || a.Entry != "" {
		return fmt.Errorf("send the details of this people change")
	}
	return a.People.Validate(command)
}

type RawCommand struct {
	League     string  `json:"league"`
	Command    string  `json:"command"`
	Args       RawArgs `json:"args"`
	Reason     string  `json:"reason"`
	Expected   int64   `json:"expected"`
	Request    string  `json:"request"`
	Effective  string  `json:"effective"`
	Visibility string  `json:"visibility"`
	Gravity    string  `json:"gravity,omitempty"`
}

func (r RawCommand) Validate() error {
	definition, err := r.EffectiveDefinition()
	if err != nil {
		return err
	}
	if strings.TrimSpace(r.League) != r.League || !strings.HasPrefix(r.League, "native.") || r.League == "native." ||
		strings.TrimSpace(r.Request) == "" || r.Expected < 0 || r.Expected == math.MaxInt64 {
		return fmt.Errorf("choose an app league and a valid request and league version")
	}
	if r.Gravity != "" && r.Gravity != definition.Gravity {
		return fmt.Errorf("this change requires a recorded reason; its safeguards cannot be lowered")
	}
	if definition.Gravity == "G3" && strings.TrimSpace(r.Reason) == "" {
		return fmt.Errorf("give a reason for this change")
	}
	if err := r.validatePlacement(); err != nil {
		return err
	}
	if err := r.Args.Validate(r.Command); err != nil {
		return err
	}
	return r.validateSchedulePlacement()
}

// publicNowChange names the commands that are never private and never wait for a rollover.
func publicNowChange(command string) string {
	switch {
	case windowCommand(command):
		return "window changes"
	case command == CalendarException:
		return "window exceptions"
	}
	return ""
}

func (r RawCommand) validatePlacement() error {
	if what := publicNowChange(r.Command); what != "" && (r.Visibility != "league" || r.Effective != "now") {
		return fmt.Errorf("%s take effect now and are visible to every owner", what)
	}
	if r.Command == RulesSet {
		return r.validateRulePlacement()
	}
	if r.Visibility != "league" && r.Visibility != "private" {
		return fmt.Errorf("choose who may read this change")
	}
	if r.Effective != "now" && r.Effective != "next_rollover" {
		at, err := time.Parse(time.RFC3339Nano, r.Effective)
		if err != nil || at.IsZero() {
			return fmt.Errorf("choose when this change takes effect")
		}
	}
	if r.Command == Create && (r.Effective != "now" || r.Visibility != "league") {
		return fmt.Errorf("a new league is created now and recorded for the league")
	}
	return r.validatePeoplePlacement()
}

func (r RawCommand) validatePeoplePlacement() error {
	if r.Args.People == nil {
		return nil
	}
	event, err := r.Args.People.event(r.Command)
	if err != nil {
		return err
	}
	if r.Effective != "now" {
		return fmt.Errorf("people changes take effect now")
	}
	private := event.Private()
	if d, ok := successionDefinitionFor(r.Command); ok {
		private = d.private
	}
	if private && r.Visibility != "private" {
		return fmt.Errorf("role changes, invitations and ballots must stay private")
	}
	if !private && r.Visibility != "league" {
		return fmt.Errorf("league-state changes must be visible to owners")
	}
	return nil
}

// Decode rejects identity/role fields and unknown arguments even for direct IPC calls.
func Decode(body string) (RawCommand, error) {
	var raw RawCommand
	decoder := json.NewDecoder(bytes.NewBufferString(body))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&raw); err != nil {
		return raw, fmt.Errorf("read commissioner command: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return raw, fmt.Errorf("send exactly one commissioner command")
	}
	raw, err := raw.Prepared()
	if err != nil {
		return raw, err
	}
	if err := raw.Validate(); err != nil {
		return raw, err
	}
	return raw, nil
}

func (r RawCommand) Canonical(principal string) (string, error) {
	definition, err := r.EffectiveDefinition()
	if err != nil {
		return "", err
	}
	r.Gravity = definition.Gravity
	if r.Effective != "now" && r.Effective != "next_rollover" {
		at, err := time.Parse(time.RFC3339Nano, r.Effective)
		if err != nil {
			return "", fmt.Errorf("canonical effective time: %w", err)
		}
		r.Effective = at.UTC().Format(time.RFC3339Nano)
	}
	body, err := json.Marshal(struct {
		Command   RawCommand `json:"command"`
		Principal string     `json:"principal"`
	}{r, principal})
	if err != nil {
		return "", fmt.Errorf("encode commissioner command: %w", err)
	}
	return string(body), nil
}

func (r RawCommand) EffectiveDefinition() (Definition, error) {
	definition, err := r.Definition()
	if err != nil {
		return Definition{}, err
	}
	if r.Command == ScheduleEdit && r.Args.Schedule != nil && r.Args.Schedule.Published {
		definition.Gravity = "G3"
	}
	return definition, nil
}

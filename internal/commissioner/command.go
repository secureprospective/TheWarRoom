// Package commissioner defines the app-owned command boundary and its replayable receipts.
package commissioner

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"strings"
	"time"
)

const (
	Create = "commish.league.create"
	Revert = "commish.correction.revert"
	Stream = "commissioner"
)

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
	default:
		if d, ok := peopleDefinitionFor(command); ok {
			return Definition{Gravity: d.gravity}, nil
		}
		return Definition{}, fmt.Errorf("this commissioner command is not available yet")
	}
}

type RawArgs struct {
	Name       string     `json:"name,omitempty"`
	Entry      string     `json:"entry,omitempty"`
	Franchises []string   `json:"franchises,omitempty"`
	People     *RawPeople `json:"people,omitempty"`
}

func (a RawArgs) Validate(command string) error {
	switch command {
	case Create:
		return a.validateCreation()
	case Revert:
		if strings.TrimSpace(a.Entry) == "" || a.Name != "" || a.People != nil || len(a.Franchises) != 0 {
			return fmt.Errorf("choose the recorded change to reverse")
		}
	default:
		return a.validatePeople(command)
	}
	return nil
}

func (a RawArgs) validateCreation() error {
	if strings.TrimSpace(a.Name) == "" || a.Entry != "" || a.People != nil {
		return fmt.Errorf("give the new league a name")
	}
	return validateFranchises(a.Franchises)
}

func (a RawArgs) validatePeople(command string) error {
	if _, ok := peopleDefinitionFor(command); !ok {
		return fmt.Errorf("this commissioner command is not available yet")
	}
	if a.People == nil || a.Name != "" || a.Entry != "" || len(a.Franchises) != 0 {
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
	definition, err := DefinitionFor(r.Command)
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
	return nil
}

func (r RawCommand) validatePlacement() error {
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
	if event.Private() && r.Visibility != "private" {
		return fmt.Errorf("role changes and invitations must stay private")
	}
	if !event.Private() && r.Visibility != "league" {
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
	if err := raw.Validate(); err != nil {
		return raw, err
	}
	return raw, nil
}

func (r RawCommand) Canonical(principal string) (string, error) {
	definition, err := DefinitionFor(r.Command)
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

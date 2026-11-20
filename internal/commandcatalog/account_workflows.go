package catalog

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/secureprospective/TheWarRoom/internal/accounts"
	"github.com/secureprospective/TheWarRoom/internal/leagueservice"
)

func accountPreferenceCommands() []Command {
	return []Command{
		{ID: "home.layout.read", Method: "ReadHomeLayout", Version: "1", Role: "domain",
			Gravity: "G0", Availability: "server",
			Arguments:  Shape(reflect.TypeFor[accounts.RawHomeLayoutRead](), "RawHomeLayoutRead.Validate"),
			Permission: "account-owner", Modes: []Mode{Read}, Undo: "none", Facts: "private-account-data",
			Safeguards: "session account scope; no request account ID; agents denied; uncached"},
		{ID: "home.layout.set", Method: "SetHomeLayout", Version: "1", Role: "domain",
			Gravity: "G0", Availability: "server",
			Arguments:  Shape(reflect.TypeFor[accounts.RawHomeLayoutSet](), "RawHomeLayoutSet.Validate"),
			Permission: "account-owner", Modes: []Mode{Execute}, Undo: "replace-own-account-preference",
			Facts:      "private-account-data",
			Safeguards: "session account scope; no request account ID; agent denied; revision checked"},
		{ID: "job.snoozes.read", Method: "ReadJobSnoozes", Version: "1", Role: "domain",
			Gravity: "G0", Availability: "server",
			Arguments:  Shape(reflect.TypeFor[accounts.RawJobSnoozesRead](), "RawJobSnoozesRead.Validate"),
			Permission: "account-owner", Modes: []Mode{Read}, Undo: "none", Facts: "private-account-data",
			Safeguards: "session account scope; no request account ID; agents denied; uncached"},
		{ID: "job.snooze.set", Method: "SetJobSnooze", Version: "1", Role: "domain",
			Gravity: "G0", Availability: "server",
			Arguments:  Shape(reflect.TypeFor[accounts.RawJobSnoozeSet](), "RawJobSnoozeSet.Validate"),
			Permission: "account-owner", Modes: []Mode{Execute}, Undo: "replace-own-account-snooze",
			Facts:      "private-account-data",
			Safeguards: "session account scope; no request account ID; agent denied"},
		{ID: "job.snooze.wake", Method: "WakeJobSnooze", Version: "1", Role: "domain",
			Gravity: "G0", Availability: "server",
			Arguments:  Shape(reflect.TypeFor[accounts.RawJobSnoozeWake](), "RawJobSnoozeWake.Validate"),
			Permission: "account-owner", Modes: []Mode{Execute}, Undo: "clear-own-account-snooze",
			Facts:      "private-account-data",
			Safeguards: "session account scope; no request account ID; agent denied"},
	}
}

func accountPreferenceWorkflows() map[string]WorkflowCommand {
	workflows := accountHomeLayoutWorkflows()
	for id, workflow := range accountJobSnoozeWorkflows() {
		workflows[id] = workflow
	}
	return workflows
}

func accountHomeLayoutWorkflows() map[string]WorkflowCommand {
	return map[string]WorkflowCommand{
		"home.layout.read": {
			Validate: decodeHomeLayoutRead,
			Run:      accountOnlyWorkflow,
			RunAccount: func(ctx context.Context, repository accounts.PreferenceRepository, accountID string,
				raw RawWorkflow) (json.RawMessage, error) {
				if err := decodeHomeLayoutRead(raw); err != nil {
					return nil, err
				}
				return workflowJSON(repository.ReadHomeLayout(ctx, accountID))
			},
		},
		"home.layout.set": {
			Validate: func(raw RawWorkflow) error { _, err := decodeHomeLayoutSet(raw); return err },
			Run:      accountOnlyWorkflow,
			RunAccount: func(ctx context.Context, repository accounts.PreferenceRepository, accountID string,
				raw RawWorkflow) (json.RawMessage, error) {
				input, err := decodeHomeLayoutSet(raw)
				if err != nil {
					return nil, err
				}
				return workflowJSON(repository.SetHomeLayout(ctx, accountID, input))
			},
		},
	}
}

func accountJobSnoozeWorkflows() map[string]WorkflowCommand {
	return map[string]WorkflowCommand{
		"job.snoozes.read": {
			Validate: decodeJobSnoozesRead,
			Run:      accountOnlyWorkflow,
			RunAccount: func(ctx context.Context, repository accounts.PreferenceRepository, accountID string,
				raw RawWorkflow) (json.RawMessage, error) {
				if err := decodeJobSnoozesRead(raw); err != nil {
					return nil, err
				}
				return workflowJSON(repository.ReadJobSnoozes(ctx, accountID))
			},
		},
		"job.snooze.set": {
			Validate: func(raw RawWorkflow) error { _, err := decodeJobSnoozeSet(raw); return err },
			Run:      accountOnlyWorkflow,
			RunAccount: func(ctx context.Context, repository accounts.PreferenceRepository, accountID string,
				raw RawWorkflow) (json.RawMessage, error) {
				input, err := decodeJobSnoozeSet(raw)
				if err != nil {
					return nil, err
				}
				return workflowJSON(repository.SetJobSnooze(ctx, accountID, input))
			},
		},
		"job.snooze.wake": {
			Validate: func(raw RawWorkflow) error { _, err := decodeJobSnoozeWake(raw); return err },
			Run:      accountOnlyWorkflow,
			RunAccount: func(ctx context.Context, repository accounts.PreferenceRepository, accountID string,
				raw RawWorkflow) (json.RawMessage, error) {
				input, err := decodeJobSnoozeWake(raw)
				if err != nil {
					return nil, err
				}
				return workflowJSON(repository.WakeJobSnooze(ctx, accountID, input))
			},
		},
	}
}

func decodeHomeLayoutRead(raw RawWorkflow) error {
	if raw.Mode != string(Read) {
		return fmt.Errorf("this is a reading only")
	}
	var input accounts.RawHomeLayoutRead
	return decodeWorkflow(raw.Arguments, &input)
}

func decodeHomeLayoutSet(raw RawWorkflow) (accounts.RawHomeLayoutSet, error) {
	var input accounts.RawHomeLayoutSet
	if raw.Mode != string(Execute) {
		return input, fmt.Errorf("replace your saved home layout")
	}
	return input, decodeWorkflow(raw.Arguments, &input)
}

func decodeJobSnoozesRead(raw RawWorkflow) error {
	if raw.Mode != string(Read) {
		return fmt.Errorf("this is a reading only")
	}
	var input accounts.RawJobSnoozesRead
	return decodeWorkflow(raw.Arguments, &input)
}

func decodeJobSnoozeSet(raw RawWorkflow) (accounts.RawJobSnoozeSet, error) {
	var input accounts.RawJobSnoozeSet
	if raw.Mode != string(Execute) {
		return input, fmt.Errorf("snooze this job")
	}
	return input, decodeWorkflow(raw.Arguments, &input)
}

func decodeJobSnoozeWake(raw RawWorkflow) (accounts.RawJobSnoozeWake, error) {
	var input accounts.RawJobSnoozeWake
	if raw.Mode != string(Execute) {
		return input, fmt.Errorf("wake this job")
	}
	return input, decodeWorkflow(raw.Arguments, &input)
}

func accountOnlyWorkflow(context.Context, *leagueservice.Service, RawWorkflow) (json.RawMessage, error) {
	return nil, fmt.Errorf("this preference requires a signed-in account")
}

// IsAccountWorkflow identifies requests whose trusted owner scope is an account, not a league.
func IsAccountWorkflow(command string) bool {
	switch command {
	case "home.layout.read", "home.layout.set", "job.snoozes.read", "job.snooze.set", "job.snooze.wake":
		return true
	default:
		return false
	}
}

// RunAccountWorkflow executes a private preference with the authenticated account supplied by the
// server; no account identity is decoded from workflow arguments.
func RunAccountWorkflow(ctx context.Context, repository accounts.PreferenceRepository, accountID string,
	raw RawWorkflow) (json.RawMessage, error) {
	if err := raw.Validate(); err != nil {
		return nil, err
	}
	command := WorkflowCommands()[raw.Command]
	if command.RunAccount == nil {
		return nil, fmt.Errorf("workflow %s is not account-scoped", raw.Command)
	}
	if repository == nil || accountID == "" {
		return nil, fmt.Errorf("account preference store unavailable")
	}
	return command.RunAccount(ctx, repository, accountID, raw)
}

package catalog

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/secureprospective/TheWarRoom/internal/accounts"
	"github.com/secureprospective/TheWarRoom/internal/leagueservice"
)

func accountHomeLayoutCommands() []Command {
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
	}
}

func accountPreferenceWorkflows() map[string]WorkflowCommand {
	decodeRead := func(raw RawWorkflow) error {
		if raw.Mode != string(Read) {
			return fmt.Errorf("this is a reading only")
		}
		var input accounts.RawHomeLayoutRead
		return decodeWorkflow(raw.Arguments, &input)
	}
	decodeSet := func(raw RawWorkflow) (accounts.RawHomeLayoutSet, error) {
		var input accounts.RawHomeLayoutSet
		if raw.Mode != string(Execute) {
			return input, fmt.Errorf("replace your saved home layout")
		}
		return input, decodeWorkflow(raw.Arguments, &input)
	}
	accountOnly := func(context.Context, *leagueservice.Service, RawWorkflow) (json.RawMessage, error) {
		return nil, fmt.Errorf("this preference requires a signed-in account")
	}
	return map[string]WorkflowCommand{
		"home.layout.read": {
			Validate: decodeRead,
			Run:      accountOnly,
			RunAccount: func(ctx context.Context, repository accounts.HomeLayoutRepository, accountID string,
				raw RawWorkflow) (json.RawMessage, error) {
				if err := decodeRead(raw); err != nil {
					return nil, err
				}
				return workflowJSON(repository.ReadHomeLayout(ctx, accountID))
			},
		},
		"home.layout.set": {
			Validate: func(raw RawWorkflow) error { _, err := decodeSet(raw); return err },
			Run:      accountOnly,
			RunAccount: func(ctx context.Context, repository accounts.HomeLayoutRepository, accountID string,
				raw RawWorkflow) (json.RawMessage, error) {
				input, err := decodeSet(raw)
				if err != nil {
					return nil, err
				}
				return workflowJSON(repository.SetHomeLayout(ctx, accountID, input))
			},
		},
	}
}

// IsAccountWorkflow identifies requests whose trusted owner scope is an account, not a league.
func IsAccountWorkflow(command string) bool {
	switch command {
	case "home.layout.read", "home.layout.set":
		return true
	default:
		return false
	}
}

// RunAccountWorkflow executes a private preference with the authenticated account supplied by the
// server; no account identity is decoded from workflow arguments.
func RunAccountWorkflow(ctx context.Context, repository accounts.HomeLayoutRepository, accountID string,
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

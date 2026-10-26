package catalog

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/secureprospective/TheWarRoom/internal/leaguerecovery"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
)

func nativeHandlers(c Command, in RegistryInputs) (map[Mode]Handler, error) {
	out := map[Mode]Handler{}
	for _, mode := range c.Modes {
		switch {
		case mode == Read:
			b, ok := in.Desktop["commissioner"][c.Method]
			if !ok || b.Handler == nil {
				return nil, fmt.Errorf("native reading %s has no typed handler", c.ID)
			}
			out[mode] = readHandler(c, b)
		case c.ID == commissioner.NativeSeed:
			method := c.Method
			if mode == Preview {
				method = "TargetCommissionerPreview"
			}
			b, ok := in.Desktop["commissioner"][method]
			if !ok || b.Handler == nil {
				return nil, fmt.Errorf("seed %s has no typed handler", mode)
			}
			out[mode] = seedHandler(mode, b)
		default:
			out[mode] = nativeChangeHandler(c.ID, mode, in)
		}
	}
	return out, nil
}

func nativeChangeHandler(id string, mode Mode, in RegistryInputs) Handler {
	return func(ctx context.Context, body json.RawMessage) (json.RawMessage, error) {
		if id == commissioner.DraftPick {
			var raw commissioner.DraftPickInput
			if err := decodeJSON(body, &raw); err != nil {
				return nil, err
			}
			return result(in.Service.Pick(ctx, raw))
		}
		var raw commissioner.RawCommand
		if err := decodeJSON(body, &raw); err != nil {
			return nil, err
		}
		if raw.Command != id {
			return nil, fmt.Errorf("choose one league action")
		}
		if mode == Preview {
			//nolint:contextcheck // Preview uses the injected service host's existing context.
			return result(in.Service.Preview(raw))
		}
		if id == commissioner.LineupSet {
			return result(in.Service.SetLineup(ctx, raw))
		}
		return result(in.Service.Execute(ctx, raw))
	}
}

func seedHandler(mode Mode, b Binding) Handler {
	return func(ctx context.Context, body json.RawMessage) (json.RawMessage, error) {
		argument := body
		if mode == Preview {
			var err error
			argument, err = json.Marshal(string(body))
			if err != nil {
				return nil, fmt.Errorf("encode seed preview: %w", err)
			}
		}
		return callBinding(ctx, b, []json.RawMessage{argument})
	}
}

func readHandler(c Command, b Binding) Handler {
	return func(ctx context.Context, body json.RawMessage) (json.RawMessage, error) {
		fields, err := argumentFields(body, c.Arguments)
		if err != nil {
			return nil, err
		}
		if len(b.Arguments.Fields) == 1 && b.Arguments.Fields[0].Schema.Type == "object" {
			return callBinding(ctx, b, []json.RawMessage{body})
		}
		args := make([]json.RawMessage, 0, len(c.Arguments.Fields))
		for _, field := range c.Arguments.Fields {
			value, ok := fields[field.Name]
			if !ok {
				return nil, fmt.Errorf("provide %s", field.Name)
			}
			if c.Method == "TargetCommissionerLedgerFiltered" && field.Name == "filter" {
				value, err = json.Marshal(string(value))
				if err != nil {
					return nil, fmt.Errorf("encode ledger filter: %w", err)
				}
			}
			args = append(args, value)
		}
		return callBinding(ctx, b, args)
	}
}

func mflHandler(c Command, b Binding) Handler {
	return func(ctx context.Context, body json.RawMessage) (json.RawMessage, error) {
		fields, err := argumentFields(body, c.Arguments)
		if err != nil {
			return nil, err
		}
		switch c.ID {
		case "mfl.pool.read", "mfl.bid.preview", "mfl.bid.submit":
			return callBinding(ctx, b, []json.RawMessage{body})
		case "mfl.bid.revise":
			id := fields["id"]
			delete(fields, "id")
			bid, err := json.Marshal(fields)
			if err != nil {
				return nil, fmt.Errorf("encode bid revision: %w", err)
			}
			return callBinding(ctx, b, []json.RawMessage{id, bid})
		default:
			args := make([]json.RawMessage, 0, len(c.Arguments.Fields))
			for _, field := range c.Arguments.Fields {
				args = append(args, fields[field.Name])
			}
			return callBinding(ctx, b, args)
		}
	}
}

// RawArguments is a named-field transport adapter, not domain state. Existing Raw validators own values.
type RawArguments map[string]json.RawMessage

func (r RawArguments) Validate(schema Schema) error {
	if r == nil {
		return fmt.Errorf("provide command arguments as an object")
	}
	allowed := map[string]bool{}
	for _, field := range schema.Fields {
		allowed[field.Name] = true
	}
	for name := range r {
		if !allowed[name] {
			return fmt.Errorf("unexpected argument %s", name)
		}
	}
	return nil
}

func argumentFields(body json.RawMessage, schema Schema) (RawArguments, error) {
	var fields RawArguments
	if err := decodeJSON(body, &fields); err != nil {
		return nil, err
	}
	if err := fields.Validate(schema); err != nil {
		return nil, err
	}
	return fields, nil
}

func callBinding(ctx context.Context, b Binding, args []json.RawMessage) (json.RawMessage, error) {
	body, err := json.Marshal(args)
	if err != nil {
		return nil, fmt.Errorf("encode desktop arguments: %w", err)
	}
	return b.Handler(ctx, body)
}

func operatorEntries() []Entry {
	definition := Command{ID: "league.stand_up", Method: "StandUpRosters", Version: "1", Role: "operator", Gravity: "G3", Availability: "server",
		Arguments:  Shape(reflect.TypeFor[leaguerecovery.RawRequest](), "leaguerecovery.RawRequest.Validate"),
		Permission: "offline-operator", Modes: []Mode{Execute}, Undo: "fresh-folder-only", Facts: "validated-rosters-and-contracts", Safeguards: "exclusive-private-folder; no-listener; read-only-league"}
	handler := func(ctx context.Context, body json.RawMessage) (json.RawMessage, error) {
		var raw leaguerecovery.RawRequest
		if err := decodeJSON(body, &raw); err != nil {
			return nil, err
		}
		if err := raw.Validate(); err != nil {
			return nil, fmt.Errorf("check roster recovery request: %w", err)
		}
		return result(leaguerecovery.StandUp(ctx, raw))
	}
	return []Entry{{ID: definition.ID, Label: "Restore league rosters and contracts offline", Definition: definition, Console: map[Mode]Handler{Execute: handler}}}
}

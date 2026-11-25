package catalog

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"reflect"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/leaguerecovery"
	"github.com/secureprospective/TheWarRoom/internal/leagueservice"
)

// Handler receives a transport-decoded body; the bound API retains Raw validation and admission.
type Handler func(context.Context, json.RawMessage) (json.RawMessage, error)

func nonExecutableHandler(context.Context, json.RawMessage) (json.RawMessage, error) {
	return nil, fmt.Errorf("generation registry cannot execute unbound handlers")
}

func decodeJSON[T any](body []byte, out *T) error {
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(out); err != nil {
		return fmt.Errorf("read command arguments: %w", err)
	}
	var trailing json.RawMessage
	if err := decoder.Decode(&trailing); err != io.EOF {
		return fmt.Errorf("send one command argument value")
	}
	return nil
}

func result[T any](value T, err error) (json.RawMessage, error) {
	if err != nil {
		return nil, fmt.Errorf("command: %w", err)
	}
	body, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("encode command result: %w", err)
	}
	return body, nil
}

func nativeHandlers(c Command, in RegistryInputs) map[Mode]Handler {
	// Server-only workflow commands do not create a NativeDoor route.
	if c.Availability == "server" {
		return nil
	}
	out := map[Mode]Handler{}
	for _, mode := range c.Modes {
		switch {
		case c.ID == leagueservice.PlayerNoteRead || c.ID == leagueservice.PlayerNoteSet ||
			c.ID == leagueservice.PlayerWatchRead || c.ID == leagueservice.PlayerWatchSet ||
			c.ID == leagueservice.PlayerWatchOrder:
			out[mode] = nativePlayerPrivateHandler(c.ID, in.Service)
		case mode == Read || c.ID == commissioner.NativeSeed:
			out[mode] = nonExecutableHandler
		default:
			out[mode] = nativeChangeHandler(c.ID, mode, in)
		}
	}
	return out
}

func nativePlayerPrivateHandler(id string, service *leagueservice.Service) Handler {
	return func(ctx context.Context, body json.RawMessage) (json.RawMessage, error) {
		switch id {
		case leagueservice.PlayerNoteRead:
			var input leagueservice.RawPlayerNotesScope
			if err := decodeJSON(body, &input); err != nil {
				return nil, err
			}
			return result(service.TargetPlayerNotesContext(ctx, input))
		case leagueservice.PlayerNoteSet:
			var input leagueservice.RawPlayerNoteSet
			if err := decodeJSON(body, &input); err != nil {
				return nil, err
			}
			return result(service.SetPlayerNoteContext(ctx, input))
		case leagueservice.PlayerWatchRead:
			var input leagueservice.RawPlayerWatchReadScope
			if err := decodeJSON(body, &input); err != nil {
				return nil, err
			}
			return result(service.TargetPlayerWatchesContext(ctx, input))
		case leagueservice.PlayerWatchSet:
			var input leagueservice.RawPlayerWatchSet
			if err := decodeJSON(body, &input); err != nil {
				return nil, err
			}
			return result(service.SetPlayerWatchContext(ctx, input))
		case leagueservice.PlayerWatchOrder:
			var input leagueservice.RawPlayerWatchOrder
			if err := decodeJSON(body, &input); err != nil {
				return nil, err
			}
			return result(service.SetPlayerWatchOrderContext(ctx, input))
		default:
			return nil, fmt.Errorf("unknown private player command")
		}
	}
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

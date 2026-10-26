package catalog

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
)

// Handler receives a transport-decoded body; the bound API retains Raw validation and admission.
type Handler func(context.Context, json.RawMessage) (json.RawMessage, error)

// Binding is a typed desktop method adapter. Reflection describes shape only, never invocation.
type Binding struct {
	Arguments Schema
	Handler   Handler
}

func binding(fields []Field, run func([]json.RawMessage) (json.RawMessage, error)) Binding {
	schema := Schema{Type: "object", Validator: "desktop adapter Raw validation", Fields: fields}
	return Binding{Arguments: schema, Handler: func(_ context.Context, body json.RawMessage) (json.RawMessage, error) {
		var args []json.RawMessage
		if err := decodeJSON(body, &args); err != nil {
			return nil, err
		}
		if len(args) != len(fields) {
			return nil, fmt.Errorf("provide %d desktop arguments", len(fields))
		}
		return run(args)
	}}
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

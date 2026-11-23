package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/serverhttp"
	"github.com/spf13/cobra"
)

// readCommand runs one read-only league command. It never changes the league: the server runs it in
// read mode only.
func (e env) readCommand() *cobra.Command {
	return &cobra.Command{Use: "read COMMAND.read [SCOPE.json|-]", Short: "Print one read-only league answer as JSON", Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			var scope serverhttp.RawWorkflowScope
			if len(args) == 2 {
				var err error
				if scope, err = e.readScope(args[1]); err != nil {
					return err
				}
			}
			reply, err := e.read(cmd.Context(), args[0], scope)
			if err != nil {
				return err
			}
			var pretty bytes.Buffer
			if err := json.Indent(&pretty, reply, "", "  "); err != nil {
				return fmt.Errorf("print the reply: %w", err)
			}
			pretty.WriteByte('\n')
			if _, err := e.out.Write(pretty.Bytes()); err != nil {
				return fmt.Errorf("print the reply: %w", err)
			}
			return nil
		}}
}

// read is shared by the CLI and the MCP tool. A command that is not a read is refused before the
// saved connection is even loaded, so it never reaches the league.
func (e env) read(ctx context.Context, command string, scope serverhttp.RawWorkflowScope) (json.RawMessage, error) {
	if command == ".read" || !strings.HasSuffix(command, ".read") {
		return nil, fmt.Errorf("twr read only runs read commands, whose names end in .read (for example standings.read); use twr preview or twr execute for changes")
	}
	agent, _, err := e.agent()
	if err != nil {
		return nil, err
	}
	reply, err := agent.Read(ctx, command, scope)
	if err != nil {
		return nil, err //nolint:wrapcheck // already a plain message for the person
	}
	return reply, nil
}

// readScope reads the optional scope as JSON from a file, or from stdin when the file is "-".
func (e env) readScope(name string) (serverhttp.RawWorkflowScope, error) {
	var data []byte
	var err error
	if name == "-" {
		data, err = io.ReadAll(io.LimitReader(e.in, 1<<20))
	} else {
		data, err = os.ReadFile(filepath.Clean(name))
	}
	if err != nil {
		return serverhttp.RawWorkflowScope{}, fmt.Errorf("read the scope: %w", err)
	}
	return decodeScope(data)
}

// decodeScope is strict like readChange: an unknown field is a mistake, not something to ignore.
// A league in the scope is accepted but replaced by the saved league.
func decodeScope(data []byte) (serverhttp.RawWorkflowScope, error) {
	var scope serverhttp.RawWorkflowScope
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&scope); err != nil {
		return scope, fmt.Errorf("the scope is not valid JSON for a league read: %w", err)
	}
	return scope, nil
}

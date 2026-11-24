package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

func (e env) workflowCommand() *cobra.Command {
	var mode, rawJSON, file string
	command := &cobra.Command{Use: "workflow COMMAND [--mode read|execute] [--json OBJECT | --file PATH|-]",
		Short: "Run any league workflow command", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			jsonSet, fileSet := cmd.Flags().Changed("json"), cmd.Flags().Changed("file")
			if jsonSet && fileSet {
				return fmt.Errorf("choose either --json or --file")
			}
			data := []byte("{}")
			if jsonSet {
				data = []byte(rawJSON)
			} else if fileSet {
				var err error
				if file == "-" {
					data, err = io.ReadAll(io.LimitReader(e.in, 1<<20))
				} else {
					data, err = os.ReadFile(filepath.Clean(file))
				}
				if err != nil {
					return fmt.Errorf("read workflow arguments: %w", err)
				}
			}
			if !json.Valid(data) || !strings.HasPrefix(strings.TrimSpace(string(data)), "{") {
				return fmt.Errorf("workflow arguments must be a JSON object")
			}
			agent, _, err := e.agent()
			if err != nil {
				return err
			}
			reply, err := agent.Workflow(cmd.Context(), args[0], mode, data)
			if err != nil {
				return err //nolint:wrapcheck // already a plain message for the person
			}
			return e.printReply(reply)
		}}
	command.Flags().StringVar(&mode, "mode", "read", "workflow mode: read or execute")
	command.Flags().StringVar(&rawJSON, "json", "", "workflow arguments as a JSON object")
	command.Flags().StringVar(&file, "file", "", "read workflow arguments from a file or - for stdin")
	return command
}

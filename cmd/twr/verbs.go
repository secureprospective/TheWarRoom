package main

import (
	"fmt"
	"slices"

	"github.com/spf13/cobra"
)

type commandVerb struct {
	ID, Label string
	Modes     []string
}

// The named verbs use exactly the existing JSON dispatch and human-confirmation path.
func (e env) commandVerbs() []*cobra.Command {
	var commands []*cobra.Command
	for _, verb := range generatedVerbs() {
		mode, confirmation := "preview", ""
		command := &cobra.Command{Use: verb.ID + " CHANGE.json", Short: verb.Label, Args: cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				if !slices.Contains(verb.Modes, mode) {
					return fmt.Errorf("choose an available mode for %s", verb.ID)
				}
				raw, err := e.readChange(args[0])
				if err != nil {
					return err
				}
				if raw.Command != "" && raw.Command != verb.ID {
					return fmt.Errorf("the file names a different league action")
				}
				raw.Command = verb.ID
				reply, err := e.change(cmd.Context(), raw, mode, confirmation)
				if err != nil {
					return err
				}
				return e.print(reply)
			}}
		command.Flags().StringVar(&mode, "mode", "preview", "preview or execute")
		command.Flags().StringVar(&confirmation, "confirmation", "", "the code the commissioner confirmed in The War Room")
		commands = append(commands, command)
	}
	return commands
}

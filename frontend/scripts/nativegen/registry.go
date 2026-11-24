package main

import (
	"fmt"

	catalog "github.com/secureprospective/TheWarRoom/internal/commandcatalog"
)

func registryCommands(door catalog.Door) ([]catalog.Command, error) {
	if door == catalog.MFLDoor {
		return nil, fmt.Errorf("MFL commands exist only in the desktop registry (internal/app/offers)")
	}
	entries, err := catalog.PublishedEntries()
	if err != nil {
		return nil, fmt.Errorf("load catalog entries: %w", err)
	}
	var commands []catalog.Command
	for _, entry := range entries {
		if door == catalog.NativeDoor && entry.Native {
			commands = append(commands, entry.Definition)
		}
	}
	return commands, nil
}

func registryServerWorkflowCommands() ([]catalog.Command, error) {
	entries, err := catalog.PublishedEntries()
	if err != nil {
		return nil, fmt.Errorf("load server workflow catalog: %w", err)
	}
	var commands []catalog.Command
	for _, entry := range entries {
		if entry.Workflow && entry.Definition.Availability == "server" {
			commands = append(commands, entry.Definition)
		}
	}
	return commands, nil
}

func registryBrowserCommands() ([]catalog.BrowserCommand, error) {
	entries, err := catalog.PublishedEntries()
	if err != nil {
		return nil, fmt.Errorf("load browser catalog: %w", err)
	}
	var commands []catalog.BrowserCommand
	for _, entry := range entries {
		if entry.Browser != nil {
			commands = append(commands, *entry.Browser)
		}
	}
	return commands, nil
}

func catalogEntries(published []catalog.PublishedEntry) []catalog.Entry {
	entries := make([]catalog.Entry, 0, len(published))
	for _, p := range published {
		var workflow *catalog.WorkflowCommand
		if p.Workflow {
			workflow = &catalog.WorkflowCommand{}
		}
		entries = append(entries, catalog.Entry{
			ID:         p.ID,
			Label:      p.Label,
			Definition: p.Definition,
			CLI:        p.CLI,
			Browser:    p.Browser,
			Workflow:   workflow,
		})
	}
	return entries
}

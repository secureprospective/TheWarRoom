package main

import (
	"fmt"

	catalog "github.com/secureprospective/TheWarRoom/internal/commandcatalog"
	"github.com/secureprospective/TheWarRoom/internal/composition"
)

func registryCommands(door catalog.Door) ([]catalog.Command, error) {
	registry, err := composition.CommandRegistry()
	if err != nil {
		return nil, fmt.Errorf("bind generation registry: %w", err)
	}
	var commands []catalog.Command
	for _, entry := range registry.Entries() {
		native := door == catalog.NativeDoor && entry.Native != nil
		mfl := door == catalog.MFLDoor && entry.MFL != nil
		if native || mfl {
			commands = append(commands, entry.Definition)
		}
	}
	return commands, nil
}

func registryServerWorkflowCommands() ([]catalog.Command, error) {
	registry, err := composition.CommandRegistry()
	if err != nil {
		return nil, fmt.Errorf("bind server workflow registry: %w", err)
	}
	var commands []catalog.Command
	for _, entry := range registry.Entries() {
		if entry.Workflow != nil && entry.Definition.Availability == "server" {
			commands = append(commands, entry.Definition)
		}
	}
	return commands, nil
}

func registryBrowserCommands() ([]catalog.BrowserCommand, error) {
	registry, err := composition.CommandRegistry()
	if err != nil {
		return nil, fmt.Errorf("bind browser registry: %w", err)
	}
	var commands []catalog.BrowserCommand
	for _, entry := range registry.Entries() {
		if entry.Browser != nil {
			commands = append(commands, *entry.Browser)
		}
	}
	return commands, nil
}

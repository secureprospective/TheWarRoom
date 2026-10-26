package serverhttp

import (
	"context"
	"encoding/json"

	catalog "github.com/secureprospective/TheWarRoom/internal/commandcatalog"
	"github.com/secureprospective/TheWarRoom/internal/leagueservice"
)

type workflowCommand struct {
	validate func(RawWorkflow) error
	run      func(context.Context, *leagueservice.Service, RawWorkflow) (json.RawMessage, error)
}

func workflowCommands() map[string]workflowCommand {
	out := map[string]workflowCommand{}
	for id, entry := range catalog.WorkflowCommands() {
		out[id] = workflowCommand{validate: entry.Validate, run: entry.Run}
	}
	return out
}

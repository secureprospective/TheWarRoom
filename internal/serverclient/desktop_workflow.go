package serverclient

import (
	"context"
	"fmt"

	"github.com/secureprospective/TheWarRoom/internal/serverhttp"
)

const DesktopWorkflowPath = "/desktop/workflow"

// Workflow keeps the device session in Go and binds every request to this window's league.
func (d *Desktop) Workflow(ctx context.Context, raw serverhttp.RawWorkflow) (serverhttp.WorkflowResponse, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	var result serverhttp.WorkflowResponse
	if err := raw.Validate(); err != nil {
		return result, fmt.Errorf("check league action: %w", err)
	}
	league, err := raw.League()
	if err != nil || league != d.league {
		return result, fmt.Errorf("choose this window's league")
	}
	d.restore()
	if d.bearer == "" {
		return result, fmt.Errorf("sign in before opening your franchise")
	}
	body, err := postDesktop(ctx, d, serverhttp.WorkflowPath, serverhttp.RawNativeRequest{Protocol: 1, Workflow: &raw})
	if err != nil {
		return result, err
	}
	if err := decodeDesktop(body, &result); err != nil {
		return result, err
	}
	return result, nil
}

package commissioner

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"

	catalog "github.com/secureprospective/TheWarRoom/internal/commandcatalog"
	"github.com/secureprospective/TheWarRoom/internal/commissioner"
)

// TargetCommissionerPreview checks the execute body without recording the change.
func (a *API) TargetCommissionerPreview(body string) (commissioner.Preview, error) {
	raw, err := a.decodeCommissionerPreview(body)
	if err != nil {
		return Blocked(err), nil
	}
	//nolint:wrapcheck // The shared service supplies the existing user-facing context.
	return a.reads.Preview(raw)
}

// A seed preview accepts the exact TargetNativeSeed request, never client-supplied starting assets.
func (a *API) decodeCommissionerPreview(body string) (commissioner.RawCommand, error) {
	raw, err := commissioner.DecodePreview(body)
	if err == nil {
		if raw.Command == commissioner.NativeSeed {
			return raw, fmt.Errorf("preview starting assets through the held-data request")
		}
		return raw, nil
	}
	if raw.Command != "" {
		return raw, fmt.Errorf("decode preview: %w", err)
	}
	var input RawNativeSeedRequest
	decoder := json.NewDecoder(bytes.NewBufferString(body))
	decoder.DisallowUnknownFields()
	if decodeErr := decoder.Decode(&input); decodeErr != nil {
		return raw, fmt.Errorf("decode preview: %w", err)
	}
	if decodeErr := decoder.Decode(&struct{}{}); decodeErr != io.EOF {
		return raw, fmt.Errorf("decode preview: %w", err)
	}
	if validateErr := input.ValidatePreview(); validateErr != nil {
		return raw, validateErr
	}
	if readyErr := a.host.Ready(); readyErr != nil {
		return raw, readyErr //nolint:wrapcheck // Preserve the existing refusal text and error identity.
	}
	ctx, cancel := context.WithTimeout(a.host.CommissionerContext(), refreshTimeout)
	defer cancel()
	return a.seedCommand(ctx, input)
}

// WorkflowResponse is the local form of the hosted workflow envelope. The league still owns its
// wording and whether it may be changed; the desktop does not supply either.
type WorkflowResponse struct {
	Label    string          `json:"label"`
	Writable bool            `json:"writable"`
	Football string          `json:"football"`
	Data     json.RawMessage `json:"data"`
}

// TargetWorkflow runs the same bounded workflow the hosted door uses, against the local app league.
func (a *API) TargetWorkflow(body string) (WorkflowResponse, error) {
	var raw catalog.RawWorkflow
	if err := json.Unmarshal([]byte(body), &raw); err != nil {
		return WorkflowResponse{}, fmt.Errorf("read league action: %w", err)
	}
	league, err := raw.League()
	if err != nil {
		return WorkflowResponse{}, err //nolint:wrapcheck // Preserve the workflow's refusal text.
	}
	location, err := a.reads.Location(league)
	if err != nil {
		return WorkflowResponse{}, err //nolint:wrapcheck // Preserve the league's authority wording.
	}
	data, err := catalog.RunWorkflow(a.host.CommissionerContext(), a.reads, raw)
	if err != nil {
		return WorkflowResponse{}, err //nolint:wrapcheck // Preserve the workflow's refusal text.
	}
	return WorkflowResponse{Label: location.Label, Writable: location.Writable, Football: "local league",
		Data: data}, nil
}

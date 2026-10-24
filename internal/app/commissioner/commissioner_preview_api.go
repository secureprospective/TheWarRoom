package commissioner

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"

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
	//nolint:wrapcheck // Preserve the existing refusal text and error identity.
	return a.host.NativeSeedCommand(ctx, input)
}

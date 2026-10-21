package main

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"

	commissionerapp "github.com/secureprospective/TheWarRoom/internal/app/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/commissioner"
)

//go:embed docs/ui/endpoint-registry.csv
var commissionerPreviewRegistry string

// TargetCommissionerPreview checks the execute body without recording the change.
func (a *App) TargetCommissionerPreview(body string) (commissioner.Preview, error) {
	raw, err := a.decodeCommissionerPreview(body)
	if err != nil {
		return commissionerapp.Blocked(err), nil
	}
	if err := a.ready(); err != nil {
		return commissioner.Preview{}, fmt.Errorf("league preview startup: %w", err)
	}
	principal, err := a.commissionerPrincipal()
	if err != nil {
		return commissioner.Preview{}, err
	}
	catalogue, err := commissioner.ReadPreviewCatalogue(commissionerPreviewRegistry)
	if err != nil {
		return commissioner.Preview{}, fmt.Errorf("league preview catalogue: %w", err)
	}
	ctx, cancel := context.WithTimeout(a.core.Ctx, refreshTimeout)
	defer cancel()
	preview, err := a.core.History.PreviewCommissionerWithFacts(ctx, raw, principal, a.core.CommissionerAuthorizer,
		catalogue, a.nativeFacts(), a.ownerZone())
	if err != nil {
		return commissioner.Preview{}, fmt.Errorf("league preview: %w", err)
	}
	return preview, nil
}

// A seed preview accepts the exact TargetNativeSeed request, never client-supplied starting assets.
func (a *App) decodeCommissionerPreview(body string) (commissioner.RawCommand, error) {
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
	if readyErr := a.ready(); readyErr != nil {
		return raw, readyErr
	}
	ctx, cancel := context.WithTimeout(a.core.Ctx, refreshTimeout)
	defer cancel()
	return a.nativeSeedCommand(ctx, input)
}

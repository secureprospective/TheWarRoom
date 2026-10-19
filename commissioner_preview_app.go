package main

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
)

//go:embed docs/ui/endpoint-registry.csv
var commissionerPreviewRegistry string

// TargetCommissionerPreview checks the execute body without recording the change.
func (a *App) TargetCommissionerPreview(body string) (commissioner.Preview, error) {
	raw, err := a.decodeCommissionerPreview(body)
	if err != nil {
		var refusal *commissioner.Refusal
		err = commissioner.Refused(err)
		words := err.Error()
		if errors.As(err, &refusal) {
			words = refusal.Reason()
		}
		return commissioner.Preview{Fields: []commissioner.PreviewField{},
			Affects:  commissioner.PreviewAffects{Readings: []string{}, Words: "Not known yet"},
			Blockers: []string{words}}, nil
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
	ctx, cancel := context.WithTimeout(a.ctx, refreshTimeout)
	defer cancel()
	preview, err := a.history.PreviewCommissionerWithFacts(ctx, raw, principal, a.commissionerAuthorizer,
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
	ctx, cancel := context.WithTimeout(a.ctx, refreshTimeout)
	defer cancel()
	return a.nativeSeedCommand(ctx, input)
}

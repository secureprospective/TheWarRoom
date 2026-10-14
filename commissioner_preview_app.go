package main

import (
	"context"
	_ "embed"
	"errors"
	"fmt"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
)

//go:embed docs/ui/endpoint-registry.csv
var commissionerPreviewRegistry string

// TargetCommissionerPreview checks the execute body without recording the change.
func (a *App) TargetCommissionerPreview(body string) (commissioner.Preview, error) {
	raw, err := commissioner.Decode(body)
	if err != nil {
		var refusal *commissioner.Refusal
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
	preview, err := a.history.PreviewCommissioner(ctx, raw, principal, a.commissionerAuthorizer, catalogue)
	if err != nil {
		return commissioner.Preview{}, fmt.Errorf("league preview: %w", err)
	}
	return preview, nil
}

package leagueservice

import (
	"context"
	"fmt"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/leagueweek"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

type footballDigestKey struct{}

// WithFootballDigest carries the evidence the caller reviewed, not permission to use it.
func WithFootballDigest(ctx context.Context, digest string) context.Context {
	return context.WithValue(ctx, footballDigestKey{}, digest)
}

// NewFootballHost selects server-held reference evidence at startup, before concurrent
// requests. Invalid selection leaves the store's prior environment untouched.
func NewFootballHost(ctx context.Context, host Host, store *history.Store, digest string) (Host, error) {
	evidence, err := store.LoadFootball(ctx, digest)
	if err != nil {
		return nil, fmt.Errorf("select hosted football evidence: %w", err)
	}
	store.SetLineupEnvironment(evidence)
	return WithPinnedFootball(host, evidence)
}

type footballHost struct {
	Host
	evidence *history.PinnedFootball
}

func (h footballHost) NativeFacts() nativecheck.Facts { return h.evidence.PlayerFacts() }

func (h footballHost) NativeScheduleDates(ctx context.Context, season int, _ time.Time) ([]leagueweek.Week, error) {
	weeks, err := h.evidence.ScheduleDates(ctx, season)
	if err != nil {
		return nil, fmt.Errorf("hosted schedule: %w", err)
	}
	return weeks, nil
}

func (h footballHost) check(ctx context.Context) error {
	digest, _ := ctx.Value(footballDigestKey{}).(string)
	if err := h.evidence.CheckDigest(digest); err != nil {
		return fmt.Errorf("hosted football evidence: %w", err)
	}
	return nil
}

func (h footballHost) ExecuteCommissionerWithFacts(ctx context.Context, raw commissioner.RawCommand,
	principal string, auth commissioner.Authorizer, _ nativecheck.Facts, zone *time.Location) (commissioner.Receipt, error) {
	if err := h.check(ctx); err != nil {
		return commissioner.Receipt{}, err
	}
	guarded := commissioner.WithCredentialAdmission(ctx, footballAdmission{host: h, credential: func(access commissioner.Access) error {
		return commissioner.AdmitCredential(ctx, access)
	}})
	receipt, err := h.Host.ExecuteCommissionerWithFacts(guarded, raw, principal, auth, h.NativeFacts(), zone)
	if err != nil {
		return commissioner.Receipt{}, fmt.Errorf("hosted football change: %w", err)
	}
	return receipt, nil
}

func (h footballHost) PreviewCommissionerWithFacts(ctx context.Context, raw commissioner.RawCommand,
	principal string, auth commissioner.Authorizer, catalogue commissioner.PreviewCatalogue,
	_ nativecheck.Facts, zone *time.Location) (commissioner.Preview, error) {
	if err := h.check(ctx); err != nil {
		return commissioner.Preview{}, err
	}
	preview, err := h.Host.PreviewCommissionerWithFacts(ctx, raw, principal, auth, catalogue, h.NativeFacts(), zone)
	if err != nil {
		return commissioner.Preview{}, fmt.Errorf("hosted football preview: %w", err)
	}
	return preview, nil
}

type footballAdmission struct {
	host       footballHost
	credential func(commissioner.Access) error
}

func (a footballAdmission) Admit(ctx context.Context, access commissioner.Access) error {
	if err := a.credential(access); err != nil {
		return fmt.Errorf("admit football credential: %w", err)
	}
	return a.host.check(ctx)
}

func (h footballHost) RecordDraftPick(ctx context.Context, permit commissioner.Permit, in commissioner.DraftPickInput, principal string, _ nativecheck.Facts, zone *time.Location) (commissioner.Receipt, error) {
	if err := h.check(ctx); err != nil {
		return commissioner.Receipt{}, err
	}
	host, ok := h.Host.(draftPickHost)
	if !ok {
		return commissioner.Receipt{}, fmt.Errorf("draft room unavailable")
	}
	guarded := commissioner.WithCredentialAdmission(ctx, footballAdmission{host: h, credential: func(access commissioner.Access) error { return commissioner.AdmitCredential(ctx, access) }})
	receipt, err := host.RecordDraftPick(guarded, permit, in, principal, h.NativeFacts(), zone)
	if err != nil {
		return commissioner.Receipt{}, fmt.Errorf("hosted football pick: %w", err)
	}
	return receipt, nil
}

// WithPinnedFootball shares immutable startup evidence across request-scoped hosts.
func WithPinnedFootball(host Host, evidence *history.PinnedFootball) (Host, error) {
	if evidence == nil {
		return nil, fmt.Errorf("select football evidence at startup")
	}
	return footballHost{Host: host, evidence: evidence}, nil
}

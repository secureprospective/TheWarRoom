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

// StoredHost binds a request's trusted identity and lifetime to server-owned history.
// It supplies no installation identity, domain override, network facts or local schedule.
type StoredHost struct {
	*history.Store
	Context   func() context.Context
	Principal string
	Registry  string
}

func (h StoredHost) Ready() error {
	if h.Store == nil || h.Context == nil || h.Principal == "" {
		return fmt.Errorf("server session is unavailable")
	}
	return nil
}
func (h StoredHost) CommissionerContext() context.Context { return h.Context() }
func (h StoredHost) CommissionerPrincipal() (string, error) {
	if err := h.Ready(); err != nil {
		return "", err
	}
	return h.Principal, nil
}
func (StoredHost) CommissionerAuthorization() commissioner.Authorizer { return nil }
func (StoredHost) CommissionerNow() time.Time                         { return time.Now() }
func (StoredHost) CommissionerZone() *time.Location                   { return time.UTC }
func (h StoredHost) CommissionerPreviewRegistry() string              { return h.Registry }
func (StoredHost) NativeFacts() nativecheck.Facts                     { return nil }
func (h StoredHost) CommissionerRecords(ctx context.Context, league string) ([]commissioner.Record, error) {
	events, err := h.Events(ctx, league, commissioner.Stream, 0)
	if err != nil {
		return nil, fmt.Errorf("server league records: %w", err)
	}
	records := make([]commissioner.Record, 0, len(events))
	for _, event := range events {
		if event.Payload.Commissioner == nil {
			return nil, fmt.Errorf("invalid server league record")
		}
		records = append(records, *event.Payload.Commissioner)
	}
	// The same verified mappings, when supplied by trusted composition, apply to policy and retries.
	resolved, err := commissioner.ResolvePrincipals(ctx, league, records)
	if err != nil {
		return nil, fmt.Errorf("resolve server identities: %w", err)
	}
	return resolved, nil
}
func (StoredHost) NativeScheduleDates(context.Context, int, time.Time) ([]leagueweek.Week, error) {
	return nil, fmt.Errorf("use the server-held league schedule")
}

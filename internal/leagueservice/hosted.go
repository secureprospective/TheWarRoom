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
func (h StoredHost) CommissionerNow() time.Time                       { return h.Now() }
func (StoredHost) CommissionerZone() *time.Location                   { return time.UTC }
func (h StoredHost) CommissionerPreviewRegistry() string              { return h.Registry }
func (StoredHost) NativeFacts() nativecheck.Facts                     { return nil }

// CommissionerRecords serves the store's revision-keyed shared snapshot, which carries the replay and
// people-state memos; rebuilding records from a deep-cloned event list on every request defeated both and
// was most of each hosted read's server time (lab CT 254 profile, 11-08). The snapshot is read-only by
// contract (TestSharedRecordsStayUnchangedByReadings).
func (h StoredHost) CommissionerRecords(ctx context.Context, league string) ([]commissioner.Record, error) {
	records, err := h.SharedCommissionerRecords(ctx, league)
	if err != nil {
		return nil, fmt.Errorf("server league records: %w", err)
	}
	return records, nil
}
func (StoredHost) NativeScheduleDates(context.Context, int, time.Time) ([]leagueweek.Week, error) {
	return nil, fmt.Errorf("use the server-held league schedule")
}

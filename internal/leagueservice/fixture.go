// Package leagueservice serves native league readings through the shared commissioner engine.
package leagueservice

import (
	"context"
	_ "embed"
	"fmt"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/leagueweek"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

//go:embed fixture.json
var fixtureCommand string

// FixtureRecords returns fresh validated records, never a live database or owner identity.
func FixtureRecords() ([]commissioner.Record, error) {
	raw, err := commissioner.Decode(fixtureCommand)
	if err != nil {
		return nil, fmt.Errorf("decode fixture: %w", err)
	}
	raw, err = raw.Prepared()
	if err != nil {
		return nil, fmt.Errorf("prepare fixture: %w", err)
	}
	raw.Gravity = "G3"
	canonical, err := raw.Canonical("fixture.creator")
	if err != nil {
		return nil, fmt.Errorf("canonicalize fixture: %w", err)
	}
	record := commissioner.Record{
		Command: raw, Principal: "fixture.creator", Role: "commissioner", Gravity: "G3", Canonical: canonical,
		Receipt: commissioner.Receipt{Entry: "fixture-create", League: raw.League, Revision: 1,
			RecordedAt: time.Date(2026, 10, 23, 12, 0, 0, 0, time.UTC)},
	}
	if err := record.Validate(); err != nil {
		return nil, fmt.Errorf("validate fixture: %w", err)
	}
	return []commissioner.Record{record}, nil
}

type fixtureObserver struct{}

func (fixtureObserver) Allow(_, _, _ string) (string, error) {
	return "", fmt.Errorf("the fixture observer has no command permissions")
}

// FixtureLedger has no arguments that could select live state or elevate the fixture observer.
func FixtureLedger(ctx context.Context) (commissioner.Ledger, error) {
	if err := ctx.Err(); err != nil {
		return commissioner.Ledger{}, fmt.Errorf("read fixture: %w", err)
	}
	records, err := FixtureRecords()
	if err != nil {
		return commissioner.Ledger{}, err
	}
	reading, err := commissioner.Reading("native.https-fixture", "fixture.observer", records, fixtureObserver{})
	if err != nil {
		return commissioner.Ledger{}, fmt.Errorf("read fixture ledger: %w", err)
	}
	return reading, nil
}

// NewFixtureChanges supplies only the fixed development identity and scope. The caller
// owns a disposable store; no account, owner identity or MFL credentials are accepted.
func NewFixtureChanges(store *history.Store) *Service {
	return NewHosted(fixtureChangeHost{Store: store}, fixtureChangeAdmission{})
}

// NewFixtureChangesWithWorker keeps the fixture transport on the hosted scheduling path.
func NewFixtureChangesWithWorker(store *history.Store, worker *DueWorker) *Service {
	return NewHosted(worker.Host(fixtureChangeHost{Store: store}), fixtureChangeAdmission{})
}

type fixtureChangeAdmission struct{}

func (fixtureChangeAdmission) Admit(_ context.Context, access commissioner.Access) error {
	if access.Principal != "fixture.creator" || access.League != "native.https-fixture" ||
		(access.Operation != commissioner.NotePost && access.Operation != "commish.ledger.read") {
		return fmt.Errorf("outside fixture authority")
	}
	return nil
}

type fixtureChangeHost struct{ *history.Store }

func (fixtureChangeHost) Ready() error                                       { return nil }
func (fixtureChangeHost) CommissionerContext() context.Context               { return context.Background() }
func (fixtureChangeHost) CommissionerPrincipal() (string, error)             { return "fixture.creator", nil }
func (fixtureChangeHost) CommissionerAuthorization() commissioner.Authorizer { return nil }
func (fixtureChangeHost) CommissionerNow() time.Time                         { return time.Now() }
func (fixtureChangeHost) CommissionerZone() *time.Location                   { return time.UTC }
func (fixtureChangeHost) CommissionerPreviewRegistry() string                { return "" }
func (fixtureChangeHost) NativeFacts() nativecheck.Facts                     { return nil }
func (h fixtureChangeHost) CommissionerRecords(ctx context.Context, league string) ([]commissioner.Record, error) {
	events, err := h.Events(ctx, league, commissioner.Stream, 0)
	if err != nil {
		return nil, fmt.Errorf("fixture records: %w", err)
	}
	records := make([]commissioner.Record, 0, len(events))
	for _, event := range events {
		if event.Payload.Commissioner == nil {
			return nil, fmt.Errorf("invalid fixture record")
		}
		records = append(records, *event.Payload.Commissioner)
	}
	return records, nil
}
func (fixtureChangeHost) NativeScheduleDates(context.Context, int, time.Time) ([]leagueweek.Week, error) {
	return nil, fmt.Errorf("the fixture has no football schedule")
}

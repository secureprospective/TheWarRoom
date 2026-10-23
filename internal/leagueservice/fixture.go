// Package leagueservice serves native league readings through the shared commissioner engine.
package leagueservice

import (
	"context"
	_ "embed"
	"fmt"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
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

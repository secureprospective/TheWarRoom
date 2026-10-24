package history

import (
	"context"
	"fmt"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

// The native legality readings judge the recorded league against its own rules at the store's clock.
// Player facts arrive from the caller, because the ledger holds none.

func (s *Store) NativeCapCheck(ctx context.Context, scope commissioner.RawNativeCapScope) (commissioner.NativeCapCheck, error) {
	records, err := s.nativeRecords(ctx, scope.League)
	if err != nil {
		return commissioner.NativeCapCheck{}, err
	}
	reading, err := commissioner.NativeCapCheckReading(scope, records, s.now())
	if err != nil {
		return commissioner.NativeCapCheck{}, fmt.Errorf("read native salary cap check: %w", err)
	}
	reading.Revision = commissioner.ViewRevision(ctx, records, reading.Revision)
	return reading, nil
}

func (s *Store) NativeRosterCheck(ctx context.Context, league string, facts nativecheck.Facts) (commissioner.NativeRosterCheck, error) {
	records, err := s.nativeRecords(ctx, league)
	if err != nil {
		return commissioner.NativeRosterCheck{}, err
	}
	reading, err := commissioner.NativeRosterCheckReading(league, records, s.now(), facts)
	if err != nil {
		return commissioner.NativeRosterCheck{}, fmt.Errorf("read native roster check: %w", err)
	}
	reading.Revision = commissioner.ViewRevision(ctx, records, reading.Revision)
	return reading, nil
}

func (s *Store) NativeReserveCheck(ctx context.Context, league string, facts nativecheck.Facts) (commissioner.NativeReserveCheck, error) {
	records, err := s.nativeRecords(ctx, league)
	if err != nil {
		return commissioner.NativeReserveCheck{}, err
	}
	reading, err := commissioner.NativeReserveCheckReading(league, records, s.now(), facts)
	if err != nil {
		return commissioner.NativeReserveCheck{}, fmt.Errorf("read native reserve check: %w", err)
	}
	reading.Revision = commissioner.ViewRevision(ctx, records, reading.Revision)
	return reading, nil
}

func (s *Store) NativeLineupCheck(ctx context.Context, in commissioner.RawNativeLineup, facts nativecheck.Facts) (commissioner.NativeLineupCheck, error) {
	records, err := s.nativeRecords(ctx, in.League)
	if err != nil {
		return commissioner.NativeLineupCheck{}, err
	}
	reading, err := commissioner.NativeLineupCheckReading(in, records, s.now(), facts)
	if err != nil {
		return commissioner.NativeLineupCheck{}, fmt.Errorf("read native lineup check: %w", err)
	}
	reading.Revision = commissioner.ViewRevision(ctx, records, reading.Revision)
	return reading, nil
}

func (s *Store) NativeHouseRules(ctx context.Context, league string) (commissioner.NativeHouseRules, error) {
	records, err := s.nativeRecords(ctx, league)
	if err != nil {
		return commissioner.NativeHouseRules{}, err
	}
	reading, err := commissioner.NativeHouseRulesReading(league, records, s.now())
	if err != nil {
		return commissioner.NativeHouseRules{}, fmt.Errorf("read native house rules: %w", err)
	}
	reading.Revision = commissioner.ViewRevision(ctx, records, reading.Revision)
	return reading, nil
}

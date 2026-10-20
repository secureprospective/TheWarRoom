package history

import (
	"context"
	"fmt"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
)

func (s *Store) nativeRecords(ctx context.Context, league string) ([]commissioner.Record, error) {
	if !strings.HasPrefix(league, "native.") || league == "native." || strings.TrimSpace(league) != league {
		return nil, fmt.Errorf("choose an app league")
	}
	tx, err := s.pools.Read().BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("read native league events: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	records, err := s.transactionRecords(ctx, tx, league)
	if err != nil {
		return nil, fmt.Errorf("read native league records: %w", err)
	}
	return records, nil
}

func (s *Store) NativeAssets(ctx context.Context, league string) (commissioner.NativeAssets, error) {
	records, err := s.nativeRecords(ctx, league)
	if err != nil {
		return commissioner.NativeAssets{}, err
	}
	reading, err := commissioner.NativeAssetsReading(league, records, s.now())
	if err != nil {
		return commissioner.NativeAssets{}, fmt.Errorf("read native assets: %w", err)
	}
	return reading, nil
}

func (s *Store) NativeDirectory(ctx context.Context, league string) (commissioner.NativeDirectory, error) {
	records, err := s.nativeRecords(ctx, league)
	if err != nil {
		return commissioner.NativeDirectory{}, err
	}
	reading, err := commissioner.NativeDirectoryReading(league, records, s.now())
	if err != nil {
		return commissioner.NativeDirectory{}, fmt.Errorf("read native league directory: %w", err)
	}
	return reading, nil
}

// NativeSeedRetry returns a recorded command, not a receipt: ExecuteCommissioner must still
// check the current principal's permission and the complete request identity on every retry.
func (s *Store) NativeSeedRetry(ctx context.Context, league, request string) (commissioner.RawCommand, bool, error) {
	records, err := s.nativeRecords(ctx, league)
	if err != nil {
		return commissioner.RawCommand{}, false, err
	}
	raw, found := commissioner.NativeSeedRetry(records, request)
	return raw, found, nil
}

// Corrections reads public compensations without exposing private owner claims.
func (s *Store) Corrections(ctx context.Context, league string) (commissioner.Corrections, error) {
	records, err := s.nativeRecords(ctx, league)
	if err != nil {
		return commissioner.Corrections{}, err
	}
	out, err := commissioner.CorrectionsReading(league, records, s.now())
	if err != nil {
		return commissioner.Corrections{}, fmt.Errorf("read native corrections: %w", err)
	}
	return out, nil
}

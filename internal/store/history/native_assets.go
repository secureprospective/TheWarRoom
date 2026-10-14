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
	events, err := s.Events(ctx, league, commissioner.Stream, 0)
	if err != nil {
		return nil, fmt.Errorf("read native league events: %w", err)
	}
	records, err := commissionerRecords(events)
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
	reading, err := commissioner.NativeAssetsReading(league, records)
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

package history

import (
	"context"
	"fmt"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
)

// MirrorLedger answers reads for the MFL-mirror league with records built in memory from the held
// MFL data: a Create from its settings and a Seed from its rosters. The mirror is a league setting, so
// every native reading replays it unchanged. It is never written; commands still refuse the league.
type MirrorLedger struct {
	League  string
	Records func(ctx context.Context) ([]commissioner.Record, error)
}

// SetMirrorLedger lets reads answer the held MFL league.
func (s *Store) SetMirrorLedger(m MirrorLedger) { s.mirror = &m }

func (s *Store) mirrorRecords(ctx context.Context, league string) ([]commissioner.Record, bool, error) {
	if s.mirror == nil || s.mirror.League == "" || league != s.mirror.League {
		return nil, false, nil
	}
	records, err := s.mirror.Records(ctx)
	if err != nil {
		return nil, true, fmt.Errorf("read the MFL league: %w", err)
	}
	return records, true, nil
}

// refuseMirror keeps the in-memory mirror league read-only: its moves happen on MFL.
func (s *Store) refuseMirror(raws ...commissioner.RawCommand) error {
	for _, raw := range raws {
		if s.mirror != nil && raw.League == s.mirror.League {
			return commissioner.Refused(fmt.Errorf("this league is run on MFL; make the change there")) //nolint:wrapcheck // Refused is the refusal wrapper.
		}
	}
	return nil
}

package history

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/fantasyscore/source"
)

// heldAbsenceProofs reads only held affirmative evidence. An unsupported held secondary
// statistics source blocks zero admission rather than silently ignoring potential credits.
func (s *Store) heldAbsenceProofs(ctx context.Context, season int) ([]source.RawAbsenceProof, string, time.Time, error) {
	blocked, err := s.unsupportedZeroSources(ctx)
	if err != nil {
		return nil, "", time.Time{}, err
	}
	if blocked {
		return nil, "", time.Time{}, nil
	}
	gsis, err := s.SourceIDs(ctx, "gsis")
	if err != nil {
		return nil, "", time.Time{}, err
	}
	pfr, err := s.SourceIDs(ctx, "pfr")
	if err != nil {
		return nil, "", time.Time{}, err
	}
	bridge := make(map[string]string)
	for player, id := range pfr {
		if gsis[player] != "" {
			bridge[id] = gsis[player]
		}
	}
	var out []source.RawAbsenceProof
	var hash bytes.Buffer
	var at time.Time
	for _, name := range []string{"snap_counts", "injuries"} {
		rows, body, stamp, err := s.heldProofFeed(ctx, season, name)
		if err != nil {
			return nil, "", time.Time{}, err
		}
		for _, r := range rows {
			if r.GSIS == "" {
				r.GSIS = bridge[r.PFR]
			}
			if r.GSIS != "" && r.Season == fmt.Sprint(season) {
				out = append(out, r)
			}
		}
		hash.Write(body)
		if stamp.After(at) {
			at = stamp
		}
	}
	encoded, err := json.Marshal(out)
	if err != nil {
		return nil, "", time.Time{}, fmt.Errorf("fingerprint absence evidence: %w", err)
	}
	hash.Write(encoded)
	sum := sha256.Sum256(hash.Bytes())
	return out, hex.EncodeToString(sum[:]), at, nil
}

func (s *Store) heldProofFeed(ctx context.Context, season int, name string) ([]source.RawAbsenceProof, []byte, time.Time, error) {
	feed, ok := s.reg.Feed(name)
	if !ok {
		return nil, nil, time.Time{}, nil
	}
	var rows []source.RawAbsenceProof
	var body []byte
	var at time.Time
	var parseErr error
	_, err := s.ExactArchivedBodies(ctx, feed.URLFor(season), func(_ string, b []byte, stamp time.Time) bool {
		body, at = b, stamp
		if name == "snap_counts" {
			rows, parseErr = source.ReadSnapProofs(bytes.NewReader(b))
		} else {
			rows, parseErr = source.ReadOutProofs(bytes.NewReader(b))
		}
		return true
	})
	if err != nil {
		return nil, nil, time.Time{}, fmt.Errorf("read held absence evidence: %w", err)
	}
	if parseErr != nil {
		return nil, nil, time.Time{}, fmt.Errorf("validate held absence evidence: %w", parseErr)
	}
	return rows, body, at, nil
}

func (s *Store) unsupportedZeroSources(ctx context.Context) (bool, error) {
	for _, pattern := range []string{"espn", "sleeper"} {
		found, err := s.ArchivedBodies(ctx, pattern, func(_ string, _ []byte, _ time.Time) bool { return true })
		if err != nil {
			return false, err
		}
		if found {
			return true, nil
		}
	}
	return false, nil
}

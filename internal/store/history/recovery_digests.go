package history

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
)

// RecoveryDigests fingerprints what a restore must bring back unchanged: every stored
// evidence body and each league's replayed ledger. A restore compares them before and
// after its sign-in reset; operators compare them with the old server's report.
type RecoveryDigests struct {
	EvidenceFiles int            `json:"evidenceFiles"`
	Evidence      string         `json:"evidence"`
	Leagues       []LeagueReplay `json:"leagues"`
}

type LeagueReplay struct {
	League  string `json:"league"`
	Name    string `json:"name"`
	Entries int    `json:"entries"`
	Replay  string `json:"replay"`
}

func (s *Store) RecoveryDigests(ctx context.Context) (RecoveryDigests, error) {
	var out RecoveryDigests
	var err error
	if out.EvidenceFiles, out.Evidence, err = s.evidenceDigest(ctx); err != nil {
		return out, err
	}
	leagues, err := s.NativeLeagues(ctx)
	if err != nil {
		return out, err
	}
	for _, league := range leagues {
		records, err := s.leagueRecords(ctx, league.ID)
		if err != nil {
			return out, err
		}
		body, err := json.Marshal(commissioner.Replay(records))
		if err != nil {
			return out, fmt.Errorf("history: encode league replay: %w", err)
		}
		sum := sha256.Sum256(body)
		out.Leagues = append(out.Leagues, LeagueReplay{league.ID, league.Name, len(records), hex.EncodeToString(sum[:])})
	}
	return out, nil
}

// evidenceDigest hashes each stored body, keyed by its recorded identity and size.
func (s *Store) evidenceDigest(ctx context.Context) (int, string, error) {
	rows, err := s.pools.Read().QueryContext(ctx, `SELECT sha256, size, body FROM raw_archive ORDER BY sha256`)
	if err != nil {
		return 0, "", fmt.Errorf("history: read evidence: %w", err)
	}
	defer func() { _ = rows.Close() }()
	all, count := sha256.New(), 0
	for rows.Next() {
		var key string
		var size int64
		var body []byte
		if err := rows.Scan(&key, &size, &body); err != nil {
			return 0, "", fmt.Errorf("history: read evidence: %w", err)
		}
		stored := sha256.Sum256(body)
		_, _ = fmt.Fprintf(all, "%s %d %x\n", key, size, stored)
		count++
	}
	if err := rows.Err(); err != nil {
		return 0, "", fmt.Errorf("history: read evidence: %w", err)
	}
	return count, hex.EncodeToString(all.Sum(nil)), nil
}

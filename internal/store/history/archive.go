package history

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"

	"github.com/secureprospective/TheWarRoom/internal/archive"
)

// UnregisteredSource is the source recorded for a fetch whose URL matches no registered source.
// The fetch is still archived; a test keeps every URL the app fetches registered.
const UnregisteredSource = "unregistered"

// Record stores one fetch: its body once per distinct sha256, and a fetch_log row every time.
// It implements archive.Sink.
func (s *Store) Record(ctx context.Context, f archive.Fetch) error {
	source := UnregisteredSource
	if u, err := url.Parse(f.URL); err == nil {
		if id, ok := s.reg.SourceFor(u); ok {
			source = id
		}
	}
	var sha sql.NullString
	if f.SHA256 != "" {
		sha = sql.NullString{String: f.SHA256, Valid: true}
	}

	s.wmu.Lock()
	defer s.wmu.Unlock()
	tx, err := s.pools.Write().BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("history: record fetch: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	if sha.Valid {
		if _, err := tx.ExecContext(ctx,
			`INSERT OR IGNORE INTO raw_archive (sha256, size, body) VALUES (?, ?, ?)`,
			f.SHA256, f.Size, f.Gzip); err != nil {
			return fmt.Errorf("history: archive body %s: %w", f.URL, err)
		}
	}
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO fetch_log (source, url, status, sha256, error, fetched_at) VALUES (?, ?, ?, ?, ?, ?)`,
		source, f.URL, f.Status, sha, f.Err, formatTime(f.FetchedAt)); err != nil {
		return fmt.Errorf("history: log fetch %s: %w", f.URL, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("history: record fetch commit: %w", err)
	}
	return nil
}

// ErrNoBody means no archived body has that sha256.
var ErrNoBody = errors.New("history: no archived body with that sha256")

// Body returns an archived body exactly as it was received, for replaying it through a parser.
func (s *Store) Body(ctx context.Context, sha256 string) ([]byte, error) {
	var gz []byte
	err := s.pools.Read().QueryRowContext(ctx, `SELECT body FROM raw_archive WHERE sha256 = ?`, sha256).Scan(&gz)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("%w: %s", ErrNoBody, sha256)
	}
	if err != nil {
		return nil, fmt.Errorf("history: read body %s: %w", sha256, err)
	}
	body, err := archive.Gunzip(gz)
	if err != nil {
		return nil, fmt.Errorf("history: body %s: %w", sha256, err)
	}
	return body, nil
}

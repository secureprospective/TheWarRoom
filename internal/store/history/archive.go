package history

import (
	"bytes"
	"compress/gzip"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"net/url"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/archive"
)

// UnregisteredSource is the source recorded for a fetch whose URL matches no registered source.
// The fetch is still archived; a test keeps every URL the app fetches registered.
const UnregisteredSource = "unregistered"

// Record stores one fetch: its body once per distinct sha256, and a fetch_log row every time.
// It implements archive.Sink. A fetch that names its source must name a registered one.
func (s *Store) Record(ctx context.Context, f archive.Fetch) error {
	source, err := s.sourceOf(f)
	if err != nil {
		return err
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

func (s *Store) sourceOf(f archive.Fetch) (string, error) {
	if f.Source != "" {
		if _, ok := s.reg.Source(f.Source); !ok {
			return "", fmt.Errorf("history: record %s: unregistered source %q", f.URL, f.Source)
		}
		return f.Source, nil
	}
	if u, err := url.Parse(f.URL); err == nil {
		if id, ok := s.reg.SourceFor(u); ok {
			return id, nil
		}
	}
	return UnregisteredSource, nil
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

// ArchivedBodies offers the bodies of successful fetches whose URL contains part, newest first,
// to accept, until accept takes one. found is false when none was taken. Fallbacks read their
// feed's last good copy this way, so a body that later proved bad is passed over. Imported
// bodies are never offered: a compile reads them through its manifest.
func (s *Store) ArchivedBodies(ctx context.Context, part string,
	accept func(fetchedURL string, body []byte, fetchedAt time.Time) bool) (bool, error) {
	return s.archivedBodies(ctx, part, accept, false, false)
}

// ReferenceBodies also admits imported league-archive bodies for read-only reference surfaces.
// Live-fetch fallbacks still use ArchivedBodies and never relabel an imported capture as live.
func (s *Store) ReferenceBodies(ctx context.Context, part string,
	accept func(fetchedURL string, body []byte, fetchedAt time.Time) bool) (bool, error) {
	return s.archivedBodies(ctx, part, accept, true, false)
}

// ExactArchivedBodies reads one registered URL without admitting similarly named files.
func (s *Store) ExactArchivedBodies(ctx context.Context, wanted string,
	accept func(fetchedURL string, body []byte, fetchedAt time.Time) bool) (bool, error) {
	return s.archivedBodies(ctx, wanted, accept, false, true)
}

const archiveContainsQuery = `
SELECT f.source, f.url, f.fetched_at, r.body FROM fetch_log f JOIN raw_archive r ON r.sha256 = f.sha256
WHERE f.status = 200 AND instr(f.url, ?) > 0 ORDER BY f.fetched_at DESC, f.fetch_id DESC`

const archiveExactQuery = `
SELECT f.source, f.url, f.fetched_at, r.body FROM fetch_log f JOIN raw_archive r ON r.sha256 = f.sha256
WHERE f.status = 200 AND f.url = ? ORDER BY f.fetched_at DESC, f.fetch_id DESC`

func (s *Store) archivedBodies(ctx context.Context, part string,
	accept func(fetchedURL string, body []byte, fetchedAt time.Time) bool, includeImports, exact bool) (found bool, err error) {
	query := archiveContainsQuery
	if exact {
		query = archiveExactQuery
	}
	rows, err := s.pools.Read().QueryContext(ctx, query, part)
	if err != nil {
		return false, fmt.Errorf("history: archived bodies: %w", err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var source, src, at string
		var gz []byte
		if err := rows.Scan(&source, &src, &at, &gz); err != nil {
			return false, fmt.Errorf("history: scan archived body: %w", err)
		}
		if reg, ok := s.reg.Source(source); ok && reg.Imported() && !includeImports {
			continue
		}
		when, err := parseTime(at)
		if err != nil {
			return false, err
		}
		body, err := gunzip(gz)
		if err != nil {
			return false, fmt.Errorf("history: archived body of %s: %w", src, err)
		}
		if accept(src, body, when) {
			return true, nil
		}
	}
	if err := rows.Err(); err != nil {
		return false, fmt.Errorf("history: archived bodies: %w", err)
	}
	return false, nil
}

func gunzip(gz []byte) ([]byte, error) {
	r, err := gzip.NewReader(bytes.NewReader(gz))
	if err != nil {
		return nil, fmt.Errorf("gunzip: %w", err)
	}
	defer func() { _ = r.Close() }()
	body, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("gunzip: %w", err)
	}
	return body, nil
}

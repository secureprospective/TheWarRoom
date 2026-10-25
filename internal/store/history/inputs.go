package history

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// FetchRecord is one fetch_log row of a source. SHA256 is empty when no body was kept.
type FetchRecord struct {
	ID        int64
	URL       string
	SHA256    string
	Error     string
	FetchedAt time.Time
}

// SourceFetches returns every fetch recorded under source, oldest first.
func (s *Store) SourceFetches(ctx context.Context, source string) ([]FetchRecord, error) {
	rows, err := s.pools.Read().QueryContext(ctx, `
SELECT fetch_id, url, COALESCE(sha256, ''), error, fetched_at FROM fetch_log
WHERE source = ? ORDER BY fetched_at, fetch_id`, source)
	if err != nil {
		return nil, fmt.Errorf("history: fetches of %s: %w", source, err)
	}
	defer func() { _ = rows.Close() }()
	var out []FetchRecord
	for rows.Next() {
		var f FetchRecord
		var at string
		if err := rows.Scan(&f.ID, &f.URL, &f.SHA256, &f.Error, &at); err != nil {
			return nil, fmt.Errorf("history: scan fetch of %s: %w", source, err)
		}
		if f.FetchedAt, err = parseTime(at); err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("history: fetches of %s: %w", source, err)
	}
	return out, nil
}

// HeldBodies reports which of shas have an archived body.
func (s *Store) HeldBodies(ctx context.Context, shas []string) (map[string]bool, error) {
	held := map[string]bool{}
	stmt, err := s.pools.Read().PrepareContext(ctx, `SELECT 1 FROM raw_archive WHERE sha256 = ?`)
	if err != nil {
		return nil, fmt.Errorf("history: held bodies: %w", err)
	}
	defer func() { _ = stmt.Close() }()
	for _, sha := range shas {
		var one int
		switch err := stmt.QueryRowContext(ctx, sha).Scan(&one); {
		case err == nil:
			held[sha] = true
		case !errors.Is(err, sql.ErrNoRows):
			return nil, fmt.Errorf("history: held body %s: %w", sha, err)
		}
	}
	return held, nil
}

// RecordImport logs a successful import run against source, so its health shows when it last
// loaded. files is what the run looked at; added is the fetches it recorded.
func (s *Store) RecordImport(ctx context.Context, source string, files, added int) error {
	s.wmu.Lock()
	defer s.wmu.Unlock()
	if _, err := s.pools.ExecWrite(ctx, func(tx *sql.Tx) (sql.Result, error) {
		return tx.ExecContext(ctx, `
INSERT INTO loads (source, loaded_at, facts, added, unresolved) VALUES (?, ?, ?, ?, 0)`,
			source, formatTime(s.now()), files, added)
	}); err != nil {
		return fmt.Errorf("history: record import %s: %w", source, err)
	}
	return nil
}

// ManifestRecord is a published input manifest: its sha256, its name and its encoded entries.
type ManifestRecord struct {
	SHA256      string
	Name        string
	Entries     []byte
	PublishedAt time.Time
}

// PublishManifest stores m once per sha256; published is false when it was already held.
func (s *Store) PublishManifest(ctx context.Context, m ManifestRecord) (published bool, err error) {
	s.wmu.Lock()
	defer s.wmu.Unlock()
	res, err := s.pools.ExecWrite(ctx, func(tx *sql.Tx) (sql.Result, error) {
		return tx.ExecContext(ctx, `
INSERT OR IGNORE INTO input_manifests (manifest_sha, name, entries, published_at) VALUES (?, ?, ?, ?)`,
			m.SHA256, m.Name, string(m.Entries), formatTime(s.now()))
	})
	if err != nil {
		return false, fmt.Errorf("history: publish manifest %s: %w", m.Name, err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("history: publish manifest %s: %w", m.Name, err)
	}
	return n == 1, nil
}

// LatestManifest returns the manifest of that name published last. ok is false when none is.
func (s *Store) LatestManifest(ctx context.Context, name string) (m ManifestRecord, ok bool, err error) {
	var entries, at string
	err = s.pools.Read().QueryRowContext(ctx, `
SELECT manifest_sha, name, entries, published_at FROM input_manifests WHERE name = ?
ORDER BY published_at DESC, rowid DESC LIMIT 1`, name).Scan(&m.SHA256, &m.Name, &entries, &at)
	if errors.Is(err, sql.ErrNoRows) {
		return ManifestRecord{}, false, nil
	}
	if err != nil {
		return ManifestRecord{}, false, fmt.Errorf("history: latest manifest %s: %w", name, err)
	}
	m.Entries = []byte(entries)
	if m.PublishedAt, err = parseTime(at); err != nil {
		return ManifestRecord{}, false, err
	}
	return m, true, nil
}

// SnapshotRecord is one compile attempt over a published manifest. A good attempt has ResultSHA
// and no Error; a failed one has an Error and no ResultSHA.
type SnapshotRecord struct {
	ManifestSHA string
	Compiler    string
	RulesHash   string
	ResultSHA   string
	Error       string
	BuiltAt     time.Time
	// Checks is the compiler's check results as JSON, kept beside the attempt (good or failed) so
	// Control Room shows them without compiling again; empty when the compile stopped before them.
	Checks string
}

// RecordSnapshot appends a compile attempt. Attempts are never changed, so a failed build leaves
// the last good one in place.
func (s *Store) RecordSnapshot(ctx context.Context, r SnapshotRecord) error {
	var result sql.NullString
	if r.ResultSHA != "" {
		result = sql.NullString{String: r.ResultSHA, Valid: true}
	}
	s.wmu.Lock()
	defer s.wmu.Unlock()
	tx, err := s.pools.BeginWrite(ctx)
	if err != nil {
		return fmt.Errorf("history: record snapshot of %s: %w", r.ManifestSHA, err)
	}
	defer func() { _ = tx.Rollback() }()
	res, err := tx.ExecContext(ctx, `
INSERT INTO compiled_snapshots (manifest_sha, compiler, rules_hash, result_sha, error, built_at)
VALUES (?, ?, ?, ?, ?, ?)`, r.ManifestSHA, r.Compiler, r.RulesHash, result, r.Error, formatTime(r.BuiltAt))
	if err != nil {
		return fmt.Errorf("history: record snapshot of %s: %w", r.ManifestSHA, err)
	}
	if r.Checks != "" {
		id, err := res.LastInsertId()
		if err != nil {
			return fmt.Errorf("history: record snapshot of %s: %w", r.ManifestSHA, err)
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO snapshot_checks (snapshot_id, checks) VALUES (?, ?)`, id, r.Checks); err != nil {
			return fmt.Errorf("history: record checks of %s: %w", r.ManifestSHA, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("history: record snapshot of %s: %w", r.ManifestSHA, err)
	}
	return nil
}

// Snapshots returns every compile attempt over manifests of that name, newest first.
func (s *Store) Snapshots(ctx context.Context, name string) ([]SnapshotRecord, error) {
	rows, err := s.pools.Read().QueryContext(ctx, `
SELECT c.manifest_sha, c.compiler, c.rules_hash, COALESCE(c.result_sha, ''), c.error, c.built_at,
       COALESCE(k.checks, '')
FROM compiled_snapshots c JOIN input_manifests m ON m.manifest_sha = c.manifest_sha
LEFT JOIN snapshot_checks k ON k.snapshot_id = c.snapshot_id
WHERE m.name = ? ORDER BY c.built_at DESC, c.snapshot_id DESC`, name)
	if err != nil {
		return nil, fmt.Errorf("history: snapshots of %s: %w", name, err)
	}
	defer func() { _ = rows.Close() }()
	var out []SnapshotRecord
	for rows.Next() {
		var r SnapshotRecord
		var at string
		if err := rows.Scan(&r.ManifestSHA, &r.Compiler, &r.RulesHash, &r.ResultSHA, &r.Error, &at, &r.Checks); err != nil {
			return nil, fmt.Errorf("history: scan snapshot of %s: %w", name, err)
		}
		if r.BuiltAt, err = parseTime(at); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("history: snapshots of %s: %w", name, err)
	}
	return out, nil
}

func (s *Store) ManifestBySHA(ctx context.Context, sha string) (ManifestRecord, error) {
	var m ManifestRecord
	var entries, at string
	if err := s.pools.Read().QueryRowContext(ctx, `SELECT manifest_sha,name,entries,published_at FROM input_manifests WHERE manifest_sha=?`, sha).Scan(&m.SHA256, &m.Name, &entries, &at); err != nil {
		return m, fmt.Errorf("history: manifest %s: %w", sha, err)
	}
	m.Entries = []byte(entries)
	var err error
	m.PublishedAt, err = parseTime(at)
	return m, err
}

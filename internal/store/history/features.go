package history

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"io"
	"strings"
	"time"
)

// latestSQL is each source's latest value per player, period and measure, as of the bound or
// literal time asOf. where adds further conditions on observations.
// SQLite takes bare columns from the MAX row; the primary key makes that row unique.
// This avoids sorting every observation through a second window before source selection.
func latestSQL(asOf, where string) string {
	return `
SELECT player_id, season, week, measure, source, MAX(as_of) AS as_of, value, text, 1 AS newest
FROM observations WHERE as_of <= ` + asOf + where + `
GROUP BY player_id, season, week, measure, source`
}

// featuresSQL keeps one row per player, period and measure: the highest-priority source's
// latest value. A source with no source_fields row for the measure is not read.
func featuresSQL(asOf, where string) string {
	return `
SELECT player_id, season, week, measure, source, as_of, value, text FROM (
	SELECT l.*, ROW_NUMBER() OVER (
		PARTITION BY l.player_id, l.season, l.week, l.measure ORDER BY f.priority) AS pick
	FROM (` + latestSQL(asOf, where) + `) l
	JOIN source_fields f ON f.source = l.source AND f.measure = l.measure
	WHERE l.newest = 1
) WHERE pick = 1`
}

// featureViewsDDL creates the current-knowledge views for reading history.db directly. The
// store's own reads use Features, which takes any as-of time.
func featureViewsDDL() string {
	now := "'9999-12-31'"
	return `
CREATE VIEW IF NOT EXISTS season_features AS ` + featuresSQL(now, " AND week = 0") + `;
CREATE VIEW IF NOT EXISTS week_features AS ` + featuresSQL(now, " AND week > 0") + `;`
}

// Feature is the value the engine reads for one player, period and measure.
type Feature struct {
	PlayerID string
	Season   int
	Week     int
	Measure  string
	Source   string
	AsOf     time.Time
	Value    float64
	Text     string // set instead of Value for a text measure
}

// FeatureQuery selects features known on or before AsOf, for one season and the listed
// measures.
type FeatureQuery struct {
	AsOf     time.Time
	Season   int
	Measures []string
}

// Features returns one row per player, period and measure, as known at q.AsOf.
func (s *Store) Features(ctx context.Context, q FeatureQuery) ([]Feature, error) {
	if len(q.Measures) == 0 {
		return nil, nil
	}
	where := " AND season = ? AND measure IN (?" + strings.Repeat(", ?", len(q.Measures)-1) + ")"
	args := []any{formatTime(q.AsOf), q.Season}
	for _, m := range q.Measures {
		args = append(args, m)
	}
	rows, err := s.pools.Read().QueryContext(ctx, featuresSQL("?", where)+` ORDER BY player_id, week, measure`, args...)
	if err != nil {
		return nil, fmt.Errorf("history: features: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var out []Feature
	for rows.Next() {
		var f Feature
		var asOf string
		var v sql.NullFloat64
		var t sql.NullString
		if err := rows.Scan(&f.PlayerID, &f.Season, &f.Week, &f.Measure, &f.Source, &asOf, &v, &t); err != nil {
			return nil, fmt.Errorf("history: scan feature: %w", err)
		}
		if f.AsOf, err = parseTime(asOf); err != nil {
			return nil, err
		}
		f.Value, f.Text = v.Float64, t.String
		out = append(out, f)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("history: features rows: %w", err)
	}
	return out, nil
}

// NextFeatureAt is the first future observation that could change an as-of book without a write.
func (s *Store) NextFeatureAt(ctx context.Context, after time.Time) (time.Time, error) {
	var at sql.NullString
	if err := s.pools.Read().QueryRowContext(ctx, `SELECT MIN(as_of) FROM observations WHERE as_of > ?`,
		formatTime(after)).Scan(&at); err != nil {
		return time.Time{}, fmt.Errorf("history: next feature time: %w", err)
	}
	if !at.Valid {
		return time.Time{}, nil
	}
	return parseTime(at.String)
}

// Disagreement is two sources holding different latest values for the same player, period and
// measure.
type Disagreement struct {
	PlayerID     string
	Season, Week int
	Measure      string
	SourceA      string
	ValueA       string
	SourceB      string
	ValueB       string
}

// Disagreements lists every pair of sources whose latest values differ, as known at asOf.
// Numbers differ when they are more than a billionth apart, relative to their size.
func (s *Store) Disagreements(ctx context.Context, asOf time.Time) ([]Disagreement, error) {
	rows, err := s.pools.Read().QueryContext(ctx, `
WITH latest AS (`+latestSQL("?", "")+`)
SELECT a.player_id, a.season, a.week, a.measure,
	a.source, COALESCE(a.text, CAST(a.value AS TEXT)), b.source, COALESCE(b.text, CAST(b.value AS TEXT))
FROM latest a JOIN latest b
	ON b.player_id = a.player_id AND b.season = a.season AND b.week = a.week
	AND b.measure = a.measure AND b.source > a.source
WHERE a.newest = 1 AND b.newest = 1
	AND (a.text IS NOT b.text OR ABS(a.value - b.value) > 1e-9 * MAX(1, ABS(a.value)))
ORDER BY a.measure, a.season, a.week, a.player_id, a.source, b.source`, formatTime(asOf))
	if err != nil {
		return nil, fmt.Errorf("history: disagreements: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var out []Disagreement
	for rows.Next() {
		var d Disagreement
		if err := rows.Scan(&d.PlayerID, &d.Season, &d.Week, &d.Measure, &d.SourceA, &d.ValueA, &d.SourceB, &d.ValueB); err != nil {
			return nil, fmt.Errorf("history: scan disagreement: %w", err)
		}
		out = append(out, d)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("history: disagreements rows: %w", err)
	}
	return out, nil
}

// valueEvidenceTables are the mutable tables a league valuation reads besides observations.
// Each is small enough to hash whole; observations and loads are append-only, so their newest
// row ids name their content.
func valueEvidenceTables() []string {
	return []string{"player_record", "player_external_id", "source_fields", "measures", "sources"}
}

// ValueEvidence fingerprints everything a valuation reads from this database. Unlike
// PRAGMA data_version it survives reopening, so a saved pass can be reused after a relaunch;
// any added observation or changed record, link or source priority changes it.
func (s *Store) ValueEvidence(ctx context.Context) (string, error) {
	h := sha256.New()
	var obs, loads int64
	if err := s.pools.Read().QueryRowContext(ctx,
		`SELECT (SELECT COALESCE(MAX(rowid), 0) FROM observations), (SELECT COALESCE(MAX(load_id), 0) FROM loads)`).
		Scan(&obs, &loads); err != nil {
		return "", fmt.Errorf("history: value evidence: %w", err)
	}
	_, _ = fmt.Fprintf(h, "observations %d loads %d\n", obs, loads)
	for _, table := range valueEvidenceTables() {
		if err := s.hashTable(ctx, h, table); err != nil {
			return "", err
		}
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func (s *Store) hashTable(ctx context.Context, h io.Writer, table string) error {
	rows, err := s.pools.Read().QueryContext(ctx, "SELECT * FROM "+table+" ORDER BY 1, 2")
	if err != nil {
		return fmt.Errorf("history: value evidence %s: %w", table, err)
	}
	defer func() { _ = rows.Close() }()
	cols, err := rows.Columns()
	if err != nil {
		return fmt.Errorf("history: value evidence %s: %w", table, err)
	}
	cells := make([]sql.RawBytes, len(cols))
	dest := make([]any, len(cols))
	for i := range cells {
		dest[i] = &cells[i]
	}
	_, _ = fmt.Fprintf(h, "table %s %d\n", table, len(cols))
	for rows.Next() {
		if err := rows.Scan(dest...); err != nil {
			return fmt.Errorf("history: value evidence %s: %w", table, err)
		}
		for _, c := range cells {
			_, _ = fmt.Fprintf(h, "%d:", len(c))
			_, _ = h.Write(c)
		}
		_, _ = h.Write([]byte{'\n'})
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("history: value evidence %s: %w", table, err)
	}
	return nil
}

package history

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// latestSQL is each source's latest value per player, period and measure, as of the bound or
// literal time asOf. where adds further conditions on observations.
func latestSQL(asOf, where string) string {
	return `
SELECT player_id, season, week, measure, source, as_of, value, text,
	ROW_NUMBER() OVER (PARTITION BY player_id, season, week, measure, source ORDER BY as_of DESC) AS newest
FROM observations WHERE as_of <= ` + asOf + where
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
CREATE VIEW season_features AS ` + featuresSQL(now, " AND week = 0") + `;
CREATE VIEW week_features AS ` + featuresSQL(now, " AND week > 0") + `;`
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

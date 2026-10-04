package history

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/measures"
)

// SourceState is a source's observed condition.
type SourceState string

const (
	SourceActive  SourceState = "active"
	SourceLost    SourceState = "lost"
	SourceRetired SourceState = "retired"
)

// SourceHealth is one source's state and the load history it was judged on.
type SourceHealth struct {
	Source      string
	State       SourceState
	LastSuccess time.Time // zero when no load has succeeded
	LastFailure time.Time // zero when no load has failed
	LastError   string
}

func (s *Store) source(id string) (measures.Source, bool) { return s.reg.Source(id) }

// SourceHealth judges every registered source at the store clock's now. A source is lost when
// its latest load failed and it has no successful load within its max age, or none at all. A
// source with no loads yet is active: nothing has failed.
func (s *Store) SourceHealth(ctx context.Context) ([]SourceHealth, error) {
	now := s.now()
	out := make([]SourceHealth, 0, len(s.reg.Sources))
	for _, src := range s.reg.Sources {
		h := SourceHealth{Source: src.ID, State: SourceActive}
		var ok, failed sql.NullString
		var lastErr string
		err := s.pools.Read().QueryRowContext(ctx, `
SELECT
	(SELECT MAX(loaded_at) FROM loads WHERE source = ?1 AND error = ''),
	(SELECT MAX(loaded_at) FROM loads WHERE source = ?1 AND error <> ''),
	COALESCE((SELECT error FROM loads WHERE source = ?1 AND error <> '' ORDER BY loaded_at DESC LIMIT 1), '')`,
			src.ID).Scan(&ok, &failed, &lastErr)
		if err != nil {
			return nil, fmt.Errorf("history: health of %s: %w", src.ID, err)
		}
		if ok.Valid {
			if h.LastSuccess, err = parseTime(ok.String); err != nil {
				return nil, err
			}
		}
		if failed.Valid {
			if h.LastFailure, err = parseTime(failed.String); err != nil {
				return nil, err
			}
			h.LastError = lastErr
		}
		switch {
		case src.Status == measures.SourceRetired:
			h.State = SourceRetired
		case failed.Valid && h.LastFailure.After(h.LastSuccess) && now.Sub(h.LastSuccess) > src.MaxAge:
			h.State = SourceLost
		}
		out = append(out, h)
	}
	return out, nil
}

// FlowingMeasures returns every measure fed by at least one active source.
func (s *Store) FlowingMeasures(ctx context.Context) (map[string]bool, error) {
	health, err := s.SourceHealth(ctx)
	if err != nil {
		return nil, err
	}
	active := map[string]bool{}
	for _, h := range health {
		active[h.Source] = h.State == SourceActive
	}
	flowing := map[string]bool{}
	for _, f := range s.reg.Fields {
		if active[f.Source] {
			flowing[f.Measure] = true
		}
	}
	return flowing, nil
}

// missingFrom returns the measures in need that are not flowing, sorted.
func missingFrom(need []string, flowing map[string]bool) []string {
	missing := []string{}
	for _, m := range need {
		if !flowing[m] {
			missing = append(missing, m)
		}
	}
	sort.Strings(missing)
	return missing
}

// LastLoadOf returns when a body fetched from url last loaded without error. ok is false when it
// never has.
func (s *Store) LastLoadOf(ctx context.Context, url string) (at time.Time, ok bool, err error) {
	var last sql.NullString
	if err := s.pools.Read().QueryRowContext(ctx, `
SELECT MAX(loaded_at) FROM loads WHERE error = '' AND sha256 <> ''
	AND sha256 IN (SELECT sha256 FROM fetch_log WHERE url = ? AND sha256 IS NOT NULL)`, url).Scan(&last); err != nil {
		return time.Time{}, false, fmt.Errorf("history: last load of %s: %w", url, err)
	}
	if !last.Valid {
		return time.Time{}, false, nil
	}
	at, err = parseTime(last.String)
	return at, err == nil, err
}

// PlayersWithData returns every player holding a value from source for any of the measures in
// season (0 for player-grain facts), and how many source ids still wait for a match.
func (s *Store) PlayersWithData(ctx context.Context, source string, season int, measures []string) (map[string]bool, int, error) {
	if len(measures) == 0 {
		return map[string]bool{}, 0, nil
	}
	in := "(?" + strings.Repeat(", ?", len(measures)-1) + ")"
	args := []any{source, season}
	for _, m := range measures {
		args = append(args, m)
	}
	rows, err := s.pools.Read().QueryContext(ctx, `
SELECT DISTINCT player_id FROM observations WHERE source = ? AND season = ? AND measure IN `+in, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("history: players with %s data: %w", source, err)
	}
	defer func() { _ = rows.Close() }()
	out := map[string]bool{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, 0, fmt.Errorf("history: scan player: %w", err)
		}
		out[id] = true
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("history: players with %s data: %w", source, err)
	}
	var waiting int
	if err := s.pools.Read().QueryRowContext(ctx, `
SELECT COUNT(DISTINCT id_type || ':' || id_value) FROM unresolved_observations
WHERE source = ? AND season = ? AND measure IN `+in, args...).Scan(&waiting); err != nil {
		return nil, 0, fmt.Errorf("history: waiting %s ids: %w", source, err)
	}
	return out, waiting, nil
}

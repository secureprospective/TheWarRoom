package history

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
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

func (s *Store) source(id string) (measures.Source, bool) {
	for _, src := range s.reg.Sources {
		if src.ID == id {
			return src, true
		}
	}
	return measures.Source{}, false
}

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

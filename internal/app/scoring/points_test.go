package scoring

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/playerscores"
	"github.com/secureprospective/TheWarRoom/internal/measures"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

// The last and current seasons are scored under today's rules; older ones only exist under
// their own.
func TestScoringYear(t *testing.T) {
	t.Parallel()
	for y, want := range map[int]int{2021: 2021, 2024: 2024, 2025: 2026, 2026: 2026} {
		if got := scoringYear(y, 2026); got != want {
			t.Errorf("scoringYear(%d) = %d, want %d", y, got, want)
		}
	}
}

type pointsHistory struct {
	queries                        []int
	ingested                       int
	failures                       int
	readErr, ingestErr, failureErr error
}

func (h *pointsHistory) Features(_ context.Context, q history.FeatureQuery) ([]history.Feature, error) {
	h.queries = append(h.queries, q.Season)
	if h.readErr != nil {
		return nil, h.readErr
	}
	return []history.Feature{{}}, nil
}
func (h *pointsHistory) Ingest(context.Context, measures.Batch) (history.LoadReport, error) {
	h.ingested++
	return history.LoadReport{}, h.ingestErr
}
func (h *pointsHistory) LoadFailed(context.Context, string, error) error {
	h.failures++
	return h.failureErr
}

func TestPointsRefreshSkipsHeldHistoryAndToleratesEmptyCurrent(t *testing.T) {
	t.Parallel()
	h := &pointsHistory{}
	var fetched []string
	src := PointsSource{History: h, Season: 2026, Fetch: func(_ context.Context, year string, season int) (measures.Batch, error) {
		fetched = append(fetched, fmt.Sprintf("%s/%d", year, season))
		if season == 2026 {
			return measures.Batch{}, playerscores.ErrEmptyScores
		}
		return measures.Batch{}, nil
	}}
	warning, err := src.Load(t.Context())
	if err != nil || warning != "" || fmt.Sprint(fetched) != "[2026/2025 2026/2026]" || h.ingested != 1 || h.failures != 0 || len(h.queries) != 2025-ingestion.LeagueFirstSeason {
		t.Fatalf("warning %q error %v fetched %v history %+v", warning, err, fetched, h)
	}
}

func TestPointsRefreshErrorsKeepTheirLabels(t *testing.T) {
	t.Parallel()
	down := errors.New("down")
	for _, tc := range []struct {
		name     string
		h        pointsHistory
		fetchErr error
		prefix   string
		fatal    bool
	}{
		{name: "read", h: pointsHistory{readErr: down}, prefix: "app: read", fatal: true},
		{name: "store", h: pointsHistory{ingestErr: down}, prefix: "app: store", fatal: true},
		{name: "fetch", fetchErr: down, prefix: "MFL fantasy points not refreshed"},
		{name: "health", h: pointsHistory{failureErr: down}, fetchErr: down, prefix: "down", fatal: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			src := PointsSource{History: &tc.h, Season: 2026, Fetch: func(context.Context, string, int) (measures.Batch, error) { return measures.Batch{}, tc.fetchErr }}
			warning, err := src.Load(t.Context())
			if tc.fatal {
				if !errors.Is(err, down) || !strings.HasPrefix(err.Error(), tc.prefix) {
					t.Fatalf("warning %q error %v", warning, err)
				}
			} else if err != nil || !strings.HasPrefix(warning, tc.prefix) || tc.h.failures != 2 {
				t.Fatalf("warning %q error %v history %+v", warning, err, tc.h)
			}
		})
	}
}

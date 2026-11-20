package history

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/leagueweek"
)

// ExtendFootballSchedule creates a new immutable football-evidence body with only additional NFL weeks.
// Existing weeks and all non-schedule evidence are preserved; callers pin the returned digest separately.
func (s *Store) ExtendFootballSchedule(ctx context.Context, digest string, additions []leagueweek.Week, asOf time.Time) (string, error) {
	if len(additions) == 0 {
		return digest, nil
	}
	if asOf.IsZero() {
		return "", fmt.Errorf("extend football schedule: observation time is required")
	}
	body, err := s.Body(ctx, digest)
	if err != nil {
		return "", fmt.Errorf("extend football schedule: read current evidence: %w", err)
	}
	raw, err := decodeFootball(body, s.storeParts(ctx))
	if err != nil {
		return "", fmt.Errorf("extend football schedule: decode current evidence: %w", err)
	}
	if err := validateFootballWeeks(additions); err != nil {
		return "", fmt.Errorf("extend football schedule: validate additions: %w", err)
	}
	present := make(map[int]struct{}, len(raw.Weeks)+len(additions))
	for _, week := range raw.Weeks {
		present[week.Number] = struct{}{}
	}
	for _, week := range additions {
		if _, exists := present[week.Number]; exists {
			return "", fmt.Errorf("extend football schedule: week %d already exists", week.Number)
		}
		present[week.Number] = struct{}{}
	}
	raw.Weeks = append(raw.Weeks, additions...)
	slices.SortFunc(raw.Weeks, func(left, right leagueweek.Week) int { return left.Number - right.Number })
	const scheduleSource = "; live MFL nflSchedule dates"
	if !strings.Contains(raw.Source, scheduleSource) {
		if len(raw.Source)+len(scheduleSource) > 200 {
			return "", fmt.Errorf("extend football schedule: provenance label exceeds 200 characters")
		}
		raw.Source += scheduleSource
	}
	raw.AsOf = asOf.UTC()
	updated, err := json.Marshal(raw)
	if err != nil {
		return "", fmt.Errorf("extend football schedule: encode evidence: %w", err)
	}
	next := footballDigest(updated)
	if err := s.ImportFootball(ctx, updated, next); err != nil {
		return "", fmt.Errorf("extend football schedule: import new evidence: %w", err)
	}
	return next, nil
}

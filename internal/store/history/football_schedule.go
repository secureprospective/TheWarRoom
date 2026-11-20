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
	return s.ReviseFootballSchedule(ctx, digest, nil, additions, asOf)
}

// ReviseFootballSchedule creates a new immutable football-evidence body in which replacements overwrite
// held weeks of the same number and additions fill weeks not yet held. A replacement must name a week
// already held; an addition must name one that is not. Nothing is written unless every check passes.
func (s *Store) ReviseFootballSchedule(ctx context.Context, digest string, replacements, additions []leagueweek.Week, asOf time.Time) (string, error) {
	if len(replacements) == 0 && len(additions) == 0 {
		return digest, nil
	}
	if asOf.IsZero() {
		return "", fmt.Errorf("revise football schedule: observation time is required")
	}
	body, err := s.Body(ctx, digest)
	if err != nil {
		return "", fmt.Errorf("revise football schedule: read current evidence: %w", err)
	}
	raw, err := decodeFootball(body, s.storeParts(ctx))
	if err != nil {
		return "", fmt.Errorf("revise football schedule: decode current evidence: %w", err)
	}
	if err := applyWeekRevisions(&raw, replacements, additions); err != nil {
		return "", fmt.Errorf("revise football schedule: %w", err)
	}
	slices.SortFunc(raw.Weeks, func(left, right leagueweek.Week) int { return left.Number - right.Number })
	const scheduleSource = "; live MFL nflSchedule dates"
	if !strings.Contains(raw.Source, scheduleSource) {
		if len(raw.Source)+len(scheduleSource) > 200 {
			return "", fmt.Errorf("revise football schedule: provenance label exceeds 200 characters")
		}
		raw.Source += scheduleSource
	}
	raw.AsOf = asOf.UTC()
	updated, err := json.Marshal(raw)
	if err != nil {
		return "", fmt.Errorf("revise football schedule: encode evidence: %w", err)
	}
	next := footballDigest(updated)
	if err := s.ImportFootball(ctx, updated, next); err != nil {
		return "", fmt.Errorf("revise football schedule: import new evidence: %w", err)
	}
	return next, nil
}

// applyWeekRevisions overwrites held weeks named by replacements and appends additions. Every week
// must validate, and a replacement must name a held week while an addition must name a new one.
func applyWeekRevisions(raw *RawFootballEvidence, replacements, additions []leagueweek.Week) error {
	if len(replacements) > 0 {
		if err := validateFootballWeeks(replacements); err != nil {
			return fmt.Errorf("validate replacements: %w", err)
		}
	}
	if len(additions) > 0 {
		if err := validateFootballWeeks(additions); err != nil {
			return fmt.Errorf("validate additions: %w", err)
		}
	}
	held := make(map[int]int, len(raw.Weeks))
	for i, week := range raw.Weeks {
		held[week.Number] = i
	}
	replaced := make(map[int]struct{}, len(replacements))
	for _, week := range replacements {
		index, ok := held[week.Number]
		if !ok {
			return fmt.Errorf("week %d is not held, so it cannot be replaced", week.Number)
		}
		if _, dup := replaced[week.Number]; dup {
			return fmt.Errorf("week %d is replaced twice", week.Number)
		}
		replaced[week.Number] = struct{}{}
		raw.Weeks[index] = week
	}
	for _, week := range additions {
		if _, exists := held[week.Number]; exists {
			return fmt.Errorf("week %d already exists", week.Number)
		}
		held[week.Number] = len(raw.Weeks)
		raw.Weeks = append(raw.Weeks, week)
	}
	return nil
}

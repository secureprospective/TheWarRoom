package history

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
)

const problemReportReadLimit = 100

// ErrProblemReportRateLimit means this principal has already submitted ten reports in the last hour.
var ErrProblemReportRateLimit = errors.New("problem report rate limit reached")

// AppendProblemReport stores one report in its own league-event stream. The writer transaction owns
// both the hourly count and append, so concurrent requests cannot pass the limit together.
func (s *Store) AppendProblemReport(ctx context.Context, report commissioner.ProblemReport) (commissioner.ProblemReport, error) {
	var idBytes [16]byte
	if _, err := rand.Read(idBytes[:]); err != nil {
		return commissioner.ProblemReport{}, fmt.Errorf("problem report: generate id: %w", err)
	}
	report.ID = "feedback." + hex.EncodeToString(idBytes[:])
	report.CreatedAt = s.now().UTC()
	if err := report.Validate(); err != nil {
		return commissioner.ProblemReport{}, fmt.Errorf("validate problem report: %w", err)
	}

	s.wmu.Lock()
	defer s.wmu.Unlock()
	tx, err := s.pools.BeginWrite(ctx)
	if err != nil {
		return commissioner.ProblemReport{}, fmt.Errorf("problem report: begin append: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	cutoff := formatTime(report.CreatedAt.Add(-time.Hour))
	var recent int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM league_event WHERE league=? AND stream=? AND actor=? AND kind=? AND recorded_at>=?`,
		report.League, commissioner.ProblemReportStream, report.Principal, commissioner.ProblemReportKind, cutoff).Scan(&recent); err != nil {
		return commissioner.ProblemReport{}, fmt.Errorf("problem report: count recent submissions: %w", err)
	}
	if recent >= 10 {
		return commissioner.ProblemReport{}, ErrProblemReportRateLimit
	}

	var revision int64
	if err := tx.QueryRowContext(ctx, `SELECT COALESCE(MAX(revision),0) FROM league_event WHERE league=? AND stream=?`,
		report.League, commissioner.ProblemReportStream).Scan(&revision); err != nil {
		return commissioner.ProblemReport{}, fmt.Errorf("problem report: read stream revision: %w", err)
	}
	details := map[string]string{
		"text": report.Text, "route": report.Route, "appRelease": strconv.Itoa(report.AppRelease),
		"viewportWidth": strconv.Itoa(report.ViewportWidth), "userAgent": report.UserAgent,
	}
	payload, err := json.Marshal(EventPayload{Version: eventPayloadVersion, Details: details})
	if err != nil {
		return commissioner.ProblemReport{}, fmt.Errorf("problem report: encode event: %w", err)
	}
	hash := sha256.Sum256(payload)
	event := LeagueEvent{ID: report.ID, League: report.League, Stream: commissioner.ProblemReportStream,
		EffectiveAt: report.CreatedAt, RecordedAt: report.CreatedAt, Actor: report.Principal,
		Scope: "league", Kind: commissioner.ProblemReportKind,
		Payload:   EventPayload{Version: eventPayloadVersion, Details: details},
		InputHash: hex.EncodeToString(hash[:]), CorrelationID: report.ID}
	if _, err := s.insertEvent(ctx, tx, event, revision); err != nil {
		return commissioner.ProblemReport{}, fmt.Errorf("problem report: append event: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return commissioner.ProblemReport{}, fmt.Errorf("problem report: commit: %w", err)
	}
	return report, nil
}

// ProblemReports reads only report events, newest first, for the given league.
func (s *Store) ProblemReports(ctx context.Context, league string) ([]commissioner.ProblemReport, error) {
	if err := (commissioner.RawProblemReportsScope{League: league}).Validate(); err != nil {
		return nil, fmt.Errorf("validate problem reports league: %w", err)
	}
	rows, err := s.pools.Read().QueryContext(ctx, problemReportsQuery, league,
		commissioner.ProblemReportStream, commissioner.ProblemReportKind, problemReportReadLimit)
	if err != nil {
		return nil, fmt.Errorf("problem reports: read: %w", err)
	}
	events, err := scanEvents(rows)
	if err != nil {
		return nil, fmt.Errorf("problem reports: decode events: %w", err)
	}
	reports := make([]commissioner.ProblemReport, 0, len(events))
	for _, event := range events {
		report, err := problemReportFromEvent(event)
		if err != nil {
			return nil, err
		}
		reports = append(reports, report)
	}
	return reports, nil
}

const problemReportsQuery = `SELECT id,league,stream,revision,effective_at,recorded_at,actor,scope,kind,payload,input_hash,correlation_id,COALESCE(supersedes,'') FROM league_event WHERE league=? AND stream=? AND kind=? ORDER BY revision DESC LIMIT ?`

func problemReportFromEvent(event LeagueEvent) (commissioner.ProblemReport, error) {
	details := event.Payload.Details
	if len(details) != 5 {
		return commissioner.ProblemReport{}, fmt.Errorf("problem reports: event %s has invalid detail fields", event.ID)
	}
	release, releaseErr := strconv.Atoi(details["appRelease"])
	width, widthErr := strconv.Atoi(details["viewportWidth"])
	if releaseErr != nil || widthErr != nil {
		return commissioner.ProblemReport{}, fmt.Errorf("problem reports: event %s has invalid client metadata", event.ID)
	}
	report := commissioner.ProblemReport{ID: event.ID, League: event.League, Principal: event.Actor,
		Text: details["text"], Route: details["route"], AppRelease: release, ViewportWidth: width,
		UserAgent: details["userAgent"], CreatedAt: event.RecordedAt}
	if err := report.Validate(); err != nil {
		return commissioner.ProblemReport{}, fmt.Errorf("problem reports: event %s: %w", event.ID, err)
	}
	return report, nil
}

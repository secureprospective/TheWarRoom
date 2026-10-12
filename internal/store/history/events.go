package history

import (
	"context"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"
)

const eventsDDL = `
CREATE TABLE IF NOT EXISTS league_event (
 id TEXT PRIMARY KEY CHECK(length(id)>0),
 league TEXT NOT NULL CHECK(length(league)>0),
 stream TEXT NOT NULL CHECK(length(stream)>0),
 revision INTEGER NOT NULL CHECK(revision>0),
 effective_at TEXT NOT NULL,
 recorded_at TEXT NOT NULL,
 actor TEXT NOT NULL CHECK(length(actor)>0),
 scope TEXT NOT NULL CHECK(length(scope)>0),
 kind TEXT NOT NULL CHECK(length(kind)>0),
 payload TEXT NOT NULL CHECK(json_valid(payload) AND json_extract(payload,'$.version')>0),
 input_hash TEXT NOT NULL CHECK(length(input_hash)=64),
 correlation_id TEXT NOT NULL CHECK(length(correlation_id)>0),
 supersedes TEXT REFERENCES league_event(id),
 UNIQUE(league,stream,revision)
);
CREATE TRIGGER IF NOT EXISTS league_event_revision BEFORE INSERT ON league_event
WHEN NEW.revision <> COALESCE((SELECT MAX(revision) FROM league_event WHERE league=NEW.league AND stream=NEW.stream),0)+1
BEGIN SELECT RAISE(ABORT,'league event revision conflict'); END;
CREATE TRIGGER IF NOT EXISTS league_event_supersedes BEFORE INSERT ON league_event
WHEN NEW.supersedes IS NOT NULL AND NOT EXISTS(SELECT 1 FROM league_event WHERE id=NEW.supersedes AND league=NEW.league AND stream=NEW.stream)
BEGIN SELECT RAISE(ABORT,'superseded event must be in the same stream'); END;
CREATE TABLE IF NOT EXISTS authority_policy (
 league TEXT NOT NULL CHECK(length(league)>0),
 domain TEXT NOT NULL CHECK(length(domain)>0),
 cutover_at TEXT,
 PRIMARY KEY(league,domain)
);`

// ErrRevisionConflict means the caller must reread the stream before appending.
var ErrRevisionConflict = errors.New("league event revision conflict")

// Authority answers which system is the record for a league domain at a moment.
const (
	AuthorityMFL   = "MFL"   // MFL is the record (every domain, until a cutover is set)
	AuthorityLocal = "local" // this app's league_event stream is the record, from the cutover on
)

// EventPayload is an event's versioned body. Money is whole cents, never a float.
type EventPayload struct {
	Version    int               `json:"version"`
	MoneyCents *int64            `json:"moneyCents,omitempty"`
	Details    map[string]string `json:"details,omitempty"`
}

// LeagueEvent is one row of the append-only league event log. Revision orders a stream; a
// correction is a new event that supersedes an earlier one in the same stream, never an edit.
type LeagueEvent struct {
	ID, League, Stream                   string
	Revision                             int64
	EffectiveAt, RecordedAt              time.Time
	Actor, Scope, Kind                   string
	Payload                              EventPayload
	InputHash, CorrelationID, Supersedes string
}

// Validate checks the identity, input hash, effective time and payload version an event needs.
func (e LeagueEvent) Validate() error {
	for _, text := range []string{e.ID, e.League, e.Stream, e.Actor, e.Scope, e.Kind, e.CorrelationID} {
		if strings.TrimSpace(text) == "" {
			return fmt.Errorf("league event: required identity is empty")
		}
	}
	hash, err := hex.DecodeString(e.InputHash)
	if err != nil || len(hash) != 32 || e.EffectiveAt.IsZero() || e.Payload.Version < 1 {
		return fmt.Errorf("league event: invalid hash, effective time or payload version")
	}
	return nil
}

// AppendEvent adds e as the stream's next revision when the stream holds exactly expected
// revisions, and returns it with its revision and recorded time. A stale expected revision
// returns ErrRevisionConflict: reread the stream before appending.
func (s *Store) AppendEvent(ctx context.Context, e LeagueEvent, expected int64) (LeagueEvent, error) {
	if err := e.Validate(); err != nil {
		return LeagueEvent{}, err
	}
	if expected < 0 || expected == math.MaxInt64 {
		return LeagueEvent{}, fmt.Errorf("league event: invalid expected revision")
	}
	s.wmu.Lock()
	defer s.wmu.Unlock()
	tx, err := s.pools.Write().BeginTx(ctx, nil)
	if err != nil {
		return LeagueEvent{}, fmt.Errorf("league event: begin append: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	var revision int64
	if err := tx.QueryRowContext(ctx, `SELECT COALESCE(MAX(revision),0) FROM league_event WHERE league=? AND stream=?`, e.League, e.Stream).Scan(&revision); err != nil {
		return LeagueEvent{}, fmt.Errorf("league event: read revision: %w", err)
	}
	if revision != expected {
		return LeagueEvent{}, fmt.Errorf("league event: expected %d, held %d: %w", expected, revision, ErrRevisionConflict)
	}
	e.Revision, e.RecordedAt = expected+1, s.now().UTC()
	payload, err := json.Marshal(e.Payload)
	if err != nil {
		return LeagueEvent{}, fmt.Errorf("league event: encode payload: %w", err)
	}
	var supersedes sql.NullString
	if e.Supersedes != "" {
		supersedes = sql.NullString{String: e.Supersedes, Valid: true}
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO league_event (id,league,stream,revision,effective_at,recorded_at,actor,scope,kind,payload,input_hash,correlation_id,supersedes) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)`, e.ID, e.League, e.Stream, e.Revision, formatTime(e.EffectiveAt), formatTime(e.RecordedAt), e.Actor, e.Scope, e.Kind, string(payload), e.InputHash, e.CorrelationID, supersedes); err != nil {
		return LeagueEvent{}, fmt.Errorf("league event: append: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return LeagueEvent{}, fmt.Errorf("league event: commit: %w", err)
	}
	return e, nil
}

// Events returns a stream's events after revision after, in revision order.
func (s *Store) Events(ctx context.Context, league, stream string, after int64) ([]LeagueEvent, error) {
	rows, err := s.pools.Read().QueryContext(ctx, `SELECT id,league,stream,revision,effective_at,recorded_at,actor,scope,kind,payload,input_hash,correlation_id,COALESCE(supersedes,'') FROM league_event WHERE league=? AND stream=? AND revision>? ORDER BY revision`, league, stream, after)
	if err != nil {
		return nil, fmt.Errorf("league event: read: %w", err)
	}
	defer func() { _ = rows.Close() }()
	out := []LeagueEvent{}
	for rows.Next() {
		var e LeagueEvent
		var effective, recorded, payload string
		if err := rows.Scan(&e.ID, &e.League, &e.Stream, &e.Revision, &effective, &recorded, &e.Actor, &e.Scope, &e.Kind, &payload, &e.InputHash, &e.CorrelationID, &e.Supersedes); err != nil {
			return nil, fmt.Errorf("league event: scan: %w", err)
		}
		if e.EffectiveAt, err = parseTime(effective); err != nil {
			return nil, err
		}
		if e.RecordedAt, err = parseTime(recorded); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(payload), &e.Payload); err != nil {
			return nil, fmt.Errorf("league event: payload: %w", err)
		}
		if err := e.Validate(); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("league event: rows: %w", err)
	}
	return out, nil
}

// AuthorityPolicy sets when a league domain moves from MFL to the local event log; a nil
// cutover keeps MFL the record.
type AuthorityPolicy struct {
	League, Domain string
	CutoverAt      *time.Time
}

// SetAuthorityPolicy saves or replaces a domain's cutover.
func (s *Store) SetAuthorityPolicy(ctx context.Context, p AuthorityPolicy) error {
	if strings.TrimSpace(p.League) == "" || strings.TrimSpace(p.Domain) == "" || (p.CutoverAt != nil && p.CutoverAt.IsZero()) {
		return fmt.Errorf("authority policy: invalid league, domain or cutover")
	}
	var cutoff sql.NullString
	if p.CutoverAt != nil {
		cutoff = sql.NullString{String: formatTime(*p.CutoverAt), Valid: true}
	}
	if _, err := s.pools.Write().ExecContext(ctx, `INSERT INTO authority_policy(league,domain,cutover_at) VALUES(?,?,?) ON CONFLICT(league,domain) DO UPDATE SET cutover_at=excluded.cutover_at`, p.League, p.Domain, cutoff); err != nil {
		return fmt.Errorf("authority policy: save: %w", err)
	}
	return nil
}

// Authority returns AuthorityMFL or AuthorityLocal for a league domain at time at.
func (s *Store) Authority(ctx context.Context, league, domain string, at time.Time) (string, error) {
	var cutover sql.NullString
	err := s.pools.Read().QueryRowContext(ctx, `SELECT cutover_at FROM authority_policy WHERE league=? AND domain=?`, league, domain).Scan(&cutover)
	if errors.Is(err, sql.ErrNoRows) {
		return AuthorityMFL, nil
	}
	if err != nil {
		return "", fmt.Errorf("authority policy: read: %w", err)
	}
	if !cutover.Valid {
		return AuthorityMFL, nil
	}
	cutoff, err := parseTime(cutover.String)
	if err != nil {
		return "", err
	}
	if at.Before(cutoff) {
		return AuthorityMFL, nil
	}
	return AuthorityLocal, nil
}

package history

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
)

func commissionerRecords(events []LeagueEvent) ([]commissioner.Record, error) {
	records := make([]commissioner.Record, 0, len(events))
	for _, e := range events {
		if e.Payload.Commissioner == nil {
			return nil, fmt.Errorf("league ledger has an unreadable entry")
		}
		r := *e.Payload.Commissioner
		if err := r.Validate(); err != nil {
			return nil, fmt.Errorf("read league ledger: %w", err)
		}
		if r.Receipt.Entry != e.ID || r.Receipt.Revision != e.Revision || r.Principal != e.Actor ||
			r.Command.League != e.League || r.Command.Visibility != e.Scope ||
			!r.Receipt.RecordedAt.Equal(e.RecordedAt) {
			return nil, fmt.Errorf("league ledger receipt does not match its entry")
		}
		records = append(records, r)
	}
	return records, nil
}

func transactionRecords(ctx context.Context, tx *sql.Tx, league string) ([]commissioner.Record, error) {
	rows, err := tx.QueryContext(ctx, eventsQuery, league, commissioner.Stream, 0)
	if err != nil {
		return nil, fmt.Errorf("read league ledger: %w", err)
	}
	events, err := scanEvents(rows)
	if err != nil {
		return nil, err
	}
	return commissionerRecords(events)
}

// ExecuteCommissioner performs authorization, retry lookup and append in one
// immediate SQLite transaction, including across independently opened stores.
func (s *Store) ExecuteCommissioner(ctx context.Context, raw commissioner.RawCommand,
	principal string, auth commissioner.Authorizer) (commissioner.Receipt, error) {
	if err := raw.Validate(); err != nil {
		return commissioner.Receipt{}, fmt.Errorf("validate league change: %w", err)
	}
	if strings.TrimSpace(principal) == "" {
		return commissioner.Receipt{}, fmt.Errorf("your local identity is unavailable")
	}
	canonical, err := raw.Canonical(principal)
	if err != nil {
		return commissioner.Receipt{}, fmt.Errorf("canonical league change: %w", err)
	}
	s.wmu.Lock()
	defer s.wmu.Unlock()
	tx, err := s.pools.Write().BeginTx(ctx, nil)
	if err != nil {
		return commissioner.Receipt{}, fmt.Errorf("begin league change: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	records, err := transactionRecords(ctx, tx, raw.League)
	if err != nil {
		return commissioner.Receipt{}, err
	}
	role, err := commandRole(raw, principal, auth, records)
	if err != nil {
		return commissioner.Receipt{}, err
	}
	for _, r := range records {
		if r.Command.Request != raw.Request {
			continue
		}
		if r.Canonical != canonical {
			return commissioner.Receipt{}, fmt.Errorf("this request was already used for a different change")
		}
		return r.Receipt, nil
	}
	if err := checkCommissionerAppend(raw, records); err != nil {
		return commissioner.Receipt{}, err
	}
	receipt, err := s.appendCommissioner(ctx, tx, raw, principal, role, canonical)
	if err != nil {
		return commissioner.Receipt{}, err
	}
	if err := tx.Commit(); err != nil {
		return commissioner.Receipt{}, fmt.Errorf("record league change: %w", err)
	}
	return receipt, nil
}

func commandRole(raw commissioner.RawCommand, principal string, auth commissioner.Authorizer,
	records []commissioner.Record) (string, error) {
	creator := commissioner.Replay(records).Creator
	if auth == nil {
		if creator == "" && raw.Command == commissioner.Create {
			creator = principal
		}
		auth = commissioner.CreatorAuthorizer{Creator: creator}
	}
	role, err := auth.Allow(raw.League, principal, raw.Command)
	if err != nil {
		return "", fmt.Errorf("league change refused: %w", err)
	}
	if role == "" {
		return "", fmt.Errorf("you do not have permission to make this league change")
	}
	return role, nil
}

func checkCommissionerAppend(raw commissioner.RawCommand, records []commissioner.Record) error {
	current := int64(len(records))
	if raw.Expected != current {
		return fmt.Errorf("league version is %d; reread before changing it (expected %d): %w",
			current, raw.Expected, ErrRevisionConflict)
	}
	if raw.Command == commissioner.Create {
		if current != 0 {
			return fmt.Errorf("this app league already exists")
		}
		return nil
	}
	if current == 0 {
		return fmt.Errorf("create this app league before changing it")
	}
	for _, r := range records {
		if r.Receipt.Entry != raw.Args.Entry {
			continue
		}
		if r.Command.Visibility == "private" && raw.Visibility != "private" {
			return fmt.Errorf("a private recorded change must be reversed privately")
		}
		return nil
	}
	return fmt.Errorf("that recorded change is not in this league")
}

func (s *Store) appendCommissioner(ctx context.Context, tx *sql.Tx, raw commissioner.RawCommand,
	principal, role, canonical string) (commissioner.Receipt, error) {
	hash := sha256.Sum256([]byte(canonical))
	identity := sha256.Sum256([]byte(raw.League + "\x00" + raw.Request))
	now := s.now().UTC().Truncate(time.Microsecond)
	receipt := commissioner.Receipt{Entry: "act." + hex.EncodeToString(identity[:]), League: raw.League,
		Revision: raw.Expected + 1, RecordedAt: now, Message: "Recorded in the league ledger."}
	definition, err := commissioner.DefinitionFor(raw.Command)
	if err != nil {
		return commissioner.Receipt{}, fmt.Errorf("define league change: %w", err)
	}
	raw.Gravity = definition.Gravity
	record := commissioner.Record{Command: raw, Principal: principal, Role: role,
		Gravity: definition.Gravity, Canonical: canonical, Receipt: receipt}
	effective := now
	if raw.Effective != "now" && raw.Effective != "next_rollover" {
		parsed, err := time.Parse(time.RFC3339Nano, raw.Effective)
		if err != nil {
			return commissioner.Receipt{}, fmt.Errorf("read effective time: %w", err)
		}
		effective = parsed
	}
	event := LeagueEvent{ID: receipt.Entry, League: raw.League, Stream: commissioner.Stream,
		EffectiveAt: effective, RecordedAt: now, Actor: principal, Scope: raw.Visibility, Kind: raw.Command,
		Payload: EventPayload{Version: 1, Commissioner: &record}, InputHash: hex.EncodeToString(hash[:]),
		CorrelationID: raw.Request, Supersedes: raw.Args.Entry}
	if _, err := s.insertEvent(ctx, tx, event, raw.Expected); err != nil {
		return commissioner.Receipt{}, err
	}
	return receipt, nil
}

func (s *Store) CommissionerLedger(ctx context.Context, league, principal string,
	auth commissioner.Authorizer) (commissioner.Ledger, error) {
	events, err := s.Events(ctx, league, commissioner.Stream, 0)
	if err != nil {
		return commissioner.Ledger{}, err
	}
	records, err := commissionerRecords(events)
	if err != nil {
		return commissioner.Ledger{}, err
	}
	if auth == nil {
		auth = commissioner.CreatorAuthorizer{Creator: commissioner.Replay(records).Creator}
	}
	return commissioner.Reading(league, principal, records, auth), nil
}

func (s *Store) NativeLeagues(ctx context.Context) ([]commissioner.NativeLeague, error) {
	rows, err := s.pools.Read().QueryContext(ctx, `SELECT league,json_extract(payload,'$.commissioner.command.args.name') FROM league_event WHERE stream=? AND kind=? ORDER BY league`, commissioner.Stream, commissioner.Create)
	if err != nil {
		return nil, fmt.Errorf("read app leagues: %w", err)
	}
	defer func() { _ = rows.Close() }()
	out := []commissioner.NativeLeague{}
	for rows.Next() {
		var league commissioner.NativeLeague
		if err := rows.Scan(&league.ID, &league.Name); err != nil {
			return nil, fmt.Errorf("read app league: %w", err)
		}
		out = append(out, league)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read app leagues: %w", err)
	}
	return out, nil
}

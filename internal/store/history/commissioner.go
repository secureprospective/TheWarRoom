package history

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
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

	return s.ExecuteCommissionerWithFacts(ctx, raw, principal, auth, nil, time.UTC)
}

// ExecuteCommissionerWithFacts keeps held football facts inside the same atomic native append path.
func (s *Store) ExecuteCommissionerWithFacts(ctx context.Context, raw commissioner.RawCommand,
	principal string, auth commissioner.Authorizer, facts nativecheck.Facts, zone *time.Location) (commissioner.Receipt, error) {
	raw, canonical, err := preparedCommand(raw, principal)
	if err != nil {
		return commissioner.Receipt{}, err
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
	if prior, retried, err := retriedReceipt(raw, canonical, records); err != nil || retried {
		return prior, err
	}
	if err := checkCommissionerAppend(raw, records); err != nil {
		return commissioner.Receipt{}, fmt.Errorf("check league version: %w", commissioner.Refused(err))
	}
	if err := commissioner.PrepareWaiverAppend(&raw, records, s.now(), facts, zone); err != nil {
		return commissioner.Receipt{}, fmt.Errorf("check waiver action: %w", commissioner.Refused(err))
	}
	if err := s.checkAppend(raw, principal, records); err != nil {
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

// retriedReceipt returns the stored receipt when this request was already recorded, and refuses a
// request key reused for a different change.
func retriedReceipt(raw commissioner.RawCommand, canonical string,
	records []commissioner.Record) (commissioner.Receipt, bool, error) {
	for _, r := range records {
		if r.Command.Request != raw.Request {
			continue
		}
		if r.Canonical != canonical {
			return commissioner.Receipt{}, false, fmt.Errorf("league change refused: %w",
				commissioner.Refused(errors.New("this request was already used for a different change")))
		}
		return r.Receipt, true, nil
	}
	return commissioner.Receipt{}, false, nil
}

// checkAppend runs every check a league change must pass before it is recorded, in order.
func (s *Store) checkAppend(raw commissioner.RawCommand, principal string, records []commissioner.Record) error {
	if err := checkCommissionerAppend(raw, records); err != nil {
		return fmt.Errorf("check league version: %w", commissioner.Refused(err))
	}
	checks := []struct {
		what  string
		check func() error
	}{
		{"native correction", func() error { return commissioner.CheckOverrideAppend(raw, records, s.now()) }},
		{"starting snapshot", func() error { return commissioner.CheckNativeAppend(raw, records) }},
		{"league change", func() error { return commissioner.CheckGovernanceAppend(raw, principal, records, s.now()) }},
		{"league calendar", func() error { return commissioner.CheckCalendarAppend(raw, records, s.now()) }},
		{"league schedule", func() error { return commissioner.CheckScheduleAppend(raw, records) }},
		{"league rules", func() error { return commissioner.CheckRulesAppend(raw, records, s.now()) }},
	}
	for _, c := range checks {
		if err := c.check(); err != nil {
			return fmt.Errorf("check %s: %w", c.what, err)
		}
	}
	return nil
}

// preparedCommand fills what Go owns on the command, validates it and fixes its retry identity.
func preparedCommand(raw commissioner.RawCommand, principal string) (commissioner.RawCommand, string, error) {
	if raw.Args.WaiverOutcome != nil {
		return raw, "", fmt.Errorf("waiver decisions are made by the app")
	}
	raw, err := raw.Prepared()
	if err != nil {
		return raw, "", fmt.Errorf("validate league change: %w", commissioner.Refused(err))
	}
	if err := raw.Validate(); err != nil {
		return raw, "", fmt.Errorf("validate league change: %w", commissioner.Refused(err))
	}
	if strings.TrimSpace(principal) == "" {
		return raw, "", fmt.Errorf("your local identity is unavailable")
	}
	canonical, err := raw.Canonical(principal)
	if err != nil {
		return raw, "", fmt.Errorf("canonical league change: %w", err)
	}
	return raw, canonical, nil
}

func commandRole(raw commissioner.RawCommand, principal string, auth commissioner.Authorizer,
	records []commissioner.Record) (string, error) {
	if auth == nil || commissioner.OwnerWaiverAction(raw.Command) != "" {
		role, err := commissioner.AuthorizeRecords(raw, principal, records)
		if err != nil {
			return "", fmt.Errorf("league change refused: %w", commissioner.Refused(err))
		}
		return role, nil
	}
	role, err := auth.Allow(raw.League, principal, raw.Command)
	if err != nil {
		return "", fmt.Errorf("league change refused: %w", commissioner.Refused(err))
	}
	if role == "" {
		return "", fmt.Errorf("league change refused: %w",
			commissioner.Refused(errors.New("you do not have permission to make this league change")))
	}
	return role, nil
}

// staleVersion refuses a change made against an older league version. It is ErrRevisionConflict, and its
// own message is the reason the owner reads.
type staleVersion struct{ current, expected int64 }

func (e staleVersion) Error() string {
	return fmt.Sprintf("league version is %d; reread before changing it (expected %d)", e.current, e.expected)
}
func (e staleVersion) Is(target error) bool { return target == ErrRevisionConflict }

func checkCommissionerAppend(raw commissioner.RawCommand, records []commissioner.Record) error {
	current := int64(len(records))
	if raw.Expected != current {
		return staleVersion{current: current, expected: raw.Expected}
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
	if raw.Command != commissioner.Revert {
		return nil
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
	definition, err := raw.EffectiveDefinition()
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
		state, err := commissioner.GovernanceState(records)
		if err != nil {
			return commissioner.Ledger{}, fmt.Errorf("read league permissions: %w", err)
		}
		auth = commissioner.GovernanceAuthorizer{State: state}
	}
	reading, err := commissioner.Reading(league, principal, records, auth)
	if err != nil {
		return commissioner.Ledger{}, fmt.Errorf("read league ledger: %w", err)
	}
	return reading, nil
}

func (s *Store) CommissionerPeople(ctx context.Context, league, principal string) (commissioner.People, error) {
	events, err := s.Events(ctx, league, commissioner.Stream, 0)
	if err != nil {
		return commissioner.People{}, err
	}
	records, err := commissionerRecords(events)
	if err != nil {
		return commissioner.People{}, err
	}
	people, err := commissioner.PeopleReading(league, principal, records)
	if err != nil {
		return commissioner.People{}, fmt.Errorf("read league people: %w", err)
	}
	return people, nil
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

func (s *Store) CommissionerRules(ctx context.Context, league string) (commissioner.Rules, error) {
	events, err := s.Events(ctx, league, commissioner.Stream, 0)
	if err != nil {
		return commissioner.Rules{}, err
	}
	records, err := commissionerRecords(events)
	if err != nil {
		return commissioner.Rules{}, err
	}
	reading, err := commissioner.RulesReading(league, records, s.now())
	if err != nil {
		return commissioner.Rules{}, fmt.Errorf("read league rules: %w", err)
	}
	return reading, nil
}

package history

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"slices"
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

// Decoded records are shared internally, read-only; each caller owns its outer slice.
type recordRevision struct {
	revision int64
	records  []commissioner.Record
}

func (s *Store) transactionRecords(ctx context.Context, tx *sql.Tx, league string) ([]commissioner.Record, error) {
	var revision int64
	if err := tx.QueryRowContext(ctx, `SELECT COALESCE(MAX(revision),0) FROM league_event WHERE league=? AND stream=?`, league, commissioner.Stream).Scan(&revision); err != nil {
		return nil, fmt.Errorf("read league revision: %w", err)
	}
	s.recordsMu.Lock()
	defer s.recordsMu.Unlock()
	hit := s.records[league]
	if hit.revision > revision {
		hit = recordRevision{}
	}
	if hit.revision == revision {
		return slices.Clone(hit.records), nil
	}
	rows, err := tx.QueryContext(ctx, eventsQuery, league, commissioner.Stream, hit.revision)
	if err != nil {
		return nil, fmt.Errorf("read league ledger: %w", err)
	}
	events, err := scanEvents(rows)
	if err != nil {
		return nil, err
	}
	tail, err := commissionerRecords(events)
	if err != nil {
		return nil, err
	}
	records := append(slices.Clone(hit.records), tail...)
	if s.records == nil {
		s.records = map[string]recordRevision{}
	}
	s.records[league] = recordRevision{revision: revision, records: records}
	return slices.Clone(records), nil
}

// ExecuteCommissioner performs authorization, retry lookup and append in one
// immediate SQLite transaction, including across independently opened stores.
func (s *Store) ExecuteCommissioner(ctx context.Context, raw commissioner.RawCommand,
	principal string, auth commissioner.Authorizer) (commissioner.Receipt, error) {
	return s.ExecuteCommissionerWithFacts(ctx, raw, principal, auth, nil, time.UTC)
}

// ExecuteCommissionerWithFacts keeps held football facts inside the same atomic native append path: the
// app's waiver decision is made under the ledger transaction, after every shared check.
func (s *Store) ExecuteCommissionerWithFacts(ctx context.Context, raw commissioner.RawCommand,
	principal string, auth commissioner.Authorizer, facts nativecheck.Facts, zone *time.Location) (commissioner.Receipt, error) {
	raw, canonical, err := preparedCommand(raw, principal, true)
	if err != nil {
		return commissioner.Receipt{}, err
	}
	gathered, err := s.appendFactsFor(ctx, raw)
	if err != nil {
		return commissioner.Receipt{}, err
	}
	gathered.players = facts
	s.wmu.Lock()
	defer s.wmu.Unlock()
	tx, err := s.pools.Write().BeginTx(ctx, nil)
	if err != nil {
		return commissioner.Receipt{}, fmt.Errorf("begin league change: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	records, err := s.transactionRecords(ctx, tx, raw.League)
	if err != nil {
		return commissioner.Receipt{}, err
	}
	role, retry, err := s.checkCommissionerCommand(raw, principal, auth, records, canonical, gathered)
	if err != nil {
		return commissioner.Receipt{}, err
	}
	if retry != nil {
		return *retry, nil
	}
	now := s.now().UTC().Truncate(time.Microsecond)
	before, plan, err := s.prepareCommissionerTrial(ctx, &raw, principal, records, now, facts, zone, gathered)
	if err != nil {
		return commissioner.Receipt{}, err
	}

	if raw.Command == commissioner.SeasonRollover && raw.Args.Rollover.DryRun {
		return commissioner.Receipt{League: raw.League, Revision: raw.Expected, RecordedAt: before.At,
			Message: "Preview only; nothing was recorded.", Plan: plan}, nil
	}
	if before != nil {
		now = before.At
	}
	receipt, err := s.appendCommissioner(ctx, tx, appendRequest{raw: raw, principal: principal, role: role,
		canonical: canonical, at: now, before: before, plan: plan, records: records})
	if err != nil {
		return commissioner.Receipt{}, err
	}
	if err := tx.Commit(); err != nil {
		return commissioner.Receipt{}, fmt.Errorf("record league change: %w", err)
	}
	return receipt, nil
}

// checkCommissionerCommand is every check a league change must pass, shared by execute and the dry-run
// preview so the two can never disagree. A retried request returns its stored receipt.
func (s *Store) checkCommissionerCommand(raw commissioner.RawCommand, principal string,
	auth commissioner.Authorizer, records []commissioner.Record, canonical string,
	facts appendFacts) (string, *commissioner.Receipt, error) {
	role, err := commandRole(raw, principal, auth, records)
	if err != nil {
		return "", nil, err
	}
	prior, retried, err := retriedReceipt(raw, canonical, records)
	if err != nil {
		return "", nil, err
	}
	if retried {
		return role, &prior, nil
	}
	if err := s.checkAppend(raw, principal, records, facts); err != nil {
		return role, nil, err
	}
	return role, nil, nil
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
func (s *Store) checkAppend(raw commissioner.RawCommand, principal string, records []commissioner.Record,
	facts appendFacts) error {
	if err := checkCommissionerAppend(raw, records); err != nil {
		return fmt.Errorf("check league version: %w", commissioner.Refused(err))
	}
	checks := []struct {
		what  string
		check func() error
	}{
		{"asset correction", func() error { return commissioner.CheckCorrectionAppend(raw, records, s.now()) }},
		{"native correction", func() error { return commissioner.CheckOverrideAppend(raw, records, s.now()) }},
		{"starting snapshot", func() error { return commissioner.CheckNativeAppend(raw, records) }},
		{"league change", func() error { return commissioner.CheckGovernanceAppend(raw, principal, records, s.now()) }},
		{"league calendar", func() error { return commissioner.CheckCalendarAppend(raw, records, s.now()) }},
		{"rookie draft", func() error { return commissioner.CheckDraftAppend(raw, records, s.now(), facts.draft) }},
		{"league schedule", func() error { return commissioner.CheckScheduleAppend(raw, records) }},
		{"season rollover", func() error { return commissioner.CheckRolloverAppend(raw, records) }},
		{"league rules", func() error { return commissioner.CheckRulesAppend(raw, records, s.now()) }},
		{"lineup", func() error { return commissioner.CheckLineupAppend(raw, records, s.now(), facts.lineup) }},
		{"playoffs", func() error { return commissioner.CheckPlayoffAppend(raw, records, s.now(), facts.lineup) }},
		{"statistic", func() error { return commissioner.CheckStatAppend(raw, records, facts.stat, s.now()) }},
	}
	for _, c := range checks {
		if err := c.check(); err != nil {
			return fmt.Errorf("check %s: %w", c.what, err)
		}
	}
	return nil
}

// preparedCommand fills what Go owns on the command, validates it and fixes its retry identity. A dry run
// passes requireReason false and reports the missing reason itself.
func preparedCommand(raw commissioner.RawCommand, principal string,
	requireReason bool) (commissioner.RawCommand, string, error) {
	if raw.Args.WaiverOutcome != nil || raw.Args.BidOutcome != nil || raw.Args.TradeOutcome != nil {
		return raw, "", fmt.Errorf("waiver and bid decisions are made by the app")
	}
	raw, err := raw.Prepared()
	if err != nil {
		return raw, "", fmt.Errorf("validate league change: %w", commissioner.Refused(err))
	}
	validate := raw.ValidatePreview
	if requireReason {
		validate = raw.Validate
	}
	if err := validate(); err != nil {
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
	if commissioner.TradeCommand(raw.Command) || auth == nil || commissioner.OwnerWaiverAction(raw.Command) != "" || commissioner.OwnerBidAction(raw.Command) != "" {
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

// appendRequest is one league change ready to be recorded, with the instant it is recorded at, the ledger
// it is appended to and, for a season rollover, its before-state and plan.
type appendRequest struct {
	raw                        commissioner.RawCommand
	principal, role, canonical string
	at                         time.Time
	before                     *commissioner.RolloverBefore
	plan                       *commissioner.RolloverPlan
	records                    []commissioner.Record
}

func (s *Store) appendCommissioner(ctx context.Context, tx *sql.Tx, req appendRequest) (commissioner.Receipt, error) {
	raw, principal, now := req.raw, req.principal, req.at
	hash := sha256.Sum256([]byte(req.canonical))
	identity := sha256.Sum256([]byte(raw.League + "\x00" + raw.Request))
	receipt := commissioner.Receipt{Entry: "act." + hex.EncodeToString(identity[:]), League: raw.League,
		Revision: raw.Expected + 1, RecordedAt: now, Plan: req.plan, Message: "Recorded in the league ledger."}
	definition, err := raw.EffectiveDefinition()
	if err != nil {
		return commissioner.Receipt{}, fmt.Errorf("define league change: %w", err)
	}
	raw.Gravity = definition.Gravity
	record := commissioner.Record{Command: raw, Principal: principal, Role: req.role,
		Gravity: definition.Gravity, Canonical: req.canonical, Receipt: receipt, Before: req.before}
	if raw.Command == commissioner.ScheduleGenerate {
		record.GeneratedSchedule, err = commissioner.FreezeSchedule(raw, req.records)
		if err != nil {
			return commissioner.Receipt{}, fmt.Errorf("freeze league schedule: %w", err)
		}
	}
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
	return s.CommissionerLedgerFiltered(ctx, league, principal, auth, commissioner.RawLedgerFilter{})
}

func (s *Store) CommissionerLedgerFiltered(ctx context.Context, league, principal string,
	auth commissioner.Authorizer, filter commissioner.RawLedgerFilter) (commissioner.Ledger, error) {
	if err := filter.Validate(); err != nil {
		return commissioner.Ledger{}, fmt.Errorf("validate ledger filters: %w", err)
	}
	records, err := s.nativeRecords(ctx, league)
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
	reading, err := commissioner.FilteredReading(league, principal, records, auth, filter)
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

type nativeLeagueRevision struct {
	revision int64
	names    []commissioner.NativeLeague
	loaded   bool
}

func (s *Store) NativeLeagues(ctx context.Context) ([]commissioner.NativeLeague, error) {
	tx, err := s.pools.Read().BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("read app leagues: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	var revision int64
	if err := tx.QueryRowContext(ctx, `SELECT COALESCE(MAX(rowid),0) FROM league_event`).Scan(&revision); err != nil {
		return nil, fmt.Errorf("read app league revision: %w", err)
	}
	s.leaguesMu.Lock()
	defer s.leaguesMu.Unlock()
	if s.leagues.loaded && s.leagues.revision == revision {
		return slices.Clone(s.leagues.names), nil
	}
	rows, err := tx.QueryContext(ctx, `SELECT league,json_extract(payload,'$.commissioner.command.args.name') FROM league_event WHERE stream=? AND kind=? ORDER BY league`, commissioner.Stream, commissioner.Create)
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
	s.leagues = nativeLeagueRevision{revision: revision, names: out, loaded: true}
	return slices.Clone(out), nil
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

// CommissionerJobs reads the league's jobs as of the store's clock.
func (s *Store) CommissionerJobs(ctx context.Context, league string) (commissioner.Jobs, error) {
	events, err := s.Events(ctx, league, commissioner.Stream, 0)
	if err != nil {
		return commissioner.Jobs{}, err
	}
	records, err := commissionerRecords(events)
	if err != nil {
		return commissioner.Jobs{}, err
	}
	reading, err := commissioner.JobsReading(league, records, s.now())
	if err != nil {
		return commissioner.Jobs{}, fmt.Errorf("read league jobs: %w", err)
	}
	return reading, nil
}

// CommissionerNotes reads the posted notices and league notes from the ledger.
func (s *Store) CommissionerNotes(ctx context.Context, league string) (commissioner.LeagueNotes, error) {
	records, err := s.leagueRecords(ctx, league)
	if err != nil {
		return commissioner.LeagueNotes{}, err
	}
	reading, err := commissioner.NotesReading(league, records)
	if err != nil {
		return commissioner.LeagueNotes{}, fmt.Errorf("read league notices: %w", err)
	}
	return reading, nil
}

// CommissionerDigests writes the digest sentences for the changes this principal may read about.
func (s *Store) CommissionerDigests(ctx context.Context, league, principal string,
	auth commissioner.Authorizer) (commissioner.Digests, error) {
	records, err := s.leagueRecords(ctx, league)
	if err != nil {
		return commissioner.Digests{}, err
	}
	if auth == nil {
		state, err := commissioner.GovernanceState(records)
		if err != nil {
			return commissioner.Digests{}, fmt.Errorf("read league permissions: %w", err)
		}
		auth = commissioner.GovernanceAuthorizer{State: state}
	}
	reading, err := commissioner.DigestsReading(league, principal, records, auth)
	if err != nil {
		return commissioner.Digests{}, fmt.Errorf("read league digests: %w", err)
	}
	return reading, nil
}

func (s *Store) leagueRecords(ctx context.Context, league string) ([]commissioner.Record, error) {
	events, err := s.Events(ctx, league, commissioner.Stream, 0)
	if err != nil {
		return nil, err
	}
	return commissionerRecords(events)
}

func prepareMarketAppend(raw *commissioner.RawCommand, principal string, records []commissioner.Record, now time.Time,
	facts nativecheck.Facts, zone *time.Location, env commissioner.BidEnvironment) error {
	if err := commissioner.PrepareWaiverAppend(raw, records, now, facts, zone); err != nil {
		return fmt.Errorf("check waiver action: %w", commissioner.Refused(err))
	}
	// The bid clock's scan is repeated here, under the append lock, against the current ledger.
	if principal == bidClockPrincipal {
		if err := commissioner.CheckDueBid(*raw, records, now, facts); err != nil {
			return fmt.Errorf("check automatic bid award: %w", commissioner.Refused(err))
		}
	}
	if err := commissioner.PrepareBidAppend(raw, records, now, facts, zone, env); err != nil {
		return fmt.Errorf("check bid action: %w", commissioner.Refused(err))
	}
	if err := commissioner.PrepareTradeAppend(raw, principal, records, now, facts, zone); err != nil {
		return fmt.Errorf("check trade action: %w", commissioner.Refused(err))
	}
	return nil
}

// prepareCommissionerTrial prepares a private command copy with exactly the append decisions, without writing.
func (s *Store) prepareCommissionerTrial(ctx context.Context, raw *commissioner.RawCommand, principal string,
	records []commissioner.Record, now time.Time, facts nativecheck.Facts, zone *time.Location,
	gathered appendFacts) (*commissioner.RolloverBefore, *commissioner.RolloverPlan, error) {
	if err := prepareMarketAppend(raw, principal, records, now, facts, zone, gathered.bid); err != nil {
		return nil, nil, err
	}
	if err := commissioner.PrepareDraftAppend(raw, records, now, facts, gathered.draft); err != nil {
		return nil, nil, fmt.Errorf("check draft change: %w", commissioner.Refused(err))
	}
	return s.prepareRollover(ctx, *raw, records, now)
}

package history

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

// ExecuteCommissionerBatch records several league changes as one: each is checked against the ledger
// as the ones before it leave it, and either every change is recorded or none is. The first change is
// checked against the version its caller read; each later one is checked at the version the batch has
// reached, under the same lock, so nothing else can land between them. An MFL import (creation and
// starting snapshot) and an invitation send use it, so a failure at any step leaves no half league.
func (s *Store) ExecuteCommissionerBatch(ctx context.Context, raws []commissioner.RawCommand,
	principal string, auth commissioner.Authorizer) ([]commissioner.Receipt, error) {
	return s.ExecuteCommissionerBatchWithFacts(ctx, raws, principal, auth, nil, time.UTC)
}

// ExecuteCommissionerBatchWithFacts uses the ordinary preparation for every step.
// Football facts are supplied before the writer is held; eligibility is judged
// against each step's current ledger, never accepted as a caller's outcome.
func (s *Store) ExecuteCommissionerBatchWithFacts(ctx context.Context, raws []commissioner.RawCommand,
	principal string, auth commissioner.Authorizer, facts nativecheck.Facts, zone *time.Location) ([]commissioner.Receipt, error) {
	if len(raws) == 0 {
		return nil, fmt.Errorf("choose at least one league change")
	}
	s.wmu.Lock()
	defer s.wmu.Unlock()
	tx, err := s.pools.BeginWrite(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin league changes: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	league := raws[0].League
	records, err := s.transactionRecords(ctx, tx, league)
	if err != nil {
		return nil, err
	}
	receipts := make([]commissioner.Receipt, 0, len(raws))
	for i, raw := range raws {
		if raw.League != league {
			return nil, fmt.Errorf("record changes to one league at a time")
		}
		if !batchable(raw.Command, raws[0].Command == commissioner.Create) {
			return nil, fmt.Errorf("%s is recorded on its own, not in a batch", raw.Command)
		}
		now := s.now().UTC()
		if i > 0 {
			raw.Expected = batchExpectedRevision(ctx, raw, principal, records, now)
		}
		receipt, err := s.batchStep(ctx, tx, raw, principal, auth, records, now, facts, zone)
		if err != nil {
			return nil, fmt.Errorf("league change %d of %d: %w", i+1, len(raws), err)
		}
		if s.batchFault != nil {
			if err := s.batchFault(i); err != nil {
				return nil, err
			}
		}
		receipts = append(receipts, receipt)
		if records, err = s.extendBatchRecords(ctx, tx, league, records); err != nil {
			return nil, err
		}
	}
	// Intermediate states cannot commit; capture recovery once for the whole batch.
	if err := saveRosterRestorePoint(ctx, tx, league, records); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("record league changes: %w", err)
	}
	s.signalCommissionerCommit()
	for i := range receipts {
		receipts[i] = viewReceipt(ctx, receipts[i], records)
	}
	return receipts, nil
}

func (s *Store) extendBatchRecords(ctx context.Context, tx *sql.Tx, league string, records []commissioner.Record) ([]commissioner.Record, error) {
	tail, err := s.eventsAfterInTransaction(ctx, tx, league, int64(len(records)))
	if err != nil {
		return nil, err
	}
	return append(records, tail...), nil
}

// batchStep checks and appends one change of a batch inside the batch's transaction. A retried request
// returns its stored receipt.
func (s *Store) batchStep(ctx context.Context, tx *sql.Tx, raw commissioner.RawCommand, principal string,
	auth commissioner.Authorizer, records []commissioner.Record, now time.Time, facts nativecheck.Facts, zone *time.Location) (commissioner.Receipt, error) {
	raw, canonical, err := preparedCommand(raw, principal, true)
	if err != nil {
		return commissioner.Receipt{}, err
	}
	if err := commissioner.AdmitCredential(ctx, commissioner.Access{Principal: principal, League: raw.League,
		Operation: raw.Command, Mode: "execute", Raw: raw, Records: records}); err != nil {
		return commissioner.Receipt{}, fmt.Errorf("admit league change: %w", err)
	}
	role, retry, err := s.checkCommissionerCommand(ctx, raw, principal, auth, records, appendFacts{players: facts}, now)
	if err != nil {
		return commissioner.Receipt{}, err
	}
	if retry != nil {
		return *retry, nil
	}
	before, plan, err := s.prepareCommissionerTrial(ctx, &raw, principal, records, now, facts, zone, appendFacts{players: facts})
	if err != nil {
		return commissioner.Receipt{}, err
	}
	return s.appendCommissioner(ctx, tx, appendRequest{raw: raw, principal: principal, role: role,
		canonical: canonical, at: now, before: before, plan: plan, records: records, deferRosterRestore: true})
}

// batchable excludes operations whose other evidence has no batch preparation port.
func batchable(command string, creating bool) bool {
	switch command {
	case commissioner.RulesSet:
		return creating
	case commissioner.RosterReserve, commissioner.CorrectRoster, commissioner.CorrectPick,
		commissioner.Create, commissioner.NativeSeed, commissioner.ScheduleGenerate, commissioner.SchedulePublish, "commish.seat.invite":
		return true
	}
	return false
}

func batchExpectedRevision(ctx context.Context, raw commissioner.RawCommand, principal string, records []commissioner.Record, now time.Time) int64 {
	for _, r := range records {
		if r.Command.Request == raw.Request && commissioner.SamePrincipal(records, principal, r.Principal) {
			return r.Command.Expected
		}
	}
	return expectedViewRevision(ctx, raw, principal, records, now)
}

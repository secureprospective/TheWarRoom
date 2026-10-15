package history

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/governance"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

// NativeDraft reads the rookie draft for one franchise at the instant the caller names.
func (s *Store) NativeDraft(ctx context.Context, scope commissioner.RawNativeDraftScope,
	local *time.Location) (commissioner.NativeDraft, error) {
	records, err := s.nativeRecords(ctx, scope.League)
	if err != nil {
		return commissioner.NativeDraft{}, err
	}
	reading, err := commissioner.NativeDraftReading(scope, records, local)
	if err != nil {
		return commissioner.NativeDraft{}, fmt.Errorf("read rookie draft: %w", err)
	}
	return reading, nil
}

// RecordDraftPick records one owner's pick. The permit is the calendar's yes for this franchise and the
// draft's pick; the owner is checked against the franchise's seat, and the draft, the roster and the cap
// are judged inside the same immediate transaction that appends the pick, so two picks cannot both land.
func (s *Store) RecordDraftPick(ctx context.Context, permit commissioner.Permit, in commissioner.DraftPickInput,
	principal string, facts nativecheck.Facts, local *time.Location) (commissioner.Receipt, error) {
	if permit.League() != in.League || permit.Franchise() != in.Franchise || permit.Action() != "draft_pick" {
		return commissioner.Receipt{}, fmt.Errorf("league action closed: %w",
			commissioner.Refused(errors.New("the calendar has not opened this pick for your franchise")))
	}
	s.wmu.Lock()
	defer s.wmu.Unlock()
	tx, err := s.pools.Write().BeginTx(ctx, nil)
	if err != nil {
		return commissioner.Receipt{}, fmt.Errorf("begin rookie pick: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	records, err := transactionRecords(ctx, tx, in.League)
	if err != nil {
		return commissioner.Receipt{}, err
	}
	if err := ownsFranchise(records, principal, in.Franchise); err != nil {
		return commissioner.Receipt{}, fmt.Errorf("rookie pick refused: %w", commissioner.Refused(err))
	}
	prior, retried, err := commissioner.RetriedDraftPick(in, principal, records)
	if err != nil {
		return commissioner.Receipt{}, fmt.Errorf("check rookie pick request: %w", err)
	}
	if retried {
		return prior, nil
	}
	if current := int64(len(records)); in.Expected != current {
		return commissioner.Receipt{}, fmt.Errorf("check league version: %w", commissioner.Refused(
			staleVersion{current: current, expected: in.Expected}))
	}
	now := s.now().UTC().Truncate(time.Microsecond)
	raw, err := commissioner.PlanDraftPick(in, records, facts, now, local)
	if err != nil {
		return commissioner.Receipt{}, fmt.Errorf("rookie pick refused: %w", err)
	}
	raw, canonical, err := preparedCommand(raw, principal, true)
	if err != nil {
		return commissioner.Receipt{}, err
	}
	receipt, err := s.appendCommissioner(ctx, tx, appendRequest{raw: raw, principal: principal,
		role: string(governance.Owner), canonical: canonical, at: now, records: records})
	if err != nil {
		return commissioner.Receipt{}, err
	}
	if err := tx.Commit(); err != nil {
		return commissioner.Receipt{}, fmt.Errorf("record rookie pick: %w", err)
	}
	return receipt, nil
}

// ownsFranchise holds a pick to the franchise's own seated owner: identity comes from the session, never
// from the request.
func ownsFranchise(records []commissioner.Record, principal, franchise string) error {
	state, err := commissioner.GovernanceState(records)
	if err != nil {
		return fmt.Errorf("read league seats: %w", err)
	}
	seat, ok := state.Franchise(governance.FranchiseID(franchise))
	if !ok || principal == "" || seat.Seat != governance.Principal(principal) {
		return errors.New("only the owner of this franchise may make its picks")
	}
	return nil
}

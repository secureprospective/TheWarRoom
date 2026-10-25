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
	local *time.Location, facts nativecheck.Facts) (commissioner.NativeDraft, error) {
	records, err := s.nativeRecords(ctx, scope.League)
	if err != nil {
		return commissioner.NativeDraft{}, err
	}
	reading, err := commissioner.NativeDraftReading(scope, records, local, s.draftStandings(ctx, scope.League), facts)
	if err != nil {
		return commissioner.NativeDraft{}, fmt.Errorf("read rookie draft: %w", err)
	}
	reading.Revision = commissioner.ViewRevision(ctx, records, reading.Revision)
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
	tx, err := s.pools.BeginWrite(ctx)
	if err != nil {
		return commissioner.Receipt{}, fmt.Errorf("begin rookie pick: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	records, err := s.transactionRecords(ctx, tx, in.League)
	if err != nil {
		return commissioner.Receipt{}, err
	}
	if err := commissioner.AdmitCredential(ctx, commissioner.Access{Principal: principal, League: in.League,
		Operation: commissioner.DraftPick, Mode: "execute", Franchise: in.Franchise, DraftPick: &in, Records: records}); err != nil {
		return commissioner.Receipt{}, fmt.Errorf("admit league change: %w", err)
	}
	if err := ownsFranchise(records, principal, in.Franchise); err != nil {
		return commissioner.Receipt{}, fmt.Errorf("rookie pick refused: %w", commissioner.Refused(err))
	}
	prior, retried, err := commissioner.RetriedDraftPick(in, principal, records)
	if err != nil {
		return commissioner.Receipt{}, fmt.Errorf("check rookie pick request: %w", err)
	}
	if retried {
		return viewReceipt(ctx, prior, records), nil
	}
	if current := commissioner.ViewRevisionAt(ctx, records, int64(len(records)), s.now()); in.Expected != current {
		return commissioner.Receipt{}, fmt.Errorf("check league version: %w", commissioner.Refused(
			staleVersion{current: current, expected: in.Expected}))
	}
	now := s.now().UTC().Truncate(time.Microsecond)
	raw, err := commissioner.PlanDraftPick(in, records, facts, now, local, s.draftStandings(ctx, in.League))
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
	return s.committedReceipt(ctx, receipt, append(records, commissioner.Record{Command: raw, Principal: principal, Receipt: receipt})), nil
}

// ownsFranchise holds a pick to the franchise's own seated owner: identity comes from the session, never
// from the request.
func ownsFranchise(records []commissioner.Record, principal, franchise string) error {
	state, err := commissioner.GovernanceState(records)
	if err != nil {
		return fmt.Errorf("read league seats: %w", err)
	}
	if !state.HoldsSeat(governance.FranchiseID(franchise), governance.Principal(commissioner.PrincipalFor(records, principal))) {
		return errors.New("only the owner of this franchise may make its picks")
	}
	return nil
}

// draftStandings is how a draft in this league reads the final standings of the season before it: the
// native standings, from this store's own result sources. They are read only when a first round needs them.
func (s *Store) draftStandings(ctx context.Context, league string) commissioner.DraftStandings {
	return &commissioner.LazyStandings{Read: func(records []commissioner.Record, season int,
		through time.Time) (commissioner.NativeStandings, error) {
		return commissioner.FinalStandingsReading(ctx, league, records, s.nativeSources(league, season), season, through)
	}}
}

// clockDraftAuthorizer is deliberately narrower than any human role: the app itself may only close a draft
// that has used its last pick.
type clockDraftAuthorizer struct{}

func (clockDraftAuthorizer) Allow(_ string, principal, command string) (string, error) {
	if principal != commissioner.DraftClockPrincipal || command != commissioner.DraftArchive {
		return "", fmt.Errorf("the draft clock can only close a finished draft")
	}
	return "automatic", nil
}

// NextDraftDeadline is when the league's next draft clock starts or runs out, or zero when none is counting.
func (s *Store) NextDraftDeadline(ctx context.Context, league string, at time.Time) (time.Time, error) {
	records, err := s.nativeRecords(ctx, league)
	if err != nil {
		return time.Time{}, err
	}
	return commissioner.NextDraftDeadline(records, at, s.draftStandings(ctx, league)), nil
}

// CloseFinishedDrafts closes every draft of the league whose last pick has been used, as the app and not a
// person: the same record, checks and request keys as a commissioner closing it by hand, so it lands once.
func (s *Store) CloseFinishedDrafts(ctx context.Context, league string, facts nativecheck.Facts) error {
	records, err := s.nativeRecords(ctx, league)
	if err != nil {
		return err
	}
	for _, season := range commissioner.DueDrafts(records, s.now(), s.draftStandings(ctx, league)) {
		records, err = s.nativeRecords(ctx, league)
		if err != nil {
			return err
		}
		raw := commissioner.RawCommand{League: league, Command: commissioner.DraftArchive,
			Args:     commissioner.RawArgs{Draft: &commissioner.RawDraftArgs{Season: season}},
			Expected: int64(len(records)), Request: fmt.Sprintf("draft-clock.%s.%d.%d", league, season, len(records)),
			Effective: "now", Visibility: "league"}
		if _, err := s.ExecuteCommissionerWithFacts(ctx, raw, commissioner.DraftClockPrincipal,
			clockDraftAuthorizer{}, facts, time.UTC); err != nil {
			return fmt.Errorf("close the %d rookie draft: %w", season, err)
		}
	}
	return nil
}

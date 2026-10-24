package history

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

// PreviewCommissioner uses a read-only snapshot and the execute checks, without the append.
// It never opens the writer pool, reserves a request key or records a trial event.
func (s *Store) PreviewCommissioner(ctx context.Context, raw commissioner.RawCommand,
	principal string, auth commissioner.Authorizer, catalogue commissioner.PreviewCatalogue) (commissioner.Preview, error) {
	return s.PreviewCommissionerWithFacts(ctx, raw, principal, auth, catalogue, nil, time.UTC)
}

// PreviewCommissionerWithFacts uses the same held football facts and owner time zone as execution.
func (s *Store) PreviewCommissionerWithFacts(ctx context.Context, raw commissioner.RawCommand,
	principal string, auth commissioner.Authorizer, catalogue commissioner.PreviewCatalogue,
	playerFacts nativecheck.Facts, zone *time.Location) (commissioner.Preview, error) {
	raw, canonical, err := preparedCommand(raw, principal, false)
	if err != nil {
		return previewRefusal(err)
	}
	facts, err := s.appendFactsFor(ctx, raw)
	if err != nil {
		return previewRefusal(err)
	}
	facts.players = playerFacts
	tx, err := s.pools.Read().BeginTx(ctx, nil)
	if err != nil {
		return commissioner.Preview{}, fmt.Errorf("begin league preview: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	records, err := s.transactionRecords(ctx, tx, raw.League)
	if err != nil {
		return commissioner.Preview{}, err
	}
	if err := commissioner.AdmitCredential(ctx, commissioner.Access{Principal: principal, League: raw.League,
		Operation: raw.Command, Mode: "preview", Raw: raw, Records: records}); err != nil {
		return previewRefusal(err)
	}
	role, retry, checkErr := s.checkCommissionerCommand(raw, principal, auth, records, canonical, facts)
	if role == "" && checkErr != nil {
		return previewRefusal(checkErr)
	}
	var before *commissioner.RolloverBefore
	var plan *commissioner.RolloverPlan
	if checkErr == nil && retry == nil {
		before, plan, checkErr = s.prepareCommissionerTrial(ctx, &raw, principal, records,
			s.now().UTC().Truncate(time.Microsecond), playerFacts, zone, facts)
	}
	blockers, err := previewBlockers(raw, checkErr)
	if err != nil {
		return commissioner.Preview{}, err
	}

	if retry != nil {
		return recordedPreview(raw, records, *retry), nil
	}

	preview, err := s.describePreparedChange(ctx, raw, principal, records, catalogue, facts, before, plan, checkErr == nil)
	if err != nil {
		if len(blockers) > 0 {
			return blockedPreview(blockers), nil
		}
		return commissioner.Preview{}, fmt.Errorf("describe league preview: %w", err)
	}

	preview.Blockers = append(preview.Blockers, blockers...)
	return preview, nil
}

// reasonOf is the sentence a refusal gives the person reading it.
func reasonOf(err error) string {
	var refusal *commissioner.Refusal
	if errors.As(err, &refusal) {
		return refusal.Reason()
	}
	return err.Error()
}

func previewRefusal(err error) (commissioner.Preview, error) {
	var refusal *commissioner.Refusal
	if !errors.As(err, &refusal) {
		return commissioner.Preview{}, fmt.Errorf("league preview: %w", err)
	}
	return blockedPreview([]string{refusal.Reason()}), nil
}

// blockedPreview is what a change gets when it cannot be described: only the reasons it is refused.
func blockedPreview(blockers []string) commissioner.Preview {
	return commissioner.Preview{Fields: []commissioner.PreviewField{},
		Affects:  commissioner.PreviewAffects{Readings: []string{}, Words: "Not known yet"},
		Blockers: blockers}
}

// describeChange is the preview of one change. A live draft change is described with the standings and the
// player names the draft room shows, where the store holds them.
func (s *Store) describeChange(raw commissioner.RawCommand, records []commissioner.Record,
	catalogue commissioner.PreviewCatalogue, facts appendFacts) (commissioner.Preview, error) {
	if !commissioner.DraftTrayCommand(raw.Command) {
		preview, err := commissioner.CommandPreview(raw, records, catalogue, s.now())
		if err != nil {
			return commissioner.Preview{}, fmt.Errorf("describe change: %w", err)
		}
		return preview, nil
	}
	var names nativecheck.Facts
	if s.lineupEnv != nil {
		names = s.lineupEnv.PlayerFacts()
	}
	preview, err := commissioner.DraftPreview(raw, records, catalogue, s.now(), facts.draft, names)
	if err != nil {
		return commissioner.Preview{}, fmt.Errorf("describe draft change: %w", err)
	}
	warning := preview.Affects.Words
	preview.Affects, err = commissioner.PreviewAffectedFranchises(raw, records, s.now(), preview.Affects)
	if err != nil {
		return commissioner.Preview{}, fmt.Errorf("describe affected draft franchises: %w", err)
	}
	if warning != "" {
		preview.Affects.Words += ". " + warning
	}
	return preview, nil
}

func previewBlockers(raw commissioner.RawCommand, checkErr error) ([]string, error) {
	var refusal *commissioner.Refusal
	if checkErr != nil && !errors.As(checkErr, &refusal) {
		return nil, fmt.Errorf("check league preview: %w", checkErr)
	}
	blockers := []string{}
	if missing := raw.MissingReason(); missing != nil {
		blockers = append(blockers, reasonOf(missing))
	}
	if refusal != nil {
		blockers = append(blockers, refusal.Reason())
	}
	return blockers, nil
}

func recordedPreview(raw commissioner.RawCommand, records []commissioner.Record,
	receipt commissioner.Receipt) commissioner.Preview {
	return commissioner.Preview{Fields: []commissioner.PreviewField{{Label: "League ledger",
		Before: receipt.Message, BeforeSource: "Recorded league ledger", After: receipt.Message,
		AfterSource: "Recorded league ledger", When: "Already recorded"}},
		Target:   commissioner.PreviewTarget{League: raw.League, Name: commissioner.Replay(records).Name, Version: raw.Expected},
		Blockers: []string{}, Affects: commissioner.PreviewAffects{Readings: []string{}, Words: "Already recorded; nothing changes"}}
}

func (s *Store) describePreparedChange(ctx context.Context, raw commissioner.RawCommand, principal string,
	records []commissioner.Record, catalogue commissioner.PreviewCatalogue, facts appendFacts,
	before *commissioner.RolloverBefore, plan *commissioner.RolloverPlan, accepted bool) (commissioner.Preview, error) {
	preview, err := s.describeChange(raw, records, catalogue, facts)
	if err != nil {
		return preview, err
	}
	if plan != nil {
		preview.Fields = commissioner.RolloverPreviewFields(before, plan)
		return preview, nil
	}
	if accepted {
		fields, err := s.previewReadingFields(ctx, raw, principal, records, facts)
		if err != nil {
			return commissioner.Preview{}, err
		}
		preview.Fields = append(preview.Fields, fields...)
		commissioner.PreviewFieldSources(raw, preview.Fields)
		draftFields, err := commissioner.DraftSetupFields(raw, records, s.now(), facts.draft)
		if err != nil {
			return commissioner.Preview{}, fmt.Errorf("describe draft order: %w", err)
		}
		preview.Fields = append(preview.Fields, draftFields...)
		if err := commissioner.NamePreviewFields(preview.Fields, records, s.now()); err != nil {
			return commissioner.Preview{}, fmt.Errorf("name preview franchises: %w", err)
		}
	}
	return preview, nil
}

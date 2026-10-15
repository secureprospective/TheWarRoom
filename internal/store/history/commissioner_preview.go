package history

import (
	"context"
	"errors"
	"fmt"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
)

// PreviewCommissioner uses a read-only snapshot and the execute checks, without the append.
// It never opens the writer pool, reserves a request key or records a trial event.
func (s *Store) PreviewCommissioner(ctx context.Context, raw commissioner.RawCommand,
	principal string, auth commissioner.Authorizer, catalogue commissioner.PreviewCatalogue) (commissioner.Preview, error) {
	raw, canonical, err := preparedCommand(raw, principal, false)
	if err != nil {
		return previewRefusal(err)
	}
	tx, err := s.pools.Read().BeginTx(ctx, nil)
	if err != nil {
		return commissioner.Preview{}, fmt.Errorf("begin league preview: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	records, err := transactionRecords(ctx, tx, raw.League)
	if err != nil {
		return commissioner.Preview{}, err
	}
	_, retry, checkErr := s.checkCommissionerCommand(raw, principal, auth, records, canonical)
	var refusal *commissioner.Refusal
	if checkErr != nil && !errors.As(checkErr, &refusal) {
		return commissioner.Preview{}, fmt.Errorf("check league preview: %w", checkErr)
	}
	blockers := []string{}
	if missing := raw.MissingReason(); missing != nil {
		blockers = append(blockers, reasonOf(missing))
	}
	if refusal != nil {
		blockers = append(blockers, refusal.Reason())
	}
	if retry != nil {
		blockers = append(blockers, "This change is already recorded in the league ledger.")
	}
	preview, err := commissioner.CommandPreview(raw, records, catalogue, s.now())
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

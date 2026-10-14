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
	raw, canonical, err := preparedCommand(raw, principal)
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
	preview, err := commissioner.RulePreview(raw, records, catalogue)
	if err != nil {
		if refusal != nil {
			return previewRefusal(checkErr)
		}
		return commissioner.Preview{}, fmt.Errorf("describe league preview: %w", err)
	}
	if refusal != nil {
		preview.Blockers = append(preview.Blockers, refusal.Reason())
	}
	if retry != nil {
		preview.Blockers = append(preview.Blockers, "This change is already recorded in the league ledger.")
	}
	return preview, nil
}

func previewRefusal(err error) (commissioner.Preview, error) {
	var refusal *commissioner.Refusal
	if !errors.As(err, &refusal) {
		return commissioner.Preview{}, fmt.Errorf("league preview: %w", err)
	}
	return commissioner.Preview{Fields: []commissioner.PreviewField{},
		Affects:  commissioner.PreviewAffects{Readings: []string{}, Words: "Not known yet"},
		Blockers: []string{refusal.Reason()}}, nil
}

package moves

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/secureprospective/TheWarRoom/internal/mflref/envelope"
)

func (s *Store) SaveContract(ctx context.Context, e envelope.Envelope, f envelope.ContractFacts) error {
	spec := e.Receipt().Spec
	if !envelope.IsContract(spec.Intent) || spec.Intent != f.Intent || spec.Expected.Player != f.Player || f.Season < 2000 {
		return fmt.Errorf("moves: contract facts do not belong to plan")
	}
	return s.save(ctx, e, func(tx *sql.Tx) error {
		body, err := json.Marshal(f)
		if err != nil {
			return fmt.Errorf("moves: encode contract facts: %w", err)
		}
		if _, err := tx.ExecContext(ctx, `
INSERT INTO contract_facts (correlation_id,facts) VALUES (?,?)
ON CONFLICT(correlation_id) DO UPDATE SET facts=excluded.facts`, e.ID(), string(body)); err != nil {
			return fmt.Errorf("moves: save contract facts: %w", err)
		}
		return nil
	})
}
func (s *Store) ContractFacts(ctx context.Context, id string) (envelope.ContractFacts, error) {
	var body string
	if err := s.pools.Read().QueryRowContext(ctx, `SELECT facts FROM contract_facts WHERE correlation_id=?`, id).Scan(&body); err != nil {
		return envelope.ContractFacts{}, fmt.Errorf("moves: read contract facts: %w", err)
	}
	var f envelope.ContractFacts
	if err := json.Unmarshal([]byte(body), &f); err != nil {
		return f, fmt.Errorf("moves: decode contract facts: %w", err)
	}
	if !envelope.IsContract(f.Intent) || f.Player.String() == "" || f.Season < 2000 {
		return f, fmt.Errorf("moves: invalid contract facts")
	}
	return f, nil
}

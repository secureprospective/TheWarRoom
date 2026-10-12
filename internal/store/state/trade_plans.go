package state

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/secureprospective/TheWarRoom/internal/db"
)

type TradePlan struct {
	ID        string   `json:"id"`
	Mine      string   `json:"mine"`
	Other     string   `json:"other"`
	Give      []string `json:"give"`
	Get       []string `json:"get"`
	Rationale string   `json:"rationale"`
}

type TradePlans struct{ pools *db.Pools }

func NewTradePlans(pools *db.Pools) *TradePlans { return &TradePlans{pools: pools} }
func (s *TradePlans) schema(ctx context.Context) error {
	if _, err := s.pools.Write().ExecContext(ctx, `CREATE TABLE IF NOT EXISTS trade_plans (
 id TEXT PRIMARY KEY, franchise_id TEXT NOT NULL, plan TEXT NOT NULL)`); err != nil {
		return fmt.Errorf("trade plan schema: %w", err)
	}
	return nil
}
func (s *TradePlans) Save(ctx context.Context, p TradePlan) error {
	if err := s.schema(ctx); err != nil {
		return err
	}
	body, err := json.Marshal(p)
	if err != nil {
		return fmt.Errorf("trade plan encode: %w", err)
	}
	if _, err := s.pools.Write().ExecContext(ctx, `INSERT INTO trade_plans (id, franchise_id, plan)
 VALUES (?, ?, ?) ON CONFLICT(id) DO UPDATE SET plan=excluded.plan`, p.ID, p.Mine, string(body)); err != nil {
		return fmt.Errorf("trade plan save: %w", err)
	}
	return nil
}
func (s *TradePlans) List(ctx context.Context, mine string) ([]TradePlan, error) {
	if err := s.schema(ctx); err != nil {
		return nil, err
	}
	rows, err := s.pools.Read().QueryContext(ctx, `SELECT plan FROM trade_plans WHERE franchise_id=? ORDER BY id`, mine)
	if err != nil {
		return nil, fmt.Errorf("trade plans read: %w", err)
	}
	defer func() { _ = rows.Close() }()
	out := []TradePlan{}
	for rows.Next() {
		var body string
		if err := rows.Scan(&body); err != nil {
			return nil, fmt.Errorf("trade plan scan: %w", err)
		}
		var p TradePlan
		if err := json.Unmarshal([]byte(body), &p); err != nil {
			return nil, fmt.Errorf("trade plan decode: %w", err)
		}
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("trade plans iterate: %w", err)
	}
	return out, nil
}

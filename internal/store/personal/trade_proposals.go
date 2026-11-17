package personal

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

const (
	MaxSavedTradeProposals = 50
	MaxTradeProposalBytes  = 16 * 1024
)

// TradeProposalsForFranchise reads only proposal rows for one authenticated franchise.
func (s *Store) TradeProposalsForFranchise(ctx context.Context, franchiseID string) ([]Entry, error) {
	if !s.league {
		return nil, fmt.Errorf("personal: trade proposals require a league store")
	}
	if err := validateFranchiseScope(franchiseID); err != nil {
		return nil, err
	}
	rows, err := s.pools.Read().QueryContext(ctx, `SELECT kind,entity,entity_id,text,at,ordinal FROM private_entries
WHERE franchise_id=? AND mine=? AND kind='proposal' AND entity='trade' ORDER BY ordinal,at,entity_id`,
		franchiseID, franchiseID)
	if err != nil {
		return nil, fmt.Errorf("personal: read trade proposals: %w", err)
	}
	defer func() { _ = rows.Close() }()
	entries := []Entry{}
	for rows.Next() {
		var entry Entry
		if err := rows.Scan(&entry.Kind, &entry.Entity, &entry.ID, &entry.Text, &entry.At, &entry.Order); err != nil {
			return nil, fmt.Errorf("personal: scan trade proposal: %w", err)
		}
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("personal: trade proposal rows: %w", err)
	}
	return entries, nil
}

// SaveTradeProposalForFranchise stores one bounded proposal for its authenticated owner.
func (s *Store) SaveTradeProposalForFranchise(ctx context.Context, franchiseID, id, text string, at time.Time) error {
	if !s.league {
		return fmt.Errorf("personal: trade proposals require a league store")
	}
	if err := validateFranchiseScope(franchiseID); err != nil {
		return err
	}
	if !validTradeProposalID(id) {
		return fmt.Errorf("personal: invalid proposal id")
	}
	if len(text) == 0 || len(text) >= MaxTradeProposalBytes || !json.Valid([]byte(text)) {
		return fmt.Errorf("personal: saved proposal must be valid JSON and under 16 KB")
	}
	tx, err := s.pools.Write().BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("personal: begin trade proposal save: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	var count int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM private_entries
WHERE franchise_id=? AND mine=? AND kind='proposal' AND entity='trade'`, franchiseID, franchiseID).Scan(&count); err != nil {
		return fmt.Errorf("personal: count trade proposals: %w", err)
	}
	if count >= MaxSavedTradeProposals {
		return fmt.Errorf("personal: saved proposals are limited to %d per team", MaxSavedTradeProposals)
	}
	stamp := at.UTC().Format(time.RFC3339Nano)
	if _, err := tx.ExecContext(ctx, `INSERT INTO private_entries(franchise_id,mine,kind,entity,entity_id,text,at,ordinal)
SELECT ?,?,'proposal','trade',?,?,?,COALESCE(MAX(ordinal),0)+1 FROM private_entries
WHERE franchise_id=? AND mine=? AND kind='proposal' AND entity='trade'`,
		franchiseID, franchiseID, id, text, stamp, franchiseID, franchiseID); err != nil {
		return fmt.Errorf("personal: save trade proposal: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("personal: commit trade proposal: %w", err)
	}
	return nil
}

// DeleteTradeProposalForFranchise deletes one proposal without addressing another owner's rows.
func (s *Store) DeleteTradeProposalForFranchise(ctx context.Context, franchiseID, id string) error {
	if !s.league {
		return fmt.Errorf("personal: trade proposals require a league store")
	}
	if err := validateFranchiseScope(franchiseID); err != nil {
		return err
	}
	if !validTradeProposalID(id) {
		return fmt.Errorf("personal: invalid proposal id")
	}
	result, err := s.pools.Write().ExecContext(ctx, `DELETE FROM private_entries
WHERE franchise_id=? AND mine=? AND kind='proposal' AND entity='trade' AND entity_id=?`,
		franchiseID, franchiseID, id)
	if err != nil {
		return fmt.Errorf("personal: delete trade proposal: %w", err)
	}
	removed, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("personal: count deleted trade proposal: %w", err)
	}
	if removed == 0 {
		return fmt.Errorf("personal: saved proposal not found")
	}
	return nil
}

func validTradeProposalID(id string) bool {
	if len(id) != 32 || strings.ToLower(id) != id {
		return false
	}
	_, err := hex.DecodeString(id)
	return err == nil
}

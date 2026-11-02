package leaguemigration

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
)

// SyncTransactions appends each transaction atomically through the ordinary ledger.
// Stable command requests make a successful transaction a no-op on subsequent reads.
// A failed transaction stops replay; earlier successful transactions remain recorded.
func SyncTransactions(ctx context.Context, ledger Ledger, league, principal string, groups [][]commissioner.RawCommand) ([]commissioner.Receipt, error) {
	records, err := ledger.Records(ctx, league)
	if err != nil {
		return nil, fmt.Errorf("read league before updating transactions: %w", err)
	}
	if len(records) == 0 {
		return nil, fmt.Errorf("create the league before updating transactions")
	}
	seen := map[string]bool{}
	for _, record := range records {
		seen[record.Command.Request] = true
	}
	receipts := []commissioner.Receipt{}
	for _, group := range groups {
		if len(group) == 0 {
			continue
		}
		pending, err := pendingTransactionCommands(group, league, seen)
		if err != nil {
			return receipts, err
		}
		if pending == 0 {
			continue
		}
		if pending != len(group) {
			return receipts, fmt.Errorf("transaction is only partly recorded; review its history before retrying")
		}
		batch := append([]commissioner.RawCommand(nil), group...)
		batch[0].Expected = int64(len(records) + len(receipts))
		appended, err := ledger.Batch(ctx, batch, principal)
		if err != nil {
			return receipts, fmt.Errorf("record transaction %s (transaction unchanged): %w", group[0].Request, err)
		}
		receipts = append(receipts, appended...)
		for _, raw := range group {
			seen[raw.Request] = true
		}
	}
	return receipts, nil
}

func pendingTransactionCommands(group []commissioner.RawCommand, league string, seen map[string]bool) (int, error) {
	pending := 0
	requests := map[string]bool{}
	for _, raw := range group {
		if raw.League != league || !strings.HasPrefix(raw.Request, "transactions.sync.") || requests[raw.Request] {
			return 0, fmt.Errorf("transaction commands need this league and unique stable requests")
		}
		requests[raw.Request] = true
		if !slices.Contains([]string{commissioner.CorrectRoster, commissioner.CorrectPick, commissioner.RosterReserve}, raw.Command) {
			return 0, fmt.Errorf("transaction replay accepts only roster, pick and reserve changes")
		}
		if err := raw.Validate(); err != nil {
			return 0, fmt.Errorf("check transaction command: %w", err)
		}
		if !seen[raw.Request] {
			pending++
		}
	}
	return pending, nil
}

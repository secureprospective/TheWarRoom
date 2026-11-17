package personal

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/db"
)

func TestHostedTradeProposalsAreScopedAndDeletable(t *testing.T) {
	ctx := context.Background()
	store := openTradeProposalStore(ctx, t)
	at := time.Date(2026, 11, 17, 12, 0, 0, 0, time.UTC)
	text := `{"partner":"franchise.02","give":["0101"],"get":[]}`
	if err := store.SaveTradeProposalForFranchise(ctx, "franchise.01", strings.Repeat("a", 32), text, at); err != nil {
		t.Fatal(err)
	}
	first, err := store.TradeProposalsForFranchise(ctx, "franchise.01")
	if err != nil || len(first) != 1 || first[0].Kind != "proposal" || first[0].Entity != "trade" ||
		first[0].Text != text || first[0].At != at.Format(time.RFC3339Nano) {
		t.Fatalf("owner proposal rows = %+v, %v", first, err)
	}
	other, err := store.TradeProposalsForFranchise(ctx, "franchise.02")
	if err != nil || len(other) != 0 {
		t.Fatalf("other franchise read proposal rows = %+v, %v", other, err)
	}
	if err := store.DeleteTradeProposalForFranchise(ctx, "franchise.02", first[0].ID); err == nil {
		t.Fatal("another franchise deleted the proposal")
	}
	remaining, err := store.TradeProposalsForFranchise(ctx, "franchise.01")
	if err != nil || len(remaining) != 1 {
		t.Fatalf("cross-franchise delete changed owner row: %+v, %v", remaining, err)
	}
	if err := store.DeleteTradeProposalForFranchise(ctx, "franchise.01", first[0].ID); err != nil {
		t.Fatal(err)
	}
	remaining, err = store.TradeProposalsForFranchise(ctx, "franchise.01")
	if err != nil || len(remaining) != 0 {
		t.Fatalf("deleted owner proposal remains: %+v, %v", remaining, err)
	}
}

func TestHostedTradeProposalCountAndSizeLimitsPreserveRows(t *testing.T) {
	ctx := context.Background()
	store := openTradeProposalStore(ctx, t)
	at := time.Date(2026, 11, 17, 12, 0, 0, 0, time.UTC)
	text := `{"partner":"franchise.02","give":["0101"],"get":[]}`
	for i := 0; i < MaxSavedTradeProposals; i++ {
		if err := store.SaveTradeProposalForFranchise(ctx, "franchise.01", fmt.Sprintf("%032x", i+1), text, at); err != nil {
			t.Fatalf("save %d: %v", i, err)
		}
	}
	if err := store.SaveTradeProposalForFranchise(ctx, "franchise.01", fmt.Sprintf("%032x", MaxSavedTradeProposals+1), text, at); err == nil || !strings.Contains(err.Error(), "limited to 50") {
		t.Fatalf("51st proposal error = %v", err)
	}
	oversize, err := json.Marshal(strings.Repeat("x", MaxTradeProposalBytes))
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SaveTradeProposalForFranchise(ctx, "franchise.02", strings.Repeat("b", 32), string(oversize), at); err == nil || !strings.Contains(err.Error(), "under 16 KB") {
		t.Fatalf("oversize proposal error = %v", err)
	}
	rows, err := store.TradeProposalsForFranchise(ctx, "franchise.01")
	if err != nil || len(rows) != MaxSavedTradeProposals {
		t.Fatalf("rejected proposal changed existing rows: %d, %v", len(rows), err)
	}
	other, err := store.TradeProposalsForFranchise(ctx, "franchise.02")
	if err != nil || len(other) != 0 {
		t.Fatalf("rejected proposal left another row: %+v, %v", other, err)
	}
}

func openTradeProposalStore(ctx context.Context, t *testing.T) *Store {
	t.Helper()
	pools, err := db.Open(ctx, filepath.Join(t.TempDir(), "league.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := pools.Close(); err != nil {
			t.Error(err)
		}
	})
	_, err = pools.Write().ExecContext(ctx, `CREATE TABLE private_entries (
 franchise_id TEXT NOT NULL, mine TEXT NOT NULL, kind TEXT NOT NULL, entity TEXT NOT NULL,
 entity_id TEXT NOT NULL, text TEXT NOT NULL, at TEXT NOT NULL, ordinal INTEGER NOT NULL,
 PRIMARY KEY (franchise_id, mine, kind, entity, entity_id, at), CHECK (franchise_id = mine));
 CREATE UNIQUE INDEX private_entries_marker ON private_entries(franchise_id, mine, kind, entity, entity_id)
 WHERE kind != 'note';`)
	if err != nil {
		t.Fatal(err)
	}
	return NewLeague(pools)
}

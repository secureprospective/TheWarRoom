package history

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/transactions"
	"github.com/secureprospective/TheWarRoom/internal/leaguemigration"
	"github.com/secureprospective/TheWarRoom/internal/leaguemigration/mflsource"
	"github.com/secureprospective/TheWarRoom/internal/leaguemigration/migrationtest"
)

func TestMigrationTransactionSync(t *testing.T) {
	t.Parallel()
	export, err := migrationtest.Legacy()
	if err != nil {
		t.Fatal(err)
	}
	svc, store, _ := migrationService(t, export)
	if _, err := svc.Import(t.Context(), export.LeagueID, "Import", "creator"); err != nil {
		t.Fatal(err)
	}
	league := leaguemigration.AppLeagueID(export.LeagueID, export.Season)
	before, err := store.NativeAssets(t.Context(), league)
	if err != nil {
		t.Fatal(err)
	}
	p := before.Snapshot.Players[0]
	var other string
	mapping := map[string]string{}
	for _, f := range before.Snapshot.Franchises {
		mapping[f.HeldID] = f.ID
		if f.ID != p.Franchise {
			other = f.HeldID
		}
	}
	var from string
	for held, id := range mapping {
		if id == p.Franchise {
			from = held
		}
	}
	raw := transactions.RawTransactions{Transactions: []transactions.RawTransaction{{Kind: "TRADE", Timestamp: fmt.Sprint(migrationtest.ReadAt.Add(30 * time.Minute).Unix()), Franchise: from,
		Counterparty: other, Gave: new(p.ID + ","), CounterpartyGave: new(""), Expires: fmt.Sprint(migrationtest.ReadAt.Unix())}}}
	groups, err := mflsource.TransactionCommands(raw, league, "Import a checked trade", export.Season, mapping, map[string]string{p.ID: p.ID})
	if err != nil {
		t.Fatal(err)
	}
	ledger := migrationLedger{store}
	// Authorization failure must leave both sides and the ledger unchanged.
	if _, err := leaguemigration.SyncTransactions(t.Context(), ledger, league, "outsider", groups); err == nil {
		t.Fatal("unauthorized import accepted")
	}
	unchanged, err := store.NativeAssets(t.Context(), league)
	if err != nil || !reflect.DeepEqual(before, unchanged) {
		t.Fatalf("denied import changed state: %v", err)
	}
	for pass := range 2 {
		receipts, err := leaguemigration.SyncTransactions(t.Context(), ledger, league, "creator", groups)
		want := 2
		if pass == 1 {
			want = 0
		}
		if err != nil || len(receipts) != want {
			t.Fatalf("pass %d: %d receipts, %v", pass, len(receipts), err)
		}
	}
	after, err := store.NativeAssets(t.Context(), league)
	if err != nil {
		t.Fatal(err)
	}
	index := after.Snapshot.PlayerOf(mapping[other], p.ID)
	if index < 0 {
		t.Fatal("trade did not move player")
	}
	got := after.Snapshot.Players[index]
	if got.Salary != p.Salary || got.ExpirationYear != p.ExpirationYear || got.ContractStatus != p.ContractStatus {
		t.Fatal("trade invented a contract")
	}
	records, err := store.nativeRecords(t.Context(), league)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range records[len(records)-2:] {
		at, err := r.Command.EffectiveTime(r.Receipt.RecordedAt)
		if err != nil || !at.Before(r.Receipt.RecordedAt) || r.Principal != "creator" {
			t.Fatalf("lost effective/receipt time or actor: %+v, %v", r, err)
		}
	}
	bad := groups[0][1]
	bad.Request += ".invalid-franchise"
	bad.Args = commissioner.RawArgs{Correction: &commissioner.RawCorrection{Action: "add", Player: "999999", Franchise: "franchise.missing"}}
	removal := groups[0][0]
	removal.Request += ".rollback"
	removal.Args = commissioner.RawArgs{Correction: &commissioner.RawCorrection{Action: "remove", Player: p.ID, Franchise: mapping[other]}}
	if _, err := leaguemigration.SyncTransactions(t.Context(), ledger, league, "creator", [][]commissioner.RawCommand{{removal, bad}}); err == nil {
		t.Fatal("invalid franchise admitted")
	}
	unchanged, err = store.NativeAssets(t.Context(), league)
	if err != nil || !reflect.DeepEqual(after, unchanged) {
		t.Fatalf("failed transaction was not atomic: %v", err)
	}
}

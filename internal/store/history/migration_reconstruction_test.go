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

func TestMigrationPreObservationReconstruction(t *testing.T) {
	export := migrationtest.Synthetic("Historical league", migrationtest.Specs()["8 teams"])
	svc, store, _ := migrationService(t, export)
	if _, err := svc.Import(t.Context(), export.LeagueID, "Import", "creator"); err != nil {
		t.Fatal(err)
	}
	league := leaguemigration.AppLeagueID(export.LeagueID, export.Season)
	original, err := store.NativeAssets(t.Context(), league)
	if err != nil {
		t.Fatal(err)
	}
	seed := *original.Snapshot
	p := seed.Players[0]
	mapping := map[string]string{}
	from, other := "", ""
	for _, f := range seed.Franchises {
		mapping[f.HeldID] = f.ID
		if f.ID == p.Franchise {
			from = f.HeldID
		} else {
			other = f.HeldID
		}
	}
	at := seed.RosterAt.Add(-30 * 24 * time.Hour).Truncate(time.Second)
	stamp := func(d time.Duration) string { return fmt.Sprint(at.Add(d).Unix()) }
	raw := transactions.RawTransactions{Transactions: []transactions.RawTransaction{
		{Kind: "FREE_AGENT", Timestamp: stamp(0), Franchise: from, Transaction: new("|999999,")},
		{Kind: "FREE_AGENT", Timestamp: stamp(time.Hour), Franchise: other, Transaction: new(p.ID + ",|")},
		{Kind: "TRADE", Timestamp: stamp(2 * time.Hour), Franchise: other, Counterparty: from, Gave: new(p.ID + ","), CounterpartyGave: new(""), Expires: stamp(3 * time.Hour)},
		{Kind: "IR", Timestamp: stamp(3 * time.Hour), Franchise: from, Deactivated: new(p.ID + ","), Activated: new("")},
		{Kind: "IR", Timestamp: stamp(4 * time.Hour), Franchise: from, Deactivated: new(""), Activated: new(p.ID + ",")},
	}}
	players := map[string]string{p.ID: p.ID, "999999": "999999"}
	groups, err := mflsource.TransactionReplayCommands(raw, league, "Reconstruct recorded roster history", seed, mapping, players)
	if err != nil {
		t.Fatal(err)
	}
	before, err := store.nativeRecords(t.Context(), league)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := leaguemigration.SyncTransactions(t.Context(), migrationLedger{store}, league, "outsider", groups); err == nil {
		t.Fatal("unauthorized historical replay accepted")
	}
	unchanged, err := store.nativeRecords(t.Context(), league)
	if err != nil || !reflect.DeepEqual(before, unchanged) {
		t.Fatal("denied reconstruction changed ledger", err)
	}
	for pass := range 2 {
		receipts, err := leaguemigration.SyncTransactions(t.Context(), migrationLedger{store}, league, "creator", groups)
		if err != nil || (pass == 0 && len(receipts) == 0) || (pass == 1 && len(receipts) != 0) {
			t.Fatalf("pass %d: %d appended, %v", pass, len(receipts), err)
		}
	}
	records, err := store.nativeRecords(t.Context(), league)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name    string
		at      time.Time
		owner   string
		unknown bool
	}{
		{"reconstructed baseline", at.Add(-time.Nanosecond), "", true},
		{"after historical drop", at, "", false},
		{"after acquisition", at.Add(time.Hour), mapping[other], false},
		{"after trade", at.Add(2 * time.Hour), p.Franchise, false},
		{"at observation", seed.RosterAt, p.Franchise, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assets, err := commissioner.NativeAssetsAsOf(league, records, tc.at)
			if err != nil {
				t.Fatal(err)
			}
			present := assets.Snapshot.PlayerOn("", p.ID)
			owner := ""
			if present >= 0 {
				owner = assets.Snapshot.Players[present].Franchise
			}
			if owner != tc.owner {
				t.Fatalf("owner %q, want %q", owner, tc.owner)
			}
			index := assets.Snapshot.PlayerOf(p.Franchise, "999999")
			if (index >= 0) != tc.unknown {
				t.Fatal("historical dropped player missing or retained")
			}
			if index >= 0 && !assets.Snapshot.Players[index].ContractMissing {
				t.Fatal("historical contract invented")
			}
		})
	}
	observed, err := commissioner.NativeAssetsAsOf(league, records, seed.RosterAt)
	if err != nil {
		t.Fatal(err)
	}
	got := observed.Snapshot.Players[observed.Snapshot.PlayerOf(p.Franchise, p.ID)]
	if got.Salary != p.Salary || got.ExpirationYear != p.ExpirationYear || got.Status != p.Status || len(observed.Snapshot.Players) != len(seed.Players) {
		t.Fatal("replay does not reproduce observed roster and contract")
	}
}

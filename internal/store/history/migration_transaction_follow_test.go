package history

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/transactions"
	"github.com/secureprospective/TheWarRoom/internal/leaguemigration"
	"github.com/secureprospective/TheWarRoom/internal/leaguemigration/migrationtest"
)

func TestMigrationTransactionFollowSetting(t *testing.T) {
	for _, follow := range []bool{false, true} {
		t.Run(fmt.Sprint(follow), func(t *testing.T) {
			export := migrationtest.Synthetic("Follow", migrationtest.Specs()["8 teams"])
			svc, s, src := migrationService(t, export)
			if _, err := svc.Import(t.Context(), export.LeagueID, "Import", "creator"); err != nil {
				t.Fatal(err)
			}
			league := leaguemigration.AppLeagueID(export.LeagueID, export.Season)
			records, err := s.nativeRecords(t.Context(), league)
			if err != nil {
				t.Fatal(err)
			}
			value, err := json.Marshal(follow)
			if err != nil {
				t.Fatal(err)
			}
			if !follow {
				_, err = s.ExecuteCommissioner(t.Context(), commissioner.RawCommand{League: league, Command: commissioner.RulesSet, Args: commissioner.RawArgs{Field: "transactions.follow", Value: value}, Expected: int64(len(records)), Request: "follow", Reason: "Choose reference refresh", Effective: "now", Visibility: "league"}, "creator", nil)
				if err != nil {
					t.Fatal(err)
				}
			}
			raw := transactions.RawTransactions{Transactions: []transactions.RawTransaction{{Kind: "TRADE", Timestamp: fmt.Sprint(export.ReadAt.Add(-time.Hour).Unix()), Franchise: "0001", Counterparty: "0002", Gave: new("FP_0001_2026_3,"), CounterpartyGave: new(""), Expires: fmt.Sprint(export.ReadAt.Unix())}}}
			src.Export.Transactions = &raw
			receipts, err := svc.Import(t.Context(), export.LeagueID, "Refresh historical evidence", "creator")
			if err != nil {
				t.Fatal(err)
			}
			want := 0
			if follow {
				want = 2
			}
			if len(receipts) != want {
				t.Fatalf("appended=%d want=%d", len(receipts), want)
			}
			receipts, err = svc.Import(t.Context(), export.LeagueID, "Repeat", "creator")
			if err != nil || len(receipts) != 0 {
				t.Fatalf("repeat=%d %v", len(receipts), err)
			}
		})
	}
}

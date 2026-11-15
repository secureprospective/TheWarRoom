package history

import (
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/commissioner/market"
	"github.com/secureprospective/TheWarRoom/internal/leaguemigration"
	"github.com/secureprospective/TheWarRoom/internal/leaguemigration/mflsource"
	"github.com/secureprospective/TheWarRoom/internal/leaguemigration/migrationtest"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

func TestImportedCutPricingReadsRecordedContractAndRules(t *testing.T) {
	for _, known := range []bool{false, true} {
		name := "missing rate"
		if known {
			name = "known rate"
		}
		t.Run(name, func(t *testing.T) {
			fixture, err := migrationtest.Legacy()
			if err != nil {
				t.Fatal(err)
			}
			fixture.Base = leaguerules.LegacyNFL()
			if !known {
				fixture.Base.Contracts.DeadCap = leaguerules.Needs[leaguerules.Decimal]("not supplied")
			}
			svc, store, _ := migrationService(t, fixture)
			plan, err := mflsource.Prepare(fixture)
			if err != nil {
				t.Fatal(err)
			}
			league := leaguemigration.AppLeagueID(fixture.LeagueID, fixture.Season)
			if _, err := svc.Ledger.Batch(t.Context(), []commissioner.RawCommand{
				{League: league, Command: commissioner.Create, Args: plan.Create, Request: "create", Reason: "Check pricing", Effective: "now", Visibility: "league"},
				{League: league, Command: commissioner.NativeSeed, Args: commissioner.RawArgs{Native: &plan.Seed}, Request: "seed", Reason: "Check pricing", Effective: "now", Visibility: "league"},
			}, "creator"); err != nil {
				t.Fatal(err)
			}
			records, err := store.nativeRecords(t.Context(), league)
			if err != nil {
				t.Fatal(err)
			}
			// The source fixture has blank contract statuses; these must not prevent release pricing.
			var player string
			for _, p := range fixture.Rosters {
				if p.ContractYear == "2028" && p.Salary != "0" && p.Salary != "0.00" {
					player = p.Player
					break
				}
			}
			if player == "" {
				t.Fatal("fixture needs a salaried contract through 2028")
			}
			quote, err := market.QuoteContract(market.RawContractQuote{League: league, Player: player, Action: "cut", ViewerZone: "UTC"}, records, svc.Now(), time.UTC, nil)
			if err != nil {
				t.Fatal(err)
			}
			if known {
				if len(quote.DeadMoney) != 1 || quote.DeadMoney[0].Cents <= 0 || quote.Cap == nil {
					t.Fatalf("recorded terms should price even behind a closed window: %+v", quote)
				}
			} else if len(quote.DeadMoney) != 0 || !strings.Contains(quote.Reason, "dead money rate") {
				t.Fatalf("missing release rate must not become a zero: %+v", quote)
			}
			after, err := store.nativeRecords(t.Context(), league)
			if err != nil || len(after) != len(records) {
				t.Fatalf("quote changed ledger: %d -> %d, %v", len(records), len(after), err)
			}
			t.Logf("player=%s allowed=%t dead=%v reason=%s", player, quote.Allowed, quote.DeadMoney, quote.Reason)
		})
	}
}

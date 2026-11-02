package mflsource

import (
	"reflect"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/transactions"
)

func TestTransactionCommands(t *testing.T) {
	t.Parallel()
	franchises := map[string]string{"0001": "franchise.01", "0002": "franchise.02"}
	players := map[string]string{"0099": "0099", "1000": "1000"}
	for _, tc := range []struct {
		name, row, want string
		count           int
	}{
		{"add and drop", `"type":"FREE_AGENT","transaction":"0099,|1000,"`, "", 2},
		{"baseline only", `"type":"LOAD_ROSTERS","transaction":"0099,|"`, "", 0},
		{"trade players and pick", `"type":"TRADE","franchise2":"0002","franchise1_gave_up":"0099,","franchise2_gave_up":"FP_0002_2027_4,","expires":"1791000000"`, "", 3},
		{"trade current pick", `"type":"TRADE","franchise2":"0002","franchise1_gave_up":"DP_2_0,","franchise2_gave_up":"0099,","expires":"1791000000"`, "", 3},
		{"IR both ways", `"type":"IR","activated":"0099,","deactivated":"1000,"`, "", 2},
		{"taxi both ways", `"type":"TAXI","promoted":"0099,","demoted":"1000,"`, "", 2},
		{"unknown player", `"type":"FREE_AGENT","transaction":"9999,|"`, "unknown player", 0},
		{"unparsed kind", `"type":"SOMETHING_NEW"`, "unsupported transaction", 0},
		{"malformed", `"type":"FREE_AGENT","transaction":"0099,"`, "exactly one pipe", 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw, err := transactions.Parse([]byte(`{"transactions":{"transaction":{"timestamp":"1791000000","franchise":"0001",` + tc.row + `}}}`))
			var groups [][]commissioner.RawCommand
			if err == nil {
				groups, err = TransactionCommands(raw, "native.test", "Import checked transactions", 2026, franchises, players)
			}
			if tc.want != "" {
				if err == nil || !strings.Contains(err.Error(), tc.want) || len(groups) != 0 {
					t.Fatalf("groups %v err %v, want %s", groups, err, tc.want)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			count := 0
			for _, group := range groups {
				count += len(group)
				for _, command := range group {
					if command.Effective != "2026-10-03T04:00:00Z" || command.Visibility != "league" {
						t.Fatalf("timing/visibility: %+v", command)
					}
				}
			}
			if count != tc.count {
				t.Fatalf("got %d commands, want %d", count, tc.count)
			}
			again, err := TransactionCommands(raw, "native.test", "Import checked transactions", 2026, franchises, players)
			if err != nil || !reflect.DeepEqual(groups, again) {
				t.Fatalf("unstable retry: %v", err)
			}
			if tc.name == "trade current pick" {
				pick := groups[0][0].Args.Correction.Pick
				if pick.Season != 2026 || pick.Round != 3 || pick.Slot != 1 || pick.Original != "" {
					t.Fatalf("lost current draft identity: %+v", pick)
				}
			}
			if tc.name == "add and drop" && (groups[0][0].Args.Correction.Action != "remove" || groups[0][1].Args.Correction.Player != "0099") {
				t.Fatal("lost drop-before-add or leading zero ID")
			}
		})
	}
}

func TestTransactionIdentitySurvivesFeedReordering(t *testing.T) {
	t.Parallel()
	first := transactions.RawTransaction{Kind: "FREE_AGENT", Timestamp: "1791000000", Franchise: "0001", Transaction: new("0099,|")}
	second := first
	second.Timestamp = "1791000001"
	raw := transactions.RawTransactions{Transactions: []transactions.RawTransaction{second, first}}
	mapping := map[string]string{"0001": "franchise.01"}
	a, err := TransactionCommands(raw, "native.test", "Import", 2026, mapping, map[string]string{"0099": "0099"})
	if err != nil {
		t.Fatal(err)
	}
	raw.Transactions[0], raw.Transactions[1] = first, second
	b, err := TransactionCommands(raw, "native.test", "Import", 2026, mapping, map[string]string{"0099": "0099"})
	if err != nil || !reflect.DeepEqual(a, b) || a[0][0].Request == a[1][0].Request {
		t.Fatalf("unstable or colliding identity: %v", err)
	}
	for _, tc := range []struct {
		season  int
		mapping map[string]string
	}{{2025, mapping}, {2026, nil}} {
		if _, err := TransactionCommands(raw, "native.test", "Import", tc.season, tc.mapping, map[string]string{"0099": "0099"}); err == nil {
			t.Fatal("accepted wrong season or unknown franchise")
		}
	}
}

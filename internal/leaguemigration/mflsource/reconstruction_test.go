package mflsource_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/ingestion/transactions"
	"github.com/secureprospective/TheWarRoom/internal/leaguemigration/mflsource"
	"github.com/secureprospective/TheWarRoom/internal/leaguemigration/migrationtest"
)

func TestReconstructionRefusesUnprovedHistory(t *testing.T) {
	for _, tc := range []struct{ name, row, want string }{
		{"unobserved acquisition", `"type":"FREE_AGENT","transaction":"999999,|"`, "after the observed acquisition"},
		{"consumed original pick has no slot link", `"type":"TRADE","franchise2":"0002","franchise1_gave_up":"FP_0001_2026_3,","franchise2_gave_up":"","expires":"1773014400"`, "has no observation"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			export := migrationtest.Synthetic("Refusal", migrationtest.Specs()["8 teams"])
			plan, err := mflsource.Prepare(export)
			if err != nil {
				t.Fatal(err)
			}
			before, err := json.Marshal(plan.Seed)
			if err != nil {
				t.Fatal(err)
			}
			raw, err := transactions.Parse([]byte(`{"transactions":{"transaction":{"timestamp":"1772888911","franchise":"0001",` + tc.row + `}}}`))
			if err != nil {
				t.Fatal(err)
			}
			groups, err := mflsource.TransactionReplayCommands(raw, "native.test", "Repair history", plan.Seed, plan.Mapping, map[string]string{"999999": "999999"})
			if err == nil || !strings.Contains(err.Error(), tc.want) || len(groups) != 0 {
				t.Fatalf("unproved history admitted: %d groups, %v", len(groups), err)
			}
			after, err := json.Marshal(plan.Seed)
			if err != nil || string(before) != string(after) {
				t.Fatal("failed reconstruction changed the observation", err)
			}
		})
	}
}

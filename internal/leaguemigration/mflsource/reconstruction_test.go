package mflsource_test

import (
	"encoding/json"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/draftreports"

	"github.com/secureprospective/TheWarRoom/internal/ingestion/transactions"
	"github.com/secureprospective/TheWarRoom/internal/leaguemigration/mflsource"
	"github.com/secureprospective/TheWarRoom/internal/leaguemigration/migrationtest"
)

func TestReconstructionRefusesUnprovedHistory(t *testing.T) {
	for _, tc := range []struct{ name, row, want string }{
		{"unobserved acquisition", `"type":"FREE_AGENT","transaction":"999999,|"`, "after the observed acquisition"},
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

func TestHistoricalPickReconstruction(t *testing.T) {
	for _, tc := range []struct {
		name, comments string
		slot           int
		unverified     bool
	}{
		{"no evidence", "", 0, true},
		{"checked original", "[Pick traded from Franchise 1.] ", 2, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			seed := commissioner.RawNativeSnapshot{Season: 2026, RosterAt: time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC),
				Franchises: []commissioner.NativeFranchise{{ID: "franchise.01", Name: "Franchise 1"}, {ID: "franchise.02", HeldID: "0002"}},
				Picks:      []commissioner.NativePick{{Season: 2026, Round: 3, Slot: 2, Owner: "franchise.02", Player: "0099"}}}
			raw, err := transactions.Parse([]byte(`{"transactions":{"transaction":{"timestamp":"1772888911","franchise":"0001","type":"TRADE","franchise2":"0002","franchise1_gave_up":"FP_0001_2026_3,","franchise2_gave_up":"","expires":"1773014400"}}}`))
			if err != nil {
				t.Fatal(err)
			}
			picks, err := draftreports.ParseResults([]byte(`{"draftResults":{"draftUnit":{"draftPick":{"round":"3","pick":"2","player":"0099","franchise":"0002","comments":` + strconv.Quote(tc.comments) + `}}}}`))
			if err != nil {
				t.Fatal(err)
			}
			groups, err := mflsource.TransactionReplayCommands(raw, "native.test", "Import history", seed, map[string]string{"0001": "franchise.01", "0002": "franchise.02"}, nil, picks)
			if err != nil {
				t.Fatal(err)
			}
			if len(groups) != 1 || len(groups[0]) != 2 {
				t.Fatalf("groups=%+v", groups)
			}
			for _, cmd := range groups[0] {
				c := cmd.Args.Correction
				if c.History == nil || c.History.Unverified != tc.unverified || c.History.Slot != tc.slot {
					t.Fatalf("evidence=%+v", c)
				}
			}
			if len(seed.PickHistory) != 0 || seed.Picks[0].Owner != "franchise.02" {
				t.Fatal("mutated held seed")
			}
		})
	}
}

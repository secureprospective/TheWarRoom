package composition

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
)

func TestStartingSettingsUseOrdinaryLedgerChanges(t *testing.T) {
	app := commissionerTestApp(t)
	const league = "native.starting-settings"
	executeCommissionerTest(t, app, commissioner.RawCommand{League: league, Command: commissioner.Create,
		Args:   commissioner.RawArgs{Name: "Settings", Preset: commissioner.PresetEmpty, FranchiseCount: 4, Season: 2026},
		Reason: "Start", Request: "create", Effective: "now", Visibility: "league"})
	for i, tc := range []struct{ field, value string }{
		{"lineup.lock", `"first_kickoff"`}, {"lineup.lock", `"each_player_game"`},
		{"league.startingAssets", `"held"`}, {"league.startingAssets", `"empty"`},
	} {
		before, err := newCommissionerAPI(app).TargetCommissionerRules(league)
		if err != nil {
			t.Fatal(err)
		}
		receipt := executeCommissionerTest(t, app, commissioner.RawCommand{League: league, Command: commissioner.RulesSet,
			Args: commissioner.RawArgs{Field: tc.field, Value: json.RawMessage(tc.value)}, Expected: before.Revision,
			Reason: "Commissioner changed the league setting", Request: fmt.Sprintf("setting.%d", i), Effective: "now", Visibility: "league"})
		reading, err := newCommissionerAPI(app).TargetCommissionerRules(league)
		if err != nil {
			t.Fatal(err)
		}
		found := false
		for _, section := range reading.Sections {
			for _, row := range section.Rows {
				if row.Field != tc.field {
					continue
				}
				found = true
				if string(row.Current) != tc.value || row.Citation != "Changed in the league ledger" || row.ChangedAt == "" || receipt.Entry == "" {
					t.Fatalf("setting not recorded and exposed: %+v %+v", receipt, row)
				}
			}
		}
		if !found || reading.Revision != before.Revision+1 {
			t.Fatal("setting missing from the ordinary rules screen", reading.Revision)
		}
	}
}

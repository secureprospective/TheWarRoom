package source_test

import (
	"encoding/json"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/fantasyscore"
	"github.com/secureprospective/TheWarRoom/internal/fantasyscore/source"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

func TestSleeperOwnedQuantities(t *testing.T) {
	var raw source.RawSleeperStats
	body := `{"rush_yac":10,"rec_yar":-2,"rec_air_yd":-3,"pass_air_yd":-4,"rush_btkl":2,"rec_drop":1,"penalty":1,"penalty_yd":5,"idp_sack_yd":0.5,"st_tkl_solo":2,"off_snp":3,"def_snp":4,"st_snp":5,"rec_rz_tgt":99,"rush_tkl_loss":99,"passing_yards":999}`
	if err := json.Unmarshal([]byte(body), &raw); err != nil {
		t.Fatal(err)
	}
	line, err := raw.Convert()
	if err != nil {
		t.Fatal(err)
	}
	want := map[leaguerules.StatID]leaguerules.Decimal{
		"rushing_yards_after_contact": 100000, "receiving_yards_after_catch": -20000,
		"receiving_air_yards": -30000, "passing_air_yards": -40000,
		"rushing_broken_tackle": 20000, "receiving_drop": 10000, "penalty": 10000,
		"penalty_yards": 50000, "defensive_sack_yards": 5000, "special_teams_solo_tackle": 20000,
		"offensive_snap": 30000, "defensive_snap": 40000, "special_teams_snap": 50000,
	}
	if len(line.Quantities) != len(want) {
		t.Fatalf("unexpected stat ownership: %+v", line)
	}
	for id, qty := range want {
		if line.Quantities[id] != qty {
			t.Errorf("%s = %d, want %d", id, line.Quantities[id], qty)
		}
	}
}

// These prove the retained quantities' unrounded value, not player-week attachment (E2b).
func TestSleeperNamedLossContributions(t *testing.T) {
	rules := leaguerules.Default32Scoring()
	rules.RoundStatPoints.Value = false
	for _, tc := range []struct {
		name, body string
		pos        leaguerules.Position
		want       leaguerules.Decimal
	}{
		{"Mausi_2025_aggregate", `{"st_snp":219,"st_tkl_solo":4,"penalty":1}`, leaguerules.LB, 14070},
		{"DJ_Johnson_2025_week_1", `{"def_snp":16,"st_snp":13}`, leaguerules.DE, 1670},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var raw source.RawSleeperStats
			if err := json.Unmarshal([]byte(tc.body), &raw); err != nil {
				t.Fatal(err)
			}
			line, err := raw.Convert()
			if err != nil {
				t.Fatal(err)
			}
			got, _, err := fantasyscore.Score(rules, tc.pos, line)
			if err != nil || got.State != leaguerules.IsKnown || got.Value != tc.want {
				t.Fatalf("points = %+v, want %d (%v)", got, tc.want, err)
			}
		})
	}
}

func TestSleeperRejectsInvalidEvidence(t *testing.T) {
	for _, body := range []string{`null`, `[]`, `{"12944":null}`, `{"12944":{"st_snp":-1}}`, `{"12944":{"idp_sack_yd":1.12345}}`, `{"12944":{"rec_drop":"oops"}}`, `{"12944":{"penalty":{}}}`} {
		if err := (source.RawSleeperWeek{Season: 2025, Week: 1, Body: body}).Validate(); err == nil {
			t.Errorf("accepted %s", body)
		}
	}
	for _, raw := range []source.RawSleeperWeek{{Season: 1919, Week: 1, Body: `{}`}, {Season: 2025, Week: 23, Body: `{}`}} {
		if err := raw.Validate(); err == nil {
			t.Errorf("accepted %+v", raw)
		}
	}
	var missing source.RawSleeperStats
	if _, err := missing.Convert(); err != nil {
		t.Fatalf("missing fields are zero: %v", err)
	}
}

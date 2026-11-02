package source_test

import (
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/fantasyscore/source"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

func TestLiveSleeperScoringFieldMap(t *testing.T) {
	rows, err := source.LiveSleeperLines(source.RawSleeperWeek{Season: 2026, Week: 8, Body: `{"1":{"fgm_0_19":0,"fgm_20_29":1,"fgm_30_39":2,"fgm_40_49":1,"fgm_50_59":2,"fgm_60p":1,"fgmiss_0_19":0,"fgmiss_20_29":1,"fgmiss_30_39":0,"fgmiss_40_49":1,"fgmiss_50_59":0,"fgmiss_60p":1,"pr_td":1,"kr_td":2,"idp_int_td":3,"idp_fum_td":4,"fum_rec_td":5,"pass_rz_att":2,"rec_rz_tgt":3,"rush_rz_att":4,"rush_tkl_loss":5,"pts_ppr":999}}`})
	if err != nil {
		t.Fatal(err)
	}
	q := rows["1"].Quantities
	for id, want := range map[leaguerules.StatID]leaguerules.Decimal{"fg_made_0_39": 30000, "fg_made_60_99": 10000, "fg_missed_0_29": 10000, "fg_missed_30_99": 20000, "punt_return_td": 10000, "kickoff_return_td": 20000, "interception_return_td": 30000, "fumble_recovery_tds": 40000, "offensive_fumble_recovery_td": 50000, "red_zone_pass_attempt": 20000, "red_zone_target": 30000, "red_zone_carry": 40000, "rushing_tfl": 50000} {
		if got, ok := q[id]; !ok || got != want {
			t.Fatalf("%s: %d present %v, want %d", id, got, ok, want)
		}
	}
	if _, ok := q["pts_ppr"]; ok {
		t.Fatal("fantasy total leaked into raw stats")
	}
}

func TestLiveSleeperSparseBandsNeedCompleteEvidence(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		present    bool
		want       leaguerules.Decimal
	}{
		{"reconciled", `{"fgm":2,"fgm_30_39":1,"fgm_50_59":1}`, true, 10000},
		{"partial", `{"fgm":3,"fgm_30_39":1,"fgm_50_59":1}`, false, 0},
		{"total absent", `{"fgm_30_39":1}`, false, 0},
		{"zero", `{"fgm":0}`, true, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rows, err := source.LiveSleeperLines(source.RawSleeperWeek{Season: 2026, Week: 8, Body: `{"1":` + tc.body + `}`})
			if err != nil {
				t.Fatal(err)
			}
			got, ok := rows["1"].Quantities["fg_made_0_39"]
			if ok != tc.present || got != tc.want {
				t.Fatal(rows)
			}
		})
	}
	for _, body := range []string{`{"1":{"fgm_0_19":-1}}`, `{"1":{"fgm_20_29":1.12345}}`} {
		if _, err := source.LiveSleeperLines(source.RawSleeperWeek{Season: 2026, Week: 8, Body: body}); err == nil {
			t.Fatal("accepted invalid partial bands", body)
		}
	}
}

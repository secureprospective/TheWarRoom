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

func TestLiveSleeperAllowsNegativeYardageAverages(t *testing.T) {
	rows, err := source.LiveSleeperLines(source.RawSleeperWeek{Season: 2026, Week: 8,
		Body: `{"1":{"rush_yd":-3,"rush_ypa":-1.5,"rec_ypr":-2,"pass_ypc":-1}}`})
	if err != nil {
		t.Fatal(err)
	}
	for id, want := range map[leaguerules.StatID]leaguerules.Decimal{"rushing_average": -15000, "receiving_average": -20000, "passing_average": -10000} {
		if rows["1"].Quantities[id] != want {
			t.Fatalf("%s: %v", id, rows["1"])
		}
	}
	if _, err := source.LiveSleeperLines(source.RawSleeperWeek{Season: 2026, Week: 8, Body: `{"1":{"rush_att":-1}}`}); err == nil {
		t.Fatal("negative count accepted")
	}
}

// Isaiah Likely, week 10 2026: nine catches, one of 66 yards. MFL scored C20 and C40; the live line had only C40.
func TestLiveSleeperCountsLongReceptions(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		want20     leaguerules.Decimal
		present20  bool
		want40     leaguerules.Decimal
	}{
		{"reconciled bins", `{"rec":9,"rec_5_9":6,"rec_10_19":2,"rec_40p":1,"rec_lng":66}`, 10000, true, 10000},
		{"two long catches", `{"rec":3,"rec_0_4":1,"rec_20_29":1,"rec_30_39":1,"rec_lng":31}`, 20000, true, 0},
		{"no bins, short longest", `{"rec":4,"rec_lng":13}`, 0, true, 0},
		{"no bins, long longest", `{"rec":4,"rec_lng":25}`, 0, false, 0},
	} {
		rows, err := source.LiveSleeperLines(source.RawSleeperWeek{Season: 2026, Week: 10, Body: `{"1":` + tc.body + `}`})
		if err != nil {
			t.Fatal(tc.name, err)
		}
		q := rows["1"].Quantities
		if got, ok := q["long_reception_20"]; ok != tc.present20 || got != tc.want20 {
			t.Errorf("%s: long_reception_20 %d present %v, want %d present %v", tc.name, got, ok, tc.want20, tc.present20)
		}
		if got := q["long_reception_40"]; got != tc.want40 {
			t.Errorf("%s: long_reception_40 %d, want %d", tc.name, got, tc.want40)
		}
	}
}

// Sleeper has no run bins, but a longest run under 20 (or 40) yards proves no R20 (or R40). A longest run of
// 20 or more leaves the count pending: the play-by-play or the final feed must say how many.
func TestLiveSleeperLongestRunProvesNoLongRuns(t *testing.T) {
	for _, tc := range []struct {
		name, body           string
		present20, present40 bool
	}{
		{"short longest", `{"rush_att":12,"rush_yd":41,"rush_lng":9}`, true, true},
		{"mid longest", `{"rush_att":12,"rush_yd":61,"rush_lng":27}`, false, true},
		{"long longest", `{"rush_att":12,"rush_yd":91,"rush_lng":52}`, false, false},
		{"no longest", `{"rush_att":12,"rush_yd":41}`, false, false},
	} {
		rows, err := source.LiveSleeperLines(source.RawSleeperWeek{Season: 2026, Week: 10, Body: `{"1":` + tc.body + `}`})
		if err != nil {
			t.Fatal(tc.name, err)
		}
		q := rows["1"].Quantities
		if got, ok := q["long_rush_20"]; ok != tc.present20 || got != 0 {
			t.Errorf("%s: long_rush_20 %d present %v, want 0 present %v", tc.name, got, ok, tc.present20)
		}
		if got, ok := q["long_rush_40"]; ok != tc.present40 || got != 0 {
			t.Errorf("%s: long_rush_40 %d present %v, want 0 present %v", tc.name, got, ok, tc.present40)
		}
	}
}

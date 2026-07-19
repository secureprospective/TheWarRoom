package leaguestandings

import (
	"context"
	"encoding/json"
	"testing"
)

func TestRawStanding_Validate(t *testing.T) {
	base := RawStanding{
		FranchiseID: "0001",
		H2HW:        "10", H2HL: "3", H2HT: "0",
		AllPlayW: "421", AllPlayL: "106", AllPlayT: "0",
		PF: "1850.5", PA: "1600.2", AvgPF: "142.3", AvgPA: "123.1",
		PP: "1990.0", Pwr: "48.2", AltPwr: "87.0", Salary: "250",
	}

	tests := []struct {
		name    string
		mutate  func(r *RawStanding)
		wantErr bool
	}{
		{"valid", func(*RawStanding) {}, false},
		{"valid empty all-play (league disabled)", func(r *RawStanding) { r.AllPlayW, r.AllPlayL, r.AllPlayT = "", "", "" }, false},
		{"valid decimal pwr", func(r *RawStanding) { r.Pwr = "48.27" }, false},
		{"missing franchise", func(r *RawStanding) { r.FranchiseID = "  " }, true},
		{"non-numeric pf", func(r *RawStanding) { r.PF = "1850pts" }, true},
		{"non-numeric all_play_w", func(r *RawStanding) { r.AllPlayW = "many" }, true},
		{"non-numeric altpwr", func(r *RawStanding) { r.AltPwr = "N/A" }, true},
		{"accept currency salary", func(r *RawStanding) { r.Salary = "$120.72" }, false},
		{"accept thousands-separated pf", func(r *RawStanding) { r.PF = "1,850.50" }, false},
		{"reject NaN pf", func(r *RawStanding) { r.PF = "NaN" }, true},
		{"reject Inf pwr", func(r *RawStanding) { r.Pwr = "Inf" }, true},
		{"reject +Inf pf", func(r *RawStanding) { r.PF = "+Inf" }, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := base
			tt.mutate(&r)
			if err := r.Validate(); (err != nil) != tt.wantErr {
				t.Fatalf("Validate() error = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}

func TestFlatten_MapsFields(t *testing.T) {
	env := standingsEnvelope{}
	env.LeagueStandings.Franchise = []franchiseStanding{
		{ID: "0001", H2HW: "10", H2HL: "3", AllPlayW: "421", PF: "1850.5", Pwr: "48.2", AltPwr: "87.0", Salary: "250"},
		{ID: "0002", H2HW: "12", H2HL: "1", AllPlayW: "438", PF: "1990.1", Pwr: "51.0", AltPwr: "86.0", Salary: "260"},
	}

	got, err := flatten(context.Background(), env)
	if err != nil {
		t.Fatalf("flatten: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d records, want 2", len(got))
	}
	if got[0].FranchiseID != "0001" || got[0].AllPlayW != "421" || got[0].Pwr != "48.2" {
		t.Errorf("record 0 mapped wrong: %+v", got[0])
	}
	if got[1].AltPwr != "86.0" || got[1].Salary != "260" {
		t.Errorf("record 1 mapped wrong: %+v", got[1])
	}
}

func TestFlatten_MalformedFailsLoud(t *testing.T) {
	env := standingsEnvelope{}
	env.LeagueStandings.Franchise = []franchiseStanding{{ID: "0001", PF: "not-a-number"}}
	if _, err := flatten(context.Background(), env); err == nil {
		t.Fatal("flatten should error on a non-numeric pf, got nil")
	}
}

// TestDecode_SingleElementCollapse guards the MFL bracket-strip quirk: a standings
// export with exactly one franchise arrives as a BARE OBJECT, not a one-element
// array. MFLList must still decode it as a single-element slice.
func TestDecode_SingleElementCollapse(t *testing.T) {
	const body = `{"leagueStandings":{"franchise":{"id":"0001","h2hw":"10","pf":"1850.5","all_play_w":"421"}}}`
	var env standingsEnvelope
	if err := json.Unmarshal([]byte(body), &env); err != nil {
		t.Fatalf("decode single-franchise standings: %v", err)
	}
	got, err := flatten(context.Background(), env)
	if err != nil {
		t.Fatalf("flatten: %v", err)
	}
	if len(got) != 1 || got[0].FranchiseID != "0001" || got[0].AllPlayW != "421" {
		t.Fatalf("single-element collapse not handled: %+v", got)
	}
}

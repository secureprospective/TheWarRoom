package source

import (
	"fmt"
	"strings"
	"testing"
)

func fullSnapCSV() string {
	s := "game_id,pfr_game_id,season,game_type,week,pfr_player_id,position,team,offense_snaps,offense_pct,defense_snaps,defense_pct,st_snaps,st_pct\n"
	for _, team := range []string{"BUF", "NYJ"} {
		for i := 0; i < 11; i++ {
			s += fmt.Sprintf("2026_01_BUF_NYJ,202609100nyj,2026,REG,1,%s%d,LB,%s,60,1,60,1,20,1\n", team, i, team)
		}
	}
	return s
}

func TestCompleteSnapsAdmitOnlyAbsentRosterMembers(t *testing.T) {
	roster := []RawAbsenceProof{{GSIS: "00-0030001", PFR: "Absent00", Season: "2026", Week: "1", Team: "BUF", Position: "LB", Evidence: "complete snaps + pbp, no snap"}}
	for _, tc := range []struct {
		name, body string
		want       int
	}{
		{"complete", fullSnapCSV(), 1},
		{"partial", strings.Replace(fullSnapCSV(), "2026_01_BUF_NYJ,202609100nyj,2026,REG,1,BUF0,LB,BUF,60,1,60,1,20,1\n", "", 1), 0},
		{"missing opponent", strings.ReplaceAll(fullSnapCSV(), ",NYJ,", ",BUF,"), 0},
		{"missing", strings.Split(fullSnapCSV(), "\n")[0] + "\n", 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			snaps, err := ReadSnaps(strings.NewReader(tc.body))
			if err != nil {
				t.Fatal(err)
			}
			proofs := NonParticipationProofs(snaps, roster)
			p := creditPlay()
			p.EndGame = "END GAME"
			p.GameSeconds = "0"
			got, err := AddProvenZeros(nil, []RawPlay{p}, proofs)
			if err != nil || len(got) != tc.want {
				t.Fatalf("zeros=%+v err=%v", got, err)
			}
			if len(got) > 0 && got[0].Evidence != "complete snaps + pbp, no snap" {
				t.Fatal(got)
			}
		})
	}
	snaps, err := ReadSnaps(strings.NewReader(fullSnapCSV()))
	if err != nil {
		t.Fatal(err)
	}
	for _, mutate := range []func(*RawAbsenceProof){func(r *RawAbsenceProof) { r.PFR = "BUF0" }, func(r *RawAbsenceProof) { r.GSIS = "" }, func(r *RawAbsenceProof) { r.Team = "MIA" }, func(r *RawAbsenceProof) { r.Week = "2" }} {
		r := roster[0]
		mutate(&r)
		if got := NonParticipationProofs(snaps, []RawAbsenceProof{r}); len(got) != 0 {
			t.Fatalf("unsupported proof: %+v", got)
		}
	}
}

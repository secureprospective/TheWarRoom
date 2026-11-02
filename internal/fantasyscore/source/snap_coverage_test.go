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

func TestRoundedSnapSharesCannotCertifyAnAmbiguousPartialFile(t *testing.T) {
	var rows []RawSnap
	for i := 0; i < 28; i++ {
		n, share := 60, 0.4
		if i == 27 {
			n, share = 19, 0.13
		}
		rows = append(rows, RawSnap{Proof: RawAbsenceProof{PFR: fmt.Sprint("Player", i), Season: "2026", Week: "1", GameID: "2026_01_BUF_NYJ", Team: "BUF", Evidence: "positive snaps; no play-by-play stat events"}, Date: "20260910", Counts: [3]int{n, n, n}, Shares: [3]float64{share, share, share}})
	}
	// The retained counts fit 149 snaps, but the rounded shares also fit 150.
	// Eleven missing player-snaps cannot become an absence proof.
	if completeSnapTeam(rows) {
		t.Fatal("ambiguous rounded shares certified a partial file")
	}
}

// snapCSV writes a complete eleven-man file whose BUF special-teams rows are replaced by the
// counts and shares given, so the unit can be made over-manned or short of a snap.
func snapCSV(st [][2]float64) string {
	s := "game_id,pfr_game_id,season,game_type,week,pfr_player_id,position,team,offense_snaps,offense_pct,defense_snaps,defense_pct,st_snaps,st_pct\n"
	row := "2026_01_BUF_NYJ,202609100nyj,2026,REG,1,%s,LB,%s,%d,%g,%d,%g,%g,%g\n"
	for i := 0; i < 11; i++ {
		s += fmt.Sprintf(row, fmt.Sprint("NYJ", i), "NYJ", 60, 1.0, 60, 1.0, 20.0, 1.0)
	}
	for i, unit := range st {
		offense, share := 60, 1.0
		if i >= 11 {
			offense, share = 0, 0
		}
		s += fmt.Sprintf(row, fmt.Sprint("BUF", i), "BUF", offense, share, offense, share, unit[0], unit[1])
	}
	return s
}

func elevenSpecialTeams(count float64, share float64) [][2]float64 {
	out := make([][2]float64, 11)
	for i := range out {
		out[i] = [2]float64{count, share}
	}
	return out
}

func TestSnapFileShortOfAPlayIsNotCompleteButAnOverMannedOneIs(t *testing.T) {
	short := elevenSpecialTeams(20, 1)
	short[10] = [2]float64{19, 0.95}
	for _, tc := range []struct {
		name    string
		st      [][2]float64
		admits  bool
		because string
	}{
		{"over-manned special teams", append(elevenSpecialTeams(20, 1), [2]float64{2, 0.1}), true,
			"222 counts over 20 plays cannot come from leaving a player out"},
		{"short of one snap", short, false,
			"219 counts over 20 plays leaves a snap unaccounted for"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			snaps, err := ReadSnaps(strings.NewReader(snapCSV(tc.st)))
			if err != nil {
				t.Fatal(err)
			}
			roster := []RawAbsenceProof{{GSIS: "00-0030001", PFR: "Absent00", Season: "2026", Week: "1",
				Team: "BUF", Position: "LB", Evidence: "complete snaps + pbp, no snap"}}
			got := NonParticipationProofs(snaps, roster)
			if (len(got) == 1) != tc.admits {
				t.Fatalf("%s: proofs=%+v", tc.because, got)
			}
		})
	}
}

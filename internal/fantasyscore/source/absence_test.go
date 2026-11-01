package source

import (
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/fantasyscore"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

func TestProvenZeroRequiresFinalCoverageAndPositiveEvidence(t *testing.T) {
	p := creditPlay()
	p.EndGame = "END GAME"
	p.GameSeconds = "0"
	proof := RawAbsenceProof{GSIS: "00-0030001", Season: "2026", Week: "1", GameID: p.GameID, Position: "LB", Evidence: "positive snaps; no play-by-play stat events"}
	got, err := AddProvenZeros(nil, []RawPlay{p}, []RawAbsenceProof{proof})
	if err != nil || len(got) != 1 || got[0].Evidence != proof.Evidence {
		t.Fatalf("%+v %v", got, err)
	}
	points, _, err := fantasyscore.Score(leaguerules.Default32Scoring(), leaguerules.LB, got[0].Line)
	if err != nil || points.State != leaguerules.IsKnown || points.Value != 0 {
		t.Fatalf("%+v %v", points, err)
	}
	for _, mutate := range []func(*RawPlay){
		func(p *RawPlay) { p.EndGame = "" }, func(p *RawPlay) { p.GameSeconds = "1" },
		func(p *RawPlay) { delete(p.Credits, "sack_player_id") },
		func(p *RawPlay) { delete(p.Credits, "lateral_rusher_player_id") },
		func(p *RawPlay) { p.Credits["solo_tackle_1_player_id"] = "00-0030001" },
		func(p *RawPlay) { p.Credits["lateral_receiver_player_id"] = "00-0030001" },
	} {
		q := creditPlay()
		q.EndGame = "END GAME"
		q.GameSeconds = "0"
		mutate(&q)
		got, err = AddProvenZeros(nil, []RawPlay{q}, []RawAbsenceProof{proof})
		if err != nil || len(got) != 0 {
			t.Fatalf("unsupported zero: %+v %v", got, err)
		}
	}
	proof.GSIS = ""
	proof.PFR = "unknown"
	got, err = AddProvenZeros(nil, []RawPlay{p}, []RawAbsenceProof{proof})
	if err != nil || len(got) != 0 {
		t.Fatalf("identity gap became zero: %+v %v", got, err)
	}
}

func TestOutProofIsExactWeekAndStatsWin(t *testing.T) {
	p := creditPlay()
	p.EndGame = "END GAME"
	p.GameSeconds = "0"
	proofs, err := ReadOutProofs(strings.NewReader("season,game_type,team,week,gsis_id,position,report_status\n2026,REG,BUF,1,00-0030001,LB,Out\n2026,REG,BUF,1,00-0030002,LB,Questionable\n"))
	if err != nil || len(proofs) != 1 {
		t.Fatalf("%+v %v", proofs, err)
	}
	got, err := AddProvenZeros(nil, []RawPlay{p}, proofs)
	if err != nil || len(got) != 1 {
		t.Fatalf("%+v %v", got, err)
	}
	proofs[0].Week = "2"
	got, err = AddProvenZeros(nil, []RawPlay{p}, proofs)
	if err != nil || len(got) != 0 {
		t.Fatalf("wrong-week Out admitted: %+v %v", got, err)
	}
	proofs[0].Week = "1"
	p.Credits["solo_tackle_1_player_id"] = "00-0030001"
	rows, err := MissingPlayWeeks(nil, []RawPlay{p})
	if err != nil {
		t.Fatal(err)
	}
	got, err = AddProvenZeros(rows, []RawPlay{p}, proofs)
	if err != nil || len(got) != 1 || got[0].Line.Quantities["tackle"] != 10000 || got[0].Provenance != "nflverse play-by-play" {
		t.Fatalf("%+v %v", got, err)
	}
}

func TestSnapProofBoundary(t *testing.T) {
	for _, tc := range []struct {
		count string
		want  int
		bad   bool
	}{{"1", 1, false}, {"0", 0, false}, {"-1", 0, true}, {"bad", 0, true}} {
		proofs, err := ReadSnapProofs(strings.NewReader("season,game_type,team,week,game_id,pfr_player_id,position,offense_snaps,defense_snaps,st_snaps\n2026,REG,BUF,1,2026_01_BUF_NYJ,Player01,LB,0," + tc.count + ",0\n"))
		if (err != nil) != tc.bad || len(proofs) != tc.want {
			t.Fatalf("%s %+v %v", tc.count, proofs, err)
		}
	}
}

func TestZeroStatRulesStillApply(t *testing.T) {
	p := creditPlay()
	p.EndGame = "END GAME"
	p.GameSeconds = "0"
	proof := RawAbsenceProof{GSIS: "00-0030001", Season: "2026", Week: "1", GameID: p.GameID, Position: "LB", Evidence: "positive snaps; no play-by-play stat events"}
	got, err := AddProvenZeros(nil, []RawPlay{p}, []RawAbsenceProof{proof})
	if err != nil {
		t.Fatal(err)
	}
	rules := leaguerules.Default32Scoring()
	rules.Rules = leaguerules.Known(map[leaguerules.StatID][]leaguerules.RangeRule{"tackle": {{Flat: true, Points: 20000, Range: leaguerules.StatRange{Min: 0, Max: 0}}}})
	points, _, err := fantasyscore.Score(rules, leaguerules.LB, got[0].Line)
	if err != nil || points.State != leaguerules.IsKnown || points.Value != 20000 {
		t.Fatalf("zero-stat award lost: %+v %v", points, err)
	}
}

func TestOvertimeEndGameDoesNotNeedAnExpiredClock(t *testing.T) {
	p := creditPlay()
	p.EndGame = "END GAME"
	p.GameSeconds = "323"
	p.GameHalf = "Overtime"
	proof := RawAbsenceProof{GSIS: "00-0030001", Season: "2026", Week: "1", GameID: p.GameID, Position: "LB", Evidence: "positive snaps; no play-by-play stat events"}
	got, err := AddProvenZeros(nil, []RawPlay{p}, []RawAbsenceProof{proof})
	if err != nil || len(got) != 1 {
		t.Fatalf("overtime final stayed unavailable: %+v %v", got, err)
	}
}

func TestScheduleByeNeedsIdentityAndDoesNotOverrideEvents(t *testing.T) {
	proof := RawAbsenceProof{GSIS: "00-0030001", Season: "2026", Week: "1", Team: "MIA", Position: "LB", Evidence: "team bye per schedule"}
	got, err := AddProvenZeros(nil, nil, []RawAbsenceProof{proof})
	if err != nil || len(got) != 1 || got[0].Evidence != proof.Evidence {
		t.Fatalf("bye unavailable: %+v %v", got, err)
	}
	p := creditPlay()
	p.Credits["solo_tackle_1_player_id"] = proof.GSIS
	got, err = AddProvenZeros(nil, []RawPlay{p}, []RawAbsenceProof{proof})
	if err != nil || len(got) != 0 {
		t.Fatalf("event replaced by bye: %+v %v", got, err)
	}
	proof.GSIS = ""
	proof.PFR = "Unlinked00"
	got, err = AddProvenZeros(nil, nil, []RawAbsenceProof{proof})
	if err != nil || len(got) != 0 {
		t.Fatalf("unlinked bye became zero: %+v %v", got, err)
	}
}

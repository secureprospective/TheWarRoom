package history

import (
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/fantasyscore/source"
)

func TestDepthMembershipIsDatedAndDoesNotRetainDepartedPlayers(t *testing.T) {
	cutoff := time.Date(2026, 9, 10, 0, 0, 0, 0, time.UTC)
	rows := []source.RawDepth{
		{DT: "2026-09-08T12:00:00Z", Team: "BUF", Name: "Departed"},
		{DT: "2026-09-09T12:00:00Z", Team: "NYJ", Name: "Current"},
		{DT: "2026-09-11T12:00:00Z", Team: "MIA", Name: "Future"},
	}
	got := depthAt(rows, cutoff)
	if len(got) != 1 || got[0].Name != "Current" {
		t.Fatalf("wrong historical affiliation: %+v", got)
	}
	if got := depthAt(rows, cutoff.AddDate(0, 0, 10)); len(got) != 0 {
		t.Fatalf("stale roster admitted: %+v", got)
	}
	if got := depthAt(rows, cutoff.AddDate(0, 0, -10)); len(got) != 0 {
		t.Fatalf("future roster admitted: %+v", got)
	}
}

func TestRosterByesRequireHistoricalTeamAndPositiveSchedule(t *testing.T) {
	gsis := "00-0030001"
	depth := []source.RawDepth{{DT: "2026-10-07T12:00:00Z", Team: "NE", GSIS: &gsis, Position: "LB"}, {DT: "2026-10-09T12:00:00Z", Team: "BUF", GSIS: &gsis, Position: "LB"}}
	snaps := []source.RawSnap{{Proof: source.RawAbsenceProof{Week: "5", Team: "NYJ"}, Date: "20261008"}}
	for _, tc := range []struct {
		name string
		byes []source.RawBye
		want int
	}{
		{"dated team", []source.RawBye{{Team: "NEP", Week: "5"}}, 1},
		{"future team", []source.RawBye{{Team: "BUF", Week: "5"}}, 0},
		{"no schedule", nil, 0},
		{"different week", []source.RawBye{{Team: "NEP", Week: "6"}}, 0},
		{"team playing", []source.RawBye{{Team: "NYJ", Week: "5"}}, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := rosterByes(2026, depth, snaps, tc.byes)
			if len(got) != tc.want {
				t.Fatalf("unsupported bye: %+v", got)
			}
		})
	}
	depth[0].GSIS = nil
	if got := rosterByes(2026, depth, snaps, []source.RawBye{{Team: "NEP", Week: "5"}}); len(got) != 0 {
		t.Fatal("unknown identity zeroed", got)
	}
}

func TestRosterByeLeavesLegacyNonGSISIdentifiersPending(t *testing.T) {
	id := "THO581952"
	depth := []source.RawDepth{{DT: "2026-10-07T12:00:00Z", Team: "NE", GSIS: &id, Position: "LB"}}
	snaps := []source.RawSnap{{Proof: source.RawAbsenceProof{Week: "5", Team: "NYJ"}, Date: "20261008"}}
	if got := rosterByes(2026, depth, snaps, []source.RawBye{{Team: "NEP", Week: "5"}}); len(got) != 0 {
		t.Fatal("legacy identity became zero", got)
	}
}

// unrosteredDepth lists two teams across two weeks: one player stays on NE both weeks, the
// player under test is on NE in week five only, and NYJ keeps a listing in both snapshots so
// each is covered. The second snapshot is the one that decides week six.
func unrosteredDepth(absent string) []source.RawDepth {
	stay, other := "00-0030009", "00-0030008"
	var out []source.RawDepth
	for _, dt := range []string{"2026-10-07T12:00:00Z", "2026-10-14T12:00:00Z"} {
		out = append(out, source.RawDepth{DT: dt, Team: "NE", GSIS: &stay, Position: "LB"},
			source.RawDepth{DT: dt, Team: "NYJ", GSIS: &other, Position: "LB"})
	}
	gone := absent
	out = append(out, source.RawDepth{DT: "2026-10-07T12:00:00Z", Team: "NE", GSIS: &gone, Position: "LB"})
	return out
}

func unrosteredSnaps() []source.RawSnap {
	return []source.RawSnap{
		{Proof: source.RawAbsenceProof{Week: "5", Team: "NYJ", PFR: "PlayA00"}, Date: "20261008"},
		{Proof: source.RawAbsenceProof{Week: "6", Team: "NYJ", PFR: "PlayA00"}, Date: "20261015"},
	}
}

func weeksUnrostered(proofs []source.RawAbsenceProof, gsis string) []string {
	var out []string
	for _, p := range proofs {
		if p.GSIS == gsis && p.Evidence == source.UnrosteredEvidence {
			out = append(out, p.Week)
		}
	}
	return out
}

func TestAPlayerNoTeamListsThatWeekScoresZeroOnlyUnderCompleteMembership(t *testing.T) {
	const gsis = "00-0030001"
	got := unrosteredWeeks(2026, unrosteredDepth(gsis), unrosteredSnaps(), nil)
	if weeks := weeksUnrostered(got, gsis); len(weeks) != 1 || weeks[0] != "6" {
		t.Fatalf("week six membership not read: %+v", weeks)
	}
	for _, p := range got {
		if p.Team != "" || p.GameID != "" || p.Evidence != source.UnrosteredEvidence {
			t.Fatalf("a teamless proof named a team or game: %+v", p)
		}
	}
	short := unrosteredDepth(gsis)
	short = append(short, source.RawDepth{DT: "2026-10-07T12:00:00Z", Team: "MIA", GSIS: &[]string{"00-0030007"}[0], Position: "LB"})
	if weeks := weeksUnrostered(unrosteredWeeks(2026, short, unrosteredSnaps(), nil), gsis); len(weeks) != 0 {
		t.Fatalf("a snapshot missing a team certified no roster: %+v", weeks)
	}
	bridge := map[string]string{"Snapped00": gsis}
	snapped := append(unrosteredSnaps(), source.RawSnap{
		Proof: source.RawAbsenceProof{Week: "6", Team: "NYJ", PFR: "Snapped00"}, Date: "20261015"})
	if weeks := weeksUnrostered(unrosteredWeeks(2026, unrosteredDepth(gsis), snapped, bridge), gsis); len(weeks) != 0 {
		t.Fatalf("a player who took a snap was called unrostered: %+v", weeks)
	}
	if weeks := weeksUnrostered(unrosteredWeeks(2026, unrosteredDepth("THO581952"), unrosteredSnaps(), nil), "THO581952"); len(weeks) != 0 {
		t.Fatalf("legacy identity became a zero: %+v", weeks)
	}
}

func TestUnrosteredZeroNeedsNoGameAndRefusesAPlayerWithAnEvent(t *testing.T) {
	const gsis = "00-0030001"
	proof := source.RawAbsenceProof{GSIS: gsis, Season: "2026", Week: "6", Evidence: source.UnrosteredEvidence}
	if err := proof.Validate(); err != nil {
		t.Fatalf("a teamless week proof was refused: %v", err)
	}
	named := proof
	named.Team = "NE"
	if named.Validate() == nil {
		t.Fatal("a player on no roster was given a team")
	}
	got, err := source.AddProvenZeros(nil, nil, []source.RawAbsenceProof{proof})
	if err != nil || len(got) != 1 || got[0].Evidence != source.UnrosteredEvidence || got[0].GameID != "" {
		t.Fatalf("no-roster zero not admitted without a game: %+v %v", got, err)
	}
	play := source.RawPlay{Season: "2026", Week: "6", SeasonType: "REG", GameID: "2026_06_NE_NYJ",
		PlayID: "1", Rusher: gsis, Credits: source.RawPlayerCredits{}}
	if got, err := source.AddProvenZeros(nil, []source.RawPlay{play}, []source.RawAbsenceProof{proof}); err != nil || len(got) != 0 {
		t.Fatalf("a player with an event that week was zeroed: %+v %v", got, err)
	}
}

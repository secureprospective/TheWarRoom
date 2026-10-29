package season

import (
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/ingestion/leaguestandings"
)

func TestFranchiseReportNamesPickWeightAndLeavesUnreadableBlank(t *testing.T) {
	res := PowerRankingsResult{OK: true, View: ViewFranchise, PickWeight: .6, AgeWeight: .5, AggMode: "lineup", ModelRunID: 5, Rows: []PowerRow{{FranchiseID: "a", PickValue: 20, HasPicks: true}, {FranchiseID: "b"}, {FranchiseID: "c", HasPicks: true}}}
	v := View{Names: map[string]string{}, in: Snapshot{Zone: time.UTC, Held: Held{Standings: Feed[[]leaguestandings.RawStanding]{Value: []leaguestandings.RawStanding{{FranchiseID: "a"}}, FetchedAt: time.Now()}, Power: map[string]PowerRankingsResult{ViewFranchise: res}}}}
	r := v.Power(ViewFranchise)
	if len(r.Columns) != 7 || r.Columns[4] != "Picks" || r.Rows[0].Cells[4] != "20.0" || r.Rows[1].Cells[4] != "" || r.Rows[2].Cells[4] != "0.0" {
		t.Fatal(r)
	}
	if !strings.Contains(r.How, "weight 0.6 (Pick Value v2)") || !strings.Contains(r.How, "next two drafts") {
		t.Fatal(r.How)
	}
	res.Error = "history: scan run: context deadline exceeded"
	v.in.Held.Power[ViewFranchise] = res
	stale := v.Power(ViewFranchise)
	if len(stale.Rows) != len(r.Rows) || stale.Note != "Power rankings could not be updated. Try refreshing again." {
		t.Fatal(stale)
	}
	res.OK = false
	v.in.Held.Power[ViewFranchise] = res
	if failed := v.Power(ViewFranchise); strings.Contains(failed.Note, "history:") || len(failed.Rows) != 0 {
		t.Fatal(failed)
	}
	if strings.Contains(powerHow(PowerRankingsResult{View: ViewSeason}), "Pick Value") {
		t.Fatal("this season counted picks")
	}

}

func TestPowerReportNamesReducedAndUnavailableInputs(t *testing.T) {
	why := "league points for 2024 are not ready"
	for _, view := range []string{ViewSeason, ViewFranchise} {
		t.Run(view, func(t *testing.T) {
			res := PowerRankingsResult{OK: true, View: view, Rows: []PowerRow{
				{FranchiseID: "a", RosterValue: 10, HasPicks: true, PickValue: 2, ReducedInputs: true, ValueReason: why},
				{FranchiseID: "b", Unavailable: true, ReducedInputs: true, ValueReason: "No usable season: " + why},
			}}
			v := View{Names: map[string]string{}, in: Snapshot{Zone: time.UTC, Held: Held{
				Standings: Feed[[]leaguestandings.RawStanding]{FetchedAt: time.Now()}, Power: map[string]PowerRankingsResult{view: res},
			}}}
			r := v.Power(view)
			if !strings.Contains(strings.Join(r.Rows[0].Cells, " "), "reduced inputs") || !strings.Contains(strings.Join(r.Rows[1].Cells, " "), "Unavailable") {
				t.Fatal(r.Rows)
			}
			for _, row := range r.Rows {
				if !strings.Contains(strings.Join(row.Titles, " "), why) {
					t.Fatal("lost reason", row)
				}
			}
		})
	}
}

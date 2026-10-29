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
	if strings.Contains(powerHow(PowerRankingsResult{View: ViewSeason}), "Pick Value") {
		t.Fatal("this season counted picks")
	}

}

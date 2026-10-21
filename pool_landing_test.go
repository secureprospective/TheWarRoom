package main

import (
	"slices"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/ingestion/players"
	"github.com/secureprospective/TheWarRoom/internal/normalize"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
)

func TestPoolLandingValuesAndSorting(t *testing.T) {
	t.Parallel()
	rows := []PoolRow{{ID: "missing", Name: "A"}, {ID: "zero", Name: "B"}, {ID: "best", Name: "Z"}}
	assets := map[string]AssetValue{"best": {Now: AssetCell{Text: "4.00 PPG", Held: true}, NowValue: 4,
		Dynasty: AssetCell{Text: "0.12 wins", Held: true}, DynastyValue: 0.12}}
	for i := range rows {
		valuePoolRow(&rows[i], map[string]float64{"zero": 0, "best": 12}, assets)
	}
	sortPool(rows, RawPoolScope{Sort: "engine", Desc: true})
	if !slices.Equal([]string{rows[0].ID, rows[1].ID, rows[2].ID}, []string{"best", "zero", "missing"}) {
		t.Fatal(rows)
	}
	if rows[2].Engine != nil || rows[2].EngineText != "Not valued" || rows[1].Engine == nil {
		t.Fatal("missing engine value turned into zero", rows)
	}
	if rows[0].LabText != "0.12 wins" || rows[0].PPGText != "4.00 PPG" || *rows[0].Engine != 12 {
		t.Fatal("engine, league PPG and Lab must stay separate", rows[0])
	}
	for _, key := range []string{"engine", "ppg", "lab"} {
		sortPool(rows, RawPoolScope{Sort: key})
		if poolNumber(rows[len(rows)-1], key) != nil {
			t.Fatal("unvalued must sort last", key, rows)
		}
	}
}

func TestPoolLandingNoTeamToggle(t *testing.T) {
	t.Parallel()
	a, _ := poolApp(t)
	lookup, err := normalize.NewLookup([]players.RawPlayer{
		{ID: "13593", Name: "A", Position: "WR", Team: "FA"},
		{ID: "13594", Name: "B", Position: "QB", Team: "DAL"},
	})
	if err != nil {
		t.Fatal(err)
	}
	ids := []playerid.PlayerID{}
	for _, raw := range []string{"13593", "13594"} {
		id, err := playerid.New(raw)
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, id)
	}
	for _, show := range []bool{false, true} {
		got, _ := a.poolRows(a.core.Ctx, ids, lookup, RawPoolScope{Sort: "engine", Desc: true, ShowNoTeam: show})
		want := 1
		if show {
			want = 2
		}
		if len(got.Rows) != want {
			t.Fatal(show, got.Rows)
		}
		if !show && got.Rows[0].Team != "DAL" {
			t.Fatal(got.Rows)
		}
		if !strings.Contains(got.EngineNote, "unavailable") {
			t.Fatal(got.EngineNote)
		}
	}
	if !noNFLTeam("") || noNFLTeam("DAL") {
		t.Fatal("NFL team classification")
	}
}

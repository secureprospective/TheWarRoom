package offers

import (
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/players"
	"github.com/secureprospective/TheWarRoom/internal/normalize"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
)

func TestPoolLandingValuesAndSorting(t *testing.T) {
	t.Parallel()
	rows := []PoolRow{{ID: "missing", Name: "A"}, {ID: "zero", Name: "B"}, {ID: "best", Name: "Z"}}
	assets := map[string]AssetValue{"best": {NowText: "4.00 PPG", NowHeld: true, NowValue: 4,
		DynastyText: "0.12 wins", DynastyHeld: true, DynastyValue: 0.12}}
	for i := range rows {
		valuePoolRow(&rows[i], map[string]float64{"zero": 0, "best": 12}, assets)
	}
	SortPool(rows, RawPoolScope{Sort: "engine", Desc: true})
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
		SortPool(rows, RawPoolScope{Sort: key})
		if PoolNumber(rows[len(rows)-1], key) != nil {
			t.Fatal("unvalued must sort last", key, rows)
		}
	}
}

func TestPoolLandingNoTeamToggle(t *testing.T) {
	t.Parallel()
	a := bidTestApp(t)
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

func TestPoolPlayerSharesHeldFacts(t *testing.T) {
	t.Parallel()
	id, err := playerid.New("0042")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 11, 5, 0, 0, 0, 0, time.UTC)
	f := normalize.PlayerFacts{Name: "Held receiver", Position: domain.PosWR, Team: "BUF", HasBirthdate: true,
		Birthdate: time.Date(2000, 11, 6, 0, 0, 0, 0, time.UTC).Unix()}
	got := PoolPlayer(id, f, 2026, now, map[string]float64{"0042": 12.75}, map[string]AssetValue{"0042": {String: "WR2", Points: "87.5"}})
	if got.ID != "0042" || got.Age != "25" || got.String != "WR2" || got.Points != "87.5" || got.EngineText != "12.75 points/game" {
		t.Fatal(got)
	}
	absent := PoolPlayer(id, normalize.PlayerFacts{}, 2026, now, nil, nil)
	if absent.Age != "—" || absent.String != "—" || absent.Points != "—" || absent.Engine != nil {
		t.Fatal(absent)
	}
}

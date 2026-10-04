package assembly

import (
	"math"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/model"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
)

func TestBuildCollegeReadsSharesAndBreakoutsFromHistory(t *testing.T) {
	born := time.Date(2001, 9, 1, 0, 0, 0, 0, time.UTC)
	players := map[string]*model.Player{
		"0101": {Birth: born, College: map[int]map[string]float64{ // WR: 15% as a sophomore, 30% as a junior
			2021: {"receiving_yards": 300, "team_receiving_yards": 2000},
			2022: {"receiving_yards": 900, "team_receiving_yards": 3000},
		}},
		"0102": {Birth: born, College: map[int]map[string]float64{ // RB: 0.7·rushing + 0.3·receiving
			2022: {"rushing_yards": 500, "team_rushing_yards": 2000, "receiving_yards": 100, "team_receiving_yards": 1000},
		}},
		"0103": {Birth: born, College: map[int]map[string]float64{ // DE: mean of TFL and sack shares
			2022: {"tackles_for_loss": 10, "team_tackles_for_loss": 50, "sacks": 4, "team_sacks": 40},
		}},
		"0104": {College: map[int]map[string]float64{2022: {"passing_yards": 3000}}}, // QB: no share
	}
	pos := fakePosLookup{"0101": domain.PosWR, "0102": domain.PosRB, "0103": domain.PosDE, "0104": domain.PosQB}
	got := BuildCollege(players, 2022, []int{2020, 2021, 2022}, []string{"0101", "0102", "0103", "0104", "9999"}, pos)

	id := func(s string) playerid.PlayerID { p, _ := playerid.New(s); return p }
	for mfl, want := range map[string]float64{"0101": 0.30, "0102": 0.7*0.25 + 0.3*0.10, "0103": (0.20 + 0.10) / 2} {
		if v, ok := got.Share[id(mfl)]; !ok || math.Abs(v-want) > 1e-12 {
			t.Errorf("share %s = %v (%v), want %v", mfl, v, ok, want)
		}
	}
	if _, ok := got.Share[id("0104")]; ok {
		t.Error("a QB has no college production share")
	}
	if age := got.Breakout[id("0101")]; math.Abs(age-21) > 0.01 {
		t.Errorf("WR broke out at 30%% in 2022, age 21: got %v", age)
	}
	if age := got.Breakout[id("0103")]; math.Abs(age-21) > 0.01 {
		t.Errorf("DE crossed the IDP line (0.15 ≥ 0.12) in 2022: got %v", age)
	}
	if age := got.Breakout[id("0102")]; math.Abs(age-21) > 0.01 {
		t.Errorf("RB breaks out on rushing share alone (0.25 ≥ 0.20) in 2022: got %v", age)
	}
	if _, ok := got.Breakout[id("0104")]; ok {
		t.Error("a QB never breaks out")
	}
}

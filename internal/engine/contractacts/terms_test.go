package contractacts

import (
	"slices"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/domain"
)

func TestTenderAndResignTables(t *testing.T) {
	want := []domain.Money{60 * Million / 100, Million, 15 * Million / 10, 25 * Million / 10, 5 * Million, 65 * Million / 10}
	for i, price := range want {
		got, err := Tender(0, i+1)
		if err != nil || got != price {
			t.Fatalf("level %d: %d %v want %d", i+1, got, err, price)
		}
	}
	for _, tc := range []struct {
		prior domain.Money
		level int
		want  domain.Money
	}{
		{54 * Million / 100, 1, 60 * Million / 100}, {55 * Million / 100, 1, 61 * Million / 100}, {10 * Million, 6, 11 * Million},
	} {
		got, err := Tender(tc.prior, tc.level)
		if err != nil || got != tc.want {
			t.Errorf("tender %+v: %d %v", tc, got, err)
		}
	}
	for _, years := range []int{1, 2, 3, 4} {
		got, err := Resign(Million, years)
		want := Million + domain.Money(years-1)*15*Million/100
		if err != nil || got != want {
			t.Fatalf("%d year re-sign: %d %v want %d", years, got, err, want)
		}
	}
	for _, level := range []int{0, 7} {
		if _, err := Tender(Million, level); err == nil {
			t.Error("invalid tender accepted")
		}
	}
	for _, years := range []int{0, 5} {
		if _, err := Resign(Million, years); err == nil {
			t.Error("invalid re-sign accepted")
		}
	}
	got, err := Resign(123_500_000, 2)
	if err != nil || got != 142_000_000 {
		t.Fatalf("snap after percentage: %d %v", got, err)
	}
}

func TestTagConditionalFloorAndConsecutiveLimit(t *testing.T) {
	pool := []domain.Money{10 * Million, 8 * Million, 6 * Million, 4 * Million, 2 * Million, 99}
	for _, tc := range []struct {
		previous, first domain.Money
		count           int
		want            domain.Money
		valid           bool
	}{
		{5 * Million, 0, 0, 6 * Million, true},
		{6 * Million, 0, 0, 6 * Million, true},
		{6*Million + 1, 0, 0, 720_000_000, true},
		{7 * Million, 0, 0, 840_000_000, true},
		{0, 5 * Million, 1, 6 * Million, true},
		{0, 0, 1, 0, false}, {0, 5 * Million, 2, 0, false},
	} {
		got, err := Tag(pool, tc.previous, tc.first, tc.count)
		if (err == nil) != tc.valid || got != tc.want {
			t.Errorf("%+v: %d %v", tc, got, err)
		}
	}
	if _, err := Tag(pool[:4], 0, 0, 0); err == nil {
		t.Fatal("fewer than five silently accepted")
	}
	if !slices.Equal(pool, []domain.Money{10 * Million, 8 * Million, 6 * Million, 4 * Million, 2 * Million, 99}) {
		t.Fatal("pool mutated")
	}
}

func TestExtensionFloorsTermsAndHighestRemaining(t *testing.T) {
	floors := map[string]domain.Money{"QB": 15 * Million, "WR": 10 * Million, "RB": 8 * Million, "TE": 8 * Million, "LB": 8 * Million, "DE": 7 * Million, "S": 5 * Million, "DT": 4 * Million, "CB": 3 * Million, "K": 3 * Million}
	cells := []Cell{{2026, 20 * Million}, {2027, 2 * Million}, {2028, 4 * Million}}
	for position, floor := range floors {
		got, err := Extend(cells, 2026, 3, position, 0, false)
		if err != nil || got.Price != max(6*Million, floor) || len(got.Cells) != 6 || got.Cells[5].Season != 2031 || !slices.Equal(got.Cells[:3], cells) {
			t.Errorf("%s: %+v %v", position, got, err)
		}
	}
	rich := []Cell{{2026, Million}, {2027, 12*Million + 350_000}}
	got, err := Extend(rich, 2026, 1, "QB", 0, false)
	if err != nil || got.Price != 1_801_000_000 {
		t.Errorf("highest and snap: %+v %v", got, err)
	}
	tagged, err := Extend(cells, 2026, 1, "QB", 5*Million, false)
	if err != nil || tagged.Price != 6*Million {
		t.Errorf("tagged 120%% no ordinary floor: %+v %v", tagged, err)
	}
	for _, tc := range []struct {
		cells    []Cell
		add      int
		pos      string
		extended bool
	}{
		{cells, 0, "QB", false}, {cells, 4, "QB", false}, {append(slices.Clone(cells), Cell{2029, Million}), 3, "QB", false},
		{cells, 1, "FLAG", false}, {cells, 1, "QB", true}, {cells[:1], 1, "QB", false},
	} {
		if _, err := Extend(tc.cells, 2026, tc.add, tc.pos, 0, tc.extended); err == nil {
			t.Errorf("invalid extension accepted %+v", tc)
		}
	}
}

func TestRestructureSourceTiersConservationAndSnap(t *testing.T) {
	for _, tc := range []struct{ salary, limit domain.Money }{
		{3*Million - 1, 0}, {3 * Million, Million}, {6*Million - 1, Million}, {6 * Million, 2 * Million}, {12*Million - 1, 2 * Million}, {12 * Million, 3 * Million},
	} {
		if got := MaxMove(tc.salary); got != tc.limit {
			t.Errorf("tier %d: %d want %d", tc.salary, got, tc.limit)
		}
	}
	cells := []Cell{{2026, 6 * Million}, {2027, 4 * Million}, {2028, 2 * Million}}
	got, err := Restructure(cells, 2027, 2026, Million+1, false)
	if err != nil || got.Cells[0].Salary != 7*Million || got.Cells[1].Salary != 3*Million || got.Cells[2] != cells[2] || got.DeadCapPercent != 50 {
		t.Fatalf("restructure: %+v %v", got, err)
	}
	var before, after domain.Money
	for i, c := range cells {
		before += c.Salary
		after += got.Cells[i].Salary
	}
	if before != after || cells[0].Salary != 6*Million {
		t.Fatal("total or input changed")
	}
	for _, tc := range []struct {
		from, to int
		move     domain.Money
		used     bool
	}{
		{2027, 2028, Million, false}, {2027, 2028, Million + 1_000_000, false},
		{2028, 2027, Million, false}, {2027, 2027, Million, false}, {2025, 2027, Million, false},
		{2027, 2030, Million, false}, {2027, 2028, 0, false}, {2027, 2028, -Million, false}, {2027, 2028, Million, true},
	} {
		_, err := Restructure(cells, tc.from, tc.to, tc.move, tc.used)
		valid := tc.from == 2027 && tc.to == 2028 && tc.move == Million && !tc.used
		if (err == nil) != valid {
			t.Errorf("restructure boundary %+v: %v", tc, err)
		}
	}
}

func TestEligibleContractBoundaries(t *testing.T) {
	for _, status := range []domain.ContractStatus{domain.CStatusRFA, domain.CStatusUFA, domain.CStatusFT1, domain.CStatusFT2} {
		for _, years := range []int{1, 2, 5, 6} {
			cells := Flat(2026, years, 3*Million, "").Cells
			for _, intent := range []string{"rfa.tender", "fa.resign", "contract.tag", "contract.extend", "contract.restructure"} {
				want := false
				switch intent {
				case "rfa.tender", "fa.resign":
					want = status == domain.CStatusRFA && years == 1
				case "contract.tag":
					want = years == 1
				case "contract.extend":
					want = (years > 1 || status == domain.CStatusFT1 || status == domain.CStatusFT2) && years < 6
				case "contract.restructure":
					want = years > 1 && status != domain.CStatusFT1 && status != domain.CStatusFT2
				}
				if got := Eligible(intent, status, cells, 2026, "LB"); got != want {
					t.Fatalf("%s %s %d: %v want %v", intent, status, years, got, want)
				}
			}
		}
	}
	for _, salary := range []domain.Money{3*Million - 1, 3 * Million} {
		cells := Flat(2026, 2, salary, "").Cells
		if Eligible("contract.restructure", domain.CStatusUFA, cells, 2026, "LB") != (salary >= 3*Million) {
			t.Fatal("salary floor")
		}
	}
}

func TestNextSeasonStartsAfterTheHeldContract(t *testing.T) {
	for _, tc := range []struct {
		cells        []Cell
		season, want int
	}{
		{[]Cell{{2026, 33_000_000}}, 2026, 2027},
		{[]Cell{{2026, Million}, {2027, Million}, {2028, Million}}, 2026, 2029},
		{[]Cell{{2026, Million}}, 2027, 2028},
		{nil, 2026, 2027},
	} {
		if got := NextSeason(tc.cells, tc.season); got != tc.want {
			t.Fatalf("%+v %d: got %d want %d", tc.cells, tc.season, got, tc.want)
		}
	}
}

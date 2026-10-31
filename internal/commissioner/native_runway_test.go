package commissioner

import (
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/engine/rookiedraft"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

func TestNativeCapRunway(t *testing.T) {
	t.Parallel()
	const million = domain.Money(100_000_000)
	rules := leaguerules.LegacyNFL()
	rules.Season = 2026
	rules.Cap.Amount.Value = 90 * leaguerules.Money(million)
	cut, err := nativecheck.PriceContract(nativecheck.Input{Rules: rules,
		Contract: nativecheck.Contract{Player: "0099", Name: "Cut player", Salary: 10 * million,
			Expiration: 2028, Status: domain.CStatusUFA}}, nativecheck.ActionCut)
	if err != nil || !cut.Removed || len(cut.Dead) != 1 || cut.Dead[0].Cents != 7*million {
		t.Fatalf("existing multi-year cut price: %+v, %v", cut, err)
	}
	cutCharges := []NativeCharge{}
	for _, cell := range cut.Dead {
		cutCharges = append(cutCharges, NativeCharge{Kind: deadMoney, Franchise: "f01", Season: cell.Season, Amount: cell.Cents})
	}
	terms, err := rookiedraft.SlotTerms(rules.Draft, 2, 1)
	if err != nil {
		t.Fatal(err)
	}
	rookie := RawNativeSnapshot{}
	signRookie(&rookie, 2026, "0001", "f01", 2, 1, terms)
	if terms.Years != 3 || terms.Salary != 142*million/100 {
		t.Fatalf("existing rookie slot terms: %+v", terms)
	}
	player := NativePlayer{ID: "0002", Franchise: "f01", Salary: 10 * million,
		Status: domain.RosterActive, ExpirationYear: 2026}
	tests := []struct {
		name    string
		players []NativePlayer
		charges []NativeCharge
		want    [4][3]domain.Money // committed, dead, room, in $10,000 units
	}{
		{"cut multi-year contract", nil, cutCharges, [4][3]domain.Money{{0, 700, 8300}, {0, 0, 9000}, {0, 0, 9000}, {0, 0, 9000}}},
		{"expiring contract", []NativePlayer{player}, nil, [4][3]domain.Money{{1000, 0, 8000}, {0, 0, 9000}, {0, 0, 9000}, {0, 0, 9000}}},
		{"rookie slot contract", rookie.Players, nil, [4][3]domain.Money{{142, 0, 8858}, {142, 0, 8858}, {142, 0, 8858}, {0, 0, 9000}}},
		{"over cap", []NativePlayer{player}, []NativeCharge{{Franchise: "f01", Kind: deadMoney, Season: 2026, Amount: 85 * million}},
			[4][3]domain.Money{{1000, 8500, -500}, {0, 0, 9000}, {0, 0, 9000}, {0, 0, 9000}}},
		{"dated dead cap across years", nil, []NativeCharge{
			{Franchise: "f01", Kind: deadMoney, Season: 2026, Amount: 8 * million},
			{Franchise: "f01", Kind: deadMoney, Season: 2027, Amount: 3 * million},
			{Franchise: "f02", Kind: deadMoney, Season: 2028, Amount: 99 * million}},
			[4][3]domain.Money{{0, 800, 8200}, {0, 300, 8700}, {0, 0, 9000}, {0, 0, 9000}}},
		{"standing and dated signed adjustments", nil, []NativeCharge{
			{Franchise: "f01", Kind: deadMoney, Amount: 2 * million},
			{Franchise: "f01", Amount: -million}, {Franchise: "f01", Season: 2027, Amount: 3 * million}},
			[4][3]domain.Money{{-100, 200, 8900}, {200, 200, 8600}, {-100, 200, 8900}, {-100, 200, 8900}}},
		{"other franchise excluded", []NativePlayer{{Franchise: "f02", Salary: 99 * million, Status: domain.RosterActive, ExpirationYear: 2029}}, nil,
			[4][3]domain.Money{{0, 0, 9000}, {0, 0, 9000}, {0, 0, 9000}, {0, 0, 9000}}},
		{"already expired contract", []NativePlayer{{Franchise: "f01", Salary: 10 * million, Status: domain.RosterActive, ExpirationYear: 2025}}, nil,
			[4][3]domain.Money{{0, 0, 9000}, {0, 0, 9000}, {0, 0, 9000}, {0, 0, 9000}}},
		{"unknown end stays unchanged", []NativePlayer{{Franchise: "f01", Salary: 10 * million, Status: domain.RosterActive}}, nil,
			[4][3]domain.Money{{1000, 0, 8000}, {1000, 0, 8000}, {1000, 0, 8000}, {1000, 0, 8000}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			seed := &RawNativeSnapshot{Season: 2026, Players: tt.players, Adjustments: tt.charges}
			before, err := json.Marshal(seed)
			if err != nil {
				t.Fatal(err)
			}
			got, err := NativeCapRunway(NativeBase{Assets: NativeAssets{Snapshot: seed}, Rules: rules}, "f01")
			if err != nil || len(got) != 4 {
				t.Fatalf("runway = %+v, %v", got, err)
			}
			for i, row := range got {
				want := tt.want[i]
				if row.Season != 2026+i || row.CapCents == nil || *row.CapCents != 90*million ||
					row.CommittedCents == nil || *row.CommittedCents != want[0]*(million/100) ||
					row.DeadCents == nil || *row.DeadCents != want[1]*(million/100) ||
					row.RoomCents == nil || *row.RoomCents != want[2]*(million/100) {
					t.Fatalf("year %d: %+v, want committed/dead/room %v × $10,000", i, row, want)
				}
			}
			after, err := json.Marshal(seed)
			if err != nil || string(after) != string(before) {
				t.Fatal("runway mutated the recorded assets")
			}
		})
	}
}

func TestNativeCapRunwayHorizonFollowsLeagueSettings(t *testing.T) {
	t.Parallel()
	const million = domain.Money(100_000_000)
	runway := func(t *testing.T, maxYears int, players []NativePlayer) []NativeCapYear {
		t.Helper()
		rules := leaguerules.LegacyNFL()
		rules.Season = 2026
		rules.Cap.Amount.Value = 90 * leaguerules.Money(million)
		rules.Contracts.MaximumYears = leaguerules.Known[int](maxYears)
		seed := &RawNativeSnapshot{Season: 2026, Players: players}
		got, err := NativeCapRunway(NativeBase{Assets: NativeAssets{Snapshot: seed}, Rules: rules}, "f01")
		if err != nil {
			t.Fatal(err)
		}
		return got
	}
	for _, tt := range []struct {
		name      string
		maxYears  int
		players   []NativePlayer
		wantYears int
		wantLast  int
	}{
		{"three-year setting", 3, nil, 3, 2028},
		{"five-year setting", 5, nil, 5, 2030},
		{"contract beyond the setting", 3, []NativePlayer{
			{Franchise: "f01", Salary: million, Status: domain.RosterActive, ExpirationYear: 2031}}, 6, 2031},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got := runway(t, tt.maxYears, tt.players)
			if len(got) != tt.wantYears || got[len(got)-1].Season != tt.wantLast {
				t.Fatalf("runway = %d years ending %d, want %d ending %d",
					len(got), got[len(got)-1].Season, tt.wantYears, tt.wantLast)
			}
		})
	}
	if len(runway(t, 3, nil)) == len(runway(t, 5, nil)) {
		t.Fatal("changing maximumYears did not change the runway")
	}
}

func TestNativeCapRunwayUsesExistingSharesAndPerEntryRounding(t *testing.T) {
	t.Parallel()
	rules := leaguerules.LegacyNFL()
	rules.Season = 2031
	rules.Cap.Amount.Value, rules.Cap.Quantum.Value = 90000, 100
	rules.Cap.IRShare.Value, rules.Cap.TaxiShare.Value = 5000, 0
	seed := &RawNativeSnapshot{Players: []NativePlayer{
		{Franchise: "f01", Salary: 10051, Status: domain.RosterActive, ExpirationYear: 2031},
		{Franchise: "f01", Salary: 10101, Status: domain.RosterIR, ExpirationYear: 2032},
		{Franchise: "f01", Salary: 25000, Status: domain.RosterTaxi, ExpirationYear: 2034}}, Adjustments: []NativeCharge{
		{Franchise: "f01", Kind: deadMoney, Season: 2031, Amount: 151},
		{Franchise: "f01", Kind: deadMoney, Season: 2031, Amount: 151},
		{Franchise: "f01", Season: 2031, Amount: -151}}}
	got, err := NativeCapRunway(NativeBase{Assets: NativeAssets{Snapshot: seed}, Rules: rules}, "f01")
	if err != nil {
		t.Fatal(err)
	}
	for i, want := range [4][3]domain.Money{{15000, 400, 74600}, {5100, 0, 84900}, {0, 0, 90000}, {0, 0, 90000}} {
		row := got[i]
		if row.Season != 2031+i || *row.CommittedCents != want[0] || *row.DeadCents != want[1] || *row.RoomCents != want[2] {
			t.Fatalf("year %d: %+v, want %v cents", i, row, want)
		}
	}
}

func TestNativeCapRunwayMissingRulingsAndFailures(t *testing.T) {
	t.Parallel()
	for _, field := range []string{"cap.amount", "cap.quantum", "cap.irShare", "cap.taxiShare"} {
		t.Run(field, func(t *testing.T) {
			rules := leaguerules.LegacyNFL()
			rules.Season = 2026
			switch field {
			case "cap.amount":
				rules.Cap.Amount.State = leaguerules.NeedsRuling
			case "cap.quantum":
				rules.Cap.Quantum.State = leaguerules.NeedsRuling
			case "cap.irShare":
				rules.Cap.IRShare.State = leaguerules.NeedsRuling
			case "cap.taxiShare":
				rules.Cap.TaxiShare.State = leaguerules.NeedsRuling
			}
			got, err := NativeCapRunway(NativeBase{Assets: NativeAssets{Snapshot: &RawNativeSnapshot{}}, Rules: rules}, "f01")
			if err != nil || len(got) != 4 {
				t.Fatalf("missing ruling: %+v, %v", got, err)
			}
			for _, row := range got {
				if row.RoomCents != nil || (field != "cap.amount" && (row.CommittedCents != nil || row.DeadCents != nil)) {
					t.Fatalf("missing ruling invented money: %+v", row)
				}
			}
		})
	}
	for _, player := range []NativePlayer{
		{Franchise: "f01", Salary: -1, Status: domain.RosterActive, ExpirationYear: 2029},
		{Franchise: "f01", Salary: 1, Status: "invalid", ExpirationYear: 2029},
		{Franchise: "f01", Salary: math.MaxInt64, Status: domain.RosterActive, ExpirationYear: 2029},
	} {
		t.Run(fmt.Sprint(player.Salary, player.Status), func(t *testing.T) {
			rules := leaguerules.LegacyNFL()
			rules.Season = 2026
			seed := &RawNativeSnapshot{Players: []NativePlayer{player}}
			before := *seed
			got, err := NativeCapRunway(NativeBase{Assets: NativeAssets{Snapshot: seed}, Rules: rules}, "f01")
			if err == nil || got != nil || !reflect.DeepEqual(*seed, before) {
				t.Fatalf("invalid held contract returned success or mutated state: %+v, %v", got, err)
			}
		})
	}
}

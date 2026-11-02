package commissioner

import (
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/domain"
)

func TestRosterCorrectionRetainsTheLastRecordedContract(t *testing.T) {
	for _, salary := range []domain.Money{0, 700} {
		b := rolloverLedger(t)
		edit := RawCommand{League: rolloverLeague, Command: ContractEdit, Request: "edit-before-cut", Reason: "Correct recorded salary", Effective: "now", Visibility: "league", Args: RawArgs{Override: &RawNativeOverride{Player: "0003", Years: new(3), Salary: &salary}}}
		b.add(edit, seasonMoment(5))
		raw := RawCommand{League: rolloverLeague, Command: CorrectRoster, Request: "cut-after-edit", Reason: "Repair membership", Effective: "now", Visibility: "league", Args: RawArgs{Correction: &RawCorrection{Player: "0003", Franchise: "franchise.02", Action: "remove"}}}
		b.add(raw, seasonMoment(6))
		raw.Request = "add-after-cut"
		raw.Args = RawArgs{Correction: &RawCorrection{Player: "0003", Franchise: "franchise.01", Action: "add"}}
		if err := CheckCorrectionAppend(raw, b.records, seasonMoment(7)); err != nil {
			t.Fatal(err)
		}
		b.add(raw, seasonMoment(7))
		assets, err := NativeAssetsReading(rolloverLeague, b.records)
		if err != nil {
			t.Fatal(err)
		}
		p := assets.Snapshot.Players[assets.Snapshot.PlayerOf("franchise.01", "0003")]
		if p.ContractMissing || p.Salary != salary || p.ExpirationYear != 2028 {
			t.Fatalf("latest contract lost: %+v", p)
		}
	}
}

func TestContractTrailCapturesOnlyTheAffectedPlayer(t *testing.T) {
	for _, tc := range []struct {
		name, player string
		want         int
	}{{"one player", "0003", 1}, {"whole roster", "", 2}, {"not rostered", "9999", 0}} {
		t.Run(tc.name, func(t *testing.T) {
			assets := NativeAssets{Seeded: true, Snapshot: &RawNativeSnapshot{Players: []NativePlayer{
				{ID: "0003", Franchise: "franchise.01", Salary: 700, ExpirationYear: 2028},
				{ID: "0004", Franchise: "franchise.02", ContractMissing: true},
			}}}
			contracts := map[string]NativePlayer{}
			var target *RawCorrection
			if tc.player != "" {
				target = &RawCorrection{Player: tc.player, Franchise: "franchise.01"}
			}
			rememberNativeContracts(&assets, contracts, target)
			if len(contracts) != tc.want {
				t.Fatalf("retained %d players, want %d", len(contracts), tc.want)
			}
			for _, p := range contracts {
				if tc.player != "" && p.ID != tc.player {
					t.Fatal("unrelated contract copied")
				}
				if p.ID == "0003" && (p.Salary != 700 || p.ExpirationYear != 2028 || p.ContractMissing) {
					t.Fatal("recorded contract changed")
				}
				if p.ID == "0004" && !p.ContractMissing {
					t.Fatal("missing contract filled")
				}
			}
		})
	}
}

func TestContractTrailKeepsSideCopiesSeparate(t *testing.T) {
	assets := NativeAssets{Seeded: true, Snapshot: &RawNativeSnapshot{Sides: map[string]string{"east": "E", "west": "W"}, Players: []NativePlayer{{ID: "0003", Franchise: "east", Salary: 700}, {ID: "0003", Franchise: "west", Salary: 900}}}}
	contracts := map[string]NativePlayer{}
	rememberNativeContracts(&assets, contracts, nil)
	assets.Snapshot.Players[0].Salary = 800
	assets.Snapshot.Players[1].Salary = 1000
	rememberNativeContracts(&assets, contracts, &RawCorrection{Player: "0003", Franchise: "east"})
	if len(contracts) != 2 || contracts["E/0003"].Salary != 800 || contracts["W/0003"].Salary != 900 {
		t.Fatalf("side contracts changed: %+v", contracts)
	}
}

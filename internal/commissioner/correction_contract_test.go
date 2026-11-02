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

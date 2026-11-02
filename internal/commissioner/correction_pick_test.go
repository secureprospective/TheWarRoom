package commissioner

import (
	"reflect"
	"testing"
)

func TestCorrectCurrentPickSlot(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name              string
		selected, skipped bool
		slot              int
		bad               bool
	}{
		{"unselected slot", false, false, 2, false},
		{"different slot", false, false, 3, true},
		{"selected slot", true, false, 2, true},
		{"skipped slot", false, true, 2, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := NativePick{Season: 2026, Round: 1, Slot: 2, Owner: "franchise.01", Skipped: tc.skipped}
			if tc.selected {
				p.Player = "0099"
			}
			seed := RawNativeSnapshot{Franchises: []NativeFranchise{{ID: "franchise.01"}, {ID: "franchise.02"}}, Picks: []NativePick{p}}
			correction := RawCorrection{Owner: "franchise.02", Pick: &CorrectionPick{Season: 2026, Round: 1, Slot: tc.slot}}
			if err := correction.Validate(CorrectPick); err != nil {
				t.Fatal(err)
			}
			err := correctPick(&seed, Record{Command: RawCommand{Command: CorrectPick, Args: RawArgs{Correction: &correction}, Reason: "Correct ownership"}})
			if (err != nil) != tc.bad {
				t.Fatalf("got %v, want bad %t", err, tc.bad)
			}
			if tc.bad && !reflect.DeepEqual(seed.Picks, []NativePick{p}) {
				t.Fatal("refused correction changed the pick")
			}
			if !tc.bad && seed.Picks[0].Owner != "franchise.02" {
				t.Fatal("slot did not change owner")
			}
		})
	}
	for _, pick := range []CorrectionPick{{Season: 2026, Round: 1}, {Season: 2026, Round: 1, Slot: -1}, {Season: 2026, Round: 1, Slot: 1001}, {Season: 2026, Round: 1, Slot: 1, Original: "franchise.01"}} {
		if err := (RawCorrection{Owner: "franchise.02", Pick: &pick}).Validate(CorrectPick); err == nil {
			t.Fatalf("accepted ambiguous or invalid pick %+v", pick)
		}
	}
}

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

func TestHistoricalPickCorrection(t *testing.T) {
	for _, tc := range []struct {
		name     string
		evidence PickEvidence
		bad      bool
	}{
		{"checked selection", PickEvidence{Slot: 2}, false},
		{"unverified", PickEvidence{Unverified: true}, false},
		{"missing selection", PickEvidence{Slot: 3}, true},
		{"invented link", PickEvidence{Slot: 2, Unverified: true}, true},
		{"blank", PickEvidence{}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			held := NativePick{Season: 2026, Round: 1, Slot: 2, Owner: "franchise.01", Player: "0099"}
			seed := RawNativeSnapshot{Season: 2026, Franchises: []NativeFranchise{{ID: "franchise.01"}, {ID: "franchise.02"}}, Picks: []NativePick{held}}
			c := RawCorrection{Owner: "franchise.02", Pick: &CorrectionPick{Season: 2026, Round: 1, Original: "franchise.01"}, History: &tc.evidence}
			err := c.Validate(CorrectPick)
			if err == nil {
				err = correctPick(&seed, Record{Command: RawCommand{Command: CorrectPick, Args: RawArgs{Correction: &c}}})
			}
			if (err != nil) != tc.bad {
				t.Fatalf("error=%v", err)
			}
			if !reflect.DeepEqual(seed.Picks, []NativePick{held}) {
				t.Fatal("historical correction changed selectable assets")
			}
			jobs := historicalPickJobs(NativeAssets{Seeded: true, Snapshot: &seed})
			if tc.bad {
				if len(seed.PickHistory) != 0 {
					t.Fatal("refusal changed history")
				}
				return
			}
			if len(seed.PickHistory) != 1 || seed.PickHistory[0].Owner != c.Owner {
				t.Fatal("lost ownership history")
			}
			if (len(jobs) == 1) != tc.evidence.Unverified {
				t.Fatalf("jobs=%+v", jobs)
			}
			c.Owner = "franchise.01"
			if err := correctPick(&seed, Record{Command: RawCommand{Command: CorrectPick, Args: RawArgs{Correction: &c}}}); err != nil {
				t.Fatal(err)
			}
			if len(seed.PickHistory) != 1 || seed.PickHistory[0].Owner != c.Owner {
				t.Fatal("history duplicated")
			}
		})
	}
}

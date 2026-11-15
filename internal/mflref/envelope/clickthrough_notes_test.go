package envelope

import (
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/snapshot"
)

func TestBlockedContractNoteIsOnlyFirstBlockingSentence(t *testing.T) {
	reason := "Offseason only: can't be done until the offseason."
	for _, reasons := range [][]string{
		{reason, "Priced from his 2026 salary, $600,000.", "Not checked: offer"},
		{"Not checked: offer", reason, "Another block."},
	} {
		got := (ContractCheck{Reasons: reasons}).Evaluate(Spec{Intent: "rfa.tender"}, snapshot.Snapshot{})
		if !got.Blocked || got.Note != reason {
			t.Fatalf("%+v", got)
		}
	}
}

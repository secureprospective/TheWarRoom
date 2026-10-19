package commissioner

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestCorrectionProjectionNeverRewritesTheOriginalEntry(t *testing.T) {
	b := rolloverLedger(t)
	original, err := json.Marshal(b.records)
	if err != nil {
		t.Fatal(err)
	}
	raw := RawCommand{League: rolloverLeague, Command: CorrectProvenance,
		Args: RawArgs{Correction: &RawCorrection{Player: "0003", Via: "trade"}}, Reason: "Repair acquisition history",
		Request: "provenance", Expected: int64(len(b.records)), Effective: "now", Visibility: "league"}
	b.add(raw, rolloverMoment())
	got, err := NativeAssetsReading(rolloverLeague, b.records)
	if err != nil {
		t.Fatal(err)
	}
	if got.Snapshot.Players[2].AcquiredVia != "trade" {
		t.Fatal("correction did not apply")
	}
	after, err := json.Marshal(b.records[:len(b.records)-1])
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(original, after) {
		t.Fatal("projection rewrote the original entry")
	}
	if len(b.records) != 5 {
		t.Fatal("correction deleted an original ledger entry")
	}
}

func TestCorrectionsDecodeOnlyTheirTypedArgument(t *testing.T) {
	for _, command := range []string{CorrectRoster, CorrectProvenance, CorrectTransaction, CorrectPick, RuleException} {
		t.Run(command, func(t *testing.T) {
			if _, err := decodeCorrection(RawArgs{}, command); err == nil {
				t.Fatal("untyped correction accepted")
			}
			if _, err := decodeCorrection(RawArgs{Correction: &RawCorrection{}, Field: "cap.amount"}, command); err == nil {
				t.Fatal("foreign correction detail accepted")
			}
		})
	}
}

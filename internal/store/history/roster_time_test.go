package history

import (
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
)

func TestRosterEffectiveTimeStoredAndRefused(t *testing.T) {
	for _, tc := range []struct {
		name, principal, effective, want string
	}{
		{"omitted defaults now", "creator", "", ""},
		{"now unchanged", "creator", "now", ""},
		{"commissioner past", "creator", "past", ""},
		{"owner past", "owner", "past", "permission"},
		{"future", "creator", "future", "cannot take effect in the future"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := newRolloverDB(t)
			effective := tc.effective
			at := r.c.now
			if effective == "past" {
				at = at.Add(-time.Hour)
				effective = at.Format(time.RFC3339Nano)
			}
			if effective == "future" {
				effective = at.Add(time.Hour).Format(time.RFC3339Nano)
			}
			player := heldSeedCommand(t).Args.Native.Players[0]
			raw := commissioner.RawCommand{League: "native.test", Command: commissioner.CorrectRoster,
				Args:     commissioner.RawArgs{Correction: &commissioner.RawCorrection{Player: player.ID, Franchise: player.Franchise, Action: "remove"}},
				Expected: r.version(t), Request: "historical-remove", Effective: effective, Visibility: "league", Reason: "Repair past membership"}
			before := fingerprint(t, r.s)
			receipt, err := r.s.ExecuteCommissioner(t.Context(), raw, tc.principal, nil)
			if tc.want != "" {
				if err == nil || !strings.Contains(err.Error(), tc.want) {
					t.Fatalf("got %v, want %q", err, tc.want)
				}
				if fingerprint(t, r.s) != before {
					t.Fatal("refusal changed stored state")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			events, err := r.s.Events(t.Context(), raw.League, commissioner.Stream, raw.Expected)
			if err != nil || len(events) != 1 {
				t.Fatalf("stored events=%d: %v", len(events), err)
			}
			if !events[0].EffectiveAt.Equal(at) || !events[0].RecordedAt.Equal(r.c.now) || !receipt.RecordedAt.Equal(r.c.now) {
				t.Fatalf("effective/receipt times lost: %+v", events[0])
			}
			ledger, err := r.s.CommissionerLedger(t.Context(), raw.League, "creator", nil)
			if err != nil || !ledger.Entries[0].When.Equal(r.c.now) {
				t.Fatalf("ledger receipt: %+v %v", ledger, err)
			}
			if tc.effective == "past" && !strings.Contains(ledger.Entries[0].What, effective) {
				t.Fatal("ledger omitted effective time")
			}
		})
	}
}

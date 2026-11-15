package commissioner

import (
	"strings"
	"testing"
	"time"
)

func TestRosterEffectiveAdmission(t *testing.T) {
	now := seasonMoment(9)
	for _, command := range []string{RosterCut, RosterReserve, CorrectRoster, CorrectPick, WaiverProcess, WaiverOverride,
		BidsProcess, BidsOverride, TradeAccept, TradeReview, TradeAppealRule, TradeOverride} {
		t.Run(command, func(t *testing.T) {
			for _, tc := range []struct {
				name, effective, role, want string
			}{
				{"default", "", "owner", ""},
				{"now", "now", "owner", ""},
				{"explicit now", now.Format(time.RFC3339Nano), "owner", ""},
				{"past commissioner", seasonMoment(8).Format(time.RFC3339Nano), "commissioner", ""},
				{"past owner", seasonMoment(8).Format(time.RFC3339Nano), "owner", "only a commissioner"},
				{"future", seasonMoment(10).Format(time.RFC3339Nano), "commissioner", "cannot take effect in the future"},
			} {
				t.Run(tc.name, func(t *testing.T) {
					raw := RawCommand{Command: command, Effective: tc.effective, Visibility: "league", Reason: "Repair past membership"}
					if err := raw.validatePlacement(); err != nil {
						t.Fatal(err)
					}
					err := CheckRosterTime(raw, now, tc.role)
					if tc.want == "" && err != nil || tc.want != "" && (err == nil || !strings.Contains(err.Error(), tc.want)) {
						t.Fatalf("got %v, want %q", err, tc.want)
					}
				})
			}
		})
	}
}

func TestRosterTimingShape(t *testing.T) {
	for _, effective := range []string{"tomorrow", "next_rollover", "0001-01-01T00:00:00Z"} {
		if err := (RawCommand{Command: RosterCut, Effective: effective, Visibility: "league"}).validatePlacement(); err == nil {
			t.Fatalf("accepted %s", effective)
		}
	}
	if err := (RawCommand{Command: RosterCut, Effective: "now", Visibility: "private"}).validatePlacement(); err == nil {
		t.Fatal("private roster change accepted")
	}
}

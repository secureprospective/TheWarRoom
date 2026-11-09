package research

import (
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

func TestNewsPostedPrecision(t *testing.T) {
	t.Parallel()
	at := time.Date(2026, 10, 13, 5, 48, 37, 0, time.UTC)
	zone := time.FixedZone("CDT", -5*60*60)
	for _, tc := range []struct{ wording, want string }{
		{"26 minutes", "Oct 13, 2026, 12:22 AM"},
		{"3 hours", "Oct 12, 2026, about 9 PM"},
		{"1 day", "Oct 12, 2026"},
		{"", "Posted time not supplied"},
		{"recently", "Posted time unknown: MFL wording not recognized"},
	} {
		t.Run(tc.wording, func(t *testing.T) {
			got, order := newsPosted(tc.wording, at, zone)
			if got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
			age, known := researchNewsAge(tc.wording)
			if known && order != float64(at.Add(-age).Unix()) || !known && order != 0 {
				t.Fatal("incorrect posted ordering", order)
			}
		})
	}
	if got, order := newsPosted("3 hours", time.Time{}, zone); order != 0 ||
		got != "Posted time unknown: fetch time not held" {
		t.Fatal(got, order)
	}
}

func TestNewsAgeValidation(t *testing.T) {
	for _, word := range []string{"6 days", "15 hours", "1 day", "1 hour", "2 minutes"} {
		if _, ok := researchNewsAge(word); !ok {
			t.Fatal(word)
		}
	}
	for _, word := range []string{"", "-1 day", "1 day ago", "recently", "1.5 days"} {
		if _, ok := researchNewsAge(word); ok {
			t.Fatal(word)
		}
	}
}

func TestRosteredAssetScopeKeepsUnscoredIDsAndOrder(t *testing.T) {
	t.Parallel()
	mirror := state.MirrorSnapshot{Players: []state.MirrorPlayer{{MFLID: "0099"}, {MFLID: "0100"}, {MFLID: "0099"}}}
	ids := RosteredAssetIDs(mirror)
	if len(ids) != 3 || ids[0] != "0099" || ids[1] != "0100" || ids[2] != "0099" {
		t.Fatal(ids)
	}
	ids[0] = "changed"
	if mirror.Players[0].MFLID != "0099" || RosteredAssetIDs(state.MirrorSnapshot{}) == nil {
		t.Fatal("scope aliases input or lost empty slice")
	}
}

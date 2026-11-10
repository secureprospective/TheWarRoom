package envelope

import (
	"reflect"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/domain"
)

func TestDraftIR(t *testing.T) {
	cases := []struct {
		name      string
		status    domain.RosterStatus
		want      domain.RosterStatus
		fresh     string
		franchise string
		league    string
		note      string
	}{
		{
			name:      "rostered",
			status:    domain.RosterActive,
			want:      domain.RosterIR,
			franchise: "0001",
			league:    "1",
			note:      "Not checked: eligibility not held; MFL will decide; MFL target not verified",
		},
		{
			name:      "on IR activates",
			status:    domain.RosterIR,
			want:      domain.RosterActive,
			franchise: "0001",
			league:    "1",
			note:      "Not checked: eligibility not held; MFL will decide; MFL target not verified",
		},
		{
			name:      "other franchise",
			status:    domain.RosterActive,
			want:      domain.RosterIR,
			franchise: "0002",
			league:    "1",
			note:      "player is not on this franchise's roster; MFL target not verified",
		},
		{
			name:      "roster unavailable",
			want:      domain.RosterIR,
			fresh:     domain.FreshFail,
			franchise: "0001",
			league:    "1",
			note:      "roster unavailable; MFL target not verified",
		},
		{name: "empty franchise", want: domain.RosterIR, league: "1"},
		{name: "empty league", want: domain.RosterIR, franchise: "0001"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			original, at, snap := testEnvelope(t)
			snap.Rosters.Value[0].Players[0].RosterStatus = tc.status
			if tc.fresh != "" {
				snap.Rosters.Provenance.Freshness.State = tc.fresh
			}
			req := IRRequest{
				LeagueID:    tc.league,
				FranchiseID: tc.franchise,
				Player:      original.Receipt().Spec.Expected.Player,
			}
			got, err := DraftIR(at, req, snap, func() string { return "ir-test" })
			if tc.note == "" {
				if err == nil || err.Error() != "draft IR: envelope: intent and scope required" {
					t.Fatalf("scope error: %v", err)
				}
				if !reflect.DeepEqual(got, Envelope{}) {
					t.Fatal("invalid scope returned an envelope")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			r := got.Receipt()
			want := AuditEntry{
				At:    at,
				From:  Draft,
				Event: ChecksBlock,
				To:    Blocked,
				Note:  tc.note,
			}
			if r.State != Blocked || !reflect.DeepEqual(r.Audit, []AuditEntry{want}) {
				t.Fatalf("receipt: %+v", r)
			}
			if r.Spec.Target.Kind != Unmapped || r.Spec.Target.URL != "" || r.Spec.Deadline != nil {
				t.Fatalf("target/deadline: %+v", r.Spec)
			}
			if r.Spec.Gravity != G2 || r.Spec.Undo != Reversible || r.Spec.Intent != "roster.ir" {
				t.Fatalf("IR spec: %+v", r.Spec)
			}
			if r.Spec.Expected.Player != req.Player || r.Spec.Expected.RosterStatus != tc.want {
				t.Fatalf("expected change: %+v", r.Spec.Expected)
			}
		})
	}
}

package envelope

import (
	"reflect"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/engine/eligibility"
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

func TestDraftIRUsesEngineDirection(t *testing.T) {
	for _, status := range []domain.RosterStatus{domain.RosterActive, domain.RosterIR} {
		t.Run(string(status), func(t *testing.T) {
			original, at, snap := testEnvelope(t)
			id := original.Receipt().Spec.Expected.Player
			snap.Rosters.Value[0].Players[0].RosterStatus = status
			snap.Eligibility.Players = eligibility.Evaluate(eligibility.Input{
				IR: eligibility.Capacity{Slots: 0, Held: true},
				Rosters: []eligibility.Franchise{{ID: "0001", Players: []eligibility.RosterPlayer{
					{ID: id.String(), Status: status},
				}}},
				Injuries: eligibility.Injuries{Fresh: domain.Freshness{State: domain.FreshLive}},
			})
			e, err := DraftIR(at, IRRequest{
				LeagueID: "1", FranchiseID: "0001", Player: id,
				Target: Target{Kind: Mapped, URL: "https://www47.myfantasyleague.com/2026/options?L=1&O=18"},
			}, snap, func() string { return "engine-ir" })
			if err != nil {
				t.Fatal(err)
			}
			if status == domain.RosterActive {
				if e.State() != Blocked || e.Receipt().Spec.Expected.RosterStatus != domain.RosterIR {
					t.Fatalf("healthy placement should block: %+v", e.Receipt())
				}
				return
			}
			v := snap.Eligibility.Players[id.String()].IR
			if v.Direction != eligibility.OffIR || v.Verdict != eligibility.Eligible ||
				e.State() != Ready || e.Receipt().Spec.Expected.RosterStatus != domain.RosterActive ||
				e.Receipt().Audit[0].Note != "eligibility checked (MFL rosters)" {
				t.Fatalf("activation must use off-IR rule despite no NFL IR designation or slots: %+v", e.Receipt())
			}
			e, err = e.HandOff(at.Add(time.Minute))
			if err != nil {
				t.Fatal(err)
			}
			snap.Rosters.Value[0].Players[0].RosterStatus = domain.RosterActive
			e, err = e.Observe(at.Add(2*time.Minute), observeAt(snap, at.Add(2*time.Minute)), IRPredicate{})
			if err != nil || e.State() != Landed {
				t.Fatalf("activation landing: %+v %v", e.Receipt(), err)
			}
		})
	}
}

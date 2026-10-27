package composition

import (
	"reflect"
	"strings"
	"testing"

	platformapp "github.com/secureprospective/TheWarRoom/internal/app/platform"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/envelope"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
)

func TestRosterDraftTargetsAndDirections(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name   string
		intent string
		status domain.RosterStatus
		want   domain.RosterStatus
		option string
	}{
		{"IR", "roster.ir", domain.RosterActive, domain.RosterIR, "18"},
		{"demote", "roster.taxi", domain.RosterActive, domain.RosterTaxi, "98"},
		{"promote", "roster.taxi", domain.RosterTaxi, domain.RosterActive, "98"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			a, transport, _ := lineupDraftApp(t)
			a.core.MyFranchise = "0001"
			player, err := playerid.New("0042")
			if err != nil {
				t.Fatal(err)
			}
			a.core.TargetSnapshot.Rosters.Value = []snapshot.Roster{{
				FranchiseID: "0001", Players: []snapshot.RosterPlayer{{ID: player, RosterStatus: tc.status}},
			}}
			var r envelope.Receipt
			if tc.intent == "roster.ir" {
				r, err = a.platformAPI().TargetDraftIR("0001", "0042")
			} else {
				r, err = a.platformAPI().TargetDraftTaxi("0001", "0042")
			}
			wantURL := "https://www47.myfantasyleague.com/2026/options?L=14432&O=" + tc.option
			if err != nil || r.State != envelope.Ready || r.Spec.Target.URL != wantURL ||
				r.Spec.Expected.RosterStatus != tc.want || r.Spec.Intent != tc.intent {
				t.Fatalf("draft: %+v %v", r, err)
			}
			if err := platformapp.ValidateMoveURL(r.Spec.Target.URL); err != nil {
				t.Fatal(err)
			}
			stored, err := a.core.Moves.Get(a.core.Ctx, r.CorrelationID)
			if err != nil || !reflect.DeepEqual(stored.Receipt(), r) {
				t.Fatalf("saved: %+v %v", stored.Receipt(), err)
			}
			if len(transport.requests) != 0 {
				t.Fatal("draft fetched MFL")
			}
		})
	}
}

func TestRosterDraftUnknownHost(t *testing.T) {
	t.Parallel()
	a := targetTestApp(t)
	down := targetOfflineClient(t, a)
	if _, err := a.platformAPI().TargetSnapshot(); err != nil {
		t.Fatal(err)
	}
	calls := down.calls
	for _, draft := range []func(string, string) (envelope.Receipt, error){a.platformAPI().TargetDraftIR, a.platformAPI().TargetDraftTaxi} {
		r, err := draft("0001", "0042")
		if err != nil || r.State != envelope.Blocked || r.Spec.Target.Kind != envelope.Unmapped ||
			r.Spec.Target.URL != "" || !strings.Contains(r.Audit[0].Note, "MFL host not yet known") {
			t.Fatalf("unknown host: %+v %v", r, err)
		}
	}
	if calls != down.calls {
		t.Fatal("draft discovered host")
	}
}

func TestTargetDraftTaxiValidation(t *testing.T) {
	t.Parallel()
	a := targetTestApp(t)
	if _, err := a.platformAPI().TargetDraftTaxi("0001", "0042"); err == nil {
		t.Fatal("draft without snapshot")
	}
	targetOfflineClient(t, a)
	if _, err := a.platformAPI().TargetSnapshot(); err != nil {
		t.Fatal(err)
	}
	if _, err := a.platformAPI().TargetDraftTaxi("0001", "invalid"); err == nil {
		t.Fatal("invalid player accepted")
	}
	for _, mode := range []string{"IR", "absent"} {
		a.core.TargetSnapshot.Rosters.Value[0].Players[0].RosterStatus = domain.RosterIR
		if mode == "absent" {
			a.core.TargetSnapshot.Rosters.Value[0].Players = nil
		}
		r, err := a.platformAPI().TargetDraftTaxi("0001", "0042")
		if err != nil || r.State != envelope.Blocked {
			t.Fatalf("invalid taxi: %+v %v", r, err)
		}
		want := "player on IR"
		if mode == "absent" {
			want = "player is not on this franchise's roster"
		}
		if !strings.Contains(r.Audit[0].Note, want) {
			t.Fatal(r.Audit[0].Note)
		}
	}
}

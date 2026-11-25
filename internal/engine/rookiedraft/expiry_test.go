package rookiedraft

import (
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

func TestExpiryPoliciesAndLateSelection(t *testing.T) {
	for _, policy := range []leaguerules.ExpiryAction{
		leaguerules.SkipPick, leaguerules.AutoPick, leaguerules.WaitPick,
	} {
		t.Run(string(policy), func(t *testing.T) {
			r := testRules().Draft
			r.OnExpire = leaguerules.Known(policy)
			start := fixedStart()
			at := start.Add(7 * time.Hour)
			order := testOrder(t, r)
			d, err := ReplayActs(r, order, start, at, nil)
			if err != nil {
				t.Fatal(err)
			}
			if policy != leaguerules.SkipPick {
				if d.OnClock == nil || d.OnClock.Slot != 1 || d.Picks[0].Skipped {
					t.Fatalf("turn lost: %+v", d)
				}
				return
			}
			if !d.Owed() {
				t.Fatal("missing debt")
			}
			pick := d.Picks[0]
			act := Act{Kind: Select, At: at, Pick: first(), Owner: pick.Owner, Player: "0001"}
			d, err = ReplayActs(r, order, start, at, []Act{act})
			if err != nil {
				t.Fatal(err)
			}
			if d.Picks[0].Player != "0001" || d.Picks[0].Round != 1 || d.Picks[0].Slot != 1 || d.OnClock.Slot != 2 {
				t.Fatalf("late pick moved slot: %+v", d)
			}
			closeAct := Act{Kind: Close, At: at}
			d, err = ReplayActs(r, order, start, at, []Act{closeAct})
			if err != nil {
				t.Fatal(err)
			}
			if d.Owed() || d.Picks[0].How != HowUnmade {
				t.Fatalf("debt survived close: %+v", d.Picks[0])
			}
			act.At = at.Add(time.Second)
			if _, err = ReplayActs(r, order, start, act.At, []Act{closeAct, act}); err == nil {
				t.Fatal("closed draft accepted pick")
			}
			at = start.Add(1000 * time.Hour)
			act.At = at
			d, err = ReplayActs(r, order, start, at, []Act{act})
			if err != nil || d.Picks[0].Player != "0001" {
				t.Fatalf("exhausted late pick: %+v %v", d, err)
			}
		})
	}
}

func TestVacantTurnHeldUnderEveryExpiryPolicy(t *testing.T) {
	for _, policy := range []leaguerules.ExpiryAction{
		leaguerules.SkipPick, leaguerules.AutoPick, leaguerules.WaitPick,
	} {
		r := testRules().Draft
		r.OnExpire = leaguerules.Known(policy)
		hold := func(string, time.Time) (bool, error) { return true, nil }
		d, err := ReplayActs(r, testOrder(t, r), fixedStart(),
			fixedStart().Add(100*time.Hour), nil, hold)
		if err != nil || d.OnClock == nil || d.OnClock.Slot != 1 {
			t.Fatalf("%s: %+v %v", policy, d, err)
		}
	}
}

func TestLatePickDoesNotExpireVacantSuccessor(t *testing.T) {
	r := testRules().Draft
	order := testOrder(t, r)
	at := fixedStart().Add(20 * time.Hour)
	act := Act{Kind: Select, At: at, Pick: first(), Owner: order[0].Picks[0].Owner, Player: "0001"}
	// Hold the successor by deadline; fixture picks may share an owner.
	hold := func(_ string, deadline time.Time) (bool, error) {
		return deadline.After(fixedStart().Add(6 * time.Hour)), nil
	}
	d, err := ReplayActs(r, order, fixedStart(), at, []Act{act}, hold)
	if err != nil {
		t.Fatal(err)
	}
	if d.OnClock == nil || d.OnClock.Slot != 2 || d.Picks[1].Skipped {
		t.Fatalf("late pick lost successor: %+v", d)
	}
}

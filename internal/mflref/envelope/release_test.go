package envelope

import (
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
)

func TestReleaseCheckBoundaries(t *testing.T) {
	base := ReleaseCheck{OnRoster: true, Offseason: true, Remaining: 2, ChargeText: "$600,000"}
	for _, tc := range []struct {
		name, intent string
		check        ReleaseCheck
		blocked      bool
		note         string
	}{
		{"cut any season", CutIntent, ReleaseCheck{OnRoster: true, ChargeText: "$0"}, false, "$0"},
		{"offseason", BuyoutIntent, base, false, "$600,000"},
		{"regular season", BuyoutIntent, ReleaseCheck{OnRoster: true, Remaining: 2}, true, "Offseason only: can't be done until the offseason."},
		{"second", BuyoutIntent, ReleaseCheck{OnRoster: true, Offseason: true, Remaining: 2, OtherBuyouts: 1}, false, "Roster checked"},
		{"third", BuyoutIntent, ReleaseCheck{OnRoster: true, Offseason: true, Remaining: 2, OtherBuyouts: 2}, true, "Two buyouts"},
		{"one year", BuyoutIntent, ReleaseCheck{OnRoster: true, Offseason: true, Remaining: 1}, true, "1-year-left"},
		{"four years", BuyoutIntent, ReleaseCheck{OnRoster: true, Offseason: true, Remaining: 4}, false, "Roster checked"},
		{"five years", BuyoutIntent, ReleaseCheck{OnRoster: true, Offseason: true, Remaining: 5}, true, "2–4"},
		{"not mine", CutIntent, ReleaseCheck{}, true, "not on my roster"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.check.Evaluate(Spec{Intent: tc.intent}, snapshot.Snapshot{})
			if got.Blocked != tc.blocked || !strings.Contains(got.Note, tc.note) {
				t.Fatalf("got %+v", got)
			}
		})
	}
}
func TestReleaseConfirmationLandingAndInvalidation(t *testing.T) {
	original, at, snap := testEnvelope(t)
	spec := original.Receipt().Spec
	spec.Intent = CutIntent
	spec.Expected.RosterStatus = ""
	e, err := DraftRelease(at, spec, ReleaseCheck{OnRoster: true}, func() string { return "cut" })
	if err != nil {
		t.Fatal(err)
	}
	if _, err := e.HandOffRelease(at, false); err == nil {
		t.Fatal("cut did not require confirmation")
	}
	e, err = e.HandOffRelease(at, true)
	if err != nil {
		t.Fatal(err)
	}
	obs := observeAt(snap, at.Add(time.Minute))
	p := ReleasePredicate{}
	if got := p.Evaluate(spec, obs); got.Event != NoChange {
		t.Fatal(got)
	}
	obs.Rosters.Value[0].Players = nil
	next, err := e.Observe(at.Add(time.Minute), obs, p)
	if err != nil || next.State() != Landed {
		t.Fatalf("landing %s %v", next.State(), err)
	}
	next, err = next.Invalidate(at.Add(time.Minute), "$100 planned vs $200 now")
	if err != nil || next.State() != Landed || next.Receipt().Audit[len(next.Receipt().Audit)-1].Event != Invalidate {
		t.Fatal("mismatch became failure", err)
	}
	obs.Rosters.Value = nil
	if got := p.Evaluate(spec, obs); got.Event != Partial {
		t.Fatal(got)
	}
	obs.LeagueID = "other"
	if got := p.Evaluate(spec, obs); got.Event != Partial {
		t.Fatal(got)
	}
	obs.LeagueID = spec.LeagueID
	obs.Rosters.Provenance.Freshness.State = domain.FreshFail
	if got := p.Evaluate(spec, obs); got.Event != Partial {
		t.Fatal(got)
	}
	for _, reason := range []string{
		ReleaseInvalidation(CutIntent, false, false, true, false, false),
		ReleaseInvalidation(CutIntent, false, true, true, true, false),
		ReleaseInvalidation(CutIntent, true, true, true, false, true),
		ReleaseInvalidation(BuyoutIntent, true, true, false, false, false),
	} {
		if reason == "" {
			t.Fatal("trigger missed")
		}
		stale, err := e.Invalidate(at, reason)
		if err != nil || stale.State() != Stale {
			t.Fatal(err)
		}
		rebased, err := stale.Rebase(at, "$100 → $200")
		if err != nil || rebased.State() != Draft {
			t.Fatal(err)
		}
	}
	if ReleaseInvalidation(CutIntent, true, false, true, true, true) != "" {
		t.Fatal("handed-off absence should land")
	}
}

func TestReleaseWithdrawalUsesTerminalTransition(t *testing.T) {
	original, at, snap := testEnvelope(t)
	spec := original.Receipt().Spec
	spec.Intent = CutIntent
	e, err := DraftRelease(at, spec, ReleaseCheck{OnRoster: true}, func() string { return "withdraw-cut" })
	if err != nil {
		t.Fatal(err)
	}
	handed, err := e.HandOffRelease(at, true)
	if err != nil {
		t.Fatal(err)
	}
	pending, err := handed.Observe(at.Add(time.Minute), observeAt(snap, at.Add(time.Minute)), ReleasePredicate{})
	if err != nil {
		t.Fatal(err)
	}
	blocked, err := DraftRelease(at, spec, ReleaseCheck{}, func() string { return "blocked-cut" })
	if err != nil {
		t.Fatal(err)
	}
	for _, candidate := range []Envelope{e, handed, pending, blocked} {
		gone, err := candidate.WithdrawRelease(at.Add(2*time.Minute), "not dropped")
		if err != nil || gone.State() != Withdrawn || ReleaseOpen(gone.State()) {
			t.Fatal(gone, err)
		}
		if _, err := gone.HandOffRelease(at.Add(3*time.Minute), true); err == nil {
			t.Fatal("terminal plan handed off")
		}
	}
	if _, err := original.WithdrawRelease(at, "x"); err == nil {
		t.Fatal("non-release withdrawn")
	}
}

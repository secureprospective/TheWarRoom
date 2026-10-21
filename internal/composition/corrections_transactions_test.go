package composition

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
)

func TestTransactionCorrectionReversesNativeCutAndAward(t *testing.T) {
	t.Parallel()
	for _, award := range []bool{false, true} {
		t.Run(map[bool]string{false: "cut", true: "award"}[award], func(t *testing.T) {
			t.Parallel()
			f := waiverApp(t)
			cut := f.cut(t, "1000")
			target := cut
			if award {
				f.claim(t, cut.Entry, "owner.one", "franchise.01")
				f.now = f.now.Add(24 * time.Hour)
				target = f.process(t, cut.Entry)
			}
			for pass := range 2 {
				if pass == 1 {
					f.restart(t)
				}
				before, err := newCommissionerAPI(f.app).TargetNativeAssets(legalityLeague)
				if err != nil {
					t.Fatal(err)
				}
				fix := f.command(t, commissioner.CorrectTransaction, commissioner.RawArgs{Correction: &commissioner.RawCorrection{Entry: target.Entry}}, "league")
				after, err := newCommissionerAPI(f.app).TargetNativeAssets(legalityLeague)
				if err != nil {
					t.Fatal(err)
				}
				owner := ""
				for _, p := range after.Snapshot.Players {
					if p.ID == "1000" {
						owner = p.Franchise
					}
				}
				if (!award && owner != "franchise.02") || (award && owner != "") {
					t.Fatalf("transaction not reversed: owner %q", owner)
				}
				reading := f.reading(t, "owner.one", "franchise.01")
				if len(reading.Corrections) == 0 || (!award && len(reading.Players) != 0) ||
					(award && reading.Players[0].Winner != "") {
					t.Fatal("waivers still shows the old transaction")
				}
				f.restart(t)
				replayed, err := newCommissionerAPI(f.app).TargetNativeAssets(legalityLeague)
				if err != nil || !reflect.DeepEqual(after.Snapshot, replayed.Snapshot) {
					t.Fatal("correction lost on replay", err)
				}
				f.command(t, commissioner.Revert, commissioner.RawArgs{Entry: fix.Entry}, "league")
				restored, err := newCommissionerAPI(f.app).TargetNativeAssets(legalityLeague)
				if err != nil || !reflect.DeepEqual(before.Snapshot, restored.Snapshot) {
					t.Fatal("transaction compensation did not restore", err)
				}
			}
		})
	}
}

func TestTransactionCorrectionReversesNativeDraftSelection(t *testing.T) {
	t.Parallel()
	f := draftApp(t)
	f.scheduled(t)
	f.seat(t, draftTeam(1), "owner.one")
	f.now = draftStartAt()
	f.actAs("owner.one")
	selected := f.pick(t, draftTeam(1), "5001")
	f.actAs("creator")
	for pass := range 2 {
		if pass == 1 {
			f.reopen(t)
		}
		before := correctionAssets(t, f)
		fix := assetCorrection(t, f, commissioner.CorrectTransaction, commissioner.RawCorrection{Entry: selected.Entry})
		for _, p := range correctionAssets(t, f).Snapshot.Players {
			if p.ID == "5001" {
				t.Fatal("reversed draft selection still rostered")
			}
		}
		if draftStillShowsPlayer(f.read(t, draftTeam(1)), "5001") {
			t.Fatal("reversed selection still appears in draft")
		}
		f.reopen(t)
		if draftStillShowsPlayer(f.read(t, draftTeam(1)), "5001") {
			t.Fatal("replay restored reversed selection")
		}
		f.revert(t, fix)
		if !reflect.DeepEqual(before.Snapshot, correctionAssets(t, f).Snapshot) {
			t.Fatal("draft transaction compensation failed")
		}
	}
}

func TestRuleExceptionGovernsWaiverEligibilityAndReopen(t *testing.T) {
	t.Parallel()
	for _, rule := range []string{"roster.limits", "cap.enforcement"} {
		t.Run(rule, func(t *testing.T) {
			t.Parallel()
			f := waiverApp(t)
			cut := f.cut(t, "1000")
			if rule == "roster.limits" {
				f.set(t, "roster.limits", `{"min":0,"max":40}`, "now")
			} else {
				f.set(t, "cap.amount", `100000000`, "now")
			}
			for pass := range 2 {
				if pass == 1 {
					f.restart(t)
				}
				assertWaiverClaimRefused(t, f, cut.Entry)
				fix := f.command(t, commissioner.RuleException, commissioner.RawArgs{Correction: &commissioner.RawCorrection{
					Rule: rule, Scope: &commissioner.ExceptionScope{Franchise: "franchise.01"}}}, "league")
				claim := f.claim(t, cut.Entry, "owner.one", "franchise.01")
				f.command(t, commissioner.Revert, commissioner.RawArgs{Entry: claim.Entry}, "private")
				f.restart(t)
				claim = f.claim(t, cut.Entry, "owner.one", "franchise.01")
				f.command(t, commissioner.Revert, commissioner.RawArgs{Entry: claim.Entry}, "private")
				f.command(t, commissioner.Revert, commissioner.RawArgs{Entry: fix.Entry}, "league")
				assertWaiverClaimRefused(t, f, cut.Entry)
			}
		})
	}
}

func assertWaiverClaimRefused(t *testing.T, f *waiverFixture, window string) {
	t.Helper()
	_, err := f.owner(t, "owner.one", commissioner.WaiverClaim,
		commissioner.RawWaiverArgs{Window: window, Franchise: "franchise.01"})
	if err == nil || (!strings.Contains(err.Error(), "roster") && !strings.Contains(err.Error(), "cap")) {
		t.Fatalf("expected shared legality refusal, got %v", err)
	}
}

func draftStillShowsPlayer(reading commissioner.NativeDraft, player string) bool {
	for _, p := range reading.Draft.Picks {
		if p.Player == player {
			return true
		}
	}
	return false
}

func TestRuleExceptionPlayerScopeExpiresInReserveRule(t *testing.T) {
	t.Parallel()
	f := waiverApp(t)
	until := f.now.Add(time.Hour)
	for pass := range 2 {
		if pass == 1 {
			f.restart(t)
		}
		assertReserveException(t, f, false)
		fix := f.command(t, commissioner.RuleException, commissioner.RawArgs{Correction: &commissioner.RawCorrection{
			Rule: "ir.eligible", Scope: &commissioner.ExceptionScope{Player: "2002"}, Until: until.Format(time.RFC3339Nano)}}, "league")
		assertReserveException(t, f, true)
		f.restart(t)
		assertReserveException(t, f, true)
		before := f.now
		f.now = until
		assertReserveException(t, f, false)
		corrections, err := newCommissionerAPI(f.app).TargetCorrections(legalityLeague)
		if err != nil || len(corrections.Items) != 0 {
			t.Fatal("expired exception still in force", err)
		}
		f.now = before
		f.command(t, commissioner.Revert, commissioner.RawArgs{Entry: fix.Entry}, "league")
		assertReserveException(t, f, false)
	}
}

func assertReserveException(t *testing.T, f *waiverFixture, allowed bool) {
	t.Helper()
	reading := f.reserves(t)
	found := false
	for _, p := range reading.Players {
		if p.Player == "2002" {
			found = true
			if (p.Standing == "stays") != allowed {
				t.Fatalf("player exception not enforced in IR rule: %+v", p)
			}
			if allowed && !strings.Contains(p.Words, "Commissioner exception") {
				t.Fatal("reserve reason missing")
			}
		}
	}
	if !found {
		t.Fatal("reserve player missing")
	}
}

func TestRuleExceptionTransactionScopeGovernsAwardDecision(t *testing.T) {
	t.Parallel()
	f := waiverApp(t)
	cut := f.cut(t, "1000")
	claim := f.claim(t, cut.Entry, "owner.one", "franchise.01")
	f.set(t, "cap.amount", `100000000`, "now")
	f.now = f.now.Add(24 * time.Hour)
	for pass := range 2 {
		if pass == 1 {
			f.restart(t)
		}
		denied := f.process(t, cut.Entry)
		if f.reading(t, "owner.one", "franchise.01").Players[0].Winner != "" {
			t.Fatal("over-cap claim won without exception")
		}
		f.command(t, commissioner.Revert, commissioner.RawArgs{Entry: denied.Entry}, "league")
		fix := f.command(t, commissioner.RuleException, commissioner.RawArgs{Correction: &commissioner.RawCorrection{
			Rule: "cap.enforcement", Scope: &commissioner.ExceptionScope{Transaction: claim.Entry}}}, "league")
		award := f.process(t, cut.Entry)
		if f.reading(t, "owner.one", "franchise.01").Players[0].Winner != "franchise.01" {
			t.Fatal("transaction exception ignored at award")
		}
		f.restart(t)
		if f.reading(t, "owner.one", "franchise.01").Players[0].Winner != "franchise.01" {
			t.Fatal("award lost on replay")
		}
		f.command(t, commissioner.Revert, commissioner.RawArgs{Entry: award.Entry}, "league")
		f.command(t, commissioner.Revert, commissioner.RawArgs{Entry: fix.Entry}, "league")
	}
}

package leagueservice

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/governance"
)

func TestBackupNowUsesCurrentCommissionerSeatsAndRefusesAgentTokens(t *testing.T) {
	fixture := newDueFixture(t)
	for _, grant := range []struct {
		principal string
		role      governance.Role
	}{
		{"cocommissioner.one", governance.CoCommissioner},
		{"deputy.one", governance.Deputy},
	} {
		raw := fixture.raw(t, "commish.role.grant", commissioner.RawArgs{People: &commissioner.RawPeople{
			Principal: grant.principal, Role: grant.role,
		}})
		raw.Visibility = "private"
		if _, err := fixture.store.ExecuteCommissionerWithFacts(t.Context(), raw, "creator", nil, fixture.facts, time.UTC); err != nil {
			t.Fatal(err)
		}
	}
	var calls atomic.Int32
	runner := func(_ context.Context, league string) (BackupResult, error) {
		calls.Add(1)
		at := time.Date(2026, 11, 18, 12, 0, 0, 0, time.UTC)
		return BackupResult{League: league, Captured: at, Verified: at.Add(time.Second), Where: "Cloudflare R2",
			Checked: true, Message: "League backup saved 2026-11-18 12:00 UTC and checked in Cloudflare R2."}, nil
	}
	serviceFor := func(principal string) *Service {
		return viewService(t, fixture, principal).WithBackupRunner(runner)
	}
	raw := RawBackupNow{League: dueLeague}
	for _, principal := range []string{"creator", "cocommissioner.one"} {
		service := serviceFor(principal)
		preview, err := service.PreviewBackupNow(t.Context(), raw)
		if err != nil || preview.Safeguard != "confirm" || preview.Target.League != dueLeague {
			t.Fatalf("%s preview = %+v, %v", principal, preview, err)
		}
		result, err := service.BackupNow(t.Context(), raw)
		if err != nil || result.League != dueLeague || !result.Checked {
			t.Fatalf("%s backup = %+v, %v", principal, result, err)
		}
	}
	for _, principal := range []string{"owner.one", "deputy.one"} {
		if _, err := serviceFor(principal).BackupNow(t.Context(), raw); err == nil {
			t.Fatalf("unauthorized %s ran a backup", principal)
		}
	}
	if _, err := serviceFor("creator").BackupNow(commissioner.WithTokenAudit(t.Context(), "agent.token"), raw); err == nil {
		t.Fatal("agent token ran a backup")
	}
	if got := calls.Load(); got != 2 {
		t.Fatalf("backup calls = %d, want commissioner and co-commissioner only", got)
	}
}

func TestBackupNowRechecksPermissionBeforeExecutionAndPreservesLeague(t *testing.T) {
	fixture := newDueFixture(t)
	var calls atomic.Int32
	service := viewService(t, fixture, "owner.one").WithBackupRunner(func(context.Context, string) (BackupResult, error) {
		calls.Add(1)
		return BackupResult{}, fmt.Errorf("unexpected backup")
	})
	before, err := fixture.store.Events(t.Context(), dueLeague, commissioner.Stream, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.PreviewBackupNow(t.Context(), RawBackupNow{League: dueLeague}); err == nil {
		t.Fatal("owner preview was admitted")
	}
	if _, err := service.BackupNow(t.Context(), RawBackupNow{League: dueLeague}); err == nil {
		t.Fatal("owner execute was admitted")
	}
	after, err := fixture.store.Events(t.Context(), dueLeague, commissioner.Stream, 0)
	if err != nil || len(after) != len(before) || calls.Load() != 0 {
		t.Fatalf("rejected backup changed state or ran: before=%d after=%d calls=%d err=%v", len(before), len(after), calls.Load(), err)
	}
}

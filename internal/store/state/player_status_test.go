package state

import (
	"context"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/domain"
)

// recordStatus runs RecordStatus through a WriteTx (the surface ReleasePlayer uses).
func recordStatus(t *testing.T, s *Store, mflID string, st domain.PlayerStatus, reason string) error {
	t.Helper()
	return s.WriteTx(context.Background(), func(w TxWriter) error {
		return w.RecordStatus(context.Background(), mflID, st, reason)
	})
}

// TestRecordAndCurrentStatus pins the event log: the current status is the LATEST event, an
// unseen player reports found=false, and an invalid status / empty reason are rejected.
func TestRecordAndCurrentStatus(t *testing.T) {
	s := newStore(t, &fakeSource{rosters: baseRosters(t)})
	ctx := context.Background()

	if _, found, err := s.CurrentStatus(ctx, "9999"); err != nil || found {
		t.Fatalf("CurrentStatus of unseen player = found %v (err %v), want found=false", found, err)
	}
	if err := recordStatus(t, s, "9999", domain.PlayerFreeAgent, "cut"); err != nil {
		t.Fatalf("record FREE_AGENT: %v", err)
	}
	if err := recordStatus(t, s, "9999", domain.PlayerRetired, "retired later"); err != nil {
		t.Fatalf("record RETIRED: %v", err)
	}
	got, found, err := s.CurrentStatus(ctx, "9999")
	if err != nil || !found || got != domain.PlayerRetired {
		t.Fatalf("CurrentStatus = %q found=%v (err %v), want the latest RETIRED", got, found, err)
	}

	if err := recordStatus(t, s, "9999", domain.PlayerStatus("BOGUS"), "x"); err == nil {
		t.Fatal("recorded an invalid status, want rejection")
	}
	if err := recordStatus(t, s, "9999", domain.PlayerFreeAgent, ""); err == nil {
		t.Fatal("recorded with an empty reason, want rejection")
	}
}

// TestPlayerStatusAppendOnlyTriggers proves the double-immutability: a raw UPDATE or DELETE that
// bypasses the write-only Go API still aborts at the DB (the house audit-log idiom). The planted
// violations ARE the proof the gate has teeth.
func TestPlayerStatusAppendOnlyTriggers(t *testing.T) {
	s := newStore(t, &fakeSource{rosters: baseRosters(t)})
	ctx := context.Background()

	if err := recordStatus(t, s, "9999", domain.PlayerFreeAgent, "seed"); err != nil {
		t.Fatalf("seed status: %v", err)
	}
	if _, err := s.pools.Write().ExecContext(ctx,
		`UPDATE player_status_events SET status = 'RETIRED' WHERE league_id = ?`, s.leagueID); err == nil {
		t.Fatal("raw UPDATE on player_status_events succeeded — append-only trigger did not fire")
	} else if !strings.Contains(err.Error(), "append-only") {
		t.Fatalf("UPDATE error = %v, want an append-only abort", err)
	}
	if _, err := s.pools.Write().ExecContext(ctx,
		`DELETE FROM player_status_events WHERE league_id = ?`, s.leagueID); err == nil {
		t.Fatal("raw DELETE on player_status_events succeeded — append-only trigger did not fire")
	} else if !strings.Contains(err.Error(), "append-only") {
		t.Fatalf("DELETE error = %v, want an append-only abort", err)
	}
}

// TestFreeAgentsExcludesRostered proves the pool query: a FREE_AGENT event puts a player in the
// pool ONLY while he is on no roster. A latest status of RETIRED keeps him out even off-roster.
func TestFreeAgentsExcludesRostered(t *testing.T) {
	s := newStore(t, &fakeSource{rosters: baseRosters(t)})
	ctx := context.Background()

	// A rostered player with a stale FREE_AGENT event (e.g. signed after a prior cut) is NOT in the
	// pool — the roster exclusion wins. baseRosters seeds real players; pick one that exists.
	rostered := s.Reader().Franchises()
	if len(rostered) == 0 {
		t.Fatal("no seeded franchises")
	}
	roster, _ := s.Reader().Roster(rostered[0])
	if len(roster) == 0 {
		t.Fatal("seeded franchise has no players")
	}
	onRoster := roster[0].MFLID
	if err := recordStatus(t, s, onRoster, domain.PlayerFreeAgent, "stale"); err != nil {
		t.Fatalf("record stale FA on a rostered player: %v", err)
	}
	// An off-roster free agent and an off-roster retiree.
	if err := recordStatus(t, s, "7001", domain.PlayerFreeAgent, "cut"); err != nil {
		t.Fatalf("record FA 7001: %v", err)
	}
	if err := recordStatus(t, s, "7002", domain.PlayerRetired, "retired"); err != nil {
		t.Fatalf("record RETIRED 7002: %v", err)
	}
	pool, err := s.FreeAgents(ctx)
	if err != nil {
		t.Fatalf("FreeAgents: %v", err)
	}
	inPool := map[string]bool{}
	for _, id := range pool {
		inPool[id] = true
	}
	if inPool[onRoster] {
		t.Errorf("pool contains rostered player %q (roster exclusion failed)", onRoster)
	}
	if !inPool["7001"] {
		t.Error("pool missing off-roster free agent 7001")
	}
	if inPool["7002"] {
		t.Error("pool contains a RETIRED player 7002")
	}
}

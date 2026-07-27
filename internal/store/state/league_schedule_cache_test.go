package state

import (
	"context"
	"errors"
	"testing"
	"time"
)

// An empty cache must report ErrNoCachedLeagueSchedule, NOT an empty payload with a zero
// time — mirrors TestCachedStandingsEmptyIsTypedError for the same reason: the caller
// distinguishes "we have older data" (stale) from "we have nothing" (fail) on exactly this.
func TestCachedLeagueScheduleEmptyIsTypedError(t *testing.T) {
	s := newStore(t, &fakeSource{rosters: baseRosters(t)})

	_, _, err := s.CachedLeagueSchedule(context.Background())
	if !errors.Is(err, ErrNoCachedLeagueSchedule) {
		t.Fatalf("want ErrNoCachedLeagueSchedule, got %v", err)
	}
}

// A cached payload must round-trip byte-identically, and its timestamp must survive to the
// second — the UI prints the age literally, so a mangled timestamp is a lie about freshness.
func TestPutAndReadLeagueScheduleRoundTrip(t *testing.T) {
	s := newStore(t, &fakeSource{rosters: baseRosters(t)})
	ctx := context.Background()
	payload := `[{"Week":"1","Matchups":[{"Franchises":[{"FranchiseID":"0001","IsHome":"1"},{"FranchiseID":"0002","IsHome":"0"}]}]}]`
	at := time.Now().Add(-90 * time.Minute)

	if err := s.PutLeagueSchedule(ctx, payload, at); err != nil {
		t.Fatalf("PutLeagueSchedule: %v", err)
	}
	got, gotAt, err := s.CachedLeagueSchedule(ctx)
	if err != nil {
		t.Fatalf("CachedLeagueSchedule: %v", err)
	}
	if got != payload {
		t.Errorf("payload = %q, want %q", got, payload)
	}
	if d := gotAt.Sub(at.UTC()); d > time.Second || d < -time.Second {
		t.Errorf("fetched_at drifted by %v", d)
	}
}

// The cache keeps only the NEWEST copy per (league, season) — it is a cache, not a ledger.
func TestPutLeagueScheduleReplacesRatherThanAccumulates(t *testing.T) {
	s := newStore(t, &fakeSource{rosters: baseRosters(t)})
	ctx := context.Background()

	if err := s.PutLeagueSchedule(ctx, `["old"]`, time.Now().Add(-2*time.Hour)); err != nil {
		t.Fatalf("first PutLeagueSchedule: %v", err)
	}
	if err := s.PutLeagueSchedule(ctx, `["new"]`, time.Now()); err != nil {
		t.Fatalf("second PutLeagueSchedule: %v", err)
	}

	got, _, err := s.CachedLeagueSchedule(ctx)
	if err != nil {
		t.Fatalf("CachedLeagueSchedule: %v", err)
	}
	if got != `["new"]` {
		t.Errorf("payload = %q, want the newer copy", got)
	}
	var rows int
	if err := s.pools.Read().QueryRowContext(ctx,
		`SELECT COUNT(1) FROM league_schedule_cache`).Scan(&rows); err != nil {
		t.Fatalf("count: %v", err)
	}
	if rows != 1 {
		t.Errorf("league_schedule_cache holds %d rows, want exactly 1 (cache, not log)", rows)
	}
}

// An empty payload must be refused — mirrors TestPutStandingsRefusesEmptyPayload: a cache
// holding "[]" would arm a later outage with a board wearing a "CACHED" label over nothing.
func TestPutLeagueScheduleRefusesEmptyPayload(t *testing.T) {
	s := newStore(t, &fakeSource{rosters: baseRosters(t)})
	ctx := context.Background()

	for _, empty := range []string{"", "   ", "\n\t", "[]", " [] ", "null"} {
		if err := s.PutLeagueSchedule(ctx, empty, time.Now()); err == nil {
			t.Errorf("PutLeagueSchedule(%q) was accepted; want refusal", empty)
		}
	}
	if _, _, err := s.CachedLeagueSchedule(ctx); !errors.Is(err, ErrNoCachedLeagueSchedule) {
		t.Fatalf("a refused write still populated the cache: %v", err)
	}
}

// A cancelled context must fail the fallback read — mirrors
// TestCachedStandingsFailsOnCancelledContext, pinning the same B-5-derived contract: the App
// seam must derive a fresh context from the app-lifetime parent on the fallback path, never
// reuse the (likely-expired) fetch context.
func TestCachedLeagueScheduleFailsOnCancelledContext(t *testing.T) {
	s := newStore(t, &fakeSource{rosters: baseRosters(t)})
	if err := s.PutLeagueSchedule(context.Background(), `[{"Week":"1"}]`, time.Now()); err != nil {
		t.Fatalf("PutLeagueSchedule: %v", err)
	}

	dead, cancel := context.WithCancel(context.Background())
	cancel()

	if _, _, err := s.CachedLeagueSchedule(dead); err == nil {
		t.Fatal("expected a cancelled context to fail the read")
	}
	if _, _, err := s.CachedLeagueSchedule(context.Background()); err != nil {
		t.Fatalf("cached league schedule should be readable on a live context: %v", err)
	}
}

// A row whose fetched_at will not parse must ERROR rather than yield a zero time.
func TestCachedLeagueScheduleRejectsUnparseableTimestamp(t *testing.T) {
	s := newStore(t, &fakeSource{rosters: baseRosters(t)})
	ctx := context.Background()

	if _, err := s.pools.Write().ExecContext(ctx, `
INSERT INTO league_schedule_cache (league_id, season, payload, fetched_at)
VALUES (?, ?, ?, ?)`, s.leagueID, s.season, `["x"]`, "not-a-timestamp"); err != nil {
		t.Fatalf("seed bad row: %v", err)
	}

	if _, _, err := s.CachedLeagueSchedule(ctx); err == nil {
		t.Fatal("an unparseable fetched_at was served; want an error")
	}
}

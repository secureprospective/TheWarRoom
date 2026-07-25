package state

import (
	"context"
	"errors"
	"testing"
	"time"
)

// An empty cache must report ErrNoCachedStandings, NOT an empty payload with a zero time.
// The caller distinguishes "we have older data" (degrade to stale) from "we have nothing"
// (fail honestly) on exactly this, and collapsing them would show a blank board while
// claiming to be showing cached data.
func TestCachedStandingsEmptyIsTypedError(t *testing.T) {
	s := newStore(t, &fakeSource{rosters: baseRosters(t)})

	_, _, err := s.CachedStandings(context.Background())
	if !errors.Is(err, ErrNoCachedStandings) {
		t.Fatalf("want ErrNoCachedStandings, got %v", err)
	}
}

// A cached payload must round-trip byte-identically, and its timestamp must survive to the
// second — the UI prints the age literally, so a mangled timestamp is a lie about freshness.
func TestPutAndReadStandingsRoundTrip(t *testing.T) {
	s := newStore(t, &fakeSource{rosters: baseRosters(t)})
	ctx := context.Background()
	payload := `[{"FranchiseID":"0001","H2HW":"7"}]`
	at := time.Now().Add(-90 * time.Minute)

	if err := s.PutStandings(ctx, payload, at); err != nil {
		t.Fatalf("PutStandings: %v", err)
	}
	got, gotAt, err := s.CachedStandings(ctx)
	if err != nil {
		t.Fatalf("CachedStandings: %v", err)
	}
	if got != payload {
		t.Errorf("payload = %q, want %q", got, payload)
	}
	if d := gotAt.Sub(at.UTC()); d > time.Second || d < -time.Second {
		t.Errorf("fetched_at drifted by %v", d)
	}
}

// The cache keeps only the NEWEST copy per (league, season) — it is a cache, not a ledger.
// A second write must replace, not accumulate, or the table grows without bound on every
// refresh and the reader has to guess which row is current.
func TestPutStandingsReplacesRatherThanAccumulates(t *testing.T) {
	s := newStore(t, &fakeSource{rosters: baseRosters(t)})
	ctx := context.Background()

	if err := s.PutStandings(ctx, `["old"]`, time.Now().Add(-2*time.Hour)); err != nil {
		t.Fatalf("first PutStandings: %v", err)
	}
	if err := s.PutStandings(ctx, `["new"]`, time.Now()); err != nil {
		t.Fatalf("second PutStandings: %v", err)
	}

	got, _, err := s.CachedStandings(ctx)
	if err != nil {
		t.Fatalf("CachedStandings: %v", err)
	}
	if got != `["new"]` {
		t.Errorf("payload = %q, want the newer copy", got)
	}
	var rows int
	if err := s.pools.Read().QueryRowContext(ctx,
		`SELECT COUNT(1) FROM standings_cache`).Scan(&rows); err != nil {
		t.Fatalf("count: %v", err)
	}
	if rows != 1 {
		t.Errorf("standings_cache holds %d rows, want exactly 1 (cache, not log)", rows)
	}
}

// An empty payload must be refused. The fetcher already rejects a zero-franchise response
// as an MFL glitch; letting one into the cache would arm a later "stale" board with no rows
// — a blank screen wearing a freshness label, which is worse than an honest failure.
func TestPutStandingsRefusesEmptyPayload(t *testing.T) {
	s := newStore(t, &fakeSource{rosters: baseRosters(t)})
	ctx := context.Background()

	// "[]" and "null" matter as much as "": the fetcher guards them, but a cache holding
	// one would arm a later outage with a blank board wearing a "CACHED" label.
	for _, empty := range []string{"", "   ", "\n\t", "[]", " [] ", "null"} {
		if err := s.PutStandings(ctx, empty, time.Now()); err == nil {
			t.Errorf("PutStandings(%q) was accepted; want refusal", empty)
		}
	}
	if _, _, err := s.CachedStandings(ctx); !errors.Is(err, ErrNoCachedStandings) {
		t.Fatalf("a refused write still populated the cache: %v", err)
	}
}

// REGRESSION (GLM review, B-5 lead 1). The cache exists to survive a failed fetch, and the
// most common failure is a TIMEOUT — which leaves the caller's context already expired. If
// the fallback read is handed that dead context it fails instantly, and the app reports "no
// data" while a perfectly good board sits in SQLite: the cache defeated in exactly the case
// it was built for.
//
// This test pins the STORE half of that contract — a cancelled context genuinely does fail
// the read — which is precisely why the caller (App.standingsOrCache) must derive a fresh
// context from the app-lifetime parent rather than reusing the fetch's. If this test ever
// starts passing with a cancelled context, the guarantee moved and the caller's fix needs
// re-checking.
func TestCachedStandingsFailsOnCancelledContext(t *testing.T) {
	s := newStore(t, &fakeSource{rosters: baseRosters(t)})
	if err := s.PutStandings(context.Background(), `[{"FranchiseID":"0001"}]`, time.Now()); err != nil {
		t.Fatalf("PutStandings: %v", err)
	}

	dead, cancel := context.WithCancel(context.Background())
	cancel() // simulate the expired per-call context of a timed-out fetch

	if _, _, err := s.CachedStandings(dead); err == nil {
		t.Fatal("expected a cancelled context to fail the read; the caller relies on this " +
			"being true, which is why it must pass a FRESH context on the fallback path")
	}
	// And the same read succeeds on a live context — proving the data was there all along.
	if _, _, err := s.CachedStandings(context.Background()); err != nil {
		t.Fatalf("cached standings should be readable on a live context: %v", err)
	}
}

// A row whose fetched_at will not parse must ERROR rather than yield a zero time. An
// unlabelled stale board reads as current — the exact failure this contract exists to
// prevent — so refusing to serve it is the correct degradation.
func TestCachedStandingsRejectsUnparseableTimestamp(t *testing.T) {
	s := newStore(t, &fakeSource{rosters: baseRosters(t)})
	ctx := context.Background()

	if _, err := s.pools.Write().ExecContext(ctx, `
INSERT INTO standings_cache (league_id, season, payload, fetched_at)
VALUES (?, ?, ?, ?)`, s.leagueID, s.season, `["x"]`, "not-a-timestamp"); err != nil {
		t.Fatalf("seed bad row: %v", err)
	}

	if _, _, err := s.CachedStandings(ctx); err == nil {
		t.Fatal("an unparseable fetched_at was served; want an error")
	}
}

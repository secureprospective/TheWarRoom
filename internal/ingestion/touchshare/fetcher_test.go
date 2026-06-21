package touchshare

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

// xwalk is the default pfr->gsis fixture map: two distinct players. A func (not a
// package var) keeps the test free of globals (gochecknoglobals) and hands each
// caller its own copy.
func xwalk() map[string]string {
	return map[string]string{
		"Smit00": "00-0011000",
		"Jone00": "00-0022000",
	}
}

func serve(t *testing.T, status int, body string, m map[string]string) (map[string]RawTouchShare, error) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return Fetch(context.Background(), srv.Client(), srv.URL, m)
}

// Header carries the four bound columns among unbound ones in a mixed order — to
// prove by-name binding survives column order and that defense/st snap fields are
// simply never read.
const header = "game_id,season,pfr_player_id,position,offense_snaps,offense_pct,defense_snaps,player\n"

func TestFetch_HappyPathAggregates(t *testing.T) {
	t.Parallel()
	body := header +
		// player A: two active games -> snaps 70, pctSum 1.10, active 2
		"g1,2024,Smit00,RB,40,0.60,0,A\n" +
		"g2,2024,Smit00,RB,30,0.50,0,A\n" +
		// player B: one active game + one 0-snap game (skipped from the active count)
		"g1,2024,Jone00,RB,20,0.30,0,B\n" +
		"g3,2024,Jone00,RB,0,0,55,B\n"
	m, err := serve(t, http.StatusOK, body, xwalk())
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if len(m) != 2 {
		t.Fatalf("got %d records, want 2", len(m))
	}

	a := m["00-0011000"]
	if a.OffenseSnaps != 70 || a.GamesActive != 2 || a.OffensePctSum != 1.10 || a.Season != 2024 {
		t.Errorf("player A aggregate wrong: %+v", a)
	}
	b := m["00-0022000"]
	if b.OffenseSnaps != 20 || b.GamesActive != 1 || b.OffensePctSum != 0.30 {
		t.Errorf("player B aggregate wrong (0-snap game must not count): %+v", b)
	}
}

// A pfr id absent from the crosswalk is an ordinary non-match — skipped, not an
// error. Here only the resolvable player survives.
func TestFetch_UnresolvedPfrSkipped(t *testing.T) {
	t.Parallel()
	body := header +
		"g1,2024,Smit00,RB,40,0.60,0,A\n" +
		"g1,2024,Ghost9,RB,99,0.99,0,Z\n" // Ghost9 not in xwalk
	m, err := serve(t, http.StatusOK, body, xwalk())
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if len(m) != 1 {
		t.Fatalf("got %d, want 1 (unresolved pfr must be skipped)", len(m))
	}
	if _, ok := m["00-0011000"]; !ok {
		t.Error("resolvable player missing")
	}
}

func TestFetch_MissingPfrCellSkipped(t *testing.T) {
	t.Parallel()
	body := header +
		"g1,2024,Smit00,RB,40,0.60,0,A\n" +
		"g1,2024,,RB,15,0.20,0,?\n" + // empty pfr
		"g1,2024,NA,RB,15,0.20,0,?\n" // NA pfr
	m, err := serve(t, http.StatusOK, body, xwalk())
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if len(m) != 1 {
		t.Fatalf("got %d, want 1 (missing pfr rows skipped)", len(m))
	}
}

func TestFetch_GateViolations(t *testing.T) {
	t.Parallel()
	// Two distinct pfr ids both mapping to one gsis — the double-count collision.
	collide := map[string]string{"Smit00": "00-0011000", "Twin00": "00-0011000"}

	cases := []struct {
		name string
		body string
		m    map[string]string
	}{
		{"missing required column", "season,pfr_player_id,offense_snaps\n2024,Smit00,40\n", xwalk()},
		{"malformed offense_snaps", header + "g1,2024,Smit00,RB,xx,0.5,0,A\n", xwalk()},
		{"malformed offense_pct", header + "g1,2024,Smit00,RB,40,bad,0,A\n", xwalk()},
		{"malformed season", header + "g1,202X,Smit00,RB,40,0.5,0,A\n", xwalk()},
		{"cross-pfr collision on one gsis", header +
			"g1,2024,Smit00,RB,40,0.6,0,A\n" +
			"g1,2024,Twin00,RB,30,0.5,0,A\n", collide},
		{"two seasons for one player", header +
			"g1,2024,Smit00,RB,40,0.6,0,A\n" +
			"g9,2023,Smit00,RB,30,0.5,0,A\n", xwalk()},
		{"ragged row", header + "g1,2024,Smit00\n", xwalk()},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if _, err := serve(t, http.StatusOK, tc.body, tc.m); err == nil {
				t.Fatalf("%s: expected error, got nil", tc.name)
			}
		})
	}
}

func TestFetch_EmptyIsSentinel(t *testing.T) {
	t.Parallel()
	// Only an unresolvable row -> zero resolved records -> errEmpty.
	body := header + "g1,2024,Ghost9,RB,40,0.6,0,Z\n"
	_, err := serve(t, http.StatusOK, body, xwalk())
	if !errors.Is(err, errEmpty) {
		t.Fatalf("err = %v, want errEmpty", err)
	}
}

func TestFetch_OnlyInactiveGamesIsEmpty(t *testing.T) {
	t.Parallel()
	// A resolvable player whose only rows are 0-snap games has no active game.
	body := header + "g1,2024,Smit00,RB,0,0,30,A\n"
	_, err := serve(t, http.StatusOK, body, xwalk())
	if !errors.Is(err, errEmpty) {
		t.Fatalf("err = %v, want errEmpty (no active games)", err)
	}
}

func TestFetch_Non200(t *testing.T) {
	t.Parallel()
	if _, err := serve(t, http.StatusInternalServerError, "boom", xwalk()); err == nil {
		t.Fatal("expected error on 500, got nil")
	}
}

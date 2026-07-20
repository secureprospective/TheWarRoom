package assembly

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/playerid"
	"github.com/secureprospective/TheWarRoom/internal/scouting"
)

// cfbdTeamsFixture is a CFBD /teams response covering one school per tier the
// join needs: a Power-Four exact name (Alabama/SEC), a Power-Four ALIAS target
// (Miami/ACC — the alias for MFL's "Miami (FL)"), a Group-of-Five alias target
// (App State/Sun Belt — alias for "Appalachian State"), and an FCS school.
const cfbdTeamsFixture = `[
  {"school":"Alabama","conference":"SEC","classification":"fbs"},
  {"school":"Miami","conference":"ACC","classification":"fbs"},
  {"school":"App State","conference":"Sun Belt","classification":"fbs"},
  {"school":"Montana","conference":"Big Sky","classification":"fcs"}
]`

// fakeSchoolLookup is a map-backed SchoolLookup for tests — the same shape the
// app wires over a normalize.Lookup (mfl id → raw MFL college name).
type fakeSchoolLookup map[string]string

func (f fakeSchoolLookup) College(mflID string) (string, bool) {
	c, ok := f[mflID]
	return c, ok
}

func teamsServer(t *testing.T, body string, status int) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv
}

// TestBuildSchoolTier_EndToEnd proves the fetch→join pipeline: an exact-name
// match, two alias-reconciled matches (the whole reason the alias map exists),
// and a college CFBD does not classify (drops to a neutral miss).
func TestBuildSchoolTier_EndToEnd(t *testing.T) {
	srv := teamsServer(t, cfbdTeamsFixture, http.StatusOK)

	roster := []string{"1001", "1002", "1003", "1004"}
	sl := fakeSchoolLookup{
		"1001": "Alabama",            // exact → Power Four
		"1002": "Miami (FL)",         // alias → "Miami" → Power Four
		"1003": "Appalachian State",  // alias → "App State" → Group of Five
		"1004": "Grand Valley State", // D2 — CFBD does not classify → neutral miss
	}

	out, err := BuildSchoolTier(context.Background(), srv.Client(), srv.URL, "test-key", 2026, roster, sl)
	if err != nil {
		t.Fatalf("BuildSchoolTier: %v", err)
	}
	if len(out) != 3 {
		t.Fatalf("want 3 classified players (1004 is an unclassified miss), got %d: %+v", len(out), out)
	}

	want := map[string]scouting.SchoolTier{
		"1001": scouting.SchoolPowerFour,
		"1002": scouting.SchoolPowerFour,
		"1003": scouting.SchoolGroupOfFive,
	}
	for mfl, tier := range want {
		pid, _ := playerid.New(mfl)
		got, ok := out[pid]
		if !ok {
			t.Fatalf("player %s missing from result (want %s)", mfl, tier)
		}
		if got != tier {
			t.Fatalf("player %s: got tier %q, want %q", mfl, got, tier)
		}
	}
	// The unclassified player must be ABSENT, not present-as-SchoolUnset — absence
	// is how the rankings side reads "no school-tier signal → neutral".
	id1004, _ := playerid.New("1004")
	if _, ok := out[id1004]; ok {
		t.Fatalf("unclassified college must be absent from the map, got %q", out[id1004])
	}
}

// TestBuildSchoolTier_PlayerMissesAreOrdinary: a player with no college (lookup
// miss), an empty college string, and a malformed mfl id all drop quietly — the
// fetched feed is healthy, so these are player-level misses, never an error.
func TestBuildSchoolTier_PlayerMissesAreOrdinary(t *testing.T) {
	srv := teamsServer(t, cfbdTeamsFixture, http.StatusOK)

	roster := []string{"1001", "1002", "1003", "10X4"}
	sl := fakeSchoolLookup{
		"1001": "Alabama", // resolves
		"1002": "",        // present in lookup but empty college → miss
		// 1003 absent from lookup entirely → miss
		"10X4": "Alabama", // malformed id → miss (never reaches the college)
	}

	out, err := BuildSchoolTier(context.Background(), srv.Client(), srv.URL, "test-key", 2026, roster, sl)
	if err != nil {
		t.Fatalf("player-level misses must NOT error: %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("want exactly 1 classified player (1001), got %d: %+v", len(out), out)
	}
	id1001, _ := playerid.New("1001")
	if out[id1001] != scouting.SchoolPowerFour {
		t.Fatalf("1001: got %q, want Power Four", out[id1001])
	}
}

// TestBuildSchoolTier_FetchFailsLoud: a fetch that classifies zero schools (empty
// CFBD array → the fetcher's errEmpty) surfaces as an error — a school-tier-less
// league is visible, not silent, matching BuildRAS's posture.
func TestBuildSchoolTier_FetchFailsLoud(t *testing.T) {
	srv := teamsServer(t, `[]`, http.StatusOK)
	if _, err := BuildSchoolTier(context.Background(), srv.Client(), srv.URL, "k", 2026, []string{"1001"}, fakeSchoolLookup{"1001": "Alabama"}); err == nil {
		t.Fatal("a zero-school CFBD response should fail loud")
	}
}

// TestBuildSchoolTier_Non200FailsLoud: a non-200 from CFBD (e.g. 401 without a
// valid key) is a genuine fetch failure and must error, not return empty.
func TestBuildSchoolTier_Non200FailsLoud(t *testing.T) {
	srv := teamsServer(t, `{"error":"unauthorized"}`, http.StatusUnauthorized)
	if _, err := BuildSchoolTier(context.Background(), srv.Client(), srv.URL, "bad-key", 2026, []string{"1001"}, fakeSchoolLookup{"1001": "Alabama"}); err == nil {
		t.Fatal("a non-200 CFBD response should fail loud")
	}
}

// TestBuildSchoolTier_NilGuards: a nil dependency is a wiring error, never a
// silent no-signal league.
func TestBuildSchoolTier_NilGuards(t *testing.T) {
	if _, err := BuildSchoolTier(context.Background(), nil, "u", "k", 2026, nil, fakeSchoolLookup{}); err == nil {
		t.Fatal("nil client should error")
	}
	if _, err := BuildSchoolTier(context.Background(), http.DefaultClient, "u", "k", 2026, nil, nil); err == nil {
		t.Fatal("nil SchoolLookup should error")
	}
}

// TestCollegeAliases_NoIdentityMappings: an alias whose target equals its key is
// dead weight (exact match already handles it) and signals a copy-paste slip.
func TestCollegeAliases_NoIdentityMappings(t *testing.T) {
	for mflName, cfbdName := range collegeAliases() {
		if mflName == cfbdName {
			t.Errorf("alias %q → %q is an identity mapping (exact match already covers it)", mflName, cfbdName)
		}
		if cfbdName == "" {
			t.Errorf("alias %q maps to an empty CFBD name", mflName)
		}
	}
}

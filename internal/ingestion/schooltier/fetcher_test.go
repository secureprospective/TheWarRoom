package schooltier

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/scouting"
)

const teamsFixture = `[
  {"school":"Georgia","conference":"SEC","classification":"fbs"},
  {"school":"Memphis","conference":"American Athletic","classification":"fbs"},
  {"school":"Oregon State","conference":"Pac-12","classification":"fbs"},
  {"school":"Notre Dame","conference":"FBS Independents","classification":"fbs"},
  {"school":"UConn","conference":"FBS Independents","classification":"fbs"},
  {"school":"Montana","conference":"Big Sky","classification":"fcs"},
  {"school":"Grand Valley State","conference":"GLIAC","classification":"ii"},
  {"school":"Mystery U","conference":"???","classification":"club"}
]`

// serveTeams returns a client + URL serving body, asserting the bearer token arrived.
func serveTeams(t *testing.T, wantKey, body string) (*http.Client, string) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer "+wantKey {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv.Client(), srv.URL
}

func TestFetch_ClassifiesAndKeysBySchool(t *testing.T) {
	t.Parallel()
	client, url := serveTeams(t, "k123", teamsFixture)

	// year 2024: Pac-12 is past the power cutoff -> Group of Five.
	out, err := Fetch(context.Background(), client, url, "k123", 2024)
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}

	want := map[string]scouting.SchoolTier{
		"Georgia":            scouting.SchoolPowerFour,   // SEC
		"Memphis":            scouting.SchoolGroupOfFive, // AAC
		"Oregon State":       scouting.SchoolGroupOfFive, // Pac-12 in 2024 (post-cutoff)
		"Notre Dame":         scouting.SchoolPowerFour,   // independent special-case
		"UConn":              scouting.SchoolGroupOfFive, // other independent
		"Montana":            scouting.SchoolFCS,
		"Grand Valley State": scouting.SchoolNonFCS, // D-II
	}
	for school, wantTier := range want {
		if got := out[school]; got != wantTier {
			t.Errorf("%s: tier = %q, want %q", school, got, wantTier)
		}
	}
	// Unknown classification is skipped, not emitted as unset.
	if _, present := out["Mystery U"]; present {
		t.Errorf("Mystery U (unknown classification) should be skipped, got %q", out["Mystery U"])
	}
	if len(out) != len(want) {
		t.Errorf("emitted %d schools, want %d", len(out), len(want))
	}
}

func TestFetch_Pac12YearAware(t *testing.T) {
	t.Parallel()
	client, url := serveTeams(t, "k", teamsFixture)

	// 2022: Pac-12 was still a power conference.
	out, err := Fetch(context.Background(), client, url, "k", 2022)
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if got := out["Oregon State"]; got != scouting.SchoolPowerFour {
		t.Errorf("Oregon State in 2022: tier = %q, want POWER_FOUR", got)
	}
}

func TestFetch_BadKeyFailsLoud(t *testing.T) {
	t.Parallel()
	client, url := serveTeams(t, "right", teamsFixture)
	if _, err := Fetch(context.Background(), client, url, "wrong", 2024); err == nil {
		t.Fatal("expected a non-200 error on a bad bearer token")
	}
}

func TestFetch_EmptyFailsLoud(t *testing.T) {
	t.Parallel()
	client, url := serveTeams(t, "k", `[]`)
	if _, err := Fetch(context.Background(), client, url, "k", 2024); err == nil {
		t.Fatal("expected errEmpty on a zero-team response")
	}
}

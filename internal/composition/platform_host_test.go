package composition

import (
	"net/http"
	"net/http/httptest"
	"testing"

	marketapp "github.com/secureprospective/TheWarRoom/internal/app/market"
	raceapp "github.com/secureprospective/TheWarRoom/internal/app/race"

	"github.com/secureprospective/TheWarRoom/internal/app/core"
)

func TestPhotoRouteWaitsForStartup(t *testing.T) {
	t.Parallel()
	a := &App{marketHeld: &marketapp.Held{}, raceHeld: &raceapp.Held{}, core: &core.State{}}
	rec := httptest.NewRecorder()
	a.photoHandler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/headshot/0501", nil))
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("before startup: %d, want 503", rec.Code)
	}
}

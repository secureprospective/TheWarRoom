package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/app/core"
)

func TestPhotoViewIsGoWordedAndOffMeansNoAddress(t *testing.T) {
	t.Parallel()
	on := photoView("0501", "Kayden McDonald", false)
	if on.Src != "/headshot/0501" || on.Alt != "Photo of Kayden McDonald" {
		t.Fatalf("photos on: %+v", on)
	}
	if off := photoView("0501", "Kayden McDonald", true); off.Src != "" || off.Alt != "Kayden McDonald" {
		t.Fatalf("photos off must carry no address: %+v", off)
	}
}

func TestPhotoRouteWaitsForStartup(t *testing.T) {
	t.Parallel()
	a := &App{core: &core.State{}}
	rec := httptest.NewRecorder()
	a.photoHandler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/headshot/0501", nil))
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("before startup: %d, want 503", rec.Code)
	}
}

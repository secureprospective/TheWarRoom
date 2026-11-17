package serverhttp

import (
	"crypto/tls"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Each file's stale backup pauses only its own kind of write; a join claim needs both.
func TestBackupAdmissionIsolatesTheTwoFiles(t *testing.T) {
	stale := func() error { return errors.New("paused") }
	fresh := func() error { return nil }
	for _, tc := range []struct {
		path            string
		league, signIns func() error
		admitted        bool
	}{
		{"/v1/native/draft/pick", stale, fresh, false},
		{"/v1/native/draft/pick", fresh, stale, true},
		{LoginPath, stale, fresh, true},
		{LoginPath, fresh, stale, false},
		{PasswordChangePath, stale, fresh, true},
		{JoinClaimPath, fresh, stale, false},
		{JoinClaimPath, stale, fresh, false},
		{JoinClaimPath, fresh, fresh, true},
	} {
		h := Hosted{WriteAdmission: tc.league, SignInAdmission: tc.signIns}
		reached := false
		handler := h.backupAdmission(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { reached = true }))
		r := httptest.NewRequest(http.MethodPost, "https://league.example"+tc.path, nil)
		r.TLS = &tls.ConnectionState{}
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		if reached != tc.admitted || (!tc.admitted && w.Code != http.StatusServiceUnavailable) {
			t.Errorf("%s league=%v signIns=%v: reached=%v code=%d", tc.path, tc.league() == nil, tc.signIns() == nil, reached, w.Code)
		}
	}
}

package serverhttp

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/secureprospective/TheWarRoom/internal/accounts"
)

const EnrollmentPath = "/v1/auth/enroll"

type RawEnrollmentRequest struct {
	Protocol   int                    `json:"protocol"`
	Enrollment accounts.RawEnrollment `json:"enrollment"`
}

func (r RawEnrollmentRequest) Validate() error {
	if r.Protocol != 1 {
		return fmt.Errorf("unsupported enrollment protocol")
	}
	if err := r.Enrollment.Validate(); err != nil {
		return fmt.Errorf("validate enrollment: %w", err)
	}
	return nil
}

// EnrollmentHandler redeems personally handed codes in hosted mode, never fixture mode.
// Issuance stays local and enrollment alone grants no session or league rights.
func EnrollmentHandler(service *accounts.EnrollmentService) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		if service == nil {
			http.Error(w, "enrollment unavailable", http.StatusServiceUnavailable)
			return
		}
		if r.TLS == nil || r.Header.Get("Origin") != "" || r.URL.RawQuery != "" {
			http.Error(w, "enrollment request refused", http.StatusBadRequest)
			return
		}
		if r.URL.Path != EnrollmentPath {
			http.NotFound(w, r)
			return
		}
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", "POST")
			http.Error(w, "use POST to enroll", http.StatusMethodNotAllowed)
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, 8<<10)
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		var raw RawEnrollmentRequest
		if err := decoder.Decode(&raw); err != nil {
			http.Error(w, "invalid enrollment request", http.StatusBadRequest)
			return
		}
		if err := raw.Validate(); err != nil {
			http.Error(w, "invalid enrollment request", http.StatusBadRequest)
			return
		}
		var trailing json.RawMessage
		if err := decoder.Decode(&trailing); err != io.EOF {
			http.Error(w, "invalid enrollment request", http.StatusBadRequest)
			return
		}
		account, err := service.Redeem(r.Context(), raw.Enrollment)
		if passwordBusy(w, err) {
			return
		}
		if err != nil {
			http.Error(w, "enrollment refused; ask the commissioner for a new code", http.StatusForbidden)
			return
		}
		body, err := json.Marshal(struct {
			Protocol int              `json:"protocol"`
			Account  accounts.Account `json:"account"`
		}{Protocol: 1, Account: account})
		if err != nil {
			http.Error(w, "enrollment response failed", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		if _, err := w.Write(body); err != nil {
			return
		}
	})
}

package serverhttp

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/secureprospective/TheWarRoom/internal/accounts"
)

const RecoveryPath = "/v1/auth/recover"

type RawRecoveryRequest struct {
	Protocol int    `json:"protocol"`
	Username string `json:"username"`
	Code     string `json:"code"`
	Password string `json:"password"`
}

func (r RawRecoveryRequest) String() string   { return "[recovery request redacted]" }
func (r RawRecoveryRequest) GoString() string { return r.String() }
func (r RawRecoveryRequest) Validate() error {
	if r.Protocol != 1 {
		return fmt.Errorf("unsupported recovery protocol")
	}
	if err := (accounts.RawRecovery{Username: r.Username, Code: r.Code, Password: r.Password}).Validate(); err != nil {
		return fmt.Errorf("invalid recovery: %w", err)
	}
	return nil
}

type RecoveryResponse struct {
	Protocol int `json:"protocol"`
}

// Recovery is authenticated by the one-use code, not by a lost device session.
func (h Hosted) recoverAccount(w http.ResponseWriter, r *http.Request) {
	if h.Recovery == nil {
		http.Error(w, "recovery unavailable", http.StatusServiceUnavailable)
		return
	}
	var raw RawRecoveryRequest
	if r.Method != http.MethodPost || decodeHosted(w, r, &raw) != nil || raw.Validate() != nil {
		http.Error(w, "invalid recovery request", http.StatusBadRequest)
		return
	}
	err := h.Recovery.Recover(r.Context(), accounts.RawRecovery{Username: raw.Username, Code: raw.Code, Password: raw.Password})
	if errors.Is(err, accounts.ErrPasswordBusy) {
		w.Header().Set("Retry-After", "1")
		http.Error(w, "password service busy", http.StatusTooManyRequests)
		return
	}
	if err != nil {
		http.Error(w, "recovery refused", http.StatusUnauthorized)
		return
	}
	writeHosted(w, RecoveryResponse{Protocol: 1})
}

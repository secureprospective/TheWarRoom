package serverhttp

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/accounts"
)

const RevokePath = "/v1/auth/revoke"

type RawRevoke struct {
	Protocol   int  `json:"protocol"`
	AllDevices bool `json:"allDevices"`
}

func (r RawRevoke) Validate() error {
	if r.Protocol != 1 {
		return fmt.Errorf("unsupported revoke protocol")
	}
	return nil
}

type RevokeResponse struct {
	Protocol int `json:"protocol"`
}

func (r RevokeResponse) Validate() error {
	if r.Protocol != 1 {
		return fmt.Errorf("invalid revoke response")
	}
	return nil
}

func (h Hosted) revoke(w http.ResponseWriter, r *http.Request) {
	var raw RawRevoke
	if r.Method != http.MethodPost || r.Header.Get("Content-Type") != "application/json" || decodeHosted(w, r, &raw) != nil || raw.Validate() != nil {
		http.Error(w, "invalid revoke request", http.StatusBadRequest)
		return
	}
	authorization := r.Header.Get("Authorization")
	if len(r.Header.Values("Authorization")) != 1 || !strings.HasPrefix(authorization, "Bearer ") {
		http.Error(w, "session required", http.StatusUnauthorized)
		return
	}
	if err := h.Sessions.Revoke(r.Context(), accounts.RawSession{Bearer: strings.TrimPrefix(authorization, "Bearer ")}, raw.AllDevices); err != nil {
		http.Error(w, "session refused", http.StatusUnauthorized)
		return
	}
	writeHosted(w, RevokeResponse{Protocol: 1})
}

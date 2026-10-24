package serverhttp

import (
	"fmt"
	"net/http"

	"github.com/secureprospective/TheWarRoom/internal/accounts"
)

const (
	TOTPBeginPath   = "/v1/auth/totp/begin"
	TOTPConfirmPath = "/v1/auth/totp/confirm"
	TOTPStepUpPath  = "/v1/auth/totp/step-up"
)

type RawAuthenticator struct {
	Protocol int    `json:"protocol"`
	Username string `json:"username"`
	Password string `json:"password"`
	Code     string `json:"code,omitempty"`
}

func (r RawAuthenticator) String() string   { return "[authenticator request redacted]" }
func (r RawAuthenticator) GoString() string { return r.String() }
func (r RawAuthenticator) Validate() error {
	if r.Protocol != 1 {
		return fmt.Errorf("unsupported protocol")
	}
	if err := (accounts.RawCredentials{Username: r.Username, Password: r.Password}).Validate(); err != nil {
		return fmt.Errorf("authenticator credentials: %w", err)
	}
	if r.Code != "" {
		if err := (accounts.RawTOTP{Code: r.Code}).Validate(); err != nil {
			return fmt.Errorf("authenticator code: %w", err)
		}
	}
	return nil
}

type AuthenticatorResponse struct {
	Protocol   int    `json:"protocol"`
	Seed       string `json:"seed,omitempty"`
	RecentAuth string `json:"recent_auth,omitempty"`
}

func (r AuthenticatorResponse) String() string   { return "[authenticator response redacted]" }
func (r AuthenticatorResponse) GoString() string { return r.String() }

func (h Hosted) authenticator(w http.ResponseWriter, r *http.Request) {
	var raw RawAuthenticator
	if r.Method != http.MethodPost || decodeHosted(w, r, &raw) != nil || raw.Validate() != nil {
		http.Error(w, "invalid authenticator request", http.StatusBadRequest)
		return
	}
	credentials := accounts.RawCredentials{Username: raw.Username, Password: raw.Password}
	response := AuthenticatorResponse{Protocol: 1}
	var err error
	switch r.URL.Path {
	case TOTPBeginPath:
		if raw.Code != "" {
			http.Error(w, "setup takes no code", http.StatusBadRequest)
			return
		}
		var setup accounts.TOTPEnrollment
		setup, err = h.TOTP.Begin(r.Context(), credentials)
		response.Seed = setup.Seed
	case TOTPConfirmPath:
		err = h.TOTP.Confirm(r.Context(), credentials, accounts.RawTOTP{Code: raw.Code})
	case TOTPStepUpPath:
		var proof accounts.RecentAuth
		proof, err = h.TOTP.StepUp(r.Context(), credentials, accounts.RawTOTP{Code: raw.Code})
		response.RecentAuth = proof.Token
	}
	if passwordBusy(w, err) {
		return
	}
	if err != nil {
		http.Error(w, "authenticator refused", http.StatusForbidden)
		return
	}
	writeHosted(w, response)
}

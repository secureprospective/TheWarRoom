package serverhttp

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/secureprospective/TheWarRoom/internal/accounts"
)

// RawPasswordChangeRequest proves the current password and carries the replacement. No bearer is
// needed: knowing the current password is the authentication for this one account change. Code is
// required only for an account that has turned on an authenticator.
type RawPasswordChangeRequest struct {
	Protocol    int    `json:"protocol"`
	Username    string `json:"username"`
	OldPassword string `json:"oldPassword"`
	NewPassword string `json:"newPassword"`
	Code        string `json:"code,omitempty"`
}

func (r RawPasswordChangeRequest) String() string   { return "[password change request redacted]" }
func (r RawPasswordChangeRequest) GoString() string { return r.String() }

func (r RawPasswordChangeRequest) Validate() error {
	if r.Protocol != 1 {
		return fmt.Errorf("unsupported password change protocol")
	}
	if err := (accounts.RawPasswordChange{Username: r.Username, OldPassword: r.OldPassword, NewPassword: r.NewPassword, Code: r.Code}).Validate(); err != nil {
		return fmt.Errorf("invalid password change: %w", err)
	}
	return nil
}

// PasswordChangeResponse is sent once the stored password and every earlier session have moved.
type PasswordChangeResponse struct {
	Protocol int `json:"protocol"`
}

func (h Hosted) changePassword(w http.ResponseWriter, r *http.Request) {
	if h.Passwords == nil {
		http.Error(w, "password change unavailable", http.StatusServiceUnavailable)
		return
	}
	var raw RawPasswordChangeRequest
	if r.Method != http.MethodPost || decodeHosted(w, r, &raw) != nil || raw.Validate() != nil {
		http.Error(w, "invalid password change request", http.StatusBadRequest)
		return
	}
	var authenticator accounts.PasswordChangeAuthenticator
	if h.TOTP != nil {
		authenticator = h.TOTP
	}
	err := h.Passwords.ChangePassword(r.Context(), accounts.RawPasswordChange{
		Username: raw.Username, OldPassword: raw.OldPassword, NewPassword: raw.NewPassword, Code: raw.Code}, authenticator)
	if passwordBusy(w, err) {
		return
	}
	if errors.Is(err, accounts.ErrInvalidCredentials) {
		http.Error(w, "current password is wrong", http.StatusUnauthorized)
		return
	}
	if errors.Is(err, accounts.ErrTOTPRequired) {
		http.Error(w, "authenticator code is missing or wrong", http.StatusForbidden)
		return
	}
	if err != nil {
		http.Error(w, "password change refused; try again", http.StatusServiceUnavailable)
		return
	}
	writeHosted(w, PasswordChangeResponse{Protocol: 1})
}

package serverhttp

import (
	"fmt"
	"net/http"

	"github.com/secureprospective/TheWarRoom/internal/accounts"
)

const RecoveryCodesPath = "/v1/auth/recovery-codes"

// RawRecoveryCodesRequest asks for a new set of printed recovery codes. The recent
// authenticator proof travels in X-TWR-Recent-Auth, as for AI helper connections.
type RawRecoveryCodesRequest struct {
	Protocol int    `json:"protocol"`
	Username string `json:"username"`
	Password string `json:"password"`
}

func (r RawRecoveryCodesRequest) String() string   { return "[recovery codes request redacted]" }
func (r RawRecoveryCodesRequest) GoString() string { return r.String() }
func (r RawRecoveryCodesRequest) Validate() error {
	if r.Protocol != 1 {
		return fmt.Errorf("unsupported recovery codes protocol")
	}
	if err := (accounts.RawCredentials{Username: r.Username, Password: r.Password}).Validate(); err != nil {
		return fmt.Errorf("recovery codes credentials: %w", err)
	}
	return nil
}

// RecoveryCodesResponse carries the new codes once; the server keeps only their digests.
type RecoveryCodesResponse struct {
	Protocol int      `json:"protocol"`
	Codes    []string `json:"codes"`
}

func (r RecoveryCodesResponse) Validate() error {
	if r.Protocol != 1 || len(r.Codes) != accounts.RecoveryCodeCount {
		return fmt.Errorf("invalid recovery codes response")
	}
	seen := make(map[string]bool, len(r.Codes))
	for _, code := range r.Codes {
		if code == "" || seen[code] {
			return fmt.Errorf("invalid recovery codes response")
		}
		seen[code] = true
	}
	return nil
}

func (r RecoveryCodesResponse) String() string   { return "[recovery codes redacted]" }
func (r RecoveryCodesResponse) GoString() string { return r.String() }

// printRecoveryCodes replaces every unused code. It needs the password and a fresh authenticator proof.
func (h Hosted) printRecoveryCodes(w http.ResponseWriter, r *http.Request) {
	if h.Recovery == nil {
		http.Error(w, "recovery unavailable", http.StatusServiceUnavailable)
		return
	}
	var raw RawRecoveryCodesRequest
	if r.Method != http.MethodPost || decodeHosted(w, r, &raw) != nil || raw.Validate() != nil {
		http.Error(w, "invalid recovery codes request", http.StatusBadRequest)
		return
	}
	proof := r.Header.Values("X-TWR-Recent-Auth")
	if len(proof) != 1 {
		http.Error(w, "recent authentication required", http.StatusForbidden)
		return
	}
	codes, err := h.Recovery.PrintCodes(r.Context(), accounts.RawCredentials{Username: raw.Username, Password: raw.Password},
		accounts.RecentAuth{Token: proof[0]})
	if passwordBusy(w, err) {
		return
	}
	if err != nil {
		http.Error(w, "recovery codes refused", http.StatusForbidden)
		return
	}
	writeHosted(w, RecoveryCodesResponse{Protocol: 1, Codes: codes.Codes[:]})
}

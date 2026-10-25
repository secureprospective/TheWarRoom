package serverhttp

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/accounts"
	"github.com/secureprospective/TheWarRoom/internal/commissioner"
)

const ConfirmPath = "/v1/native/confirm"

// Confirmations signs short-lived consent to one exact command, issuer and audit ID.
// Restarting the server invalidates outstanding consent; a token cannot mint it.
type Confirmations struct{ key [32]byte }

func NewConfirmations() (*Confirmations, error) {
	c := &Confirmations{}
	if _, err := rand.Read(c.key[:]); err != nil {
		return nil, fmt.Errorf("confirmation signing key: %w", err)
	}
	return c, nil
}

func (c *Confirmations) sign(expiry, id, principal string, raw commissioner.RawCommand) (string, error) {
	if c == nil {
		return "", fmt.Errorf("human confirmation is unavailable")
	}
	prepared, err := raw.Prepared()
	if err != nil {
		return "", fmt.Errorf("confirmation command: %w", err)
	}
	canonical, err := prepared.Canonical(principal)
	if err != nil {
		return "", fmt.Errorf("confirmation command: %w", err)
	}
	mac := hmac.New(sha256.New, c.key[:])
	_, _ = fmt.Fprintf(mac, "%s\x00%s\x00%s", expiry, id, canonical)
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil)), nil
}

func (c *Confirmations) check(proof, id, principal string, raw commissioner.RawCommand) error {
	parts := strings.Split(proof, ".")
	if len(parts) != 2 {
		return fmt.Errorf("a human must confirm this exact change")
	}
	expiry, err := strconv.ParseInt(parts[0], 10, 64)
	now := time.Now().Unix()
	if err != nil || expiry <= now || expiry > now+300 {
		return fmt.Errorf("human confirmation expired")
	}
	signature, err := c.sign(parts[0], id, principal, raw)
	if err != nil {
		return err
	}
	if !hmac.Equal([]byte(signature), []byte(parts[1])) {
		return fmt.Errorf("a human must confirm this exact change")
	}
	return nil
}

type ConfirmationResponse struct {
	Confirmation string `json:"confirmation"`
}

func (ConfirmationResponse) String() string     { return "[human confirmation redacted]" }
func (c ConfirmationResponse) GoString() string { return c.String() }

func (h Hosted) confirm(w http.ResponseWriter, r *http.Request, gate accountAdmission, raw RawNativeRequest) {
	if raw.Command == nil || (accounts.RawSession{Bearer: raw.TokenID}).Validate() != nil || h.Confirmations == nil {
		http.Error(w, "invalid confirmation request", http.StatusBadRequest)
		return
	}
	principal := gate.host.Principal
	access := commissioner.Access{Principal: principal, League: raw.Command.League, Operation: raw.Command.Command, Mode: "preview", Raw: *raw.Command}
	if err := gate.Admit(r.Context(), access); err != nil {
		http.Error(w, "human confirmation refused", http.StatusForbidden)
		return
	}
	// Confirming always needs recent human authentication, even if league roles changed.
	if remaining, limited, err := gate.proofRemaining(r.Context(), raw.Command.League); err != nil || !limited || remaining <= 0 {
		http.Error(w, "human confirmation refused", http.StatusForbidden)
		return
	}
	expiry := strconv.FormatInt(time.Now().Add(5*time.Minute).Unix(), 10)
	signature, err := h.Confirmations.sign(expiry, raw.TokenID, principal, *raw.Command)
	if err != nil {
		http.Error(w, "human confirmation unavailable", http.StatusBadRequest)
		return
	}
	writeHosted(w, ConfirmationResponse{Confirmation: expiry + "." + signature})
}

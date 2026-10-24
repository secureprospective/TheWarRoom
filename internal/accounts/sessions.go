package accounts

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
)

var ErrSessionRefused = errors.New("your session has expired or was revoked")

const (
	SessionIdleLifetime     = 7 * 24 * time.Hour
	SessionAbsoluteLifetime = 30 * 24 * time.Hour
)

type RawSession struct {
	Bearer string `json:"-"`
}

func (r RawSession) String() string   { return "[device session redacted]" }
func (r RawSession) GoString() string { return r.String() }
func (r RawSession) Validate() error {
	secret, err := base64.RawURLEncoding.Strict().DecodeString(r.Bearer)
	if err != nil || len(secret) != 32 || len(r.Bearer) != 43 {
		return ErrSessionRefused
	}
	return nil
}

type DeviceSession struct {
	RawSession
	Device    string    `json:"device"`
	Account   Account   `json:"account"`
	ExpiresAt time.Time `json:"expires_at"`
}

func (s DeviceSession) String() string   { return "[device session redacted]" }
func (s DeviceSession) GoString() string { return s.String() }

type SessionRepository interface {
	CreateSession(ctx context.Context, credential Credential, digest, device string) (time.Time, error)
	SessionAccount(ctx context.Context, digest string, touch bool) (Account, error)
	RevokeSession(ctx context.Context, digest string, allDevices bool) error
}

type SessionService struct {
	passwords *Service
	store     SessionRepository
}

func NewSessions(passwords *Service, store SessionRepository) *SessionService {
	return &SessionService{passwords: passwords, store: store}
}

// Login binds issuance to the password hash actually verified, fencing concurrent recovery.
// Privileged league access still requires the separate current-policy TOTP gate.
func (s *SessionService) Login(ctx context.Context, raw RawCredentials, device string) (DeviceSession, error) {
	if !ValidDeviceName(device) {
		return DeviceSession{}, ErrSessionRefused
	}
	credential, err := s.passwords.authenticateCredential(ctx, raw)
	if err != nil {
		return DeviceSession{}, err
	}
	var secret [32]byte
	if _, err := rand.Read(secret[:]); err != nil {
		return DeviceSession{}, fmt.Errorf("generate device session: %w", err)
	}
	session := DeviceSession{RawSession: RawSession{Bearer: base64.RawURLEncoding.EncodeToString(secret[:])}, Device: device, Account: credential.Account}
	session.ExpiresAt, err = s.store.CreateSession(ctx, credential, enrollmentDigest(session.Bearer), device)
	if err != nil {
		return DeviceSession{}, fmt.Errorf("issue device session: %w", err)
	}
	return session, nil
}

func ValidDeviceName(device string) bool {
	if len(device) < 1 || len(device) > 64 {
		return false
	}
	for _, c := range device {
		if c < 32 || c > 126 {
			return false
		}
	}
	return true
}

// Authenticate records activity at server time. Call before entering the ledger writer;
// admission below is read-only and rechecks expiry/revocation without reentering it.
func (s *SessionService) Authenticate(ctx context.Context, raw RawSession) (Account, error) {
	if err := raw.Validate(); err != nil {
		return Account{}, err
	}
	account, err := s.store.SessionAccount(ctx, enrollmentDigest(raw.Bearer), true)
	if err != nil {
		return Account{}, fmt.Errorf("authenticate device session: %w", err)
	}
	return account, nil
}
func (s *SessionService) Revoke(ctx context.Context, raw RawSession, allDevices bool) error {
	if err := raw.Validate(); err != nil {
		return err
	}
	if err := s.store.RevokeSession(ctx, enrollmentDigest(raw.Bearer), allDevices); err != nil {
		return fmt.Errorf("revoke device session: %w", err)
	}
	return nil
}

// Admission checks only device identity. League grants, scope and privileged recent
// authentication remain separate mandatory gates in hosted composition.
func (s *SessionService) Admission(raw RawSession) commissioner.CredentialAdmission {
	return sessionAdmission{service: s, raw: raw}
}

type sessionAdmission struct {
	service *SessionService
	raw     RawSession
}

func (a sessionAdmission) Admit(ctx context.Context, access commissioner.Access) error {
	if err := a.raw.Validate(); err != nil {
		return err
	}
	account, err := a.service.store.SessionAccount(ctx, enrollmentDigest(a.raw.Bearer), false)
	if err != nil {
		return fmt.Errorf("check device session: %w", err)
	}
	principal, err := account.Principal()
	if err != nil || string(principal) != access.Principal {
		return ErrSessionRefused
	}
	return nil
}

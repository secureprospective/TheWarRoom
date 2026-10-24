package accounts

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
)

var ErrRecoveryRefused = errors.New("account recovery refused")

const RecoveryCodeCount = 8

type RecoveryCodes struct {
	Codes [RecoveryCodeCount]string `json:"-"`
}

func (r RecoveryCodes) String() string   { return "[recovery codes redacted]" }
func (r RecoveryCodes) GoString() string { return r.String() }

type RawRecovery struct {
	Username string `json:"username"`
	Code     string `json:"code"`
	Password string `json:"password"`
}

func (r RawRecovery) String() string               { return "[account recovery redacted]" }
func (r RawRecovery) GoString() string             { return r.String() }
func (r RawRecovery) MarshalJSON() ([]byte, error) { return []byte(`{"redacted":true}`), nil }
func (r RawRecovery) Validate() error {
	code, err := base64.RawURLEncoding.Strict().DecodeString(r.Code)
	if err != nil || len(code) != 32 || len(r.Code) != 43 {
		return ErrRecoveryRefused
	}
	return (RawCredentials{Username: r.Username, Password: r.Password}).Validate()
}

type RawConsoleRecovery struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Reason   string `json:"reason"`
}

func (r RawConsoleRecovery) String() string               { return "[console recovery redacted]" }
func (r RawConsoleRecovery) GoString() string             { return r.String() }
func (r RawConsoleRecovery) MarshalJSON() ([]byte, error) { return []byte(`{"redacted":true}`), nil }
func (r RawConsoleRecovery) Validate() error {
	if len(r.Reason) < 1 || len(r.Reason) > 256 {
		return ErrRecoveryRefused
	}
	for _, c := range r.Reason {
		if c < 32 || c > 126 {
			return ErrRecoveryRefused
		}
	}
	return (RawCredentials{Username: r.Username, Password: r.Password}).Validate()
}

type RecoveryRepository interface {
	ReplaceRecoveryCodes(ctx context.Context, accountID, proofDigest string, digests [RecoveryCodeCount]string) error
	ResetAccount(ctx context.Context, username, passwordHash, codeDigest, reason string) error
}

type RecoveryService struct {
	passwords        *Service
	store            RecoveryRepository
	offlineKeyDigest [32]byte
}

// The pinned digest is trusted operator configuration, not part of a reset request.
func NewRecovery(passwords *Service, store RecoveryRepository, offlineKeyDigest [32]byte) *RecoveryService {
	return &RecoveryService{passwords: passwords, store: store, offlineKeyDigest: offlineKeyDigest}
}

// PrintCodes returns eight codes once, after password and recent TOTP admission.
func (s *RecoveryService) PrintCodes(ctx context.Context, credentials RawCredentials, proof RecentAuth) (RecoveryCodes, error) {
	account, err := s.passwords.Authenticate(ctx, credentials)
	if err != nil {
		return RecoveryCodes{}, err
	}
	if len(proof.Token) != 43 {
		return RecoveryCodes{}, ErrRecoveryRefused
	}
	var codes RecoveryCodes
	var digests [RecoveryCodeCount]string
	for i := range codes.Codes {
		var secret [32]byte
		if _, err := rand.Read(secret[:]); err != nil {
			return RecoveryCodes{}, fmt.Errorf("generate recovery code: %w", err)
		}
		codes.Codes[i] = base64.RawURLEncoding.EncodeToString(secret[:])
		digests[i] = enrollmentDigest(codes.Codes[i])
	}
	if err := s.store.ReplaceRecoveryCodes(ctx, account.ID, enrollmentDigest(proof.Token), digests); err != nil {
		return RecoveryCodes{}, fmt.Errorf("print recovery codes: %w", err)
	}
	return codes, nil
}

// Recover replaces a forgotten password and removes lost factors. A recovery code
// is account authority, never league/seat authority. No recent-auth proof is granted.
func (s *RecoveryService) Recover(ctx context.Context, raw RawRecovery) error {
	if err := raw.Validate(); err != nil {
		return ErrRecoveryRefused
	}
	return s.reset(ctx, raw.Username, raw.Password, enrollmentDigest(raw.Code), "")
}

func (s *RecoveryService) ConsoleRecover(ctx context.Context, raw RawConsoleRecovery, offlineKey []byte) error {
	if err := raw.Validate(); err != nil {
		return ErrRecoveryRefused
	}
	digest := sha256.Sum256(offlineKey)
	if len(offlineKey) != 32 || subtle.ConstantTimeCompare(digest[:], s.offlineKeyDigest[:]) != 1 {
		return ErrRecoveryRefused
	}
	return s.reset(ctx, raw.Username, raw.Password, "", raw.Reason)
}

func (s *RecoveryService) reset(ctx context.Context, username, password, codeDigest, reason string) error {
	canonical, err := CanonicalUsername(username)
	if err != nil {
		return ErrRecoveryRefused
	}
	hash, err := HashPassword(password)
	if err != nil {
		return err
	}
	if err := s.store.ResetAccount(ctx, canonical, hash, codeDigest, reason); err != nil {
		return fmt.Errorf("reset account: %w", err)
	}
	return nil
}

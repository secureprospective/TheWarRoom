package accounts

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1" //nolint:gosec // RFC 6238 HMAC-SHA1 authenticator compatibility, not collision resistance.
	"crypto/subtle"
	"encoding/base32"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/governance"
)

var ErrTOTPRequired = errors.New("set up your authenticator and authenticate again")

const (
	TOTPEnrollmentLifetime = 10 * time.Minute
	RecentAuthLifetime     = 5 * time.Minute
)

// RawTOTP is the untrusted six-digit authenticator input.
type RawTOTP struct {
	Code string `json:"code"`
}

func (r RawTOTP) String() string               { return "[authenticator code redacted]" }
func (r RawTOTP) GoString() string             { return r.String() }
func (r RawTOTP) MarshalJSON() ([]byte, error) { return []byte(`{"redacted":true}`), nil }
func (r RawTOTP) Validate() error {
	if len(r.Code) != 6 {
		return ErrTOTPRequired
	}
	for _, c := range r.Code {
		if c < '0' || c > '9' {
			return ErrTOTPRequired
		}
	}
	return nil
}

// TOTPEnrollment exposes the base32 seed only for the authenticated account's setup UI.
type TOTPEnrollment struct {
	Seed string `json:"-"`
}

func (e TOTPEnrollment) String() string   { return "[authenticator enrollment redacted]" }
func (e TOTPEnrollment) GoString() string { return e.String() }

// RecentAuth is an opaque account-bound proof, not a session or a league grant.
type RecentAuth struct {
	Token string `json:"-"`
}

func (a RecentAuth) String() string   { return "[recent authentication redacted]" }
func (a RecentAuth) GoString() string { return a.String() }

// TOTPState contains only encrypted seed material. Now is the repository's server clock.
type TOTPState struct {
	SealedSeed []byte `json:"-"`
	Confirmed  bool
	Now        time.Time
}

func (s TOTPState) String() string   { return "[stored authenticator redacted]" }
func (s TOTPState) GoString() string { return s.String() }

type TOTPRepository interface {
	BeginTOTP(context.Context, string, []byte) error
	TOTP(context.Context, string) (TOTPState, error)
	ConsumeTOTP(ctx context.Context, accountID string, sealedSeed []byte, counter int64, confirming bool, proofDigest string) error
	RecentAuthRemaining(ctx context.Context, accountID, proofDigest string) (time.Duration, error)
}

type TOTPService struct {
	passwords *Service
	store     TOTPRepository
	cipher    cipher.AEAD
}

// NewTOTP requires an externally supplied AES-256 key; it is never stored with the seeds.
func NewTOTP(passwords *Service, store TOTPRepository, key []byte) (*TOTPService, error) {
	if passwords == nil || store == nil || len(key) != 32 {
		return nil, fmt.Errorf("authenticator requires accounts, storage and a 32-byte encryption key")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("authenticator encryption: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("authenticator encryption: %w", err)
	}
	return &TOTPService{passwords: passwords, store: store, cipher: aead}, nil
}

func (s *TOTPService) Begin(ctx context.Context, credentials RawCredentials) (TOTPEnrollment, error) {
	account, err := s.passwords.Authenticate(ctx, credentials)
	if err != nil {
		return TOTPEnrollment{}, err
	}
	seed := make([]byte, 20)
	if _, err := rand.Read(seed); err != nil {
		return TOTPEnrollment{}, fmt.Errorf("generate authenticator seed: %w", err)
	}
	nonce := make([]byte, s.cipher.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return TOTPEnrollment{}, fmt.Errorf("generate authenticator nonce: %w", err)
	}
	sealed := s.cipher.Seal(nonce, nonce, seed, []byte(account.ID))
	if err := s.store.BeginTOTP(ctx, account.ID, sealed); err != nil {
		return TOTPEnrollment{}, fmt.Errorf("begin authenticator enrollment: %w", err)
	}
	return TOTPEnrollment{Seed: base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(seed)}, nil
}

// Confirm consumes the first code but grants no step-up proof; use a fresh code to StepUp.
func (s *TOTPService) Confirm(ctx context.Context, credentials RawCredentials, raw RawTOTP) error {
	_, err := s.verify(ctx, credentials, raw, true)
	return err
}

func (s *TOTPService) StepUp(ctx context.Context, credentials RawCredentials, raw RawTOTP) (RecentAuth, error) {
	return s.verify(ctx, credentials, raw, false)
}

func (s *TOTPService) verify(ctx context.Context, credentials RawCredentials, raw RawTOTP, confirming bool) (RecentAuth, error) {
	if err := raw.Validate(); err != nil {
		return RecentAuth{}, err
	}
	account, err := s.passwords.Authenticate(ctx, credentials)
	if err != nil {
		return RecentAuth{}, err
	}
	state, err := s.store.TOTP(ctx, account.ID)
	if err != nil {
		return RecentAuth{}, fmt.Errorf("read authenticator: %w", err)
	}
	if state.Confirmed == confirming {
		return RecentAuth{}, ErrTOTPRequired
	}
	seed, err := s.open(account.ID, state.SealedSeed)
	if err != nil {
		return RecentAuth{}, err
	}
	counter := matchingCounter(seed, raw.Code, state.Now)
	if counter < 0 {
		return RecentAuth{}, ErrTOTPRequired
	}
	var proof RecentAuth
	if !confirming {
		var token [32]byte
		if _, err := rand.Read(token[:]); err != nil {
			return RecentAuth{}, fmt.Errorf("generate recent authentication: %w", err)
		}
		proof.Token = base64.RawURLEncoding.EncodeToString(token[:])
	}
	digest := ""
	if !confirming {
		digest = enrollmentDigest(proof.Token)
	}
	if err := s.store.ConsumeTOTP(ctx, account.ID, state.SealedSeed, counter, confirming, digest); err != nil {
		return RecentAuth{}, fmt.Errorf("consume authenticator code: %w", err)
	}
	return proof, nil
}

func (s *TOTPService) open(accountID string, sealed []byte) ([]byte, error) {
	size := s.cipher.NonceSize()
	if len(sealed) != size+20+s.cipher.Overhead() {
		return nil, ErrTOTPRequired
	}
	seed, err := s.cipher.Open(nil, sealed[:size], sealed[size:], []byte(accountID))
	if err != nil {
		return nil, ErrTOTPRequired
	}
	return seed, nil
}

// CheckPrivileged is a credential gate, not domain authorization. The caller supplies
// current league policy and canonical identity from trusted admission, never request fields.
func (s *TOTPService) CheckPrivileged(ctx context.Context, policy governance.State, principal governance.Principal, proof RecentAuth) error {
	_, _, err := s.PrivilegedRemaining(ctx, policy, principal, proof)
	return err
}

// PrivilegedRemaining checks like CheckPrivileged and says how long the proof has left. limited is
// false for ordinary accounts, whose access never depended on a proof.
func (s *TOTPService) PrivilegedRemaining(ctx context.Context, policy governance.State, principal governance.Principal,
	proof RecentAuth) (remaining time.Duration, limited bool, err error) {
	if !policy.HasRole(principal, governance.Commissioner) && !policy.HasRole(principal, governance.CoCommissioner) {
		return 0, false, nil
	}
	id := strings.TrimPrefix(string(principal), "acct.")
	canonical, err := governance.AccountPrincipal(id)
	if err != nil || canonical != principal || len(proof.Token) != 43 {
		return 0, true, ErrTOTPRequired
	}
	remaining, err = s.store.RecentAuthRemaining(ctx, id, enrollmentDigest(proof.Token))
	if err != nil {
		return 0, true, fmt.Errorf("recent authentication: %w", err)
	}
	return remaining, true, nil
}

// ValidTOTPCounter is rechecked after writer acquisition so a queued code cannot age in.
func ValidTOTPCounter(counter int64, now time.Time) bool {
	current := now.Unix() / 30
	return counter >= 0 && current > 0 && counter >= current-1 && counter <= current+1
}

func matchingCounter(seed []byte, code string, now time.Time) int64 {
	current := now.Unix() / 30
	matched := int64(-1)
	for counter := current - 1; counter <= current+1; counter++ {
		if counter >= 0 && subtle.ConstantTimeCompare([]byte(totpCode(seed, counter)), []byte(code)) == 1 {
			matched = counter
		}
	}
	return matched
}

func totpCode(seed []byte, counter int64) string {
	if counter < 0 {
		return ""
	}
	var message [8]byte
	binary.BigEndian.PutUint64(message[:], uint64(counter))
	mac := hmac.New(sha1.New, seed)
	_, _ = mac.Write(message[:]) // hash.Hash.Write never fails.
	sum := mac.Sum(nil)
	offset := sum[len(sum)-1] & 15
	value := binary.BigEndian.Uint32(sum[offset:offset+4]) & 0x7fffffff
	return fmt.Sprintf("%06d", value%1000000)
}

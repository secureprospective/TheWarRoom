package accounts

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/governance"
)

var ErrEnrollmentRefused = errors.New("enrollment code is invalid, expired, used or intended for a different username")

const EnrollmentLifetime = 48 * time.Hour

// EnrollmentAuthority supplies current league policy and identity from a trusted local adapter.
// Never populate either value from enrollment request fields.
type EnrollmentAuthority interface {
	EnrollmentPolicy(ctx context.Context, league string) (governance.State, governance.Principal, error)
}

type EnrollmentRepository interface {
	IssueEnrollment(ctx context.Context, digest, username, league string, issuer governance.Principal) (time.Time, error)
	RedeemEnrollment(ctx context.Context, digest string, credential Credential) error
}

type EnrollmentService struct {
	store     EnrollmentRepository
	authority EnrollmentAuthority
	work      *PasswordWork
}

func NewEnrollment(store EnrollmentRepository, authority EnrollmentAuthority) *EnrollmentService {
	return NewEnrollmentWithPasswordWork(store, authority, NewPasswordWork())
}

func NewEnrollmentWithPasswordWork(store EnrollmentRepository, authority EnrollmentAuthority, work *PasswordWork) *EnrollmentService {
	if work == nil {
		work = NewPasswordWork()
	}
	return &EnrollmentService{store: store, authority: authority, work: work}
}

// EnrollmentCode is returned only to the local issuer for personal handoff, never in a league event.
type EnrollmentCode struct {
	Code      string    `json:"-"`
	ExpiresAt time.Time `json:"expiresAt"`
}

func (e EnrollmentCode) String() string   { return "[enrollment code redacted]" }
func (e EnrollmentCode) GoString() string { return e.String() }

// Issue is local-only until authenticated commissioner routes are integrated.
// Username binds the handoff to an intended login identity, not to a league seat.
func (s *EnrollmentService) Issue(ctx context.Context, league, username string) (EnrollmentCode, error) {
	canonical, err := CanonicalUsername(username)
	if err != nil {
		return EnrollmentCode{}, err
	}
	if league == "" || len(league) > 128 || s.authority == nil {
		return EnrollmentCode{}, fmt.Errorf("a current commissioner is required to issue enrollment codes")
	}
	policy, issuer, err := s.authority.EnrollmentPolicy(ctx, league)
	if err != nil {
		return EnrollmentCode{}, fmt.Errorf("enrollment issuer: %w", err)
	}
	if !policy.HasRole(issuer, governance.Commissioner) {
		return EnrollmentCode{}, fmt.Errorf("only the league commissioner may issue enrollment codes")
	}
	canonicalIssuer, err := governance.AccountPrincipal(strings.TrimPrefix(string(issuer), "acct."))
	if err != nil || canonicalIssuer != issuer {
		return EnrollmentCode{}, fmt.Errorf("enrollment requires a canonical account issuer")
	}
	var secret [32]byte
	if _, err := rand.Read(secret[:]); err != nil {
		return EnrollmentCode{}, fmt.Errorf("generate enrollment code: %w", err)
	}
	code := base64.RawURLEncoding.EncodeToString(secret[:])
	expires, err := s.store.IssueEnrollment(ctx, enrollmentDigest(code), canonical, league, issuer)
	if err != nil {
		return EnrollmentCode{}, fmt.Errorf("issue enrollment: %w", err)
	}
	return EnrollmentCode{Code: code, ExpiresAt: expires}, nil
}

type RawEnrollment struct {
	Code     string `json:"code"`
	Username string `json:"username"`
	Password string `json:"password"`
}

func (r RawEnrollment) String() string               { return "[enrollment credentials redacted]" }
func (r RawEnrollment) GoString() string             { return r.String() }
func (r RawEnrollment) MarshalJSON() ([]byte, error) { return []byte(`{"redacted":true}`), nil }

func (r RawEnrollment) Validate() error {
	code, err := base64.RawURLEncoding.Strict().DecodeString(r.Code)
	if err != nil || len(code) != 32 || len(r.Code) != 43 {
		return ErrEnrollmentRefused
	}
	return (RawCredentials{Username: r.Username, Password: r.Password}).Validate()
}

func (s *EnrollmentService) Redeem(ctx context.Context, raw RawEnrollment) (Account, error) {
	if err := raw.Validate(); err != nil {
		return Account{}, ErrEnrollmentRefused
	}
	username, err := CanonicalUsername(raw.Username)
	if err != nil {
		return Account{}, ErrEnrollmentRefused
	}
	hash, err := s.work.Hash(ctx, raw.Password)
	if err != nil {
		return Account{}, err
	}
	id, err := newAccountID()
	if err != nil {
		return Account{}, err
	}
	account := Account{ID: id, Username: username}
	if err := s.store.RedeemEnrollment(ctx, enrollmentDigest(raw.Code), Credential{Account: account, PasswordHash: hash}); err != nil {
		return Account{}, fmt.Errorf("redeem enrollment: %w", err)
	}
	return account, nil
}

func enrollmentDigest(code string) string {
	digest := sha256.Sum256([]byte(code))
	return hex.EncodeToString(digest[:])
}

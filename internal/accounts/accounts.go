package accounts

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/governance"
)

var (
	ErrNotFound           = errors.New("account not found")
	ErrUsernameTaken      = errors.New("username already taken")
	ErrBootstrapClosed    = errors.New("local account bootstrap is already complete")
	ErrInvalidCredentials = errors.New("incorrect username or password")
)

type Account struct {
	ID       string `json:"id"`
	Username string `json:"username"`
}

func (a Account) Principal() (governance.Principal, error) {
	principal, err := governance.AccountPrincipal(a.ID)
	if err != nil {
		return "", fmt.Errorf("account principal: %w", err)
	}
	return principal, nil
}

type RawCredentials struct {
	Username string `json:"username"`
	Password string `json:"-"`
}

func (r RawCredentials) String() string   { return "[account credentials redacted]" }
func (r RawCredentials) GoString() string { return r.String() }

func (r RawCredentials) Validate() error {
	if _, err := CanonicalUsername(r.Username); err != nil {
		return err
	}
	return validatePassword(r.Password)
}

// CanonicalUsername makes username uniqueness ASCII case-insensitive, without trimming.
func CanonicalUsername(username string) (string, error) {
	if len(username) < 3 || len(username) > 64 {
		return "", fmt.Errorf("username must contain 3 to 64 ASCII letters, digits, dots, underscores or hyphens")
	}
	for _, c := range username {
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9', c == '.', c == '_', c == '-':
		default:
			return "", fmt.Errorf("username must contain only ASCII letters, digits, dots, underscores or hyphens")
		}
	}
	return strings.ToLower(username), nil
}

// Credential is private persistence data, not a wire response or league event.
type Credential struct {
	Account
	PasswordHash string `json:"-"`
}

func (c Credential) String() string   { return "[stored account credentials redacted]" }
func (c Credential) GoString() string { return c.String() }

func (c Credential) Validate() error {
	if _, err := c.Principal(); err != nil {
		return fmt.Errorf("account identity: %w", err)
	}
	username, err := CanonicalUsername(c.Username)
	if err != nil || username != c.Username {
		return fmt.Errorf("stored username must be canonical")
	}
	_, _, err = DecodePasswordHash(c.PasswordHash)
	return err
}

type Repository interface {
	Insert(ctx context.Context, credential Credential, onlyIfEmpty bool) error
	Credential(ctx context.Context, username string) (Credential, error)
}

type Service struct{ store Repository }

func New(store Repository) *Service { return &Service{store: store} }

// LocalBootstrap is a local operator door, never an enrollment or league-role grant.
// The store must atomically refuse it once any account exists.
func (s *Service) LocalBootstrap(ctx context.Context, raw RawCredentials) (Account, error) {
	if err := raw.Validate(); err != nil {
		return Account{}, err
	}
	username, err := CanonicalUsername(raw.Username)
	if err != nil {
		return Account{}, err
	}
	hash, err := HashPassword(raw.Password)
	if err != nil {
		return Account{}, err
	}
	id, err := newAccountID()
	if err != nil {
		return Account{}, err
	}
	account := Account{ID: id, Username: username}
	if err := s.store.Insert(ctx, Credential{Account: account, PasswordHash: hash}, true); err != nil {
		return Account{}, fmt.Errorf("bootstrap account: %w", err)
	}
	return account, nil
}

func (s *Service) Authenticate(ctx context.Context, raw RawCredentials) (Account, error) {
	credential, err := s.authenticateCredential(ctx, raw)
	return credential.Account, err
}

func (s *Service) authenticateCredential(ctx context.Context, raw RawCredentials) (Credential, error) {
	if err := raw.Validate(); err != nil {
		return Credential{}, ErrInvalidCredentials
	}
	username, err := CanonicalUsername(raw.Username)
	if err != nil {
		return Credential{}, ErrInvalidCredentials
	}
	credential, err := s.store.Credential(ctx, username)
	if errors.Is(err, ErrNotFound) {
		return Credential{}, ErrInvalidCredentials
	}
	if err != nil {
		return Credential{}, fmt.Errorf("load account: %w", err)
	}
	if err := credential.Validate(); err != nil {
		return Credential{}, fmt.Errorf("stored account: %w", err)
	}
	valid, err := VerifyPassword(credential.PasswordHash, raw.Password)
	if err != nil {
		return Credential{}, err
	}
	if !valid {
		return Credential{}, ErrInvalidCredentials
	}
	return credential, nil
}

func newAccountID() (string, error) {
	var id [16]byte
	if _, err := rand.Read(id[:]); err != nil {
		return "", fmt.Errorf("generate account identity: %w", err)
	}
	id[6] = id[6]&0x0f | 0x40
	id[8] = id[8]&0x3f | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", id[:4], id[4:6], id[6:8], id[8:10], id[10:]), nil
}

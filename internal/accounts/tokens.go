package accounts

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
)

var ErrTokenRefused = errors.New("agent token is unavailable or outside its allowed scope")

const (
	TokenReadLifetime    = 30 * 24 * time.Hour
	TokenExecuteLifetime = 24 * time.Hour
)

// RawTokenScope is also the immutable issuance ceiling. No wildcard or implicit mode expansion.
type RawTokenScope struct {
	League    string   `json:"league"`
	Franchise string   `json:"franchise"`
	Commands  []string `json:"commands"`
	Modes     []string `json:"modes"`
}

func (r RawTokenScope) Validate() error {
	if !tokenName(r.League) || !tokenName(r.Franchise) || len(r.Commands) == 0 || len(r.Commands) > 128 || len(r.Modes) == 0 || len(r.Modes) > 3 {
		return ErrTokenRefused
	}
	seen := map[string]bool{}
	for _, command := range r.Commands {
		if !tokenName(command) || seen[command] || !delegableCommand(command) {
			return ErrTokenRefused
		}
		seen[command] = true
	}
	return validateTokenModes(r.Modes)
}

func validateTokenModes(modes []string) error {
	seen := map[string]bool{}
	for _, mode := range modes {
		if (mode != "read" && mode != "preview" && mode != "execute") || seen[mode] {
			return ErrTokenRefused
		}
		seen[mode] = true
	}
	return nil
}

func tokenName(value string) bool {
	if len(value) == 0 || len(value) > 128 {
		return false
	}
	for _, c := range value {
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9', c == '.', c == '-', c == '_':
		default:
			return false
		}
	}
	return true
}

func delegableCommand(command string) bool {
	for _, prefix := range []string{"auth.", "account.", "token.", "commish.role.", "commish.seat.", "commish.abilities.", "commish.visibility.", "commish.deputy.", "commish.succession.", "commish.commissioner.", "commish.vote."} {
		if strings.HasPrefix(command, prefix) {
			return false
		}
	}
	// A generic correction could undo a role or seat change.
	if command == commissioner.Revert || command == commissioner.Create {
		return false
	}
	if strings.HasSuffix(command, ".read") || command == "contract.quote" {
		return true
	}
	_, err := commissioner.DefinitionFor(command)
	return err == nil
}

func (r RawTokenScope) Lifetime() time.Duration {
	if slices.Contains(r.Modes, "execute") {
		return TokenExecuteLifetime
	}
	return TokenReadLifetime
}

// TokenAuthority is trusted current league policy, not request-supplied metadata.
// CheckTokenScope must authorize every command/mode at issuance and on every use.
// CheckTokenAccess must resolve all Raw/player/trade/bid/pick references against the
// scoped franchise and current policy, using Access.Records inside append. Neither
// callback may write or reenter the ledger writer. Missing policy fails closed.
type TokenAuthority interface {
	CheckTokenScope(context.Context, Account, RawTokenScope) error
	CheckTokenAccess(context.Context, Account, string, commissioner.Access) error
}

type AgentToken struct {
	RawSession
	ID        string        `json:"id"`
	Account   Account       `json:"account"`
	Scope     RawTokenScope `json:"scope"`
	ExpiresAt time.Time     `json:"expires_at"`
}

func (t AgentToken) String() string   { return "[agent token redacted]" }
func (t AgentToken) GoString() string { return t.String() }

type TokenRepository interface {
	IssueToken(context.Context, string, string, string, RawTokenScope, func(Account) error) (AgentToken, error)
	Token(context.Context, string) (AgentToken, error)
	RevokeToken(context.Context, string, string) error
}

type TokenService struct {
	store     TokenRepository
	authority TokenAuthority
}

func NewTokens(store TokenRepository, authority TokenAuthority) *TokenService {
	return &TokenService{store: store, authority: authority}
}

// Issue requires a human device session; agent tokens cannot mint other credentials.
// The store rechecks that session and policy after acquiring the shared writer.
func (s *TokenService) Issue(ctx context.Context, session RawSession, scope RawTokenScope) (AgentToken, error) {
	if session.Validate() != nil || scope.Validate() != nil || s.authority == nil {
		return AgentToken{}, ErrTokenRefused
	}
	scope.Commands, scope.Modes = slices.Clone(scope.Commands), slices.Clone(scope.Modes)
	var secret, id [32]byte
	if _, err := rand.Read(secret[:]); err != nil {
		return AgentToken{}, fmt.Errorf("generate agent token: %w", err)
	}
	if _, err := rand.Read(id[:]); err != nil {
		return AgentToken{}, fmt.Errorf("generate agent identifier: %w", err)
	}
	bearer := base64.RawURLEncoding.EncodeToString(secret[:])
	token, err := s.store.IssueToken(ctx, enrollmentDigest(session.Bearer), enrollmentDigest(bearer), base64.RawURLEncoding.EncodeToString(id[:]), scope, func(account Account) error { return s.authority.CheckTokenScope(ctx, account, scope) })
	if err != nil {
		return AgentToken{}, fmt.Errorf("issue agent token: %w", err)
	}
	token.Bearer = bearer
	return token, nil
}

func (s *TokenService) Authenticate(ctx context.Context, raw RawSession) (AgentToken, error) {
	if raw.Validate() != nil {
		return AgentToken{}, ErrTokenRefused
	}
	token, err := s.store.Token(ctx, enrollmentDigest(raw.Bearer))
	if err != nil {
		return AgentToken{}, fmt.Errorf("authenticate agent token: %w", err)
	}
	return token, nil
}
func (s *TokenService) Revoke(ctx context.Context, session RawSession, id string) error {
	if session.Validate() != nil || !tokenName(id) {
		return ErrTokenRefused
	}
	if err := s.store.RevokeToken(ctx, enrollmentDigest(session.Bearer), id); err != nil {
		return fmt.Errorf("revoke agent token: %w", err)
	}
	return nil
}
func (s *TokenService) Admission(raw RawSession) commissioner.CredentialAdmission {
	return tokenAdmission{service: s, raw: raw}
}

type tokenAdmission struct {
	service *TokenService
	raw     RawSession
}

func (a tokenAdmission) Admit(ctx context.Context, access commissioner.Access) error {
	token, err := a.service.Authenticate(ctx, a.raw)
	if err != nil {
		return err
	}
	principal, err := token.Account.Principal()
	scope := token.Scope
	if err != nil || string(principal) != access.Principal || !scope.allows(access) || a.service.authority == nil {
		return ErrTokenRefused
	}
	current := RawTokenScope{League: scope.League, Franchise: scope.Franchise, Commands: []string{access.Operation}, Modes: []string{access.Mode}}
	if err := a.service.authority.CheckTokenScope(ctx, token.Account, current); err != nil {
		return fmt.Errorf("current token rights: %w", err)
	}
	if err := a.service.authority.CheckTokenAccess(ctx, token.Account, scope.Franchise, access); err != nil {
		return fmt.Errorf("token argument scope: %w", err)
	}
	return nil
}

func (r RawTokenScope) allows(access commissioner.Access) bool {
	if access.League != r.League || !slices.Contains(r.Commands, access.Operation) || !slices.Contains(r.Modes, access.Mode) {
		return false
	}
	if access.Franchise != "" && access.Franchise != r.Franchise {
		return false
	}
	if access.Raw.League != "" && access.Raw.League != r.League {
		return false
	}
	return access.Raw.Command == "" || access.Raw.Command == access.Operation
}

package accounts

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/secureprospective/TheWarRoom/internal/governance"
)

var (
	// ErrJoinRefused never says whether the league, the link or the code was wrong.
	ErrJoinRefused = errors.New("this join link is not valid; ask the commissioner for the current link")
	// ErrFranchiseTaken is the loser of a claim race, or a pick of a franchise that already has an owner.
	ErrFranchiseTaken = errors.New("that franchise already has an owner; pick another or ask the commissioner")
	// ErrJoinCredentials is a username or password the claimant must change; it reveals nothing else.
	ErrJoinCredentials = errors.New("choose a username of 3 to 64 letters, digits, dots, underscores or hyphens and a longer password")
)

// JoinRepository holds join link digests and creates the claimant's account.
type JoinRepository interface {
	SetJoinLink(ctx context.Context, league, digest string, issuer governance.Principal) error
	RevokeJoinLink(ctx context.Context, league string) error
	JoinLink(ctx context.Context, league string) (string, governance.Principal, error)
	Insert(ctx context.Context, credential Credential, onlyIfEmpty bool) error
	SignInName(ctx context.Context, accountID string) (string, error)
}

// JoinFranchise is what a holder of the join link sees of one franchise before claiming.
type JoinFranchise struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Abbreviation string `json:"abbreviation,omitempty"`
	Taken        bool   `json:"taken"`
	ClaimedBy    string `json:"claimedBy"`
	// Seat is the owner's principal, resolved into ClaimedBy and never sent to the browser.
	Seat string `json:"-"`
}

// JoinSeats is the league ledger as the join link uses it. Seat and Release record league events
// (an owner seated, an owner removed) as the issuing commissioner, through the normal checks.
type JoinSeats interface {
	Franchises(ctx context.Context, league string) ([]JoinFranchise, error)
	IsCommissioner(ctx context.Context, league string, p governance.Principal) (bool, error)
	Seat(ctx context.Context, league, franchise string, owner, issuer governance.Principal) error
	Release(ctx context.Context, league, franchise string, commissioner governance.Principal) error
}

// FUTURE-SEAM(join): every link-based sign-up (league recruiting, share-to-join) redeems through this
// one service: a revocable per-league secret, first pick binds, an ordinary owner seat, ledger events.
//
// JoinService is pick-your-team sign-up: one secret, revocable link per league; whoever holds it may
// pick a franchise with no owner and set a password. The first pick binds; the account is an ordinary
// owner of that franchise and nothing more.
type JoinService struct {
	store JoinRepository
	seats JoinSeats
	work  *PasswordWork
	// claims serializes check-then-create-then-seat in this process, so a losing claimant gets no
	// account. JoinSeats.Seat re-checks vacancy at the revision it appends at, so even claims from two
	// processes can never both be seated.
	claims sync.Mutex
}

func NewJoin(store JoinRepository, seats JoinSeats, work *PasswordWork) *JoinService {
	if work == nil {
		work = NewPasswordWork()
	}
	return &JoinService{store: store, seats: seats, work: work}
}

// JoinCode is returned only to the local console, never in a league event or log.
type JoinCode struct{ Code string }

func (j JoinCode) String() string   { return "[join code redacted]" }
func (j JoinCode) GoString() string { return j.String() }

func (s *JoinService) requireCommissioner(ctx context.Context, league string, p governance.Principal) error {
	ok, err := s.seats.IsCommissioner(ctx, league, p)
	if err != nil {
		return fmt.Errorf("league policy: %w", err)
	}
	if !ok {
		return fmt.Errorf("only the league commissioner may manage its join link")
	}
	return nil
}

// Rotate issues a new join link for league, replacing (and so revoking) any earlier one.
func (s *JoinService) Rotate(ctx context.Context, league string, commissioner governance.Principal) (JoinCode, error) {
	if err := s.requireCommissioner(ctx, league, commissioner); err != nil {
		return JoinCode{}, err
	}
	var secret [32]byte
	if _, err := rand.Read(secret[:]); err != nil {
		return JoinCode{}, fmt.Errorf("generate join link: %w", err)
	}
	code := base64.RawURLEncoding.EncodeToString(secret[:])
	if err := s.store.SetJoinLink(ctx, league, enrollmentDigest(code), commissioner); err != nil {
		return JoinCode{}, fmt.Errorf("save join link: %w", err)
	}
	return JoinCode{Code: code}, nil
}

// Revoke stops league's join link from working.
func (s *JoinService) Revoke(ctx context.Context, league string, commissioner governance.Principal) error {
	if err := s.requireCommissioner(ctx, league, commissioner); err != nil {
		return err
	}
	return s.store.RevokeJoinLink(ctx, league) //nolint:wrapcheck // the store words its own failure
}

// Release removes franchise's owner so it can be claimed again; the removal is a league event.
func (s *JoinService) Release(ctx context.Context, league, franchise string, commissioner governance.Principal) error {
	if err := s.requireCommissioner(ctx, league, commissioner); err != nil {
		return err
	}
	s.claims.Lock()
	defer s.claims.Unlock()
	if err := s.seats.Release(ctx, league, franchise, commissioner); err != nil {
		return fmt.Errorf("release franchise: %w", err)
	}
	return nil
}

// RawJoin is a join link presented by its holder.
type RawJoin struct {
	League string `json:"league"`
	Code   string `json:"code"`
}

func (r RawJoin) String() string   { return "[join link redacted]" }
func (r RawJoin) GoString() string { return r.String() }

func (r RawJoin) Validate() error {
	code, err := base64.RawURLEncoding.Strict().DecodeString(r.Code)
	if err != nil || len(code) != 32 || len(r.Code) != 43 || !strings.HasPrefix(r.League, "native.") || len(r.League) > 128 {
		return ErrJoinRefused
	}
	return nil
}

// check compares the presented code's digest with the live one in constant time.
func (s *JoinService) check(ctx context.Context, raw RawJoin) (governance.Principal, error) {
	if err := raw.Validate(); err != nil {
		return "", err
	}
	digest, issuer, err := s.store.JoinLink(ctx, raw.League)
	presented := enrollmentDigest(raw.Code)
	if err != nil {
		// Spend the same comparison on a missing link as on a wrong code.
		_ = subtle.ConstantTimeCompare([]byte(presented), []byte(strings.Repeat("0", len(presented))))
		if errors.Is(err, ErrJoinRefused) {
			return "", ErrJoinRefused
		}
		return "", fmt.Errorf("read join link: %w", err)
	}
	if subtle.ConstantTimeCompare([]byte(presented), []byte(digest)) != 1 {
		return "", ErrJoinRefused
	}
	return issuer, nil
}

// Franchises lists league's franchises and which are taken, to a holder of the live link.
func (s *JoinService) Franchises(ctx context.Context, raw RawJoin) ([]JoinFranchise, error) {
	if _, err := s.check(ctx, raw); err != nil {
		return nil, err
	}
	list, err := s.seats.Franchises(ctx, raw.League)
	if err != nil {
		return nil, fmt.Errorf("read franchises: %w", err)
	}
	for i := range list {
		id := strings.TrimPrefix(list[i].Seat, "acct.")
		if id == "" || id == list[i].Seat {
			continue
		}
		name, err := s.store.SignInName(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("read claimant sign-in name: %w", err)
		}
		list[i].ClaimedBy = name
	}
	return list, nil
}

// RawJoinClaim picks one franchise and sets the new owner's sign-in.
type RawJoinClaim struct {
	RawJoin
	Franchise string `json:"franchise"`
	Username  string `json:"username"`
	Password  string `json:"password"`
}

func (r RawJoinClaim) String() string   { return "[join claim redacted]" }
func (r RawJoinClaim) GoString() string { return r.String() }

func (r RawJoinClaim) Validate() error {
	if err := r.RawJoin.Validate(); err != nil {
		return err
	}
	if r.Franchise == "" || len(r.Franchise) > 64 {
		return ErrJoinRefused
	}
	if err := (RawCredentials{Username: r.Username, Password: r.Password}).Validate(); err != nil {
		return ErrJoinCredentials
	}
	return nil
}

// Claim creates the owner's account and seats it in franchise, if the link is live and the franchise
// has no owner. Exactly one of two simultaneous claims of a franchise wins; the other gets
// ErrFranchiseTaken and no account.
func (s *JoinService) Claim(ctx context.Context, raw RawJoinClaim) (Account, error) {
	if err := raw.Validate(); err != nil {
		return Account{}, err
	}
	issuer, err := s.check(ctx, raw.RawJoin)
	if err != nil {
		return Account{}, err
	}
	username, err := CanonicalUsername(raw.Username)
	if err != nil {
		return Account{}, ErrJoinRefused
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
	principal, err := account.Principal()
	if err != nil {
		return Account{}, fmt.Errorf("new owner identity: %w", err)
	}
	s.claims.Lock()
	defer s.claims.Unlock()
	if err := s.vacant(ctx, raw.League, raw.Franchise); err != nil {
		return Account{}, err
	}
	if err := s.store.Insert(ctx, Credential{Account: account, PasswordHash: hash}, false); err != nil {
		return Account{}, err //nolint:wrapcheck // ErrUsernameTaken is shown as is
	}
	if err := s.seats.Seat(ctx, raw.League, raw.Franchise, principal, issuer); err != nil {
		return Account{}, fmt.Errorf("seat the new owner (the account exists without a franchise; ask the commissioner): %w", err)
	}
	return account, nil
}

func (s *JoinService) vacant(ctx context.Context, league, franchise string) error {
	list, err := s.seats.Franchises(ctx, league)
	if err != nil {
		return fmt.Errorf("read franchises: %w", err)
	}
	for _, f := range list {
		if f.ID == franchise {
			if f.Taken {
				return ErrFranchiseTaken
			}
			return nil
		}
	}
	return ErrJoinRefused
}

package accounts

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/governance"
)

var ErrAliasTaken = errors.New("this computer is already linked to a different sign-in")

// Alias is one desktop identity a password-verified account has claimed.
type Alias struct {
	Alias   governance.Principal
	Account governance.Principal
}

// SeatStatus is what a desktop may show before sign-in: whether any local account exists and
// which username, if any, this computer's identity is already linked to.
type SeatStatus struct {
	HasAccount bool   `json:"hasAccount"`
	Username   string `json:"username,omitempty"`
}

type SeatRepository interface {
	BindAlias(ctx context.Context, accountID, alias string) error
	Aliases(ctx context.Context) ([]Alias, error)
	SeatStatus(ctx context.Context, alias string) (SeatStatus, error)
}

// LocalSeat keeps a desktop league identity across computers. The desktop's own principal
// is derived from its folder and OS user, so a restored copy elsewhere is a new identity.
// Signing in with the local account's password links that new identity to the account; the
// ledger then resolves every linked identity to one account through the verified-alias map.
// A copy without the password links nothing and keeps no one's role.
type LocalSeat struct {
	passwords *Service
	store     SeatRepository
}

func NewLocalSeat(passwords *Service, store SeatRepository) *LocalSeat {
	return &LocalSeat{passwords: passwords, store: store}
}

func (l *LocalSeat) Status(ctx context.Context, local string) (SeatStatus, error) {
	if err := validLocalAlias(local); err != nil {
		return SeatStatus{}, err
	}
	status, err := l.store.SeatStatus(ctx, local)
	if err != nil {
		return SeatStatus{}, fmt.Errorf("read sign-in: %w", err)
	}
	return status, nil
}

// SignIn creates the first local account, or verifies the existing one, then links local.
func (l *LocalSeat) SignIn(ctx context.Context, raw RawCredentials, local string) (Account, error) {
	if err := validLocalAlias(local); err != nil {
		return Account{}, err
	}
	status, err := l.store.SeatStatus(ctx, local)
	if err != nil {
		return Account{}, fmt.Errorf("read sign-in: %w", err)
	}
	var account Account
	if status.HasAccount {
		account, err = l.passwords.Authenticate(ctx, raw)
	} else {
		account, err = l.passwords.LocalBootstrap(ctx, raw)
	}
	if err != nil {
		return Account{}, err
	}
	if err := l.store.BindAlias(ctx, account.ID, local); err != nil {
		return Account{}, fmt.Errorf("link this computer: %w", err)
	}
	return account, nil
}

// VerifiedAliases supplies the ledger's identity map for one app league.
func (l *LocalSeat) VerifiedAliases(ctx context.Context, league string) ([]governance.VerifiedAlias, error) {
	aliases, err := l.store.Aliases(ctx)
	if err != nil {
		return nil, fmt.Errorf("read linked identities: %w", err)
	}
	out := make([]governance.VerifiedAlias, len(aliases))
	for i, a := range aliases {
		out[i] = governance.VerifiedAlias{League: league, Alias: a.Alias, Account: a.Account}
	}
	return out, nil
}

func validLocalAlias(local string) error {
	if local == "" || strings.TrimSpace(local) != local || strings.HasPrefix(local, "acct.") || len(local) > 128 {
		return fmt.Errorf("local identity is unavailable")
	}
	return nil
}

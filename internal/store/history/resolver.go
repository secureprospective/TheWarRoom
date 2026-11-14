package history

import (
	"context"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
)

// LeagueResolver finds the store holding a league. Leagues is the directory of native leagues;
// Stores is every file holding league history (each verifies its own audit chain); Commits names the
// league of each native ledger commit, coalesced. Per-league files arrive behind this seam.
type LeagueResolver interface {
	Leagues(ctx context.Context) ([]commissioner.NativeLeague, error)
	League(ctx context.Context, league string) (*Store, error)
	Stores(ctx context.Context) ([]*Store, error)
	Commits() <-chan string
}

// OneFile resolves every league to the single file both the hosted server and the desktop open today.
type OneFile struct{ store *Store }

func NewOneFile(store *Store) OneFile { return OneFile{store: store} }

func (o OneFile) Leagues(ctx context.Context) ([]commissioner.NativeLeague, error) {
	return o.store.NativeLeagues(ctx)
}

func (o OneFile) League(context.Context, string) (*Store, error) { return o.store, nil }

func (o OneFile) Stores(context.Context) ([]*Store, error) { return []*Store{o.store}, nil }

func (o OneFile) Commits() <-chan string { return o.store.commits }

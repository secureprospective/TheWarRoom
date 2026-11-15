package leagueservice

import (
	"context"
	"errors"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

// twoLeagues is a fake resolver over two leagues: one resolves to a real store, the other fails.
type twoLeagues struct {
	good   *history.Store
	mu     sync.Mutex
	asked  []string
	broken string
}

func (r *twoLeagues) Leagues(context.Context) ([]commissioner.NativeLeague, error) {
	return []commissioner.NativeLeague{{ID: r.broken}, {ID: dueLeague}}, nil
}

func (r *twoLeagues) League(_ context.Context, league string) (*history.Store, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.asked = append(r.asked, league)
	if league == r.broken {
		return nil, errors.New("league file unavailable")
	}
	return r.good, nil
}

func (r *twoLeagues) Stores(context.Context) ([]*history.Store, error) {
	return []*history.Store{r.good}, nil
}
func (r *twoLeagues) Commits() <-chan string { return nil }

// The worker asks the resolver for each league's store, and one league that cannot be reached is
// reported without stopping the other league's due work.
func TestDueWorkerResolvesEachLeagueAndIsolatesFailure(t *testing.T) {
	f := newDueFixture(t)
	resolver := &twoLeagues{good: f.store, broken: "league.broken"}
	f.worker.leagues = resolver
	_, err := f.worker.step(t.Context())
	if err == nil || !strings.Contains(err.Error(), "league.broken") || strings.Contains(err.Error(), dueLeague) {
		t.Fatalf("step error = %v", err)
	}
	if !slices.Equal(resolver.asked, []string{"league.broken", dueLeague}) {
		t.Fatalf("resolver asked for %v", resolver.asked)
	}
}

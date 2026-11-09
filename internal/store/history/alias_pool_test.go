package history

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/governance"
)

// Alias reads share the read pool. Readers holding a transaction while waiting for an alias read
// exhausted the pool and hung every screen; more concurrent readers than connections must finish.
func TestConcurrentLeagueReadsDoNotHoldAConnectionForAliases(t *testing.T) {
	f := statResultsTest(t)
	f.s.SetVerifiedAliases(func(ctx context.Context, _ string) ([]governance.VerifiedAlias, error) {
		var one int
		if err := f.s.pools.Read().QueryRowContext(ctx, `SELECT 1`).Scan(&one); err != nil {
			return nil, err
		}
		return nil, nil
	})
	ctx, cancel := context.WithTimeout(t.Context(), 20*time.Second)
	defer cancel()
	var wg sync.WaitGroup
	errs := make(chan error, 40)
	for range 40 {
		wg.Go(func() {
			if _, err := f.s.NativeAssets(ctx, "native.test"); err != nil {
				errs <- err
			}
		})
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatal(err)
	}
}

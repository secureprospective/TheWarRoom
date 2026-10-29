package tradeplans_test

import (
	"context"
	"errors"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/store/mflref/tradeplans"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

func TestIndependentStoreInitializesBeforeEveryOperation(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	pools, err := db.OpenScratch(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := pools.Close(); err != nil {
			t.Error(err)
		}
	}()
	calls := 0
	store := tradeplans.NewTradePlans(pools, func(ctx context.Context, pools *db.Pools) error {
		calls++
		return state.InitializeWhatIf(ctx, pools)
	})
	plan := tradeplans.TradePlan{ID: "saved", Mine: "0001", Other: "0002", Give: []string{"0042"}}
	if err := store.Save(ctx, plan); err != nil {
		t.Fatal(err)
	}
	saved, err := store.List(ctx, "0001")
	if err != nil || len(saved) != 1 || saved[0].Give[0] != "0042" || calls != 2 {
		t.Fatalf("plans=%+v calls=%d err=%v", saved, calls, err)
	}
}

func TestInitializationFailureCannotWriteOrReadPlans(t *testing.T) {
	t.Parallel()
	blocked := errors.New("migration refused")
	store := tradeplans.NewTradePlans(nil, func(context.Context, *db.Pools) error { return blocked })
	if err := store.Save(t.Context(), tradeplans.TradePlan{ID: "no-write"}); !errors.Is(err, blocked) {
		t.Fatal(err)
	}
	if plans, err := store.List(t.Context(), "0001"); !errors.Is(err, blocked) || plans != nil {
		t.Fatalf("plans=%v err=%v", plans, err)
	}
}

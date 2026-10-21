package commissioner

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/commissioner/commissionertest"
	commish "github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/leagueweek"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

type ledgerFixture struct {
	*API
	history  *history.Store
	session  string
	registry string
}

// The unused owner-action port remains nil: these tests must not enter a native gate or a held-data read.
type fixtureHost struct {
	SessionHost
	ActionHost
	LeagueHost
	ResultsHost
	RosterHost
	MutationHost
}

func commissionerTestApp(t *testing.T) *ledgerFixture {
	t.Helper()
	registry, err := os.ReadFile(commissionertest.RepoPath("docs/ui/endpoint-registry.csv"))
	if err != nil {
		t.Fatal(err)
	}
	store := commissionertest.History(t)
	fixture := &ledgerFixture{history: store, session: "creator", registry: string(registry)}
	fixture.API = New(fixtureHost{SessionHost: fixture, LeagueHost: store, ResultsHost: store,
		RosterHost: store, MutationHost: store})
	return fixture
}
func executeCommissionerTest(t *testing.T, app *ledgerFixture, raw commish.RawCommand) commish.Receipt {
	t.Helper()
	return commissionertest.Execute(t, app.TargetCommissionerExecute, raw)
}
func (f *ledgerFixture) Ready() error                                  { return nil }
func (f *ledgerFixture) CommissionerContext() context.Context          { return context.Background() }
func (f *ledgerFixture) CommissionerPrincipal() (string, error)        { return f.session, nil }
func (f *ledgerFixture) CommissionerAuthorization() commish.Authorizer { return nil }
func (f *ledgerFixture) CommissionerNow() time.Time                    { return time.Now() }
func (f *ledgerFixture) CommissionerZone() *time.Location              { return time.Local }
func (f *ledgerFixture) CommissionerPreviewRegistry() string           { return f.registry }
func (f *ledgerFixture) NativeFacts() nativecheck.Facts                { return nil }
func (f *ledgerFixture) NativeScheduleDates(context.Context, int, time.Time) ([]leagueweek.Week, error) {
	panic("unexpected held NFL read")
}
func (f *ledgerFixture) CommissionerRecords(ctx context.Context, league string) ([]commish.Record, error) {
	return Records(ctx, f.history, league, "calendar")
}

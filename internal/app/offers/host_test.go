package offers

import (
	"context"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/offers/offerstest"
	"github.com/secureprospective/TheWarRoom/internal/app/report"
	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/envelope"
	"github.com/secureprospective/TheWarRoom/internal/lab"
	"github.com/secureprospective/TheWarRoom/internal/leagueweek"
	"github.com/secureprospective/TheWarRoom/internal/normalize"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
	"github.com/secureprospective/TheWarRoom/internal/store/moves"
	"github.com/secureprospective/TheWarRoom/internal/store/rulebook"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

// Unused capabilities stay nil: accidentally adding an acquisition fails the isolated test.
type fakeHost struct {
	Host
	//nolint:containedctx // The fake retains the test-lifetime context for bounded API calls.
	Ctx           context.Context
	Pools         *db.Pools
	Rulebook      *rulebook.Store
	League        *state.Mirror
	Moves         *moves.Store
	Lookup        normalize.Lookup
	Week          leagueweek.Week
	WeekFetchedAt time.Time
	MovesNow      func() time.Time
	LabData       func() (*lab.Data, error)
	MovesWake     chan struct{}
	movesMu       sync.Mutex
}

type apiFixture struct {
	*API
	core *fakeHost
}

func newFakeHost(t *testing.T) *fakeHost {
	t.Helper()
	return &fakeHost{Ctx: t.Context(), LabData: lab.Load, MovesNow: time.Now, MovesWake: make(chan struct{}, 1)}
}

func bidTestApp(t *testing.T) *apiFixture {
	t.Helper()
	h := newFakeHost(t)
	p, err := db.Open(h.Ctx, filepath.Join(t.TempDir(), "offers.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := p.Close(); err != nil {
			t.Error(err)
		}
	})
	h.Pools = p
	h.Rulebook = rulebook.New(p)
	h.League = state.NewMirror(p, h.Rulebook)
	h.Moves = moves.New(p)
	for _, initialize := range []func(context.Context) error{h.League.Initialize, h.Moves.Initialize} {
		if err := initialize(h.Ctx); err != nil {
			t.Fatal(err)
		}
	}
	if err := h.Rulebook.Initialize(h.Ctx, offerstest.CapSource{}); err != nil {
		t.Fatal(err)
	}
	h.Lookup = offerstest.Lookup(t)
	h.Week = leagueweek.Week{Number: 5}
	h.WeekFetchedAt = time.Now()
	a := &apiFixture{API: New(h), core: h}
	setRosters(t, a)
	return a
}

func setRosters(t *testing.T, a *apiFixture, extra ...state.MirrorPlayer) {
	t.Helper()
	offerstest.SetRosters(a.core.Ctx, t, a.core.League, extra...)
}
func rawBid(salary string, years int) RawBidInput {
	return RawBidInput{FranchiseID: "0001", Player: "13593", Salary: salary, Years: years}
}
func ruleWords(plan BidPlan, rule string) string {
	for _, r := range plan.Rules {
		if r.Rule == rule {
			return r.Mark + ": " + r.Reason
		}
	}
	return ""
}
func actionIDs(plan BidPlan) string {
	ids := []string{}
	for _, a := range plan.Actions {
		ids = append(ids, a.ID)
	}
	return strings.Join(ids, ",")
}
func mustGet(t *testing.T, a *apiFixture, id string) envelope.Envelope {
	t.Helper()
	e, err := a.core.Moves.Get(a.core.Ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	return e
}

func (h *fakeHost) Ready() error                          { return nil }
func (h *fakeHost) OffersHistoryContext() context.Context { return h.Ctx }
func (h *fakeHost) OffersContext() context.Context        { return h.Ctx }
func (h *fakeHost) OffersBuildStamp() string              { return "dev dev" }
func (h *fakeHost) OffersZone() *time.Location            { return time.Local }
func (h *fakeHost) OffersLab() (*lab.Data, error)         { return h.LabData() }
func (h *fakeHost) OffersProvenance(d *lab.Data) snapshot.Provenance {
	return snapshot.Provenance{Source: "Lab compiled facts sha256:" + lab.DataSHA, Kind: "archive", Freshness: domain.Freshness{State: domain.FreshStale, FetchedAt: d.Built, Note: "Frozen historical input; not current league truth"}}
}
func (h *fakeHost) OffersPools() *db.Pools      { return h.Pools }
func (h *fakeHost) OffersMoves() *moves.Store   { return h.Moves }
func (h *fakeHost) OffersNow() time.Time        { return h.MovesNow() }
func (h *fakeHost) OffersChanged()              {}
func (h *fakeHost) OffersLog(envelope.Envelope) {}
func (h *fakeHost) OffersWake() {
	select {
	case h.MovesWake <- struct{}{}:
	default:
	}
}
func (h *fakeHost) OffersDirectory(context.Context) (normalize.Lookup, error) { return h.Lookup, nil }
func (h *fakeHost) OffersLookup() normalize.Lookup                            { return h.Lookup }
func (h *fakeHost) OffersRoster() state.MirrorSnapshot                        { m, _, _ := h.League.ContractsView(); return m }
func (h *fakeHost) OffersSeason() int                                         { return h.League.Season() }
func (h *fakeHost) OffersWeek() (int, bool)                                   { return h.Week.Number, !h.WeekFetchedAt.IsZero() }
func (h *fakeHost) OffersCapRoom(mine string) *domain.Money {
	limit, err := domain.ParseMoneyMillions(h.Rulebook.GetSalaryCap())
	if err != nil {
		return nil
	}
	used, ok := h.League.CapUsed(mine)
	if !ok {
		return nil
	}
	room := limit - used
	return &room
}
func (h *fakeHost) SerializeOffersMove(run func() error) error {
	h.movesMu.Lock()
	defer h.movesMu.Unlock()
	return run()
}
func (h *fakeHost) OffersMarketFacts() map[string][]report.MarketFact { return nil }
func (h *fakeHost) OffersEngineValues(context.Context) (map[string]float64, string) {
	return nil, "Engine unavailable"
}
func (h *fakeHost) OffersAssetValues(context.Context, []playerid.PlayerID, time.Time) (map[string]AssetValue, string) {
	return nil, ""
}

func TestAPIMethodCatalogHasThirteenDoors(t *testing.T) {
	if got := len(Commands()); got != 13 {
		t.Fatalf("commands = %d", got)
	}
}

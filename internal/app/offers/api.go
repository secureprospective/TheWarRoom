package offers

import (
	"context"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/catalog"
	"github.com/secureprospective/TheWarRoom/internal/app/report"
	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/envelope"
	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/lab"
	"github.com/secureprospective/TheWarRoom/internal/normalize"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
	"github.com/secureprospective/TheWarRoom/internal/store/moves"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

// Host supplies shared snapshots and preserves the whole-operation serializers.
type Host interface {
	SharedHost
	HistoryHost
	PersonalHost
	PoolHost
	BidHost
}

// SharedHost supplies the app-lifetime services and presentation context.
type SharedHost interface {
	Ready() error
	OffersContext() context.Context
	OffersBuildStamp() string
	OffersZone() *time.Location
	OffersPools() *db.Pools
	OffersDirectory(context.Context) (normalize.Lookup, error)
}

// HistoryHost reads the immutable archive without joining another feature's API.
type HistoryHost interface {
	OffersHistoryContext() context.Context
	OffersLab() (*lab.Data, error)
	OffersProvenance(*lab.Data) snapshot.Provenance
}

// PersonalHost supplies detached list inputs and the read-only MFL hand-off.
type PersonalHost interface {
	OffersSnapshotNames() ([]snapshot.Player, []snapshot.Franchise, bool)
	OffersTradeBait(context.Context) report.HeldFeed[[]ingestion.RawTradeBait]
	OffersRosterTarget(string) (envelope.Target, []string)
	OffersInjuries() (snapshot.Provenance, map[string]string)
	OpenBrowser(string)
}

// PoolHost reads the existing owners' market and asset values at their original capture points.
type PoolHost interface {
	OffersMarketFacts() map[string][]report.MarketFact
	OffersFreeAgents() report.HeldFeed[[]playerid.PlayerID]
	RefreshOffersPool(bool) error
	OffersEngineValues(context.Context) (map[string]float64, string)
	OffersAssetValues(context.Context, []playerid.PlayerID, time.Time) (map[string]AssetValue, string)
}

// BidHost retains the original serializer and local plan publication order.
type BidHost interface {
	OffersLookup() normalize.Lookup
	OffersSeason() int
	OffersRoster() state.MirrorSnapshot
	OffersCapRoom(string) *domain.Money
	OffersWeek() (int, bool)
	PlanHost
	OffersNow() time.Time
}

// PlanHost coordinates the local store, log and event while the move serializer is held.
type PlanHost interface {
	OffersMoves() *moves.Store
	OffersChanged()
	OffersLog(envelope.Envelope)
	OffersWake()
	SerializeOffersMove(func() error) error
}

type API struct{ host Host }

func New(host Host) *API { return &API{host: host} }

const readTimeout = 120 * time.Second

// AssetValue is the reader's port over the shared asset board, not its owner's cache.
type AssetValue struct {
	NowText, DynastyText   string
	NowHeld, DynastyHeld   bool
	NowValue, DynastyValue float64
}

type TeamOption struct {
	Fid  string `json:"fid"`
	Name string `json:"name"`
}

func (a *API) serializedMove(run func() (BidPlan, error)) (BidPlan, error) {
	var result BidPlan
	err := a.host.SerializeOffersMove(func() error {
		var err error
		result, err = run()
		return err
	})
	return result, err //nolint:wrapcheck // Preserve the existing operation's error text.
}

func Commands() []catalog.Command {
	return []catalog.Command{
		{ID: "bid.submit", Method: "TargetBidDraft", Version: catalog.Unset, Role: "gm,commish,admin", Gravity: "G1", Availability: "both"},
		{ID: catalog.Unset, Method: "TargetBidHandOff", Version: catalog.Unset, Role: "gm,commish,admin", Gravity: "G2", Availability: "both"},
		{ID: catalog.Unset, Method: "TargetBidPlans", Version: catalog.Unset, Role: "gm,commish,admin", Gravity: "G1", Availability: "both"},
		{ID: catalog.Unset, Method: "TargetBidPosted", Version: catalog.Unset, Role: "gm,commish,admin", Gravity: "G2", Availability: "both"},
		{ID: catalog.Unset, Method: "TargetBidPreview", Version: catalog.Unset, Role: "gm,commish,admin", Gravity: "G1", Availability: "both"},
		{ID: catalog.Unset, Method: "TargetBidRevise", Version: catalog.Unset, Role: "gm,commish,admin", Gravity: "G1", Availability: "both"},
		{ID: catalog.Unset, Method: "TargetBidWithdraw", Version: catalog.Unset, Role: "gm,commish,admin", Gravity: "G2", Availability: "both"},
		{ID: catalog.Unset, Method: "TargetOffers", Version: catalog.Unset, Role: "gm,commish,admin", Gravity: "G0", Availability: "both"},
		{ID: catalog.Unset, Method: "TargetPersonal", Version: catalog.Unset, Role: "gm,commish,admin", Gravity: "G0", Availability: "both"},
		{ID: catalog.Unset, Method: "TargetPersonalChange", Version: catalog.Unset, Role: "gm,commish,admin", Gravity: "G1", Availability: "both"},
		{ID: catalog.Unset, Method: "TargetPersonalHandOff", Version: catalog.Unset, Role: "gm,commish,admin", Gravity: "G1", Availability: "desktop"},
		{ID: catalog.Unset, Method: "TargetPool", Version: catalog.Unset, Role: "gm,commish,admin", Gravity: "G1", Availability: "both"},
		{ID: catalog.Unset, Method: "TargetPoolRefresh", Version: catalog.Unset, Role: "gm,commish,admin", Gravity: "G1", Availability: "both"},
	}
}

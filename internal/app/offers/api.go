// Package offers is the Offers feature: pending offers, the personal board, the player pool and bid plans.
package offers

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/report"
	catalog "github.com/secureprospective/TheWarRoom/internal/commandcatalog"
	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/mflref/envelope"
	"github.com/secureprospective/TheWarRoom/internal/normalize"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
	"github.com/secureprospective/TheWarRoom/internal/store/mflref/moves"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

// Host supplies shared snapshots and preserves the whole-operation serializers.
type Host interface {
	SharedHost
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

func (a *API) serializedMove(run func() (BidPlan, error)) (BidPlan, error) {
	var result BidPlan
	err := a.host.SerializeOffersMove(func() error {
		var err error
		result, err = run()
		return err
	})
	return result, err //nolint:wrapcheck // Preserve the existing operation's error text.
}

func Commands() []catalog.Command { return catalog.DesktopCommands("offers", MFLCommands()) }

// MFLCommands describes the solo owner's local desktop plans, not hosted league actions.
// Desktop custody is the permission boundary; these declarations grant no remote access.
func MFLCommands() []catalog.Command {
	return catalog.MFLCommands(catalog.MFLShapes{
		Bid:   catalog.Shape(reflect.TypeFor[RawBidInput](), "RawBidInput.Validate"),
		Plan:  catalog.Shape(reflect.TypeFor[RawBidPlanScope](), "RawBidPlanScope.Validate"),
		Pool:  catalog.Shape(reflect.TypeFor[RawPoolScope](), "RawPoolScope.Validate"),
		Owner: catalog.Shape(reflect.TypeFor[RawBidOwner](), "RawBidOwner.Validate"),
	})
}

// RawBidOwner selects the owner's plans on this computer, not a hosted franchise grant.
type RawBidOwner struct {
	FranchiseID string `json:"franchiseId"`
}

func (r RawBidOwner) Validate() error {
	if err := ingestion.FeedFranchise(r.FranchiseID); err != nil {
		return fmt.Errorf("choose your franchise: %w", err)
	}
	return nil
}

type RawBidPlanScope struct {
	ID string `json:"id"`
}

func (r RawBidPlanScope) Validate() error {
	if r.ID == "" || len(r.ID) > 128 || strings.TrimSpace(r.ID) != r.ID {
		return fmt.Errorf("choose a saved bid plan")
	}
	return nil
}

func (r RawBidInput) Validate() error {
	_, err := r.entry()
	return err
}

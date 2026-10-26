package contracts

import (
	"context"
	"fmt"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/report"
	catalog "github.com/secureprospective/TheWarRoom/internal/commandcatalog"
	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/envelope"
	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
	"github.com/secureprospective/TheWarRoom/internal/leagueweek"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
	"github.com/secureprospective/TheWarRoom/internal/store/moves"
	"github.com/secureprospective/TheWarRoom/internal/store/rulebook"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

// Host supplies shared stores, detached readings and the original operation serializers. It holds
// every lock; the API holds none. Contracts has no feature-private held state, so there is no Held.
type Host interface {
	PlanLedger
	Selector
	ContractsContext() context.Context
	ContractsBuildStamp() string
	WakeMoves()
	OpenMoveURL(context.Context, string)
	Serializer
	LeagueHost
	CrossHost
	AcquisitionHost
}

// PlanLedger is the local plan ledger: the self-locking stores, the plan clock and the change notice.
type PlanLedger interface {
	ContractsStores() (*state.Mirror, *state.Store, *db.Pools, *moves.Store, *rulebook.Store)
	ContractsNow() time.Time
	NotifyMoves(context.Context)
}

// Selector reads the held players directory after startup.
type Selector interface {
	RequireOwnFranchise(string) error
	Ready() error
	ContractsDirectory() (snapshot.Snapshot, bool)
}

// Serializer runs fn under each binding's original operation locks, in their original order.
type Serializer interface {
	SerializeContractsRead(fn func())
	SerializeContractActs(fn func())
	SerializeReleasePlans(fn func())
	SerializeWhatIf(fn func())
	SerializeMoves(fn func())
}

// LeagueHost supplies detached held league readings.
type LeagueHost interface {
	ContractsLineups() (leaguefeed.Lineups, report.Feed)
	ContractsWeek() leagueweek.Week
	ContractsLeagueHost() (string, int)
	RulesFreshness() domain.Freshness
}

// CrossHost reads other features' state; the composition root implements it over their owners.
type CrossHost interface {
	ContractsSeason() (snapshot.Sourced[leaguefeed.Lineups], error)
	RosterSnapshot(mine string) (snapshot.Snapshot, error)
	RosterFeeds(ctx context.Context, lastWeek int) (season, week Scores, byes Byes)
	ReleaseTarget() envelope.Target
	ValidateMoveURL(target string) error
}

// Stores are the self-locking stores the plans read and write; the Host owns their lifetime.
type Stores struct {
	League      *state.Mirror
	Whatif      *state.Store // the what-if league's phase
	WhatifPools *db.Pools
	Moves       *moves.Store
	Rulebook    *rulebook.Store
}

// StoresOf names the ledger's stores.
func StoresOf(h PlanLedger) Stores {
	var s Stores
	s.League, s.Whatif, s.WhatifPools, s.Moves, s.Rulebook = h.ContractsStores()
	return s
}

type API struct{ host Host }

func New(host Host) *API { return &API{host: host} }

func (a *API) stores() Stores { return StoresOf(a.host) }

// timeout bounds a binding's store work, as the root's m2Timeout and m4Timeout did.
const timeout = 120 * time.Second

// Concrete wire envelopes let Wails discover the nested feature model types.
type ContractActsReading report.Reading[[]ContractActView]
type ReleasePlansReading report.Reading[[]ReleasePlanView]
type WhatIfReading report.Reading[WhatIfView]
type PlayerContractReading report.Reading[snapshot.PlayerContract]

type ContractsReadings struct { //nolint:revive // the wire and generated model keep the root name ContractsReadings
	Table       report.Reading[snapshot.ContractTable]     `json:"table"`
	Cap         report.Reading[snapshot.CapSummary]        `json:"cap"`
	Adjustments report.Reading[snapshot.SalaryAdjustments] `json:"adjustments"`
}

// Reading is a contracts reading of value; a non-nil err makes it a failed one with err as the reason.
func Reading[T any](value T, at, source, how string, count int, err error, stamp string) report.Reading[T] {
	r := report.Reading[T]{Sourced: snapshot.Sourced[T]{Value: value, Provenance: snapshot.Provenance{Source: source, Kind: "live", Freshness: domain.Freshness{State: domain.FreshLive, FetchedAt: at, Note: "Held MFL mirror; no network or what-if state"}}}, Status: report.ReadingOK, Count: count, Threshold: 0, Baseline: "Current MFL mirror revision", Cutoff: at, Method: "Integer cents; Go-authored display", BuildStamp: stamp, HowCounted: how}
	if err != nil {
		reason := err.Error()
		r.Status, r.AbsenceReason = report.ReadingFailed, &reason
		r.Provenance.Freshness.State = domain.FreshFail
	}
	return r
}

func contractReading[T any](a *API, value T, at, source, how string, count int, err error) report.Reading[T] {
	return Reading(value, at, source, how, count, err, a.host.ContractsBuildStamp())
}

// Commands is data, not an API method: Wails must not bind the catalog itself. Only the lineup draft
// has a registry verb (lineup.set); the rest are endpoint.open navigation rows with no backend verb.
// A release hand-off opens MFL's drop page in the browser, so release plans are desktop-only.
func Commands() []catalog.Command { return catalog.DesktopCommands("contracts", nil) }

func (a *API) requirePlanFranchise(ctx context.Context, id, label string) error {
	if id == "" {
		return nil
	}
	e, err := a.stores().Moves.Get(ctx, id)
	if err != nil {
		return fmt.Errorf("%s: %w", label, err)
	}
	return a.host.RequireOwnFranchise(e.Receipt().Spec.FranchiseID) //nolint:wrapcheck // Preserve the shared authorization refusal.
}

// Package core owns application state and shared lock discipline, independent of transport and features.
package core

import (
	"context"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/report"
	"github.com/secureprospective/TheWarRoom/internal/archive"
	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/envelope"
	"github.com/secureprospective/TheWarRoom/internal/facts"
	"github.com/secureprospective/TheWarRoom/internal/headshot"
	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/draftreports"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/marketfeeds"
	"github.com/secureprospective/TheWarRoom/internal/lab"
	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
	"github.com/secureprospective/TheWarRoom/internal/leagueweek"
	"github.com/secureprospective/TheWarRoom/internal/mfl"
	"github.com/secureprospective/TheWarRoom/internal/mflkey"
	"github.com/secureprospective/TheWarRoom/internal/normalize"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
	"github.com/secureprospective/TheWarRoom/internal/store/moves"
	"github.com/secureprospective/TheWarRoom/internal/store/params"
	"github.com/secureprospective/TheWarRoom/internal/store/rulebook"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
	"github.com/secureprospective/TheWarRoom/internal/transactions"
)

type State struct {
	// ReadCommsArchive runs under FactsMu; its cache remains with the archive feature.
	ReadCommsArchive       func(context.Context) (facts.RosterWeekColumns, facts.NflGameColumns, error)
	RefreshTransactions    func(context.Context)
	CommissionerSession    commissioner.Session
	CommissionerAuthorizer commissioner.Authorizer
	WhatifMu               sync.Mutex
	KeyMu                  sync.Mutex
	KeyStore               *mflkey.Store
	KeyVerifiedAt          string
	KeyGeneration          uint64
	//nolint:containedctx // Key-change refreshes run under it; shutdown cancels it.
	WorkCtx    context.Context
	WorkCancel context.CancelFunc
	//nolint:containedctx // Wails supplies an app-lifetime context; bindings derive bounded contexts.
	Ctx         context.Context
	Paths       Paths
	Pools       *db.Pools // thewarroom.db: the MFL mirror and app settings
	HistPools   *db.Pools // history.db: everything that cannot be rebuilt
	WhatifPools *db.Pools // whatif.db: moves made in the app, apart from MFL truth (R2)
	LabData     func() (*lab.Data, error)
	Params      *params.Store
	Rulebook    *rulebook.Store
	League      *state.Mirror // the league as MFL states it; every score surface reads it
	Whatif      *state.Store  // the what-if league the transaction screens work on
	Moves       *moves.Store
	History     *history.Store
	Coordinator *transactions.Coordinator // the only holder of the what-if Writer
	MFLClient   *mfl.Client               // shared, so rate limit and host discovery are process-wide
	Fetches     *archive.Transport        // every outbound HTTP request goes through it
	Season      int                       // mirror season at startup; rollover is picked up next launch
	StartupErr  error                     // startup failure; shown in the shell through AppInfo
	Started     chan struct{}             // closed when startup returns; read the fields above through ready
	LockFile    *os.File                  // single-instance lock, held until shutdown
	Headshots   atomic.Pointer[headshot.Service]
	FactsMu     sync.Mutex
	PhotosOff   atomic.Bool

	ModelMu    sync.Mutex
	ModelDirty atomic.Bool
	LookupMu   sync.Mutex
	Lookup     normalize.Lookup
	HasLookup  bool
	LookupAt   time.Time

	TargetSnapshotMu  sync.Mutex
	TargetSnapshot    snapshot.Snapshot
	HasTargetSnapshot bool

	WeekMu             sync.Mutex
	Week               leagueweek.Week
	WeekFetchedAt      time.Time
	WeekRefreshError   string
	ClockChanged       func(context.Context)
	WeekCancel         context.CancelFunc
	WeekWorkers        sync.WaitGroup
	WorkerMu           sync.Mutex
	Stopping           bool
	Backups            *db.Backups
	DatabaseCheckStart sync.Once
	ShutdownOnce       sync.Once
	WeekStart          sync.Once // domReady fires again on a webview reload; one worker only

	SeasonMu            sync.Mutex
	SeasonCalendar      HeldFeed[[]ingestion.RawCalendarEvent]
	SeasonTradeBait     HeldFeed[[]ingestion.RawTradeBait]
	ShellFeedDays       map[string]string
	SeasonTransactions  HeldFeed[[]leaguefeed.Transaction]
	CommsBoards         [2]HeldFeed[[]ingestion.RawLeaguePost]
	SeasonLineups       HeldFeed[leaguefeed.Lineups]
	SeasonPendingTrades HeldFeed[[]leaguefeed.PendingTrade]
	SeasonInjuries      HeldFeed[[]leaguefeed.Injury]
	SeasonFreeAgents    HeldFeed[[]playerid.PlayerID]
	SeasonMarket        map[string]HeldFeed[[]marketfeeds.Feed]
	SeasonDraftResults  HeldFeed[[]draftreports.Pick]
	SeasonFuturePicks   HeldFeed[[]draftreports.FuturePick]
	ResearchArchiveMu   sync.Mutex
	LiveGameMu          sync.Mutex
	LineupsWeek         int
	SeasonChanged       func(context.Context)
	SeasonPause         func(context.Context, time.Duration) error
	ClockNow            func() time.Time
	LocalZone           *time.Location

	MovesMu      sync.Mutex
	MovesWake    chan struct{}
	MovesChanged func(context.Context)
	OpenURL      func(context.Context, string)
	MovesNow     func() time.Time
	MovesTimer   func(time.Duration) (<-chan time.Time, func())
	MovesRefresh func(context.Context, envelope.Source) error
	MovesLog     func(string)
	FallbackLog  func(string)
	MovesUnknown map[string]bool

	NewsMu           sync.Mutex
	RefreshMu        sync.Mutex   // one MFL refresh at a time
	RostersCheckedAt time.Time    // last successful roster refresh; refreshMu guards it
	RulesCheckedAt   atomic.Int64 // unix seconds of the last rules sync, 0 if none; read lock-free
	LaunchRefreshDue bool         // startup did not refresh, so domReady does

	CrosswalkMu sync.Mutex

	LoadingSignals sync.Mutex // one signals load at a time
	SignalsMu      sync.Mutex // guards signals
}

type Paths struct {
	DataDir string
}

// Runtime is implemented by the desktop shell; a server supplies its own adapter.
type Runtime interface {
	EventsEmit(context.Context, string)
	BrowserOpenURL(context.Context, string)
}

// HeldFeed is a feed's last good copy and why the latest refresh failed, if it did.
type HeldFeed[T any] struct {
	Value        T
	FetchedAt    time.Time
	RefreshError string
	ReadPending  bool
}

// state is the feed's held read state, without its value.
func (h HeldFeed[T]) FeedState() report.Feed {
	return report.Feed{FetchedAt: h.FetchedAt, RefreshError: h.RefreshError, ReadPending: h.ReadPending}
}

func (h HeldFeed[T]) Provenance(source, feed string) snapshot.Provenance {
	fresh := domain.Freshness{State: domain.FreshFail, Note: feed + " not fetched yet"}
	if !h.FetchedAt.IsZero() {
		fresh = domain.Freshness{State: domain.FreshLive, FetchedAt: h.FetchedAt.UTC().Format(time.RFC3339)}
	}
	if h.ReadPending {
		fresh.State, fresh.Note = domain.FreshStale, report.PendingRead(h.FetchedAt, time.Local)
	}
	if h.RefreshError != "" {
		fresh.Note = h.RefreshError
		if !h.FetchedAt.IsZero() {
			fresh.State = domain.FreshStale
		}
	}
	return snapshot.Provenance{Kind: "live", Source: source, Freshness: fresh}
}

func Pause(ctx context.Context, duration time.Duration) error {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return fmt.Errorf("season refresh pause: %w", ctx.Err())
	case <-timer.C:
		return nil
	}
}

func HoldFeed[T any](h *HeldFeed[T], value T, fresh domain.Freshness, err error) {
	if err != nil {
		h.RefreshError = err.Error()
		return
	}
	at, parseErr := time.Parse(time.RFC3339, fresh.FetchedAt)
	if parseErr != nil {
		h.RefreshError = "season refresh time: " + parseErr.Error()
		return
	}
	h.Value, h.FetchedAt, h.RefreshError, h.ReadPending = value, at, fresh.Note, false
}

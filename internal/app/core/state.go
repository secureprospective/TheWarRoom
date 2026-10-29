// Package core owns application state and shared lock discipline, independent of transport and features.
package core

import (
	"context"
	"net/http"
	"os"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/report"
	"github.com/secureprospective/TheWarRoom/internal/app/season"
	"github.com/secureprospective/TheWarRoom/internal/archive"
	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/facts"
	"github.com/secureprospective/TheWarRoom/internal/headshot"
	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/draftreports"
	"github.com/secureprospective/TheWarRoom/internal/lab"
	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
	"github.com/secureprospective/TheWarRoom/internal/leagueweek"
	"github.com/secureprospective/TheWarRoom/internal/mfl"
	"github.com/secureprospective/TheWarRoom/internal/mflkey"
	"github.com/secureprospective/TheWarRoom/internal/mflref/envelope"
	"github.com/secureprospective/TheWarRoom/internal/normalize"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
	"github.com/secureprospective/TheWarRoom/internal/store/mflref/moves"
	"github.com/secureprospective/TheWarRoom/internal/store/params"
	"github.com/secureprospective/TheWarRoom/internal/store/rulebook"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
	"github.com/secureprospective/TheWarRoom/internal/transactions"
)

type State struct {
	// CaptureRace takes SeasonMu before capturing private held and shared season data.
	CaptureRace      func(func() season.Held) season.Snapshot
	RefreshRaceFeeds func(context.Context)

	// RefreshMarketFeeds runs under RefreshMu; Market's held reports stay at the composition root.
	RefreshMarketFeeds func(context.Context, bool)

	ReadSourcesDatabaseChecks func() *report.Report
	ReadSourcesDirectory      func(context.Context) (normalize.Lookup, error)
	ReadSourcesSettings       func() (string, bool, error)
	RunSourcesLeagueRefresh   func(context.Context) (int, bool, bool, error)
	WriteSourcesSettings      func(string, bool) error
	// ReadCommsArchive runs under FactsMu; its cache remains with the archive feature.
	ReadCommsArchive       func(context.Context) (facts.RosterWeekColumns, facts.NflGameColumns, error)
	RefreshTransactions    func(context.Context)
	CommissionerSession    commissioner.Session
	CommissionerAuthorizer commissioner.Authorizer
	WhatifMu               sync.Mutex
	KeyMu                  sync.Mutex
	KeyStore               *mflkey.Store
	KeyVerifiedAt          string
	MyFranchise            string // guarded by KeyMu; empty means identity is unknown
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
	BuildStamp  string
	LabData     func() (*lab.Data, error)
	Params      *params.Store
	Rulebook    *rulebook.Store
	League      *state.Mirror // the league as MFL states it; every score surface reads it
	Whatif      *state.Store  // the what-if league the transaction screens work on
	Moves       *moves.Store
	History     *history.Store
	Coordinator *transactions.Coordinator // the only holder of the what-if Writer
	MFLClient   *mfl.Client               // shared, so rate limit and host discovery are process-wide
	Fetches     *archive.Transport        // every archived outbound HTTP request goes through it
	Network     http.RoundTripper         // the connection under every fetch; nil is http.DefaultTransport
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

	LoadingSignals sync.Mutex // one signals load at a time
}

type Paths struct {
	DataDir string
}

// Runtime is implemented by the desktop shell; a server supplies its own adapter.
type Runtime interface {
	EventsEmit(context.Context, string)
	BrowserOpenURL(context.Context, string)
}

// HeldFeed is the shared held-value contract; synchronization remains with State.
type HeldFeed[T any] = report.HeldFeed[T]

func Pause(ctx context.Context, duration time.Duration) error {
	return report.Pause(ctx, duration) //nolint:wrapcheck // Preserve pause cancellation context.
}

func HoldFeed[T any](h *HeldFeed[T], value T, fresh domain.Freshness, err error) {
	report.HoldFeed(h, value, fresh, err)
}

// SeasonFeeds copies the held transactions, lineups (with the week they were asked for) and
// pending trades under SeasonMu. The slices, each lineup row's players and starters are copied
// and nil stays nil; a refresh replaces the published rows, never edits them in place.
func (s *State) SeasonFeeds() (HeldFeed[[]leaguefeed.Transaction], HeldFeed[leaguefeed.Lineups], int,
	HeldFeed[[]leaguefeed.PendingTrade]) {
	s.SeasonMu.Lock()
	defer s.SeasonMu.Unlock()
	txs, lineups, trades := s.SeasonTransactions, s.SeasonLineups, s.SeasonPendingTrades
	txs.Value = slices.Clone(txs.Value)
	trades.Value = slices.Clone(trades.Value)
	lineups.Value = CloneLineups(lineups.Value)
	return txs, lineups, s.LineupsWeek, trades
}

// FallbackTimeout bounds a fallback read of the fetch archive.
const FallbackTimeout = report.FallbackTimeout

type ArchivedFeed[T any] = report.ArchivedFeed[T]

func LiveOrArchive[T any](ctx, fallback context.Context, season int, source ArchivedFeed[T], logLine func(string)) ([]T, domain.Freshness, error) {
	return report.LiveOrArchive(ctx, fallback, season, source, logLine) //nolint:wrapcheck // Preserve shared fallback errors.
}
func ReadArchivedFeed[T any](ctx context.Context, season int, source ArchivedFeed[T]) ([]T, time.Time, bool, error) {
	return report.ReadArchivedFeed(ctx, season, source) //nolint:wrapcheck // Preserve shared archive errors.
}
func SameExport(raw, export string, season int, leagueless bool) bool {
	return report.SameExport(raw, export, season, leagueless)
}

// LiveFreshness marks a successful fetch without changing its wire provenance.
func LiveFreshness(t time.Time) domain.Freshness { return report.LiveFreshness(t) }

func StaleFreshness(at time.Time, cause error) domain.Freshness {
	return report.StaleFreshness(at, cause)
}

func ArchivedOnly[T any](ctx context.Context, s *State, export string, leagueless bool, parse func([]byte) ([]T, error)) HeldFeed[[]T] {
	return report.ReadHeldArchive(ctx, s.History != nil, s.Season, report.ArchivedFeed[T]{Export: export, Leagueless: leagueless, Parse: parse, Bodies: s.History.ReferenceBodies})
}

func (s *State) WithTeamIdentity(run func(string) error) error {
	s.KeyMu.Lock()
	defer s.KeyMu.Unlock()
	return run(s.MyFranchise)
}

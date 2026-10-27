package composition

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/accounts"
	commissionerapp "github.com/secureprospective/TheWarRoom/internal/app/commissioner"
	nativeapp "github.com/secureprospective/TheWarRoom/internal/app/commissioner/native"
	"github.com/secureprospective/TheWarRoom/internal/app/core"
	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/leagueservice"
	"github.com/secureprospective/TheWarRoom/internal/leagueweek"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

type localCommissionerSession struct{ installation string }

func (s localCommissionerSession) Principal() (string, error) {
	return commissionerapp.LocalPrincipal(s.installation, os.Getuid()) //nolint:wrapcheck // Preserve missing local identity refusal.
}

func (a *App) commissionerPrincipal() (string, error) {
	return commissionerapp.ResolvePrincipal(a.core.CommissionerSession, func() (string, error) { //nolint:wrapcheck // Preserve local identity errors.
		return (localCommissionerSession{installation: a.core.Paths.DataDir}).Principal()
	})
}
func (a *App) localSeat() commissionerapp.LocalSeat {
	return commissionerapp.LocalSeat{Ready: a.ready, Principal: a.commissionerPrincipal, Context: func() context.Context { return a.core.Ctx },
		Status: func(ctx context.Context, local string) (accounts.SeatStatus, error) { return a.seat.Status(ctx, local) },
		SignIn: func(ctx context.Context, raw accounts.RawCredentials, local string) (accounts.Account, error) {
			return a.seat.SignIn(ctx, raw, local)
		}}
}
func (a *App) commissionerSeat() (accounts.SeatStatus, error) {
	return a.localSeat().Read() //nolint:wrapcheck // Preserve seat errors.
}
func (a *App) commissionerSeatSignIn(raw accounts.RawCredentials) (accounts.SeatStatus, error) {
	return a.localSeat().Login(raw, a.commissionerSeat) //nolint:wrapcheck // Preserve sign-in errors and re-read status.
}

// commissionerPreviewRegistry is docs/ui/endpoint-registry.csv, embedded by the root binary.
var commissionerPreviewRegistry string //nolint:gochecknoglobals // Main sets it once from the root embed.

// leagueRecords replays one app league's commissioner ledger; the calendar and the gate read the same one.
func (a *App) leagueRecords(parent context.Context, league string) ([]commissioner.Record, error) {
	ctx, cancel := context.WithTimeout(parent, refreshTimeout)
	defer cancel()
	//nolint:wrapcheck // the service's errors already name the league calendar
	return commissionerapp.Records(ctx, a.core.History, league, "calendar")
}

// nativeFacts reads the football facts the app already holds, with no network: positions and draft years
// from the held players directory under its lock, NFL listings from the held injuries feed.
func (a *App) nativeFacts() nativecheck.Facts {
	a.core.LookupMu.Lock()
	facts := nativeapp.HeldPlayerFacts(a.core.Lookup, a.core.HasLookup)
	a.core.LookupMu.Unlock()
	return nativeapp.HeldDesignations(facts, a.injuriesHeld(), func() map[string]string { return a.heldInjuries().Status })
}

type RawLeagueCalendarScope = commissionerapp.RawLeagueCalendarScope
type RawLeagueScheduleScope = commissionerapp.RawLeagueScheduleScope
type RawNativeResultsScope = commissionerapp.RawNativeResultsScope
type RawNativeScheduleScope = commissionerapp.RawNativeScheduleScope
type RawNativeWeekScope = commissionerapp.RawNativeWeekScope
type RawNativeSeedRequest = commissionerapp.RawNativeSeedRequest

type commissionerHost struct {
	*core.State
	app *App
}

func newCommissionerAPI(app *App) *commissionerapp.API {
	return commissionerapp.New(commissionerHost{State: app.core, app: app})
}
func (h commissionerHost) CommissionerContext() context.Context { return h.Ctx }
func (h commissionerHost) CommissionerPrincipal() (string, error) {
	return h.app.commissionerPrincipal()
}
func (h commissionerHost) CommissionerAuthorization() commissioner.Authorizer {
	return h.CommissionerAuthorizer
}
func (h commissionerHost) CommissionerNow() time.Time          { return h.app.timeNow() }
func (h commissionerHost) CommissionerZone() *time.Location    { return h.app.ownerZone() }
func (h commissionerHost) CommissionerPreviewRegistry() string { return commissionerPreviewRegistry }
func (h commissionerHost) CommissionerRecords(ctx context.Context, league string) ([]commissioner.Record,
	error) {
	return h.app.leagueRecords(ctx, league)
}
func (h commissionerHost) NativeFacts() nativecheck.Facts { return h.app.nativeFacts() }
func (h commissionerHost) NativeScheduleDates(ctx context.Context, season int,
	at time.Time) ([]leagueweek.Week, error) {
	return h.app.nativeScheduleDates(ctx, season, at)
}
func (h commissionerHost) RunNativeOwnerAction(league, franchise, name string,
	body json.RawMessage) (commissioner.Receipt, error) {
	return h.app.runNativeAction(nativeOwnerActions(), league, franchise, name, body)
}
func (h commissionerHost) CaptureNativeSeed() (commissioner.RawNativeSnapshot, error) {
	return h.app.heldNativeSnapshot()
}
func (h commissionerHost) CommissionerSeat() (accounts.SeatStatus, error) {
	return h.app.commissionerSeat()
}
func (h commissionerHost) CommissionerSeatSignIn(raw accounts.RawCredentials) (accounts.SeatStatus, error) {
	return h.app.commissionerSeatSignIn(raw)
}
func (h commissionerHost) CommissionerBackupNow() (commissionerapp.BackupResult, error) {
	return h.app.backupNow()
}
func (h commissionerHost) CloseFinishedDrafts(league string) { h.app.closeFinishedDrafts(league) }

func (h commissionerHost) TransferStatus(ctx context.Context, league string) (history.TransferStatus, error) {
	status, err := h.History.TransferStatus(ctx, league)
	if err != nil {
		return history.TransferStatus{}, fmt.Errorf("read league location: %w", err)
	}
	return status, nil
}

// This is a local reference read. It has no fetch path and never reads a fantasy league's schedule.
func (a *App) nativeScheduleDates(parent context.Context, season int, at time.Time) ([]leagueweek.Week,
	error) {
	ctx, cancel := context.WithTimeout(parent, refreshTimeout)
	defer cancel()
	return nativeapp.ScheduleDates(ctx, a.core.History.ReferenceBodies, season, at) //nolint:wrapcheck // Preserve the held NFL week dates refusal.
}

type appBidEnv struct{ app *App }

func (e appBidEnv) BidEnvironment(ctx context.Context, league string,
	at time.Time) (commissioner.BidEnvironment, error) {
	//nolint:wrapcheck // Preserve the bidding refusal.
	return nativeapp.ReadBidEnvironment(ctx, nativeapp.BidSource{Records: e.app.leagueRecords,
		Schedule: e.app.nativeScheduleDates, Bracket: e.app.core.History.NativeBracket}, league, at)
}

type nativeDueClock struct{ now func() time.Time }

func (c nativeDueClock) Now() time.Time { return c.now() }
func (nativeDueClock) Trusted() error   { return nil }

func (a *App) nativeWaiverClock(ctx context.Context) {
	worker := leagueservice.NewDueWorker(a.core.History, a.nativeFacts, nativeDueClock{now: a.timeNow})
	worker.Run(ctx, func(err error) { log.Printf("the war room: process native deadlines: %v", err) })
}

package composition

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/accounts"
	commissionerapp "github.com/secureprospective/TheWarRoom/internal/app/commissioner"
	nativeapp "github.com/secureprospective/TheWarRoom/internal/app/commissioner/native"
	"github.com/secureprospective/TheWarRoom/internal/app/core"
	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/leagueweek"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

type localCommissionerSession struct{ installation string }

func (s localCommissionerSession) Principal() (string, error) {
	if s.installation == "" {
		return "", fmt.Errorf("local identity is unavailable")
	}
	identity := sha256.Sum256([]byte(fmt.Sprintf("%s\x00%d", s.installation, os.Getuid())))
	return "local." + hex.EncodeToString(identity[:]), nil
}

func (a *App) commissionerPrincipal() (string, error) {
	session := a.core.CommissionerSession
	if session == nil {
		session = localCommissionerSession{installation: a.core.Paths.DataDir}
	}
	principal, err := session.Principal()
	if err != nil {
		return "", fmt.Errorf("read local identity: %w", err)
	}
	if strings.TrimSpace(principal) == "" {
		return "", fmt.Errorf("local identity is unavailable")
	}
	return principal, nil
}

// commissionerSeat reads whether this computer is linked to the local sign-in.
func (a *App) commissionerSeat() (accounts.SeatStatus, error) {
	if err := a.ready(); err != nil {
		return accounts.SeatStatus{}, err
	}
	local, err := a.commissionerPrincipal()
	if err != nil {
		return accounts.SeatStatus{}, err
	}
	//nolint:wrapcheck // the seat names its own failure
	return a.seat.Status(a.core.Ctx, local)
}

// commissionerSeatSignIn links this computer's identity to the local sign-in. On the computer that
// made the league it protects the commissioner; on a restored copy it proves the same person.
func (a *App) commissionerSeatSignIn(raw accounts.RawCredentials) (accounts.SeatStatus, error) {
	if err := a.ready(); err != nil {
		return accounts.SeatStatus{}, err
	}
	local, err := a.commissionerPrincipal()
	if err != nil {
		return accounts.SeatStatus{}, err
	}
	if _, err := a.seat.SignIn(a.core.Ctx, raw, local); err != nil {
		return accounts.SeatStatus{}, err //nolint:wrapcheck // the seat names its own failure
	}
	return a.commissionerSeat()
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
func (h commissionerHost) NativeSeedCommand(ctx context.Context,
	input commissionerapp.RawNativeSeedRequest) (commissioner.RawCommand, error) {
	return h.app.nativeSeedCommand(ctx, input)
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

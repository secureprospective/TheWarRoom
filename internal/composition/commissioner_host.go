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

	commissionerapp "github.com/secureprospective/TheWarRoom/internal/app/commissioner"
	nativeapp "github.com/secureprospective/TheWarRoom/internal/app/commissioner/native"
	"github.com/secureprospective/TheWarRoom/internal/app/core"
	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/leagueweek"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
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
	facts := nativecheck.Facts{}
	a.core.LookupMu.Lock()
	if a.core.HasLookup {
		facts = nativeapp.PlayerFacts(a.core.Lookup)
	}
	a.core.LookupMu.Unlock()
	if !a.injuriesHeld() {
		return facts
	}
	nativeapp.Designate(facts, a.heldInjuries().Status)
	return facts
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
func (h commissionerHost) SubmitNativeTrade(raw commissioner.RawCommand) (commissioner.Receipt, error) {
	return h.app.submitOwnerAction(raw, tradeFamily())
}
func (h commissionerHost) SubmitNativeWaiver(raw commissioner.RawCommand) (commissioner.Receipt, error) {
	return h.app.submitOwnerAction(raw, waiverFamily())
}
func (h commissionerHost) SubmitNativeBid(raw commissioner.RawCommand) (commissioner.Receipt, error) {
	return h.app.submitOwnerAction(raw, bidFamily())
}
func (h commissionerHost) NativeSeedCommand(ctx context.Context,
	input commissionerapp.RawNativeSeedRequest) (commissioner.RawCommand, error) {
	return h.app.nativeSeedCommand(ctx, input)
}
func (h commissionerHost) CloseFinishedDrafts(league string) { h.app.closeFinishedDrafts(league) }

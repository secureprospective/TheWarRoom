package composition

import (
	"context"
	"strconv"
	"time"

	controlapp "github.com/secureprospective/TheWarRoom/internal/app/control"
	"github.com/secureprospective/TheWarRoom/internal/app/core"
	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/league"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/pendingtrades"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/sitenews"
	"github.com/secureprospective/TheWarRoom/internal/mflkey"
)

// rulebookMarkdown is the league's own rulebook, embedded by the root binary (M-185).
var rulebookMarkdown []byte //nolint:gochecknoglobals // Main sets it once from the root embed.

// controlHost is the Control Room API's Host. It is never bound: only control.API exposes methods
// to Wails. Its reads keep their SeasonMu, WeekMu, KeyMu and params-store spans exactly as the
// bindings had; the rule names and site news are held in control.Held, not here.
type controlHost struct {
	*core.State
	app *App
}

// controlAPI is the Control Room API over this App; root callers use it, never an App wrapper.
func (a *App) controlAPI() *controlapp.API {
	return controlapp.New(controlHost{State: a.core, app: a}, a.controlHeld)
}

func (h controlHost) RulebookMarkdown() []byte { return rulebookMarkdown }

func (h controlHost) KeyUnavailable(detail string) controlapp.MFLKeyStatus {
	h.app.core.KeyMu.Lock()
	defer h.app.core.KeyMu.Unlock()
	return h.app.keyStatus("unavailable", detail)
}

func (h controlHost) ScoringConfig() league.RawConfig { return h.app.core.Rulebook.ActiveConfig() }

func (h controlHost) Zone() *time.Location { return time.Local }

func (h controlHost) FetchRuleNames() (map[string]string, Freshness, error) {
	a := h.app
	ctx, cancel := context.WithTimeout(a.core.Ctx, refreshTimeout)
	defer cancel()
	rows, fresh, err := liveOrArchive(ctx, a.fallbackParent(), a.core.Season, archivedFeed[league.RawRuleCode]{
		Export: league.RuleDictionaryExport, Leagueless: true,
		Fetch: func(ctx context.Context) ([]league.RawRuleCode, error) {
			return league.FetchRuleDictionary(ctx, a.core.MFLClient, strconv.Itoa(a.core.Season))
		},
		Parse: league.ParseRuleDictionary, Bodies: a.core.History.ArchivedBodies,
	}, a.core.FallbackLog)
	return league.RuleNames(rows), fresh, err
}

func (h controlHost) FetchSiteNews() ([]sitenews.RawItem, Freshness, error) {
	a := h.app
	ctx, cancel := context.WithTimeout(a.core.Ctx, refreshTimeout)
	defer cancel()
	return liveOrArchive(ctx, a.fallbackParent(), a.core.Season, archivedFeed[sitenews.RawItem]{
		Export: sitenews.Export, Leagueless: true,
		Fetch: func(ctx context.Context) ([]sitenews.RawItem, error) {
			return sitenews.Fetch(ctx, a.core.MFLClient, strconv.Itoa(a.core.Season))
		},
		Parse: sitenews.Parse, Bodies: a.core.History.ArchivedBodies,
	}, a.core.FallbackLog)
}

// SourceHealth runs after the binding's startup check, under the status strip's own timeout.
func (h controlHost) SourceHealth() (string, string, error) {
	a := h.app
	ctx, cancel := context.WithTimeout(a.core.Ctx, m1Timeout)
	defer cancel()
	health, err := a.healthInput(ctx)
	if err != nil {
		return "", "", err
	}
	state, label := sourceStanding(health)
	return state, label, nil
}

func (h controlHost) Week() controlapp.WeekSnapshot { return h.app.weekSnapshot() }

func (h controlHost) Now() time.Time { return h.app.timeNow() }

// SetWindow runs after the binding's validation and startup check, under the window timeout.
func (h controlHost) SetWindow(kind, at string) error {
	ctx, cancel := context.WithTimeout(h.app.core.Ctx, m2Timeout)
	defer cancel()
	//nolint:wrapcheck // Preserve window refusals and coordinator errors.
	return (controlapp.WindowSetter{StateError: h.app.core.Whatif.Err, TradeDeadline: h.app.core.Whatif.TradeDeadline,
		CalendarEvents: h.app.core.Whatif.CalendarEvents, Season: h.app.core.League.Season, Execute: h.app.core.Coordinator.Execute,
		Changed: h.app.core.ClockChanged}).Set(ctx, kind, at)
}

func (h controlHost) OpenMFLAccounting() error {
	a := h.app
	//nolint:wrapcheck // Preserve existing feature refusal text.
	return controlapp.OpenAccounting(a.ready, func() string {
		if a.core.MFLClient != nil {
			return a.core.MFLClient.LeagueHost()
		}
		return ""
	}, func() int { return a.core.Season }, func(target string) { a.core.OpenURL(a.core.Ctx, target) })

}

func (h controlHost) Params() controlapp.ParamsResult {
	return controlapp.ReadParams(h.app.ready, func() controlapp.ParameterStore { return h.app.core.Params })
}
func (h controlHost) SetParam(key, position string, value float64) controlapp.SetParamResult {
	return controlapp.SetParameter(func() context.Context { return h.app.core.Ctx }, h.app.ready, func() controlapp.ParameterStore { return h.app.core.Params }, key, position, value)
}
func (h controlHost) ResetParam(key, position string) controlapp.SetParamResult {
	return controlapp.ResetParameter(func() context.Context { return h.app.core.Ctx }, h.app.ready, func() controlapp.ParameterStore { return h.app.core.Params }, key, position)
}

func (a *App) keyStatus(state, detail string) controlapp.MFLKeyStatus {
	return controlapp.MFLKeyStatus{
		State: state, League: ingestion.LeagueID, Season: a.core.Season,
		MyFranchise: a.core.MyFranchise, TeamActionRefusal: core.OwnFranchiseRefusal(a.core.MyFranchise, ""),
		VerifiedAt: a.core.KeyVerifiedAt, Detail: detail,
	}
}

func (h controlHost) KeyStatus() (controlapp.MFLKeyStatus, error) {
	return h.app.keyWorkflow().Read() //nolint:wrapcheck // Preserve startup refusals.
}

func (h controlHost) SetKey(key string) (controlapp.MFLKeyStatus, error) {
	return h.app.keyWorkflow().Set(key) //nolint:wrapcheck // Preserve startup refusals.
}

func (h controlHost) DeleteKey() (controlapp.MFLKeyStatus, error) {
	return h.app.keyWorkflow().Delete() //nolint:wrapcheck // Preserve startup refusals.
}

// lookupMyFranchise runs under KeyMu or before startup publishes state.
func (a *App) lookupMyFranchise(ctx context.Context) {
	a.core.MyFranchise = ""
	a.core.MyFranchise = controlapp.ResolveFranchise(ctx, a.core.KeyStore, func(ctx context.Context) (string, error) {
		return a.core.MFLClient.MyFranchise(ctx, strconv.Itoa(a.core.Season), ingestion.LeagueID)
	})
}

func (a *App) keyWorkflow() controlapp.KeyWorkflow {
	return controlapp.KeyWorkflow{Ready: a.ready, Lock: a.core.KeyMu.Lock, Unlock: a.core.KeyMu.Unlock,
		Context: func() context.Context { return a.core.Ctx }, Store: func() controlapp.CredentialStore { return a.core.KeyStore }, Status: a.keyStatus,
		Verify: func(ctx context.Context, candidate mflkey.Key) (string, error) {
			c := a.core.MFLClient.WithKeySource(func(context.Context) (mflkey.Key, error) { return candidate, nil })
			verified, err := pendingtrades.Verify(ctx, c, strconv.Itoa(a.core.Season), ingestion.LeagueID)
			return verified.Detail, err //nolint:wrapcheck // The workflow classifies rejection identity.
		}, Lookup: a.lookupMyFranchise,
		Verified: func(at string) { a.core.KeyVerifiedAt = at; a.core.KeyGeneration++ },
		Clear: func() {
			a.core.MyFranchise = ""
			a.core.KeyVerifiedAt = ""
			a.core.KeyGeneration++
			a.core.SeasonMu.Lock()
			a.clearPendingTrades()
			a.core.SeasonMu.Unlock()
		}, Refresh: a.refreshPendingTradesInBackground, Now: time.Now}
}

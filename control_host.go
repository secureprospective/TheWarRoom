package main

import (
	"cmp"
	"context"
	_ "embed" // the league's own rulebook ships with the app (M-185)
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	controlapp "github.com/secureprospective/TheWarRoom/internal/app/control"
	"github.com/secureprospective/TheWarRoom/internal/app/core"
	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/league"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/pendingtrades"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/sitenews"
	"github.com/secureprospective/TheWarRoom/internal/mflkey"
)

//go:embed docs/league-rules/Official_Rulebook.md
var rulebookMarkdown []byte

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
	return h.app.core.SetLeagueWindow(ctx, kind, at) //nolint:wrapcheck // keeps the binding's refusal text
}

func (h controlHost) OpenMFLAccounting() error {
	a := h.app
	if err := a.ready(); err != nil {
		return fmt.Errorf("open accounting: %w", err)
	}
	host := ""
	if a.core.MFLClient != nil {
		host = a.core.MFLClient.LeagueHost()
	}
	target, err := controlapp.AccountingURL(host, a.core.Season)
	if err != nil {
		return err //nolint:wrapcheck // preserve the binding error text
	}
	a.core.OpenURL(a.core.Ctx, target)
	return nil
}

func (h controlHost) Params() controlapp.ParamsResult {
	a := h.app
	if err := a.ready(); err != nil {
		return controlapp.ParamsResult{OK: false, Error: err.Error()}
	}
	values := a.core.Params.Snapshot()
	defs := a.core.Params.Definitions()
	out := make([]controlapp.ParamView, len(defs))
	for i, d := range defs {
		v, err := values.GetPosition(d.Key, d.Position)
		if err != nil {
			return controlapp.ParamsResult{OK: false, Error: err.Error()}
		}
		label, group := controlapp.ParameterWords(d.Key, d.Description)
		out[i] = controlapp.ParamView{Key: d.Key, Label: label, Group: group, Position: d.Position, Description: d.Description,
			Default: d.Default, Min: d.Min, Max: d.Max, Value: v, Calibrated: d.IsCalibrated,
			Overridden: a.core.Params.Overridden(d.Key, d.Position)}
	}
	slices.SortFunc(out, func(a, b controlapp.ParamView) int {
		if n := cmp.Compare(a.Group, b.Group); n != 0 {
			return n
		}
		if n := cmp.Compare(a.Label, b.Label); n != 0 {
			return n
		}
		return cmp.Compare(a.Position, b.Position)
	})
	return controlapp.ParamsResult{OK: true, Params: out}
}

func (h controlHost) SetParam(key, position string, value float64) controlapp.SetParamResult {
	a := h.app
	if err := a.ready(); err != nil {
		return controlapp.SetParamResult{OK: false, Error: err.Error()}
	}
	ctx, cancel := context.WithTimeout(a.core.Ctx, 3*time.Second)
	defer cancel()
	if err := a.core.Params.SetOverride(ctx, key, position, value, "admin console"); err != nil {
		return controlapp.SetParamResult{OK: false, Error: err.Error()}
	}
	return controlapp.SetParamResult{OK: true}
}

func (h controlHost) ResetParam(key, position string) controlapp.SetParamResult {
	a := h.app
	if err := a.ready(); err != nil {
		return controlapp.SetParamResult{OK: false, Error: err.Error()}
	}
	ctx, cancel := context.WithTimeout(a.core.Ctx, 3*time.Second)
	defer cancel()
	if err := a.core.Params.ClearOverride(ctx, key, position); err != nil {
		return controlapp.SetParamResult{OK: false, Error: err.Error()}
	}
	return controlapp.SetParamResult{OK: true}
}

func (a *App) keyStatus(state, detail string) controlapp.MFLKeyStatus {
	return controlapp.MFLKeyStatus{
		State: state, League: ingestion.LeagueID, Season: a.core.Season,
		VerifiedAt: a.core.KeyVerifiedAt, Detail: detail,
	}
}

func (h controlHost) KeyStatus() (controlapp.MFLKeyStatus, error) {
	a := h.app
	if err := a.ready(); err != nil {
		return controlapp.MFLKeyStatus{}, err
	}
	a.core.KeyMu.Lock()
	defer a.core.KeyMu.Unlock()
	_, err := a.core.KeyStore.Get(a.core.Ctx)
	if errors.Is(err, mflkey.ErrNoKey) {
		return a.keyStatus("absent", ""), nil
	}
	if err != nil {
		return a.keyStatus("unavailable", err.Error()), nil
	}
	return a.keyStatus("connected", ""), nil
}

// SetKey runs after the bound method's panic recovery, in the binding's original order.
func (h controlHost) SetKey(key string) (controlapp.MFLKeyStatus, error) {
	a := h.app
	candidate := mflkey.Key(strings.TrimSpace(key))
	if err := a.ready(); err != nil {
		return controlapp.MFLKeyStatus{}, err
	}
	a.core.KeyMu.Lock()
	defer a.core.KeyMu.Unlock()
	// Bounds are plausibility checks, not a claim about MFL's key encoding.
	if len(candidate) < 16 || len(candidate) > 256 {
		detail := "key must contain 16 to 256 bytes after trimming whitespace"
		return a.keyStatus("rejected", detail), nil
	}
	ctx, cancel := context.WithTimeout(a.core.Ctx, time.Minute)
	defer cancel()
	c := a.core.MFLClient.WithKeySource(func(context.Context) (mflkey.Key, error) { return candidate, nil })
	verified, err := pendingtrades.Verify(ctx, c, strconv.Itoa(a.core.Season), ingestion.LeagueID)
	if errors.Is(err, pendingtrades.ErrRejected) {
		return a.keyStatus("rejected", err.Error()), nil
	}
	if err != nil {
		return a.keyStatus("unreachable", err.Error()+"; nothing was stored"), nil
	}
	if err := a.core.KeyStore.Set(ctx, candidate); err != nil {
		return a.keyStatus("unavailable", err.Error()), nil
	}
	a.core.KeyVerifiedAt = time.Now().UTC().Format(time.RFC3339)
	a.core.KeyGeneration++
	a.refreshPendingTradesInBackground()
	return a.keyStatus("connected", verified.Detail), nil
}

func (h controlHost) DeleteKey() (controlapp.MFLKeyStatus, error) {
	a := h.app
	if err := a.ready(); err != nil {
		return controlapp.MFLKeyStatus{}, err
	}
	a.core.KeyMu.Lock()
	defer a.core.KeyMu.Unlock()
	if err := a.core.KeyStore.Delete(a.core.Ctx); err != nil {
		return a.keyStatus("unavailable", err.Error()), nil
	}
	a.core.KeyVerifiedAt = ""
	a.core.KeyGeneration++
	a.core.SeasonMu.Lock()
	a.clearPendingTrades()
	a.core.SeasonMu.Unlock()
	a.refreshPendingTradesInBackground()
	return a.keyStatus("absent", ""), nil
}

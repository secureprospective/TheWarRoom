package main

import (
	"context"
	_ "embed" // the league's own rulebook ships with the app (M-185)
	"fmt"
	"maps"
	"slices"
	"strconv"
	"time"

	controlapp "github.com/secureprospective/TheWarRoom/internal/app/control"

	"github.com/secureprospective/TheWarRoom/internal/ingestion/league"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/sitenews"
)

//go:embed docs/league-rules/Official_Rulebook.md
var rulebookMarkdown []byte

// siteNewsFreshFor is how long a held site-news read serves before the next open refreshes it.
const siteNewsFreshFor = time.Hour

// controlFeeds are the Control Room's held MFL feeds, guarded by seasonMu.
type controlFeeds struct {
	siteNews  heldSeasonFeed[[]sitenews.RawItem]
	ruleNames heldSeasonFeed[map[string]string]
}

// TargetLeagueSettings is Control Room › League › League settings (M-091, M-097 merged in).
func (a *App) TargetLeagueSettings() (Report, error) {
	if err := a.ready(); err != nil {
		return Report{}, fmt.Errorf("league settings: startup: %w", err)
	}
	ctx, cancel := context.WithTimeout(a.core.Ctx, refreshTimeout)
	defer cancel()
	a.core.SeasonMu.Lock()
	held := a.control.ruleNames
	held.Value = maps.Clone(held.Value)
	a.core.SeasonMu.Unlock()
	if held.Value == nil {
		rows, fresh, err := liveOrArchive(ctx, a.fallbackParent(), a.core.Season, archivedFeed[league.RawRuleCode]{
			export: league.RuleDictionaryExport, leagueless: true,
			fetch: func(ctx context.Context) ([]league.RawRuleCode, error) {
				return league.FetchRuleDictionary(ctx, a.core.MFLClient, strconv.Itoa(a.core.Season))
			},
			parse: league.ParseRuleDictionary, bodies: a.core.History.ArchivedBodies,
		}, a.core.FallbackLog)
		a.core.SeasonMu.Lock()
		holdSeasonFeed(&a.control.ruleNames, league.RuleNames(rows), fresh, err)
		held = a.control.ruleNames
		held.Value = maps.Clone(held.Value)
		a.core.SeasonMu.Unlock()
	}
	note := ""
	if held.Value == nil {
		note = "MFL's names for its scoring codes could not be read, so codes are shown: " + held.RefreshError
	}
	return wireReport(controlapp.LeagueSettingsReport(a.core.Rulebook.ActiveConfig(), held.Value, note)), nil
}

// TargetSiteNews is Control Room › App › What's New (M-141), refreshed when an hour old.
func (a *App) TargetSiteNews() (Report, error) {
	if err := a.ready(); err != nil {
		return Report{}, fmt.Errorf("site news: startup: %w", err)
	}
	a.core.SeasonMu.Lock()
	held := a.control.siteNews
	held.Value = slices.Clone(held.Value)
	a.core.SeasonMu.Unlock()
	if held.RefreshError != "" || time.Since(held.FetchedAt) >= siteNewsFreshFor {
		ctx, cancel := context.WithTimeout(a.core.Ctx, refreshTimeout)
		defer cancel()
		rows, fresh, err := liveOrArchive(ctx, a.fallbackParent(), a.core.Season, archivedFeed[sitenews.RawItem]{
			export: sitenews.Export, leagueless: true,
			fetch: func(ctx context.Context) ([]sitenews.RawItem, error) {
				return sitenews.Fetch(ctx, a.core.MFLClient, strconv.Itoa(a.core.Season))
			},
			parse: sitenews.Parse, bodies: a.core.History.ArchivedBodies,
		}, a.core.FallbackLog)
		a.core.SeasonMu.Lock()
		holdSeasonFeed(&a.control.siteNews, rows, fresh, err)
		held = a.control.siteNews
		held.Value = slices.Clone(held.Value)
		a.core.SeasonMu.Unlock()
	}
	items, err := sitenews.Read(held.Value)
	if err != nil {
		return Report{}, err //nolint:wrapcheck // names the item
	}
	asOf := "As of: not fetched yet"
	if !held.FetchedAt.IsZero() {
		asOf = "As of: " + reportTime(held.FetchedAt, time.Local)
	}
	return wireReport(controlapp.SiteNewsReport(items, asOf, held.RefreshError, time.Local)), nil
}

// TargetAccounting is Control Room › League › Accounting (M-098, M-197).
func (a *App) TargetAccounting() Report { return wireReport(controlapp.AccountingReport()) }

// TargetOpenMFLAccounting opens MFL's accounting page for the league. The app only opens it.
func (a *App) TargetOpenMFLAccounting() error {
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

// BylawSection is one section of the league's rulebook as readable lines.
type BylawSection struct {
	Title string   `json:"title"`
	Lines []string `json:"lines"`
}

// TargetBylaws is Control Room › League › By-laws (M-185): the league's own rulebook, by section.
func (a *App) TargetBylaws() []BylawSection {
	return wireBylaws(controlapp.BylawSections(rulebookMarkdown))
}

func wireBylaws(sections []controlapp.BylawSection) []BylawSection {
	if sections == nil {
		return nil
	}
	out := make([]BylawSection, len(sections))
	for i, s := range sections {
		out[i] = BylawSection(s)
	}
	return out
}

package main

import (
	"bufio"
	"bytes"
	"context"
	_ "embed" // the league's own rulebook ships with the app (M-185)
	"fmt"
	"strconv"
	"strings"
	"time"

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
	ctx, cancel := context.WithTimeout(a.ctx, refreshTimeout)
	defer cancel()
	a.seasonMu.Lock()
	held := a.control.ruleNames
	a.seasonMu.Unlock()
	if held.value == nil {
		rows, fresh, err := liveOrArchive(ctx, a.fallbackParent(), a.season, archivedFeed[league.RawRuleCode]{
			export: league.RuleDictionaryExport, leagueless: true,
			fetch: func(ctx context.Context) ([]league.RawRuleCode, error) {
				return league.FetchRuleDictionary(ctx, a.mflClient, strconv.Itoa(a.season))
			},
			parse: league.ParseRuleDictionary, bodies: a.history.ArchivedBodies,
		}, a.fallbackLog)
		a.seasonMu.Lock()
		holdSeasonFeed(&a.control.ruleNames, league.RuleNames(rows), fresh, err)
		held = a.control.ruleNames
		a.seasonMu.Unlock()
	}
	note := ""
	if held.value == nil {
		note = "MFL's names for its scoring codes could not be read, so codes are shown: " + held.refreshError
	}
	return leagueSettingsReport(a.rulebook.ActiveConfig(), held.value, note), nil
}

// TargetSiteNews is Control Room › App › What's New (M-141), refreshed when an hour old.
func (a *App) TargetSiteNews() (Report, error) {
	if err := a.ready(); err != nil {
		return Report{}, fmt.Errorf("site news: startup: %w", err)
	}
	a.seasonMu.Lock()
	held := a.control.siteNews
	a.seasonMu.Unlock()
	if held.refreshError != "" || time.Since(held.fetchedAt) >= siteNewsFreshFor {
		ctx, cancel := context.WithTimeout(a.ctx, refreshTimeout)
		defer cancel()
		rows, fresh, err := liveOrArchive(ctx, a.fallbackParent(), a.season, archivedFeed[sitenews.RawItem]{
			export: sitenews.Export, leagueless: true,
			fetch: func(ctx context.Context) ([]sitenews.RawItem, error) {
				return sitenews.Fetch(ctx, a.mflClient, strconv.Itoa(a.season))
			},
			parse: sitenews.Parse, bodies: a.history.ArchivedBodies,
		}, a.fallbackLog)
		a.seasonMu.Lock()
		holdSeasonFeed(&a.control.siteNews, rows, fresh, err)
		held = a.control.siteNews
		a.seasonMu.Unlock()
	}
	items, err := sitenews.Read(held.value)
	if err != nil {
		return Report{}, err //nolint:wrapcheck // names the item
	}
	asOf := "As of: not fetched yet"
	if !held.fetchedAt.IsZero() {
		asOf = "As of: " + reportTime(held.fetchedAt, time.Local)
	}
	return siteNewsReport(items, asOf, held.refreshError, time.Local), nil
}

// TargetAccounting is Control Room › League › Accounting (M-098, M-197).
func (a *App) TargetAccounting() Report { return accountingReport() }

// TargetOpenMFLAccounting opens MFL's accounting page for the league. The app only opens it.
func (a *App) TargetOpenMFLAccounting() error {
	if err := a.ready(); err != nil {
		return fmt.Errorf("open accounting: %w", err)
	}
	host := ""
	if a.mflClient != nil {
		host = a.mflClient.LeagueHost()
	}
	target, err := accountingURL(host, a.season)
	if err != nil {
		return err
	}
	a.openURL(a.ctx, target)
	return nil
}

// BylawSection is one section of the league's rulebook as readable lines.
type BylawSection struct {
	Title string   `json:"title"`
	Lines []string `json:"lines"`
}

// TargetBylaws is Control Room › League › By-laws (M-185): the league's own rulebook, by section.
func (a *App) TargetBylaws() []BylawSection { return bylawSections(rulebookMarkdown) }

// bylawSections reads the rulebook's markdown: each "## " heading starts a section; "### "
// headings, bullets, paragraphs and table rows become plain lines; markup is dropped.
func bylawSections(md []byte) []BylawSection {
	var out []BylawSection
	sc := bufio.NewScanner(bytes.NewReader(md))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		switch {
		case strings.HasPrefix(line, "## "):
			out = append(out, BylawSection{Title: plainMarkdown(strings.TrimPrefix(line, "## "))})
		case len(out) == 0 || line == "" || line == "---" || isTableRule(line):
			continue
		default:
			if text := bylawLine(line); text != "" {
				out[len(out)-1].Lines = append(out[len(out)-1].Lines, text)
			}
		}
	}
	return out
}

func bylawLine(line string) string {
	switch {
	case strings.HasPrefix(line, "### "):
		return plainMarkdown(strings.TrimPrefix(line, "### ")) + ":"
	case strings.HasPrefix(line, "|"):
		var cells []string
		for _, c := range strings.Split(strings.Trim(line, "|"), "|") {
			if c = plainMarkdown(c); c != "" {
				cells = append(cells, c)
			}
		}
		return strings.Join(cells, " · ")
	case strings.HasPrefix(line, "- "):
		return "• " + plainMarkdown(strings.TrimPrefix(line, "- "))
	}
	return plainMarkdown(line)
}

func isTableRule(line string) bool {
	return strings.HasPrefix(line, "|") && strings.Trim(line, "|-: ") == ""
}

func plainMarkdown(s string) string {
	return strings.TrimSpace(strings.NewReplacer("**", "", "`", "").Replace(s))
}

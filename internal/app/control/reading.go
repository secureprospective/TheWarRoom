// Package control builds readings from caller-owned snapshots without locks or workers.
package control

import (
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/report"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/engine/bid"
	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/league"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/sitenews"
)

// LeagueSettingsReport is M-091 (with M-097 merged in): the league's settings and scoring rules in
// plain words, from the held league and rules exports and MFL's scoring-event dictionary.
func LeagueSettingsReport(cfg league.RawConfig, names map[string]string, dictionaryNote string) report.Report {
	source := "Source: MFL league and rules exports"
	if cfg.Source != "" {
		source += " (" + cfg.Source + ")"
	}
	r := report.Report{ID: "M-091", Title: "League settings", Status: "ok", Source: source,
		AsOf: "As of: the held league settings", Columns: []string{"Setting", "Value"}, Rows: []report.Row{},
		How: "Read from MFL's own league and rules exports, unchanged. Scoring events use MFL's own names for its codes."}
	if cfg.RosterSize == "" && len(cfg.ScoringRules) == 0 {
		r.Status, r.Note = "empty", "No league settings held yet: they arrive with the next league refresh."
		return r
	}
	r.Note = dictionaryNote
	row := func(group, k, v string) { r.Rows = append(r.Rows, report.Row{Group: group, Cells: []string{k, v}}) }
	row("Season", "Scoring weeks", fmt.Sprintf("Weeks %s to %s; the regular season ends after week %s", cfg.StartWeek, cfg.EndWeek, cfg.LastRegularSeasonWeek))
	row("Season", "Standings order", tiebreakText(cfg.StandingsSort))
	row("Roster", "Roster size", cfg.RosterSize+" players")
	row("Roster", "Taxi squad", cfg.TaxiSquad+" players")
	row("Roster", "Injured reserve", cfg.InjuredReserve+" players")
	row("Roster", "Position limits", limitsText(cfg.RosterLimits))
	row("Starters", "Starters each week", cfg.Starters.Count)
	row("Starters", "Offensive starters (MFL's count)", cfg.Starters.IOPStarters)
	row("Starters", "Defensive starters (MFL's count)", cfg.Starters.IDPStarters)
	for _, p := range cfg.Starters.Positions {
		row("Starters", p.Name, startText(p.Limit))
	}
	capRows(cfg, row)
	for _, set := range cfg.ScoringRules {
		group := "Scoring · " + strings.ReplaceAll(set.Positions, "|", ", ")
		for _, rule := range set.Rules {
			if scoresNothing(rule) {
				continue
			}
			row(group, ruleName(rule.Event, names), scoringText(rule))
		}
	}
	return r
}

func capRows(cfg league.RawConfig, row func(group, k, v string)) {
	const g = "Salary cap and contracts"
	if cfg.UsesSalaries != "1" {
		row(g, "Salary cap", "This league does not use salaries")
		return
	}
	if m, err := domain.ParseMoneyMillions(cfg.SalaryCapAmount); err == nil {
		row(g, "Salary cap", bid.MoneyText(m))
	} else {
		row(g, "Salary cap", "MFL's setting could not be read: "+cfg.SalaryCapAmount)
	}
	row(g, "Taxi squad salaries", percentText(cfg.IncludeTaxiWithSalary)+" of a taxi player's salary counts against the cap")
	row(g, "Injured reserve salaries", percentText(cfg.IncludeIRWithSalary)+" of an injured player's salary counts against the cap")
	contractYears := "Contracts have no set length"
	if cfg.UsesContractYear == "1" {
		contractYears = "Every contract runs for a set number of seasons"
	}
	row(g, "Contract years", contractYears)
	taxiYears := "Seasons on the taxi squad do not count toward a contract's length"
	if p, err := strconv.Atoi(cfg.IncludeTaxiWithContractYear); err == nil && p > 0 {
		taxiYears = "Seasons on the taxi squad count toward a contract's length"
	}
	row(g, "Taxi seasons", taxiYears)
}

func percentText(p string) string {
	if p == "" {
		return "None"
	}
	return p + "%"
}

// tiebreak is one of MFL's standings-sort codes in words; an unknown code stays as MFL wrote it.
func tiebreak(code string) string {
	switch code {
	case "PCT":
		return "win percentage"
	case "H2H":
		return "head-to-head record"
	case "DIVPCT":
		return "division win percentage"
	case "CONFPCT":
		return "conference win percentage"
	case "ALL_PLAY_PCT":
		return "all-play win percentage"
	case "PTS":
		return "points scored"
	case "PWR":
		return "power rank"
	}
	return code
}

// tiebreakText reads MFL's standings sort ("PCT,H2H,…") as the order in words.
func tiebreakText(sort string) string {
	var parts []string
	for _, code := range strings.Split(sort, ",") {
		if code = strings.TrimSpace(code); code == "" {
			continue
		}
		parts = append(parts, tiebreak(code))
	}
	if len(parts) == 0 {
		return "MFL's default"
	}
	return strings.Join(parts, ", then ")
}

func limitsText(limits []league.PositionLimit) string {
	var set []string
	for _, l := range limits {
		if l.Limit != "0-0" && l.Limit != "" {
			set = append(set, l.Name+" "+strings.ReplaceAll(l.Limit, "-", " to "))
		}
	}
	if len(set) == 0 {
		return "None: any mix of positions"
	}
	return strings.Join(set, "; ")
}

func startText(limit string) string {
	if lo, hi, ok := strings.Cut(limit, "-"); ok {
		return lo + " to " + hi
	}
	return "Exactly " + limit
}

func ruleName(code string, names map[string]string) string {
	if n := names[code]; n != "" {
		return n
	}
	return "MFL event " + code
}

// scoringText reads MFL's points ("6", "*.05", "-2") and range ("0-39") in words. A per-each
// rule's range is MFL's guard band (0-10, 0-99, -50-999 in the 2026 rules), so only a fixed
// award, such as a field goal's distance band, names its range.
func scoringText(r league.ScoringRule) string {
	p := strings.TrimSpace(r.Points)
	if each, ok := strings.CutPrefix(p, "*"); ok {
		return scoringPoints(each) + " for each"
	}
	text := scoringPoints(p)
	if m := regexp.MustCompile(`^(-?[0-9]+)-(-?[0-9]+)$`).FindStringSubmatch(strings.TrimSpace(r.Range)); m != nil {
		text += fmt.Sprintf(" (when it is %s to %s)", m[1], m[2])
	}
	return text
}

// scoringPoints is MFL's points in words: ".05" is "0.05 points", "1" is "1 point".
func scoringPoints(p string) string {
	if v, ok := strings.CutPrefix(p, "-."); ok {
		p = "-0." + v
	} else if v, ok := strings.CutPrefix(p, "."); ok {
		p = "0." + v
	}
	if p == "1" || p == "-1" {
		return p + " point"
	}
	return p + " points"
}

// scoresNothing is a rule that awards no points, which says nothing to a GM.
func scoresNothing(r league.ScoringRule) bool {
	v, err := strconv.ParseFloat(strings.TrimPrefix(strings.TrimSpace(r.Points), "*"), 64)
	return err == nil && v == 0
}

// SiteNewsReport is M-141: MFL's What's New, newest first, in plain sentences.
func SiteNewsReport(items []sitenews.Item, fresh, refreshError string, zone *time.Location) report.Report {
	r := report.Report{ID: "M-141", Title: "What's New on MFL", Status: "ok", Source: "Source: MFL site news (MFL-wide, not this league)",
		AsOf: fresh, Columns: []string{"Published", "Article"}, Rows: []report.Row{},
		How: "MFL's site news feed, newest first. Its articles are written for every MFL league; the text is MFL's, with the formatting removed and its paragraphs kept."}
	if refreshError != "" {
		r.Note = "The latest refresh failed: " + refreshError
	}
	if len(items) == 0 {
		r.Status = "empty"
		if r.Note == "" {
			r.Note = "MFL has published no site news."
		}
		return r
	}
	for _, it := range items {
		r.Rows = append(r.Rows, report.Row{Group: it.Headline, Cells: []string{it.Published.In(zone).Format("Jan 2, 2006"), strings.Join(it.Lines, "\n")}})
	}
	return r
}

// AccountingURL is MFL's accounting report for the league's season on its own host (found in the
// captured MFL league pages: claudeos ~/mflmap-runs/bee-mfl-ui-map-2026-10-04/nav.log).
func AccountingURL(host string, season int) (string, error) {
	if !regexp.MustCompile(`^www[0-9]{1,3}\.myfantasyleague\.com$`).MatchString(host) {
		return "", fmt.Errorf("accounting: the league's MFL host is not known yet")
	}
	u := url.URL{Scheme: "https", Host: host, Path: fmt.Sprintf("/%d/accounting_report", season),
		RawQuery: url.Values{"L": {ingestion.LeagueID}}.Encode()}
	return u.String(), nil
}

// AccountingReport is M-098 and M-197: MFL's accounting export holds nothing for this league.
func AccountingReport() report.Report {
	return report.Report{ID: "M-098", Title: "Accounting", Status: "empty", Source: "Source: MFL accounting export, every archived season",
		AsOf: "As of: the league archive", Columns: []string{"Fact", "Value"}, Rows: []report.Row{},
		Note: "MFL's accounting export holds no entries for this league in any season from 2013 to 2026. " +
			"MFL's own accounting page opens from here.",
		How: "Checked against every archived accounting.json in the league archive."}
}

func OpenAccounting(ready func() error, host func() string, season func() int, open func(string)) error {
	if err := ready(); err != nil {
		return fmt.Errorf("open accounting: %w", err)
	}
	target, err := AccountingURL(host(), season())
	if err != nil {
		return err
	}
	open(target)
	return nil
}

package control

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/report"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/league"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/sitenews"
)

func rows(r report.Report) map[string]string {
	out := map[string]string{}
	for _, row := range r.Rows {
		out[row.Group+" / "+row.Cells[0]] = row.Cells[1]
	}
	return out
}

// readTestdata reads a fixture or golden at its canonical repository path.
func readTestdata(t *testing.T, path string) []byte {
	t.Helper()
	b, err := os.ReadFile(repoPath(path))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// realSettingsReport is League settings from the real 2026 league, rules and scoring-dictionary exports.
func realSettingsReport(t *testing.T) report.Report {
	t.Helper()
	read := func(f string) []byte { return readTestdata(t, "internal/ingestion/league/testdata/"+f) }
	cfg, err := league.Parse(read("league-2026.json"), read("rules-2026.json"))
	if err != nil {
		t.Fatal(err)
	}
	codes, err := league.ParseRuleDictionary(read("allRules.json"))
	if err != nil {
		t.Fatal(err)
	}
	return LeagueSettingsReport(cfg, league.RuleNames(codes), "")
}

func TestLeagueSettingsReadTheReal2026ExportsInPlainWords(t *testing.T) {
	t.Parallel()
	r := realSettingsReport(t)
	got := rows(r)
	for k, want := range map[string]string{
		"Season / Scoring weeks":                         "Weeks 1 to 17; the regular season ends after week 13",
		"Season / Standings order":                       "win percentage, then head-to-head record, then division win percentage, then conference win percentage, then all-play win percentage, then points scored, then power rank",
		"Roster / Roster size":                           "80 players",
		"Roster / Position limits":                       "None: any mix of positions",
		"Starters / Starters each week":                  "21",
		"Starters / QB":                                  "Exactly 1",
		"Starters / WR":                                  "2 to 5",
		"Salary cap and contracts / Salary cap":          "$125,000,000",
		"Salary cap and contracts / Taxi squad salaries": "100% of a taxi player's salary counts against the cap",
		"Salary cap and contracts / Contract years":      "Every contract runs for a set number of seasons",
		"Scoring · CB, S / Tackles":                      "",
	} {
		if want == "" {
			if _, ok := got[k]; !ok {
				t.Errorf("%s missing", k)
			}
			continue
		}
		if got[k] != want {
			t.Errorf("%s = %q, want %q", k, got[k], want)
		}
	}
	for k, v := range got {
		if strings.Contains(k, "MFL event ") || strings.Contains(v, "*") || strings.Contains(v, "to 50-") ||
			strings.HasPrefix(v, ".") || strings.HasPrefix(v, "1 points") || strings.HasPrefix(v, "0 points") {
			t.Errorf("a code or MFL's points syntax reached the GM: %s = %s", k, v)
		}
	}
	if empty := LeagueSettingsReport(league.RawConfig{}, nil, ""); empty.Status != "empty" {
		t.Fatal("no held settings must say so")
	}
}

func TestSiteNewsReportAndAccountingHandOff(t *testing.T) {
	t.Parallel()
	at := time.Date(2026, 10, 8, 3, 32, 0, 0, time.UTC)
	r := SiteNewsReport([]sitenews.Item{{Headline: "Week 4 Strange Plays", Published: at, Lines: []string{"One.", "Two."}}},
		"As of: now", "", time.UTC)
	if r.Status != "ok" || r.Rows[0].Group != "Week 4 Strange Plays" || r.Rows[0].Cells[0] != "Oct 8, 2026" ||
		r.Rows[0].Cells[1] != "One.\nTwo." {
		t.Fatalf("site news = %+v", r.Rows)
	}
	if e := SiteNewsReport(nil, "As of: now", "MFL down", time.UTC); e.Status != "empty" || !strings.Contains(e.Note, "MFL down") {
		t.Fatalf("empty news = %+v", e)
	}
	u, err := AccountingURL("www47.myfantasyleague.com", 2026)
	if err != nil || u != "https://www47.myfantasyleague.com/2026/accounting_report?L=14432" {
		t.Fatalf("accounting url = %q, %v", u, err)
	}
	for _, bad := range []string{"", "api", "www47", "evil.example.com", "www47.evil", "www47.myfantasyleague.com/path", "www47.myfantasyleague.com:443"} {
		if _, err := AccountingURL(bad, 2026); err == nil {
			t.Errorf("host %q accepted", bad)
		}
	}
}

func TestBylawsReadTheRulebookBySection(t *testing.T) {
	t.Parallel()
	sections := BylawSections(readTestdata(t, "docs/league-rules/Official_Rulebook.md"))
	if len(sections) < 13 || sections[0].Title != "1. SALARY CAP" {
		t.Fatalf("%d sections, first %q", len(sections), sections[0].Title)
	}
	joined := ""
	for _, s := range sections {
		joined += s.Title + "\n" + strings.Join(s.Lines, "\n")
	}
	for _, want := range []string{"• Each team: $125M cap for 2026", "Active Roster: Minimum 35", "Passing:"} {
		if !strings.Contains(joined, want) {
			t.Errorf("rulebook text lost %q", want)
		}
	}
	for _, markup := range []string{"**", "|---", "## "} {
		if strings.Contains(joined, markup) {
			t.Errorf("markdown %q reached the GM", markup)
		}
	}
}

// TestControlRoomWire writes the Control Room's real Go output for the frontend's tests: League
// settings, the first by-laws sections and accounting from the app, and MFL's real site news.
func TestControlRoomWire(t *testing.T) {
	t.Parallel()
	raw, err := sitenews.Parse(readTestdata(t, "internal/ingestion/sitenews/testdata/siteNews.json"))
	if err != nil {
		t.Fatal(err)
	}
	items, err := sitenews.Read(raw)
	if err != nil {
		t.Fatal(err)
	}
	settings := realSettingsReport(t)
	settings.Rows = settings.Rows[:12:12]
	body, err := json.MarshalIndent(map[string]any{
		"settings": settings, "accounting": AccountingReport(),
		"bylaws": BylawSections(readTestdata(t, "docs/league-rules/Official_Rulebook.md"))[:3],
		"news":   SiteNewsReport(items[:2:2], "As of: Oct 12, 2026, 8:00 AM", "", time.UTC),
	}, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join("frontend", "src", "app", "control", "wire.testdata.json")
	if os.Getenv("TWR_WRITE_WIRE") == "1" {
		if err := os.WriteFile(repoPath(path), append(body, '\n'), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if strings.TrimSpace(string(readTestdata(t, path))) != string(body) {
		t.Fatal("Control Room wire is stale")
	}
}

func repoPath(path string) string { return filepath.Join("..", "..", "..", path) }

package bid

import (
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/domain"
)

const tenK domain.Money = 1_000_000

func millions(m float64) domain.Money { return domain.Money(m * 100_000_000) }

func TestScoreRulebookExamples(t *testing.T) {
	for _, tc := range []struct {
		salary domain.Money
		years  int
		want   string
	}{
		{millions(5), 1, "5.000"},
		{millions(5), 2, "6.000"},
		{millions(1.3), 3, "1.820"},
		{millions(24.5), 4, "39.200"},
		{millions(0.33), 1, "0.330"},
		{millions(5) + 4*tenK/10, 1, "5.000"}, // $5,000,400 snaps down
		{millions(5) + 5*tenK/10, 1, "5.010"}, // $5,000,500 snaps up
	} {
		got, err := Score(tc.salary, tc.years)
		if err != nil || got.String() != tc.want {
			t.Errorf("Score(%v, %d) = %v, %v; want %s", tc.salary, tc.years, got, err, tc.want)
		}
	}
	for _, years := range []int{0, 5, -1} {
		if _, err := Score(millions(5), years); err == nil {
			t.Errorf("Score with %d years succeeded", years)
		}
	}
}

func TestParsePoints(t *testing.T) {
	if got, err := ParsePoints(" 7.250 "); err != nil || got.String() != "7.250" {
		t.Errorf("ParsePoints with spaces = %v, %v", got, err)
	}
	for raw, want := range map[string]string{"7": "7.000", "7.5": "7.500", ".5": "0.500", "0.001": "0.001"} {
		if got, err := ParsePoints(raw); err != nil || got.String() != want {
			t.Errorf("ParsePoints(%q) = %v, %v; want %s", raw, got, err, want)
		}
	}
	for _, raw := range []string{"", ".", "7.2501", "-1", "x", "1e3", "7,5"} {
		if _, err := ParsePoints(raw); err == nil {
			t.Errorf("ParsePoints(%q) succeeded", raw)
		}
	}
}

func TestMinimumSalaryTable(t *testing.T) {
	for exp, want := range map[int]int64{-1: 330_000, 0: 330_000, 1: 380_000, 2: 430_000, 3: 480_000,
		4: 530_000, 6: 530_000, 7: 580_000, 9: 580_000, 10: 630_000, 15: 630_000} {
		if got := MinimumSalary(exp); got != domain.Money(want*100) {
			t.Errorf("MinimumSalary(%d) = %v, want $%d", exp, got, want)
		}
	}
}

// passing is a 2-year $5M bid in the offseason that every rule accepts.
func TestPostText(t *testing.T) {
	p := Post{Player: "Smith, John (WR, KCC)", Salary: millions(5), Years: 3, Points: 7000}
	want := "Player: Smith, John (WR, KCC)\nContract: 3 years at $5,000,000 a year, flat\n" +
		"Cap hit: $5,000,000 a year\nPoints: 7.000"
	if got := p.Text(); got != want {
		t.Errorf("post text:\n%s\nwant:\n%s", got, want)
	}
	p.Snipe, p.Years = true, 1
	if got := p.Text(); !strings.HasSuffix(got, "\nSnipe") || !strings.Contains(got, "1 year at") {
		t.Errorf("snipe post text:\n%s", got)
	}
}

func TestExperienceFrom(t *testing.T) {
	for _, tc := range []struct {
		name   string
		season int
		draft  int
		has    bool
		want   Experience
	}{
		{"drafted last year", 2026, 2025, true, Experience{Years: 1, Known: true}},
		{"drafted this year", 2026, 2026, true, Experience{Years: 0, Known: true}},
		{"thirty years back", 2026, 1996, true, Experience{Years: 30, Known: true}},
		{"the 1970 placeholder", 2026, 1970, true, Experience{}},
		{"a future draft year", 2026, 2027, true, Experience{}},
		{"undrafted", 2026, 0, false, Experience{}},
	} {
		if got := ExperienceFrom(tc.season, tc.draft, tc.has); got != tc.want {
			t.Errorf("%s: %+v, want %+v", tc.name, got, tc.want)
		}
	}
}

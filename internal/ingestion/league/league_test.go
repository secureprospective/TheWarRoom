package league

import (
	"errors"
	"reflect"
	"testing"
)

// canned MFL bodies. The starters.position list and the second positionRules block's
// rule are deliberately SINGLE objects (not arrays) to exercise the MFL
// array/object-collapse decode that the live API produces for one-element lists.
const leagueBody = `{"league":{
	"salaryCapAmount":"120","rosterSize":"80","taxiSquad":"8","injuredReserve":"12",
	"keeperType":"dynasty","usesSalaries":"1","usesContractYear":"1",
	"startWeek":"1","endWeek":"17","lastRegularSeasonWeek":"13",
	"rosterLimits":{"position":[{"name":"QB","limit":"0-0"},{"name":"RB","limit":"0-0"}]},
	"starters":{"count":"21","iop_starters":"8","idp_starters":"12",
		"position":{"name":"QB","limit":"1"}},
	"franchises":{"franchise":[
		{"id":"0001","name":"Gridiron Gurus","owner_name":"x"},
		{"id":"0002","name":""},
		{"name":"no id — dropped"}]}}}`

const rulesBody = `{"rules":{"positionRules":[
	{"positions":"CB|S","rule":[
		{"event":{"$t":"TK"},"points":{"$t":"*0.5"},"range":{"$t":"0-99"}},
		{"event":{"$t":"FG"},"points":{"$t":"3"},"range":{"$t":"0-39"}}]},
	{"positions":"QB|WR|TE|DT|RB","rule":
		{"event":{"$t":"#P"},"points":{"$t":"*0"},"range":{"$t":"0-0"}}}]}}`

func TestAssemble_DecodesBothExports(t *testing.T) {
	cfg, err := assemble([]byte(leagueBody), []byte(rulesBody))
	if err != nil {
		t.Fatalf("assemble: %v", err)
	}

	if cfg.SalaryCapAmount != "120" {
		t.Errorf("cap = %q, want 120", cfg.SalaryCapAmount)
	}
	if len(cfg.RosterLimits) != 2 {
		t.Errorf("roster limits = %d, want 2", len(cfg.RosterLimits))
	}
	// starters.position collapsed from a single object to one entry.
	if len(cfg.Starters.Positions) != 1 || cfg.Starters.Positions[0].Name != "QB" {
		t.Errorf("starters positions = %+v, want one QB", cfg.Starters.Positions)
	}
	if cfg.Starters.IDPStarters != "12" {
		t.Errorf("idp starters = %q, want 12", cfg.Starters.IDPStarters)
	}
}

func TestAssemble_DecodesFranchiseDirectory(t *testing.T) {
	cfg, err := assemble([]byte(leagueBody), []byte(rulesBody))
	if err != nil {
		t.Fatalf("assemble: %v", err)
	}
	// Two entries survive (the id-less row is dropped); the blank name is kept raw
	// (FranchiseNames, not the fetcher, decides to omit it).
	if len(cfg.Franchises) != 2 {
		t.Fatalf("franchises = %+v, want 2 (id-less dropped)", cfg.Franchises)
	}
	if cfg.Franchises[0].ID != "0001" || cfg.Franchises[0].Name != "Gridiron Gurus" {
		t.Errorf("franchise[0] = %+v, want 0001/Gridiron Gurus", cfg.Franchises[0])
	}
	if cfg.Franchises[1].ID != "0002" || cfg.Franchises[1].Name != "" {
		t.Errorf("franchise[1] = %+v, want 0002 with blank name", cfg.Franchises[1])
	}
}

func TestAssemble_UnwrapsAndCollapsesScoring(t *testing.T) {
	cfg, err := assemble([]byte(leagueBody), []byte(rulesBody))
	if err != nil {
		t.Fatalf("assemble: %v", err)
	}
	if len(cfg.ScoringRules) != 2 {
		t.Fatalf("rule sets = %d, want 2", len(cfg.ScoringRules))
	}

	// First block: two rules, sorted by event, $t unwrapped (incl. a flat "3" with no "*").
	first := cfg.ScoringRules[0]
	if first.Positions != "CB|S" || len(first.Rules) != 2 {
		t.Fatalf("first block = %+v", first)
	}
	if first.Rules[0].Points != "3" {
		t.Errorf("FG points = %q, want raw 3", first.Rules[0].Points)
	}
	if first.Rules[1].Event != "TK" || first.Rules[1].Points != "*0.5" || first.Rules[1].Range != "0-99" {
		t.Errorf("TK rule = %+v, want TK/*0.5/0-99", first.Rules[1])
	}

	// Second block: rule collapsed from a single object to one entry.
	second := cfg.ScoringRules[1]
	if len(second.Rules) != 1 || second.Rules[0].Event != "#P" {
		t.Errorf("second block = %+v, want one #P rule", second)
	}
}

func TestAssemble_EmptyScoringFailsLoud(t *testing.T) {
	_, err := assemble([]byte(leagueBody), []byte(`{"rules":{"positionRules":[]}}`))
	if !errors.Is(err, errEmptyScoring) {
		t.Fatalf("err = %v, want errEmptyScoring", err)
	}
}

func TestAssemble_SalaryLeagueMissingCapFailsLoud(t *testing.T) {
	body := `{"league":{"usesSalaries":"1","salaryCapAmount":""}}`
	_, err := assemble([]byte(body), []byte(rulesBody))
	if err == nil {
		t.Fatal("want error for salary league missing cap amount")
	}
}

func TestAssemble_CurrentSeasonIsNewestYearUnderThisLeaguesID(t *testing.T) {
	body := `{"league":{"id":"14432","salaryCapAmount":"125","usesSalaries":"1","history":{"league":[
		{"year":"2026","url":"https://www47.myfantasyleague.com/2026/home/14432"},
		{"year":"2031","url":"https://www48.myfantasyleague.com/2031/home/51719"},
		{"year":"2025","url":"https://www47.myfantasyleague.com/2025/home/14432"}]}}}`
	cfg, err := assemble([]byte(body), []byte(rulesBody))
	if err != nil {
		t.Fatalf("assemble: %v", err)
	}
	if cfg.CurrentSeason != 2026 {
		t.Errorf("current season = %d, want 2026 (2031 belongs to another league id)", cfg.CurrentSeason)
	}
}

// MFL shuffles the positionRules blocks between requests; the config must not change with it,
// or every refresh would write a new rulebook version.
func TestAssemble_ScoringIsTheSameInAnyOrder(t *testing.T) {
	shuffled := `{"rules":{"positionRules":[
	{"positions":"QB|WR|TE|DT|RB","rule":
		{"event":{"$t":"#P"},"points":{"$t":"*0"},"range":{"$t":"0-0"}}},
	{"positions":"CB|S","rule":[
		{"event":{"$t":"FG"},"points":{"$t":"3"},"range":{"$t":"0-39"}},
		{"event":{"$t":"TK"},"points":{"$t":"*0.5"},"range":{"$t":"0-99"}}]}]}}`
	a, err := assemble([]byte(leagueBody), []byte(rulesBody))
	if err != nil {
		t.Fatal(err)
	}
	b, err := assemble([]byte(leagueBody), []byte(shuffled))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(a.ScoringRules, b.ScoringRules) {
		t.Errorf("same rules in another order assembled differently:\n%+v\n%+v", a.ScoringRules, b.ScoringRules)
	}
}

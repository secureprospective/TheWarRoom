package league

import (
	"errors"
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

func TestAssemble_EmptyScoringFailsLoud(t *testing.T) {
	_, err := Parse([]byte(leagueBody), []byte(`{"rules":{"positionRules":[]}}`))
	if !errors.Is(err, errEmptyScoring) {
		t.Fatalf("err = %v, want errEmptyScoring", err)
	}
}

func TestAssemble_SalaryLeagueMissingCapFailsLoud(t *testing.T) {
	body := `{"league":{"usesSalaries":"1","salaryCapAmount":""}}`
	_, err := Parse([]byte(body), []byte(rulesBody))
	if err == nil {
		t.Fatal("want error for salary league missing cap amount")
	}
}

func TestParseCarriesPartialLineupPermission(t *testing.T) {
	cfg, err := Parse([]byte(`{"league":{"partialLineupAllowed":"YES"}}`), []byte(rulesBody))
	if err != nil || cfg.PartialLineupAllowed != "YES" {
		t.Fatalf("partial-lineup permission lost: %+v %v", cfg, err)
	}
}

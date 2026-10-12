package facts

import (
	"strings"
	"testing"
)

func TestRosterDoubleListingKeepsTheSpecificStatus(t *testing.T) {
	// MFL lists an IR player twice; the order varies, and the specific status wins either way.
	facts := compileFor(t, []ResolvedInput{inputFor("2020/rosters.json", `{"rosters":{"franchise":{"id":"0001","player":[
		{"id":"1","status":"INJURED_RESERVE"},{"id":"1","status":"ROSTER"},
		{"id":"2","status":"ROSTER"},{"id":"2","status":"TAXI_SQUAD","contractInfo":"  "}]}}}`)})
	r := facts.RosterWeek
	if len(r.Pid) != 2 || value(r.Status[0]) != "INJURED_RESERVE" || value(r.Status[1]) != "TAXI_SQUAD" {
		t.Fatalf("statuses %v", r.Status)
	}
	if r.Week[0] != 0 || r.ContractInfo[1] != nil {
		t.Fatal("end-of-season roster is week 0; blank contract text is null")
	}
	if len(facts.LoadNote.N) != 1 || facts.LoadNote.N[0] != 2 {
		t.Fatalf("doubled listings not counted: %+v", facts.LoadNote)
	}
}

func TestResultsGamesLineupsAndViews(t *testing.T) {
	results := `{"allWeeklyResults":{"weeklyResults":[
		{"week":"1","matchup":[
			{"franchise":[{"id":"0001","score":"100","result":"W","opt_pts":"120","starters":"10,11","nonstarters":"12",
				"player":[{"id":"11","status":"nonstarter","shouldStart":"1","score":"9"}]},
				{"id":"0002","score":"90","result":"L"}]},
			{"franchise":[{"id":"0003","score":"90","result":"W"},{"id":"BYE"}]}],
		 "franchise":{"id":"0001","score":"100"}},
		{"week":"2","franchise":[{"id":"0001","score":"","result":"T"},{"id":"0002","score":"50"}]}]}}`
	league := `{"league":{"id":"9","franchises":{"franchise":[{"id":"0001"},{"id":"0002"}]}}}`
	standings := `{"leagueStandings":{"franchise":{"id":"0002","all_play_wlt":"7-8-0"}}}`
	facts := compileFor(t, []ResolvedInput{inputFor("2020/weeklyResults_WYTD.json", results),
		inputFor("2020/league.json", league), inputFor("2020/leagueStandings.json", standings)})
	g := facts.Game
	if len(g.Fid) != 5 || g.Opp[2] != nil || value(g.Result[3]) != "T" || g.Score[3] != nil {
		t.Fatalf("games: opp %v result %v", g.Opp, g.Result)
	}
	l := facts.Lineup
	started := map[string]int64{}
	for i, pid := range l.Pid {
		started[pid] = l.Started[i]
	}
	if len(l.Pid) != 3 || started["10"] != 1 || started["11"] != 0 || started["12"] != 0 {
		t.Fatalf("an explicit status beats the starters list: %v", started)
	}
	notes := strings.Join(facts.LoadNote.Note, "|")
	if !strings.Contains(notes, "standalone copy") || !strings.Contains(notes, "BYE") {
		t.Fatalf("notes %q", notes)
	}
	v := facts.Views()
	if len(v.PlayedGame.Fid) != 4 || len(v.RegularWeek.Week) != 1 || v.RegularWeek.Week[0] != 1 {
		t.Fatalf("played %d regular %v", len(v.PlayedGame.Fid), v.RegularWeek.Week)
	}
	ts := v.TeamSeason
	if *ts.RegW[0] != 1 || *ts.RegPf[0] != 100 || *ts.AllWeeksPp[0] != 120 || ts.AllWeeksPp[1] != nil {
		t.Fatal("team season counts")
	}
	// MFL's all-play wins for 0002; 0001's is counted from the weekly scores, week 2 alone not counting.
	if ts.AllPlaySource[1] != "MFL" || *ts.AllPlayW[1] != 7 || *ts.AllPlayT[1] != 0 || ts.AllPlaySource[0] != "counted" ||
		*ts.AllPlayW[0] != 2 || *ts.AllPlayL[0] != 0 {
		t.Fatalf("all-play %v %v", ts.AllPlaySource, ts.AllPlayW)
	}
}

func TestStandaloneCopyWithADifferentScoreIsAnError(t *testing.T) {
	results := `{"allWeeklyResults":{"weeklyResults":{"week":"1",
		"matchup":{"franchise":[{"id":"0001","score":"100"},{"id":"0002","score":"90"}]},
		"franchise":{"id":"0001","score":"101"}}}}`
	_, err := Compile(t.Context(), []ResolvedInput{inputFor("2020/weeklyResults_WYTD.json", results)}, nil)
	if err == nil || !strings.Contains(err.Error(), "differs") {
		t.Fatalf("err %v", err)
	}
}

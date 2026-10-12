package facts

import (
	"os"
	"strconv"
	"strings"
	"testing"
)

// TestReducedChecks runs the nine checks on the reduced fixture. The fixture keeps every body only
// for 2013, 2020 and 2026, so a difference outside those seasons is the fixture's gap; inside them
// every difference must be pinned, in the class the reference gave it.
func TestReducedChecks(t *testing.T) {
	facts, err := Compile(t.Context(), fixtureInputs(t), nil)
	if err != nil {
		t.Fatal(err)
	}
	results, _ := facts.Check(pinnedExceptions(t))
	want := map[string][2]int{ // explained, review within the three seasons
		"Win-loss records match MFL standings": {12, 0}, "Team scores equal their starters": {11, 0},
		"Roster changes are recorded moves (2018 on)": {0, 2}, "Draft picks were used by their last owner (2017 on)": {0, 49},
		"Rostered players have an MFL birthdate": {0, 21},
	}
	for _, r := range results {
		for _, it := range r.Items {
			year, _ := strconv.Atoi(it.Key[:4])
			if it.Class == ClassUnpinned && selectedYear(int64(year)) {
				t.Errorf("%s: unpinned %s (%s)", r.Name, it.Key, it.Words)
			}
		}
		if got := [2]int{r.Explained, r.Review}; got != want[r.Name] {
			t.Errorf("%s: explained/review %v, want %v", r.Name, got, want[r.Name])
		}
	}
}

// TestAnUnexpectedIdentityFailsEvenWithTheSameCount: one difference is pinned and one occurs, but
// not the pinned one, so the check fails; a count never stands in for an identity.
func TestAnUnexpectedIdentityFailsEvenWithTheSameCount(t *testing.T) {
	results := `{"allWeeklyResults":{"weeklyResults":{"week":"1","matchup":{"franchise":[
		{"id":"0001","score":"100","result":"W","starters":"10","player":{"id":"10","status":"starter","score":"90"}},
		{"id":"0002","score":"50","result":"L","starters":"20","player":{"id":"20","status":"starter","score":"50"}}]}}}}`
	facts := compileFor(t, []ResolvedInput{inputFor("2020/weeklyResults_WYTD.json", results)})
	check := "Team scores equal their starters"
	for key, wantErr := range map[string]bool{"2020/01/0002": true, "2020/01/0001": false} {
		pinned, err := ParseExceptions([]byte("check,key,class,reason\n" + check + "," + key + ",review,A test reason.\n"))
		if err != nil {
			t.Fatal(err)
		}
		res, err := facts.Check(pinned)
		r := res[4]
		if r.Name != check || (err != nil) != wantErr || r.Passed == wantErr || len(r.Items) != 1 {
			t.Fatalf("pinned %s: err %v, result %+v", key, err, r)
		}
		if wantErr && (!strings.Contains(err.Error(), "team score 100.00, its starters sum to 90.00") || r.Items[0].Class != ClassUnpinned) {
			t.Fatalf("the error must name the difference in words: %v", err)
		}
	}
}

func TestExceptionsFileIsWellFormed(t *testing.T) {
	if x := pinnedExceptions(t); len(x.byKey) != 635 {
		t.Fatalf("pinned %d, want the reference's 635 identities", len(x.byKey))
	}
	for _, bad := range []string{"check,key,class,reason\nA,B,maybe,R\n", "check,key,class,reason\nA,B,review,\n",
		"check,key,class,reason\nA,B,review,R\nA,B,review,R\n", "check,key,reason\n"} {
		if _, err := ParseExceptions([]byte(bad)); err == nil {
			t.Errorf("accepted %q", bad)
		}
	}
}

// pinnedExceptions reads the exceptions file the app embeds (internal/lab).
func pinnedExceptions(t *testing.T) Exceptions {
	t.Helper()
	body, err := os.ReadFile("../lab/testdata/check-exceptions.csv")
	if err != nil {
		t.Fatal(err)
	}
	x, err := ParseExceptions(body)
	if err != nil {
		t.Fatal(err)
	}
	return x
}

// TestRosterLedgerWindow: a move counts in the window after one week's last kickoff up to and
// including the next week's; a move at the earlier kickoff belongs to the week before.
func TestRosterLedgerWindow(t *testing.T) {
	sched := func(kickoff string) string {
		return `{"nflSchedule":{"matchup":{"kickoff":"` + kickoff + `","team":[{"id":"BUF"},{"id":"MIA"}]}}}`
	}
	roster := func(pids ...string) string {
		var players []string
		for _, p := range pids {
			players = append(players, `{"id":"`+p+`","status":"ROSTER"}`)
		}
		return `{"rosters":{"franchise":{"id":"0001","player":[` + strings.Join(players, ",") + `]}}}`
	}
	facts := compileFor(t, []ResolvedInput{
		inputFor("2020/weekly/nflSchedule_W01.json", sched("1000")), inputFor("2020/weekly/nflSchedule_W02.json", sched("2000")),
		inputFor("2020/weekly/nflSchedule_W03.json", sched("3000")),
		inputFor("2020/weekly/rosters_W01.json", roster("10")), inputFor("2020/weekly/rosters_W02.json", roster("11")),
		inputFor("2020/weekly/rosters_W03.json", roster("11", "12")),
		inputFor("2020/transactions.json", `{"transactions":{"transaction":[
			{"type":"FREE_AGENT","franchise":"0001","transaction":"11,|10,","timestamp":"2000"},
			{"type":"FREE_AGENT","franchise":"0001","transaction":"12,|","timestamp":"2000"}]}}`),
	})
	res, err := facts.Check(Exceptions{})
	r := res[6]
	if err == nil || r.Checked != 3 || len(r.Items) != 1 || r.Items[0].Key != "2020/02/0001/12/joined" {
		t.Fatalf("checked %d, items %+v", r.Checked, r.Items)
	}
}

// TestTeamScoreWithinACentAgrees: MFL scores carry two decimals; a one-cent difference agrees.
func TestTeamScoreWithinACentAgrees(t *testing.T) {
	results := `{"allWeeklyResults":{"weeklyResults":{"week":"1","matchup":{"franchise":[
		{"id":"0001","score":"100.01","result":"W","starters":"10","player":{"id":"10","status":"starter","score":"100"}},
		{"id":"0002","score":"50","result":"L","starters":"20","player":{"id":"20","status":"starter","score":"50.02"}}]}}}}`
	res, _ := compileFor(t, []ResolvedInput{inputFor("2020/weeklyResults_WYTD.json", results)}).Check(Exceptions{})
	if r := res[4]; r.Checked != 2 || len(r.Items) != 1 || r.Items[0].Key != "2020/01/0002" {
		t.Fatalf("checked %d, items %+v", r.Checked, r.Items)
	}
}

package main

import (
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/normalize"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
	"github.com/secureprospective/TheWarRoom/internal/store/params"
)

const freeAgentBody = `{"version":"1.0","freeAgents":{"leagueUnit":{"unit":"LEAGUE","player":[` +
	`{"id":"13593"},{"id":"13594"},{"id":"13595"},{"id":"99999"}]}}}`

func poolApp(t *testing.T) (*App, *seasonTransport) {
	t.Helper()
	a, transport, _, _ := seasonTestApp(t)
	bidSetup(t, a)
	transport.bodies["freeAgents"] = freeAgentBody
	return a, transport
}

func freeAgentRequests(transport *seasonTransport) int {
	return strings.Count(strings.Join(transport.requests, ","), "freeAgents")
}

func poolNames(r Reading[Pool]) string {
	names := []string{}
	for _, row := range r.Value.Rows {
		names = append(names, row.Name)
	}
	return strings.Join(names, "|")
}

func TestPoolNeverFetchedIsFailedNotEmpty(t *testing.T) {
	t.Parallel()
	a, transport := poolApp(t)
	r := a.TargetPool(RawPoolScope{Sort: "name"})
	if r.Status != ReadingFailed || r.AbsenceReason == nil || !strings.Contains(*r.AbsenceReason, "have not been fetched") ||
		r.Value.Rows == nil || freeAgentRequests(transport) != 0 {
		t.Fatalf("reading = %+v, requests %v", r, transport.requests)
	}
}

func TestPoolReadsEveryFreeAgentWithGoFigures(t *testing.T) {
	t.Parallel()
	a, transport := poolApp(t)
	if err := a.TargetPoolRefresh(true); err != nil {
		t.Fatal(err)
	}
	r := a.TargetPool(RawPoolScope{FranchiseID: "0001", Sort: "name"})
	if r.Status != ReadingOK || r.Count != 3 || poolNames(r) != "Adams, Zed|Jones, Amy|Smith, John" ||
		r.Provenance.Freshness.State != "live" || r.Cutoff == "" ||
		!strings.Contains(r.HowCounted, "Left out of this list: 1 ids the directory does not know") {
		t.Fatalf("reading = %+v", r)
	}
	byName := map[string]PoolRow{}
	for _, row := range r.Value.Rows {
		byName[row.Name] = row
	}
	smith, jones, adams := byName["Smith, John"], byName["Jones, Amy"], byName["Adams, Zed"]
	if smith.Experience != "4 years" || smith.Minimum.Cents != 53_000_000 || smith.Minimum.Text != "$530,000" ||
		smith.Position != "WR" || smith.Team != "DAL" || !regexp.MustCompile(`^\d\d$`).MatchString(smith.Age) {
		t.Errorf("smith = %+v", smith)
	}
	if jones.Experience != "Unknown (rookie minimum)" || jones.Minimum.Cents != 33_000_000 || jones.Age != "—" ||
		adams.Experience != "16 years" || adams.Minimum.Text != "$630,000" {
		t.Errorf("jones = %+v adams = %+v", jones, adams)
	}
	if !slices.Equal(r.Value.Positions, []string{"QB", "WR", "CB"}) {
		t.Errorf("positions = %v", r.Value.Positions)
	}
	before := len(transport.requests)
	a.TargetPool(RawPoolScope{Sort: "name"})
	if len(transport.requests) != before {
		t.Error("reading the pool went to MFL")
	}
}

func TestPoolFiltersAndSorts(t *testing.T) {
	t.Parallel()
	a, _ := poolApp(t)
	if err := a.TargetPoolRefresh(true); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		scope RawPoolScope
		want  string
	}{
		{RawPoolScope{Sort: "name"}, "Adams, Zed|Jones, Amy|Smith, John"},
		{RawPoolScope{Sort: "name", Desc: true}, "Smith, John|Jones, Amy|Adams, Zed"},
		{RawPoolScope{Sort: "position"}, "Jones, Amy|Smith, John|Adams, Zed"},
		{RawPoolScope{Sort: "minimum"}, "Jones, Amy|Smith, John|Adams, Zed"},
		{RawPoolScope{Sort: "minimum", Desc: true}, "Adams, Zed|Smith, John|Jones, Amy"},
		{RawPoolScope{Sort: "name", Position: "QB"}, "Jones, Amy"},
		{RawPoolScope{Sort: "name", Position: "RB"}, ""},
	} {
		if got := poolNames(a.TargetPool(tc.scope)); got != tc.want {
			t.Errorf("%+v = %q, want %q", tc.scope, got, tc.want)
		}
	}
	none := a.TargetPool(RawPoolScope{Sort: "name", Position: "RB"})
	if none.Status != ReadingOK || none.Count != 0 || !strings.Contains(none.Value.Summary, "0 free agents at RB of 3") {
		t.Errorf("an empty filter = %+v", none)
	}
	for _, bad := range []RawPoolScope{{Sort: "age"}, {Sort: ""}, {Sort: "name", FranchiseID: "1"}} {
		if r := a.TargetPool(bad); r.Status != ReadingFailed {
			t.Errorf("scope %+v gave %v", bad, r.Status)
		}
	}
}

func TestPoolRefreshSkipsAFreshCopyUnlessForced(t *testing.T) {
	t.Parallel()
	a, transport := poolApp(t)
	for _, force := range []bool{false, false, true, false} {
		if err := a.TargetPoolRefresh(force); err != nil {
			t.Fatal(err)
		}
	}
	if got := freeAgentRequests(transport); got != 2 {
		t.Errorf("MFL was asked %d times, want 2", got)
	}
}

func TestPoolEmptyAndOfflineStates(t *testing.T) {
	t.Parallel()
	a, transport := poolApp(t)
	transport.failure = "freeAgents"
	if err := a.TargetPoolRefresh(true); err != nil {
		t.Fatal(err)
	}
	r := a.TargetPool(RawPoolScope{Sort: "name"})
	if r.Status != ReadingFailed || r.AbsenceReason == nil || !strings.Contains(*r.AbsenceReason, "MFL free agents unavailable") {
		t.Errorf("offline from the start = %+v", r)
	}
	transport.failure = ""
	transport.bodies["freeAgents"] = `{"freeAgents":{"leagueUnit":{"unit":"LEAGUE"}}}`
	if err := a.TargetPoolRefresh(true); err != nil {
		t.Fatal(err)
	}
	if r := a.TargetPool(RawPoolScope{Sort: "name"}); r.Status != ReadingThin || r.Count != 0 ||
		r.AbsenceReason == nil || !strings.Contains(*r.AbsenceReason, "no free agents") {
		t.Errorf("an empty pool = %+v", r)
	}
	transport.bodies["freeAgents"] = freeAgentBody
	if err := a.TargetPoolRefresh(true); err != nil {
		t.Fatal(err)
	}
	transport.failure = "freeAgents"
	if err := a.TargetPoolRefresh(true); err != nil {
		t.Fatal(err)
	}
	stale := a.TargetPool(RawPoolScope{Sort: "name"})
	if stale.Status != ReadingOK || stale.Count != 3 || stale.Provenance.Freshness.State != "stale" ||
		!strings.Contains(stale.Provenance.Freshness.Note, "free agents") {
		t.Errorf("offline after a good copy = %+v", stale)
	}
}

func TestPoolWithoutNamesIsFailed(t *testing.T) {
	t.Parallel()
	a, transport := poolApp(t)
	if err := a.TargetPoolRefresh(true); err != nil {
		t.Fatal(err)
	}
	a.lookup, a.hasLookup = normalize.Lookup{}, false
	transport.failure = "players"
	r := a.TargetPool(RawPoolScope{Sort: "name"})
	if r.Status != ReadingFailed || r.AbsenceReason == nil || !strings.Contains(*r.AbsenceReason, "Player names unavailable") {
		t.Errorf("no directory = %+v", r)
	}
}

func TestPoolRowShowsTheOwnersOpenBidPlan(t *testing.T) {
	t.Parallel()
	a, _ := poolApp(t)
	if err := a.TargetPoolRefresh(true); err != nil {
		t.Fatal(err)
	}
	plan, err := a.TargetBidDraft(rawBid("5", 3))
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ franchise, state, id string }{
		{"0001", "Ready", plan.Receipt.CorrelationID}, {"0002", "", ""}, {"", "", ""},
	} {
		r := a.TargetPool(RawPoolScope{FranchiseID: tc.franchise, Sort: "name", Position: "WR"})
		if len(r.Value.Rows) != 1 || r.Value.Rows[0].Plan != tc.state || r.Value.Rows[0].PlanID != tc.id {
			t.Errorf("franchise %q row = %+v", tc.franchise, r.Value.Rows)
		}
	}
	if _, err := a.TargetBidWithdraw(plan.Receipt.CorrelationID); err != nil {
		t.Fatal(err)
	}
	if r := a.TargetPool(RawPoolScope{FranchiseID: "0001", Sort: "name", Position: "WR"}); r.Value.Rows[0].Plan != "" {
		t.Errorf("a withdrawn plan still shows: %+v", r.Value.Rows[0])
	}
}

func TestFreeAgentChangesAreHonestAcrossLaunches(t *testing.T) {
	t.Parallel()
	a, transport := poolApp(t)
	recordSeasonArchive(t, a, "freeAgents", "", freeAgentBody, time.Now().Add(-time.Hour))
	if err := a.TargetPoolRefresh(true); err != nil {
		t.Fatal(err)
	}
	if a.modelDirty.Load() {
		t.Fatal("identical archived pool was marked changed on launch")
	}
	transport.bodies["freeAgents"] = `{"freeAgents":{"leagueUnit":{"player":[{"id":"13594"},{"id":"13593"},{"id":"99999"},{"id":"13595"}]}}}`
	if err := a.TargetPoolRefresh(true); err != nil {
		t.Fatal(err)
	}
	if a.modelDirty.Load() {
		t.Fatal("only list order changed")
	}
	a.seasonFreeAgents = heldSeasonFeed[[]playerid.PlayerID]{}
	if err := a.TargetPoolRefresh(true); err != nil {
		t.Fatal(err)
	}
	if a.modelDirty.Load() {
		t.Fatal("restoring the held pool on launch looked like new inputs")
	}
	transport.bodies["freeAgents"] = `{"freeAgents":{"leagueUnit":{"player":{"id":"13594"}}}}`
	if err := a.TargetPoolRefresh(true); err != nil {
		t.Fatal(err)
	}
	if !a.modelDirty.Load() {
		t.Fatal("changed pool did not mark engine inputs changed")
	}
}

func TestPoolNamesEngineRunInLocalTime(t *testing.T) {
	t.Parallel()
	a, _ := poolApp(t)
	if err := a.TargetPoolRefresh(true); err != nil {
		t.Fatal(err)
	}
	at := time.Date(2026, 10, 11, 12, 34, 0, 0, time.Local)
	if _, _, err := a.history.WriteModelRun(a.ctx, history.NewModelRun{Season: a.season, AsOf: at,
		Engine: "fixture", InputsHash: "fixture", Params: history.ParamSet{Params: params.DefaultSet().Model()},
		Scores: []history.ModelScore{{MFLID: "13594", Position: "WR"}}}); err != nil {
		t.Fatal(err)
	}
	r := a.TargetPool(RawPoolScope{Sort: "name"})
	if !strings.Contains(r.Value.EngineNote, engineRunText(at, time.Local)) ||
		strings.Contains(r.Value.Summary, "Engine run") {
		t.Fatal(r.Value.EngineNote, " | ", r.Value.Summary)
	}
}

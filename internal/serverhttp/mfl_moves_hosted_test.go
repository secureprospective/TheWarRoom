package serverhttp_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/accounts"
	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/leagueservice"
	"github.com/secureprospective/TheWarRoom/internal/mflmirror/mflread"
	"github.com/secureprospective/TheWarRoom/internal/model"
	"github.com/secureprospective/TheWarRoom/internal/serverhttp"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

// mflRewriteTransport routes MFL read-only requests to a local httptest.Server while keeping the
// original host in a header so the test can count discovery vs league-host calls separately.
type mflRewriteTransport struct {
	target *url.URL
	base   http.RoundTripper
}

func (rt mflRewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	clone := req.Clone(req.Context())
	clone.Header.Set("X-Original-Host", req.URL.Host)
	clone.URL.Scheme = rt.target.Scheme
	clone.URL.Host = rt.target.Host
	clone.Host = rt.target.Host
	return rt.base.RoundTrip(clone)
}

type fakeMFLHTTPServer struct {
	server      *httptest.Server
	totalCalls  atomic.Int32
	leagueCalls atomic.Int32
}

func newFakeMFLHTTPServer(t *testing.T) *fakeMFLHTTPServer {
	t.Helper()
	fake := &fakeMFLHTTPServer{}
	fake.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fake.totalCalls.Add(1)
		exportType := r.URL.Query().Get("TYPE")
		if exportType == "league" {
			fake.leagueCalls.Add(1)
		}
		w.Header().Set("Content-Type", "application/json")
		switch exportType {
		case "league":
			_, _ = w.Write([]byte(`{"league":{"id":"14432","name":"Fake MFL","baseURL":"https://www47.myfantasyleague.com/2026","salaryCapAmount":"120","usesSalaries":"1","usesContractYear":"1","rosterSize":"48","injuredReserve":"12","taxiSquad":"8","includeTaxiWithSalary":"0","includeIRWithSalary":"50","partialLineupAllowed":"YES","startWeek":"1","endWeek":"17","lastRegularSeasonWeek":"13","starters":{"count":"3","position":[{"name":"QB","limit":"1"},{"name":"RB","limit":"1"},{"name":"WR","limit":"1"}]},"franchises":{"franchise":[{"id":"0001","name":"Alpha"},{"id":"0002","name":"Beta"}]}}}`))
		case "rules":
			_, _ = w.Write([]byte(`{"rules":{"positionRules":[{"positions":"QB|RB|WR","rule":[{"event":{"$t":"#P"},"range":{"$t":"0-20"},"points":{"$t":"*4"}}]}]}}`))
		case "freeAgents":
			_, _ = w.Write([]byte(`{"freeAgents":{"leagueUnit":{"unit":"LEAGUE","player":[{"id":"0531"},{"id":"13593"}]}}}`))
		case "players":
			_, _ = w.Write([]byte(`{"players":{"player":[` +
				`{"id":"0531","name":"Doe, John","position":"WR","team":"BUF","draft_year":"2015","birthdate":"788918400"},` +
				`{"id":"13593","name":"Roe, Rich","position":"RB","team":"FA","draft_year":"2026","birthdate":"1041379200"},` +
				`{"id":"1001","name":"Quinn QB","position":"QB","team":"BUF","draft_year":"2021","birthdate":"915148800"},` +
				`{"id":"1002","name":"Rex RB","position":"RB","team":"DAL","draft_year":"2022","birthdate":"946684800"},` +
				`{"id":"1003","name":"Wes WR","position":"WR","team":"NYJ","draft_year":"2023","birthdate":"978307200"}]}}`))
		case "rosters":
			_, _ = w.Write([]byte(`{"rosters":{"franchise":[` +
				`{"id":"0001","player":[{"id":"1001","salary":"1.00","contractYear":"2027","status":"ROSTER"},{"id":"1002","salary":"1.00","contractYear":"2027","status":"ROSTER"},{"id":"1003","salary":"1.00","contractYear":"2027","status":"ROSTER"}]},` +
				`{"id":"0002","player":[{"id":"1007","salary":"1.00","contractYear":"2027","status":"ROSTER"}]}]}}`))
		case "salaryAdjustments":
			_, _ = w.Write([]byte(`{"salaryAdjustments":{"salaryAdjustment":[]}}`))
		case "nflByeWeeks":
			_, _ = w.Write([]byte(`{"nflByeWeeks":{"year":"2026","team":[{"id":"BUF","bye_week":"12"},{"id":"DAL","bye_week":"7"}]}}`))
		default:
			http.Error(w, fmt.Sprintf("unexpected MFL export %q", exportType), http.StatusBadRequest)
		}
	}))
	t.Cleanup(fake.server.Close)
	return fake
}

func (f *fakeMFLHTTPServer) fetcher(t *testing.T) mflread.Fetcher {
	t.Helper()
	target, err := url.Parse(f.server.URL)
	if err != nil {
		t.Fatal(err)
	}
	return mflread.Fetcher{
		Transport: mflRewriteTransport{target: target, base: f.server.Client().Transport},
		RPS:       1000,
	}
}

func invokeRaw(t *testing.T, c hostedWorkflowClient, command, mode string, args []byte, status int) []byte {
	t.Helper()
	raw := workflowRequest(t, command, mode, string(args), c.football)
	body, err := json.Marshal(serverhttp.RawNativeRequest{Protocol: 1, Workflow: &raw})
	if err != nil {
		t.Fatal(err)
	}
	return c.f.post(t, serverhttp.WorkflowPath, string(body), c.bearer, c.proof, status)
}

func TestHostedMflMovesEndToEnd(t *testing.T) {
	c := workflowClient(t)
	fakeMFL := newFakeMFLHTTPServer(t)
	c.f.mfl = fakeMFL.fetcher(t)

	// Seed a model run so pool valuation through footballHost/dueHost is exercised too.
	run := history.NewModelRun{
		Season: 2026, AsOf: time.Now().UTC(), Engine: "test", InputsHash: "h",
		Params: history.ParamSet{Params: map[string]float64{"a": 1}},
		Scores: []history.ModelScore{
			{MFLID: "0531", Position: "WR", Value: model.Value{NowPPG: 12.5, DynastyPPG: 10.2}},
		},
	}
	if _, _, err := c.f.history.WriteModelRun(t.Context(), run); err != nil {
		t.Fatal(err)
	}

	leagueScope := []byte(`{"league":"native.hosted"}`)

	// 1. No-league reply: teams.read returns 200 with empty mflLeague and plain words.
	var noLeague leagueservice.MflTeams
	if err := json.Unmarshal(c.invoke(leagueservice.MflMovesTeamsRead, "read", leagueScope, 200).Data, &noLeague); err != nil {
		t.Fatal(err)
	}
	if noLeague.MFLLeague != "" || len(noLeague.Teams) != 0 || !strings.Contains(noLeague.Words, "follows no MFL league yet") {
		t.Fatalf("no-league reply = %+v", noLeague)
	}

	// 2. Set mfl.league = "14432" through the real commish.rules.set workflow.
	setRule := c.command(commissioner.RulesSet, commissioner.RawArgs{
		Field: "mfl.league", Value: json.RawMessage(`"14432"`),
	}, "set-mfl-league")
	ruleBody, err := json.Marshal(setRule)
	if err != nil {
		t.Fatal(err)
	}
	c.invoke(commissioner.RulesSet, "execute", ruleBody, 200)

	// 2b. Before first refresh, teams.read returns 200 with mflLeague set and plain refresh words.
	var unheld leagueservice.MflTeams
	if err := json.Unmarshal(c.invoke(leagueservice.MflMovesTeamsRead, "read", leagueScope, 200).Data, &unheld); err != nil {
		t.Fatal(err)
	}
	if unheld.MFLLeague != "14432" || len(unheld.Teams) != 0 || !strings.Contains(unheld.Words, "Refresh from MFL") {
		t.Fatalf("unheld teams reply = %+v", unheld)
	}

	// 3. First refresh calls fake MFL httptest server; league export fetched once on league host.
	var refreshed leagueservice.MflRefresh
	if err := json.Unmarshal(c.invoke(leagueservice.MflMovesRefresh, "execute", leagueScope, 200).Data, &refreshed); err != nil {
		t.Fatal(err)
	}
	if !refreshed.AskedMFL || refreshed.MFLLeague != "14432" || refreshed.FreeAgents != 2 {
		t.Fatalf("first refresh = %+v", refreshed)
	}
	if fakeMFL.leagueCalls.Load() != 1 {
		t.Fatalf("MFL TYPE=league fetched %d times per refresh, want 1", fakeMFL.leagueCalls.Load())
	}
	callsAfterFirst := fakeMFL.totalCalls.Load()

	// 4. Second refresh inside 5-minute window returns held copy without asking MFL again.
	var held leagueservice.MflRefresh
	if err := json.Unmarshal(c.invoke(leagueservice.MflMovesRefresh, "execute", leagueScope, 200).Data, &held); err != nil {
		t.Fatal(err)
	}
	if held.AskedMFL || fakeMFL.totalCalls.Load() != callsAfterFirst {
		t.Fatalf("second refresh called MFL: %+v calls %d -> %d", held, callsAfterFirst, fakeMFL.totalCalls.Load())
	}

	// 5. teams.read returns Alpha & Beta.
	var teams leagueservice.MflTeams
	if err := json.Unmarshal(c.invoke(leagueservice.MflMovesTeamsRead, "read", leagueScope, 200).Data, &teams); err != nil {
		t.Fatal(err)
	}
	if len(teams.Teams) != 2 || teams.Yours != "" {
		t.Fatalf("teams.read = %+v", teams)
	}

	// 5b. bid.check before picking team returns plain-word 409 refusal.
	pickMsg := invokeRaw(t, c, leagueservice.MflMovesBidCheck, "read",
		[]byte(`{"league":"native.hosted","add":"0531","amount":1000000,"years":1}`), 409)
	if !strings.Contains(string(pickMsg), "Pick your MFL team first") {
		t.Fatalf("bid.check before pick team = %q", pickMsg)
	}

	// 6. team.set picks 0001.
	var picked leagueservice.MflTeams
	if err := json.Unmarshal(c.invoke(leagueservice.MflMovesTeamSet, "execute",
		[]byte(`{"league":"native.hosted","team":"0001"}`), 200).Data, &picked); err != nil {
		t.Fatal(err)
	}
	if picked.Yours != "0001" {
		t.Fatalf("team.set = %+v", picked)
	}

	// 7. roster.read returns team 0001's players.
	var roster leagueservice.MflRoster
	if err := json.Unmarshal(c.invoke(leagueservice.MflMovesRosterRead, "read", leagueScope, 200).Data, &roster); err != nil {
		t.Fatal(err)
	}
	if roster.Team != "0001" || len(roster.Players) != 3 {
		t.Fatalf("roster.read = %+v", roster)
	}

	// 8. pool.read returns free agents valued via LatestModelScores and sorted numerically by minimum.
	var pool leagueservice.MflPool
	if err := json.Unmarshal(c.invoke(leagueservice.MflMovesPoolRead, "read",
		[]byte(`{"league":"native.hosted","sort":"minimum"}`), 200).Data, &pool); err != nil {
		t.Fatal(err)
	}
	if len(pool.Rows) != 2 {
		t.Fatalf("pool.read = %+v", pool)
	}
	// 13593 is a 2026 rookie ($500,000), 0531 is a 2015 vet ($1,000,000+): rookie must sort first.
	if pool.Rows[0].ID != "13593" || pool.Rows[1].ID != "0531" {
		t.Fatalf("sort minimum order = [%s (%s), %s (%s)]",
			pool.Rows[0].ID, pool.Rows[0].Minimum, pool.Rows[1].ID, pool.Rows[1].Minimum)
	}
	if pool.Rows[1].NowPPG == nil || *pool.Rows[1].NowPPG != 12.5 {
		t.Fatalf("free agent 0531 not valued through hosted wrappers: %+v", pool.Rows[1])
	}

	// 9. bid.check passes a valid bid on 0531.
	var bidCheck leagueservice.MflBidCheck
	if err := json.Unmarshal(c.invoke(leagueservice.MflMovesBidCheck, "read",
		[]byte(`{"league":"native.hosted","add":"0531","amount":1500000,"years":1}`), 200).Data, &bidCheck); err != nil {
		t.Fatal(err)
	}
	if !bidCheck.OK {
		t.Fatalf("bid.check = %+v", bidCheck)
	}
	// 9b. Two drops: the owner reads the rule in plain words (409), not "invalid native request" (400).
	twoDrops := invokeRaw(t, c, leagueservice.MflMovesBidCheck, "read",
		[]byte(`{"league":"native.hosted","add":"0531","amount":1500000,"years":1,"drop":["0001","0002"]}`), 409)
	if !strings.Contains(strings.ToLower(string(twoDrops)), "a bid can hold one drop") {
		t.Fatalf("two-drop bid.check body = %q", twoDrops)
	}

	// 10. plan.save saves the bid plan; duplicate bid on same player gets plain-word 409.
	var saved leagueservice.MflPlan
	if err := json.Unmarshal(c.invoke(leagueservice.MflMovesPlanSave, "execute",
		[]byte(`{"league":"native.hosted","kind":"bid","body":{"add":"0531","amount":1500000,"years":1}}`), 200).Data, &saved); err != nil {
		t.Fatal(err)
	}
	if saved.ID == "" || saved.Kind != "bid" || !saved.CanWithdraw {
		t.Fatalf("plan.save = %+v", saved)
	}
	dupMsg := invokeRaw(t, c, leagueservice.MflMovesPlanSave, "execute",
		[]byte(`{"league":"native.hosted","kind":"bid","body":{"add":"0531","amount":1600000,"years":1}}`), 409)
	if !strings.Contains(string(dupMsg), "already have an open bid plan") {
		t.Fatalf("duplicate plan refusal = %q", dupMsg)
	}

	// 11. plans.read lists the saved plan.
	var plans leagueservice.MflPlans
	if err := json.Unmarshal(c.invoke(leagueservice.MflMovesPlansRead, "read", leagueScope, 200).Data, &plans); err != nil {
		t.Fatal(err)
	}
	if len(plans.Plans) != 1 || plans.Plans[0].ID != saved.ID {
		t.Fatalf("plans.read = %+v", plans)
	}

	// 12. plan.link returns validated MFL URL.
	planScope := []byte(fmt.Sprintf(`{"league":"native.hosted","plan":%q}`, saved.ID))
	var link leagueservice.MflPlanLink
	if err := json.Unmarshal(c.invoke(leagueservice.MflMovesPlanLink, "read", planScope, 200).Data, &link); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(link.URL, "https://www.myfantasyleague.com/") {
		t.Fatalf("plan.link = %+v", link)
	}

	// 13. Another seat (owner on franchise.02) cannot see, link, or withdraw seat 1's plan.
	other := c
	other.bearer, other.proof = c.f.login(t, "owner", "private-password"), ""
	var otherPlans leagueservice.MflPlans
	if err := json.Unmarshal(other.invoke(leagueservice.MflMovesPlansRead, "read", leagueScope, 200).Data, &otherPlans); err != nil {
		t.Fatal(err)
	}
	if len(otherPlans.Plans) != 0 {
		t.Fatalf("another seat saw seat 1's plans: %+v", otherPlans)
	}
	otherLink := invokeRaw(t, other, leagueservice.MflMovesPlanLink, "read", planScope, 409)
	if !strings.Contains(string(otherLink), "not found") {
		t.Fatalf("another seat plan.link = %q", otherLink)
	}
	invokeRaw(t, other, leagueservice.MflMovesPlanWithdraw, "execute", planScope, 409)

	// 14. Owner withdraws their own plan; withdrawing again returns plain-word 409 refusal.
	var withdrawn leagueservice.MflPlan
	if err := json.Unmarshal(c.invoke(leagueservice.MflMovesPlanWithdraw, "execute", planScope, 200).Data, &withdrawn); err != nil {
		t.Fatal(err)
	}
	if withdrawn.State != "withdrawn" || withdrawn.CanWithdraw {
		t.Fatalf("plan.withdraw = %+v", withdrawn)
	}
	againMsg := invokeRaw(t, c, leagueservice.MflMovesPlanWithdraw, "execute", planScope, 409)
	if !strings.Contains(string(againMsg), "still planned") {
		t.Fatalf("withdraw already-withdrawn refusal = %q", againMsg)
	}

	// 15. Missing native football pin does NOT block mflmoves executes.
	c.f.footballGone.Store(true)
	c.invoke(leagueservice.MflMovesTeamSet, "execute", []byte(`{"league":"native.hosted","team":"0002"}`), 200)
	c.f.footballGone.Store(false)

	// 16. Helper token delegation works for owner's own seat and refuses another franchise.
	mflCmds := []string{
		leagueservice.MflMovesRefresh, leagueservice.MflMovesTeamsRead, leagueservice.MflMovesTeamSet,
		leagueservice.MflMovesRosterRead, leagueservice.MflMovesPoolRead, leagueservice.MflMovesBidCheck,
		leagueservice.MflMovesLineupCheck, leagueservice.MflMovesPlanSave, leagueservice.MflMovesPlansRead,
		leagueservice.MflMovesPlanLink, leagueservice.MflMovesPlanDone, leagueservice.MflMovesPlanWithdraw,
	}
	token := c.f.agentToken(t, c.bearer, mflCmds, "read", "execute")
	agentClient := hostedWorkflowClient{t: t, f: c.f, bearer: token.Bearer, football: c.football}
	agentClient.invoke(leagueservice.MflMovesTeamsRead, "read", leagueScope, 200)
	agentClient.invoke(leagueservice.MflMovesTeamSet, "execute", []byte(`{"league":"native.hosted","team":"0001"}`), 200)
	if _, err := c.f.tokens.Issue(t.Context(), accounts.RawSession{Bearer: c.bearer}, accounts.RawTokenScope{
		League: "native.hosted", Franchise: "franchise.02", Commands: mflCmds, Modes: []string{"read", "execute"},
	}); err == nil {
		t.Fatal("delegated mflmoves token for another franchise")
	}
}

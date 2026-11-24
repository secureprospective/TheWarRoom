package main

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/accounts"
	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/governance"
	"github.com/secureprospective/TheWarRoom/internal/serverhttp"
	accountstore "github.com/secureprospective/TheWarRoom/internal/store/accounts"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
	"github.com/secureprospective/TheWarRoom/tools/hostedfixture"
)

// TestSeasonInABoxPlayoff runs a complete 4-team playoff bracket against the real twr-server binary:
// week-1 regular season final -> standings seeds 1-4 -> bracket.build and bracket.seed -> week-2
// semi-finals scored final -> bracket.advance -> week-3 championship final -> bracket.advance -> champion.
func TestSeasonInABoxPlayoff(t *testing.T) {
	if testing.Short() {
		t.Skip("builds and runs the twr-server binary")
	}
	owners := map[string]string{}
	b := startBinaryBoxWith(t, func(t *testing.T, store *history.Store, credentials *accountstore.Store, principal governance.Principal, _ time.Time) {
		seedPlayoffLeague(t, store, credentials, principal, owners)
	})
	owners["franchise.01"], owners["franchise.04"] = b.commishBearer, b.owner04Bearer

	b.ledgerRevision(t)
	b.quickDraft(t)
	setPlayoffLineups(t, b, owners)

	// Week 1 regular season goes final through the server's official stats job.
	b.setClock(t, b.at.Add(31*time.Hour))
	useOfficialPair(t, b, "stats_player_week_2026_playoff_w1.csv.gz", "play_by_play_2026_playoff_w1.csv.gz")
	b.awaitPoints(t, func(points int64) bool { return points > 0 })
	var standings commissioner.NativeStandings
	b.read(t, "standings.read", map[string]int{"season": 2026}, &standings)
	if standings.Status != "Ranked" || standings.Decided != 2 || standings.Pending != 0 || standings.Through != 1 {
		t.Fatalf("week-1 standings not final: status=%q decided=%d pending=%d through=%d words=%q",
			standings.Status, standings.Decided, standings.Pending, standings.Through, standings.Words)
	}
	t.Logf("PLAYOFF STEP 1 week-1 regular season final: %s", standings.Words)

	// Bracket ready from week-1 standings: build and seed through the real command path.
	ready := awaitBracket(t, b, "ready to build", func(br commissioner.NativeBracket) bool { return br.Status == "Ready to build" })
	wantSeeds := []string{"franchise.01", "franchise.02", "franchise.03", "franchise.04"}
	assertSeeds(t, ready, wantSeeds)

	commishBracket(t, b, commissioner.BracketBuild, "box.playoff.build", "Build 4-team playoff bracket after week-1 final",
		commissioner.RawBracketChange{Season: 2026, Started: true, Preset: "league_rules"})
	commishBracket(t, b, commissioner.BracketSeed, "box.playoff.seed", "Confirm seed 1 from week-1 standings",
		commissioner.RawBracketChange{Season: 2026, Started: true, Slot: 1, Team: wantSeeds[0]})

	built := awaitBracket(t, b, "built bracket", func(br commissioner.NativeBracket) bool { return br.Status == "Built" && len(br.Rounds) == 2 })
	assertSeeds(t, built, wantSeeds)
	t.Logf("PLAYOFF STEP 2 bracket built and seeded: seeds=[1:%s 2:%s 3:%s 4:%s]", wantSeeds[0], wantSeeds[1], wantSeeds[2], wantSeeds[3])

	// Week 2 semi-finals go final; verify computed winners before confirming advancement.
	b.setClock(t, b.at.Add(time.Hour))
	useOfficialPair(t, b, "stats_player_week_2026_playoff_w2.csv.gz", "play_by_play_2026_playoff_w2.csv.gz")
	semis := awaitBracket(t, b, "week-2 semi-finals final", func(br commissioner.NativeBracket) bool {
		return gameStatus(br, "League/semi_1") == commissioner.GameFinal && gameStatus(br, "League/semi_2") == commissioner.GameFinal
	})
	semi1, semi2 := requireBracketGame(t, semis, "League/semi_1"), requireBracketGame(t, semis, "League/semi_2")
	if semi1.Winner != "franchise.01" || semi2.Winner != "franchise.03" {
		t.Fatalf("week-2 semi-final winners unexpected: semi_1=%s (%s), semi_2=%s (%s)",
			semi1.Winner, semi1.Words, semi2.Winner, semi2.Words)
	}
	t.Logf("PLAYOFF STEP 3 week-2 semi-finals final: semi_1 winner=%s (%s); semi_2 winner=%s (%s)",
		semi1.Winner, semi1.Words, semi2.Winner, semi2.Words)

	commishBracket(t, b, commissioner.BracketAdvance, "box.playoff.adv.s1", "Confirm semi-final 1 winner after week-2 final",
		commissioner.RawBracketChange{Season: 2026, Game: "League/semi_1", Team: semi1.Winner})
	commishBracket(t, b, commissioner.BracketAdvance, "box.playoff.adv.s2", "Confirm semi-final 2 winner after week-2 final",
		commissioner.RawBracketChange{Season: 2026, Game: "League/semi_2", Team: semi2.Winner})
	advanced := awaitBracket(t, b, "championship paired", func(br commissioner.NativeBracket) bool {
		g, ok := findBracketGame(br, "League/championship")
		return ok && g.Home != nil && g.Away != nil && g.Home.Franchise == "franchise.01" && g.Away.Franchise == "franchise.03"
	})
	champPair := requireBracketGame(t, advanced, "League/championship")
	t.Logf("PLAYOFF STEP 4 bracket advanced to final: %s v %s", champPair.Home.Franchise, champPair.Away.Franchise)

	// Week 3 championship final goes final; verify final winner and champion, then record final advance.
	b.setClock(t, b.at.Add(time.Hour))
	useOfficialPair(t, b, "stats_player_week_2026_playoff_w3.csv.gz", "play_by_play_2026_playoff_w3.csv.gz")
	decided := awaitBracket(t, b, "week-3 final decided", func(br commissioner.NativeBracket) bool {
		return gameStatus(br, "League/championship") == commissioner.GameFinal && br.Champion != nil
	})
	finalGame := requireBracketGame(t, decided, "League/championship")
	if finalGame.Winner != "franchise.03" || decided.Champion.Franchise != "franchise.03" {
		t.Fatalf("week-3 championship unexpected: winner=%s champion=%+v (%s)",
			finalGame.Winner, decided.Champion, finalGame.Words)
	}
	commishBracket(t, b, commissioner.BracketAdvance, "box.playoff.adv.final", "Confirm championship final winner after week-3 final",
		commissioner.RawBracketChange{Season: 2026, Game: "League/championship", Team: finalGame.Winner})
	var finished commissioner.NativeBracket
	b.read(t, "bracket.read", map[string]int{"season": 2026}, &finished)
	if finished.Champion == nil || finished.Champion.Franchise != "franchise.03" {
		t.Fatalf("champion missing after final advance: %+v", finished.Champion)
	}
	t.Logf("PLAYOFF STEP 5 championship final winner=%s (%s); champion=%s (%s, %s)",
		finalGame.Winner, finalGame.Words, finished.Champion.Franchise, finished.Champion.Name, finished.Champion.Source)
}

func seedPlayoffLeague(
	t *testing.T, store *history.Store, credentials *accountstore.Store,
	principal governance.Principal, owners map[string]string,
) {
	t.Helper()
	ctx := t.Context()
	assets, err := store.NativeAssets(ctx, "native.hosted")
	if err != nil {
		t.Fatal(err)
	}
	rev := assets.Revision
	record := func(command, req, reason string, args commissioner.RawArgs) {
		raw := commissioner.RawCommand{
			League: "native.hosted", Command: command, Args: args, Expected: rev,
			Request: req, Reason: reason, Effective: "now", Visibility: "league",
		}
		receipt, err := store.ExecuteCommissioner(ctx, raw, string(principal), nil)
		if err != nil {
			t.Fatalf("playoff setup %s (%s): %v", command, req, err)
		}
		rev = receipt.Revision
	}
	graph := `[{"id":"semi_1","week":2,"homeSeed":1,"awaySeed":4},{"id":"semi_2","week":2,"homeSeed":2,"awaySeed":3},{"id":"championship","week":3,"homeWinner":"winner of semi_1","awayWinner":"winner of semi_2"}]`
	for i, rule := range []struct{ field, value string }{
		{"league.conferences", `1`},
		{"schedule.regularWeeks", `1`},
		{"playoffs.perConference", `4`},
		{"playoffs.divisionWinners", `0`},
		{"playoffs.wildCards", `4`},
		{"playoffs.byeSeed", `0`},
		{"playoffs.reseed", `false`},
		{"playoffs.startWeek", `2`},
		{"playoffs.finalWeek", `3`},
		{"playoffs.graph", graph},
		{"lineup.positions", `{"QB":{"min":0,"max":1},"WR":{"min":0,"max":2}}`},
	} {
		record(commissioner.RulesSet, fmt.Sprintf("box.playoff.rule.%d", i), "Configure 4-team playoffs",
			commissioner.RawArgs{Field: rule.field, Value: json.RawMessage(rule.value)})
	}
	fixtures := []commissioner.ScheduleWeek{{Week: 1, Games: []commissioner.ScheduleGame{
		{ID: "w1g1", Home: "franchise.01", Away: "franchise.04"},
		{ID: "w1g2", Home: "franchise.02", Away: "franchise.03"},
	}}}
	record(commissioner.ScheduleGenerate, "box.playoff.sched.gen", "One-week regular season for four teams",
		commissioner.RawArgs{Schedule: &commissioner.RawScheduleArgs{Weeks: 1, Fixtures: fixtures}})
	record(commissioner.SchedulePublish, "box.playoff.sched.pub", "Publish one-week regular season",
		commissioner.RawArgs{Schedule: &commissioner.RawScheduleArgs{}})
	record(commissioner.CorrectRoster, "box.playoff.roster.03", "Roster free agent 6001 on franchise.03",
		commissioner.RawArgs{Correction: &commissioner.RawCorrection{
			Franchise: "franchise.03", Action: "add", Player: "6001",
			Contract: &commissioner.CorrectionContract{Salary: 100000000, ExpirationYear: 2028, ContractStatus: domain.CStatusUFA},
		}})
	sessions := accounts.NewSessions(accounts.New(credentials), credentials)
	for _, seat := range []struct{ franchise, user string }{
		{"franchise.02", "playoff-owner-02"},
		{"franchise.03", "playoff-owner-03"},
	} {
		raw := accounts.RawCredentials{Username: seat.user, Password: rand.Text()}
		if err := hostedfixture.AddWorkflowOwner(ctx, store, credentials, principal, seat.franchise, raw); err != nil {
			t.Fatalf("seat %s: %v", seat.franchise, err)
		}
		session, err := sessions.Login(ctx, raw, "playoff box")
		if err != nil {
			t.Fatalf("login %s: %v", seat.franchise, err)
		}
		owners[seat.franchise] = session.Bearer
	}
	if _, err := store.LinkPlayerIDs(ctx, "dynastyprocess", []history.PlayerIDLink{
		{IDType: "gsis", IDValue: "00-0000102", PlayerID: "0102"},
		{IDType: "gsis", IDValue: "00-0006001", PlayerID: "6001"},
		{IDType: "gsis", IDValue: "00-0005002", PlayerID: "5002"},
	}); err != nil {
		t.Fatalf("link playoff player gsis ids: %v", err)
	}
}

func setPlayoffLineups(t *testing.T, b *binaryBox, owners map[string]string) {
	t.Helper()
	starters := []struct{ franchise, player string }{
		{"franchise.01", "0099"},
		{"franchise.02", "0102"},
		{"franchise.03", "6001"},
		{"franchise.04", "5002"},
	}
	for _, week := range []int{1, 2, 3} {
		for _, s := range starters {
			if week == 1 && s.franchise == "franchise.01" {
				continue
			}
			bearer := owners[s.franchise]
			cmd := commissioner.RawCommand{
				League: "native.hosted", Command: commissioner.LineupSet,
				Request:   fmt.Sprintf("box.playoff.lineup.%s.w%d", s.franchise, week),
				Effective: "now", Visibility: "private", Expected: b.ownerRevision(t, bearer),
				Args: commissioner.RawArgs{Lineup: &commissioner.RawLineupChange{
					Season: 2026, Franchise: s.franchise, Week: week, Starters: []string{s.player},
				}},
			}
			if _, err := b.ownerPost(t, serverhttp.LineupPath, bearer, serverhttp.RawNativeRequest{Protocol: 1, Command: &cmd}); err != nil {
				t.Fatalf("lineup %s week %d refused: %v", s.franchise, week, err)
			}
		}
	}
}

func commishBracket(t *testing.T, b *binaryBox, command, req, reason string, change commissioner.RawBracketChange) {
	t.Helper()
	cmd := commissioner.RawCommand{
		League: "native.hosted", Command: command, Request: req, Effective: "now", Visibility: "league",
		Expected: b.ownerRevision(t, b.commishBearer), Reason: reason,
		Args: commissioner.RawArgs{Bracket: &change},
	}
	body, err := json.Marshal(cmd)
	if err != nil {
		t.Fatal(err)
	}
	if change.Team == "" || change.Team == "franchise.01" {
		if printed, err := b.twr(t, string(body), "preview", "-"); err != nil || !strings.Contains(printed, `"blockers": []`) {
			t.Fatalf("%s twr preview refused: %v %s", command, err, printed)
		}
	}
	readReq := serverhttp.RawNativeRequest{
		Protocol: 1,
		Workflow: &serverhttp.RawWorkflow{Command: "native.assets.read", Version: "1", Mode: "read", Arguments: json.RawMessage(`{"league":"native.hosted"}`)},
	}
	reply, err := b.ownerPost(t, serverhttp.WorkflowPath, b.commishBearer, readReq)
	if err != nil {
		t.Fatalf("%s football digest read refused: %v", command, err)
	}
	var env serverhttp.WorkflowResponse
	if err := json.Unmarshal(reply, &env); err != nil {
		t.Fatalf("%s decode workflow response: %v", command, err)
	}
	preview := serverhttp.RawNativeRequest{
		Protocol: 1,
		Workflow: &serverhttp.RawWorkflow{Command: command, Version: "1", Mode: "preview", Arguments: body, Football: env.Football},
	}
	if prevReply, err := b.ownerPost(t, serverhttp.WorkflowPath, b.commishBearer, preview); err != nil || !strings.Contains(string(prevReply), `"blockers":[]`) {
		t.Fatalf("%s workflow preview refused: %v %s", command, err, prevReply)
	}
	wf := serverhttp.RawNativeRequest{
		Protocol: 1,
		Workflow: &serverhttp.RawWorkflow{Command: command, Version: "1", Mode: "execute", Arguments: body, Football: env.Football},
	}
	if _, err := b.ownerPost(t, serverhttp.WorkflowPath, b.commishBearer, wf); err != nil {
		log, _ := os.ReadFile(filepath.Join(b.dir, "server.log"))
		t.Fatalf("%s execute refused: %v\nserver log:\n%s", command, err, log)
	}
}

func useOfficialPair(t *testing.T, b *binaryBox, weekly, plays string) {
	t.Helper()
	for from, to := range map[string]string{
		weekly: "stats_player_week_2026.csv.gz",
		plays:  "play_by_play_2026.csv.gz",
	} {
		body, err := os.ReadFile(filepath.Join("testdata/official", from))
		if err != nil {
			t.Fatal(err)
		}
		temp := filepath.Join(b.officialDir, to+".tmp")
		if err := os.WriteFile(temp, body, 0o600); err != nil { //nolint:gosec // G703: fixed names under the test temp dir.
			t.Fatal(err)
		}
		if err := os.Rename(temp, filepath.Join(b.officialDir, to)); err != nil {
			t.Fatal(err)
		}
	}
}

func awaitBracket(t *testing.T, b *binaryBox, desc string, ready func(commissioner.NativeBracket) bool) commissioner.NativeBracket {
	t.Helper()
	deadline := time.Now().Add(30 * time.Second)
	var last commissioner.NativeBracket
	for {
		b.read(t, "bracket.read", map[string]int{"season": 2026}, &last)
		if ready(last) {
			return last
		}
		if time.Now().After(deadline) {
			log, _ := os.ReadFile(filepath.Join(b.dir, "server.log"))
			t.Fatalf("bracket never reached %s (status=%q words=%q); server log:\n%s", desc, last.Status, last.Words, log)
		}
		time.Sleep(200 * time.Millisecond)
	}
}

func assertSeeds(t *testing.T, br commissioner.NativeBracket, want []string) {
	t.Helper()
	if len(br.Fields) != 1 || len(br.Fields[0].Seeds) != len(want) {
		t.Fatalf("expected 1 conference with %d seeds, got %+v", len(want), br.Fields)
	}
	for i, id := range want {
		if got := br.Fields[0].Seeds[i].Franchise; got != id {
			t.Fatalf("seed %d = %s, want %s", i+1, got, id)
		}
	}
}

func findBracketGame(br commissioner.NativeBracket, id string) (commissioner.BracketGame, bool) {
	for _, r := range br.Rounds {
		for _, g := range r.Games {
			if g.ID == id {
				return g, true
			}
		}
	}
	return commissioner.BracketGame{}, false
}

func gameStatus(br commissioner.NativeBracket, id string) string {
	g, ok := findBracketGame(br, id)
	if !ok {
		return ""
	}
	return g.Status
}

func requireBracketGame(t *testing.T, br commissioner.NativeBracket, id string) commissioner.BracketGame {
	t.Helper()
	g, ok := findBracketGame(br, id)
	if !ok {
		t.Fatalf("bracket game %s missing from %+v", id, br.Rounds)
	}
	return g
}

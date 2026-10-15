package history

import (
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
)

func (f *bracketFixture) link(t *testing.T, record, franchise string) commissioner.RawCommand {
	t.Helper()
	return commissioner.RawCommand{League: bracketLeague, Command: commissioner.HistoryLink,
		Args:     commissioner.RawArgs{Link: &commissioner.RawHistoryLink{Record: record, Franchise: franchise}},
		Expected: f.rev(t), Request: "link." + record + "." + franchise + "." + fmt.Sprint(f.rev(t)), Effective: "now", Visibility: "league", Reason: "Fix the record"}
}

func (f *bracketFixture) history(t *testing.T, franchise string) commissioner.FranchiseHistory {
	t.Helper()
	out, err := commissioner.FranchiseHistoryReading(bracketLeague, scoreRecords(t, f.s), franchise)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func linkOf(b commissioner.NativeBracket, record string) commissioner.HistoryRecord {
	for _, h := range b.History {
		if h.Record == record {
			return h
		}
	}
	return commissioner.HistoryRecord{}
}

func TestHistoryLinkCM049GovernsTheChampionAndFranchiseHistoryAndReopens(t *testing.T) {
	f := bracketTest(t)
	f.do(t, commissioner.BracketBuild, rulesBuild())
	f.grantCoCommissioner(t, "cc")
	start := linkOf(f.bracket(t), "champion.2025")
	if start.Franchise != franchiseID(5) || start.ByCommissioner || !slices.Equal(f.history(t, franchiseID(5)).Titles, []int{2025}) {
		t.Fatalf("the link carried from the archive is the starting state: %+v", start)
	}
	if f.refused(t, f.link(t, "champion.2025", franchiseID(9)), "owner") == "" {
		t.Fatal("an owner relinked the history")
	}
	move := runScoreCommand(t, f.s, f.link(t, "champion.2025", franchiseID(9)), "cc")
	for range 2 {
		b := f.bracket(t)
		if l := linkOf(b, "champion.2025"); l.Franchise != franchiseID(9) || !l.ByCommissioner || l.Name != "Held team 9" {
			t.Fatalf("a link set by hand shows as set by the commissioner: %+v", l)
		}
		if !slices.Equal(f.history(t, franchiseID(9)).Titles, []int{2025}) || len(f.history(t, franchiseID(5)).Titles) != 0 {
			t.Fatalf("franchise history must read through the link: %+v %+v", f.history(t, franchiseID(9)), f.history(t, franchiseID(5)))
		}
		past, err := f.s.NativeBracket(t.Context(), bracketLeague, 2025)
		if err != nil || past.Status != "Archived" || past.Champion == nil || past.Champion.Franchise != franchiseID(9) {
			t.Fatalf("the champion of an archived season reads through the link: %+v %v", past.Champion, err)
		}
		f.reopen(t)
	}
	f.undo(t, move.Entry)
	for range 2 {
		l := linkOf(f.bracket(t), "champion.2025")
		if l.Franchise != franchiseID(5) || l.ByCommissioner || !slices.Equal(f.history(t, franchiseID(5)).Titles, []int{2025}) {
			t.Fatalf("compensation did not bring the carried link back: %+v", l)
		}
		f.reopen(t)
	}
}

func TestHistoryLinkCM049OverridesTheSeasonsOwnChampion(t *testing.T) {
	f := bracketTest(t)
	f.do(t, commissioner.BracketBuild, rulesBuild())
	if c := f.bracket(t).Champion; c == nil || c.Franchise != franchiseID(32) || c.Source != "Won the final" {
		t.Fatalf("the final decides the champion: %+v", c)
	}
	receipt := f.run(t, f.link(t, "champion.2026", franchiseID(3)))
	if c := f.bracket(t).Champion; c.Franchise != franchiseID(3) || c.Source != "Set by the commissioner's history link" {
		t.Fatalf("a link names the champion record's franchise: %+v", c)
	}
	f.reopen(t)
	if c := f.bracket(t).Champion; c.Franchise != franchiseID(3) {
		t.Fatalf("replay lost the link: %+v", c)
	}
	f.undo(t, receipt.Entry)
	if c := f.bracket(t).Champion; c.Franchise != franchiseID(32) || c.Source != "Won the final" {
		t.Fatalf("compensation did not return the final's champion: %+v", c)
	}
}

func TestHistoryLinkCM049Refusals(t *testing.T) {
	f := bracketTest(t)
	for name, raw := range map[string]commissioner.RawCommand{
		"a franchise the league lacks": f.link(t, "champion.2025", "franchise.99"),
		"a season not yet played":      f.link(t, "champion.2027", franchiseID(2)),
		"the franchise it has":         f.link(t, "champion.2025", franchiseID(5)),
	} {
		if msg := f.refused(t, raw, "creator"); msg == "" {
			t.Fatalf("%s accepted", name)
		}
	}
	if msg := f.refused(t, f.link(t, "champion.2027", franchiseID(2)), "creator"); !strings.Contains(msg, "has not been played") {
		t.Fatalf("future season: %s", msg)
	}
	if msg := f.refused(t, f.link(t, "champion.2025", franchiseID(5)), "creator"); !strings.Contains(msg, "Nothing would change") {
		t.Fatalf("no-op: %s", msg)
	}
}

func TestBracketChangesAfterKickoffNeedTheReasonThatG3Demands(t *testing.T) {
	f := bracketTest(t)
	f.do(t, commissioner.BracketBuild, rulesBuild())
	edit := func(started bool, reason string) commissioner.RawCommand {
		return f.bracketCommand(t, commissioner.BracketEdit, change{Node: "AFC/wildcard_2/away", Team: franchiseID(2), Started: started}, reason)
	}
	f.run(t, edit(false, ""))
	f.kickoff(t0().Add(48 * time.Hour))
	if msg := f.refused(t, edit(true, "Games began"), "creator"); !strings.Contains(msg, "has not kicked off") {
		t.Fatalf("a claim that the playoffs began: %s", msg)
	}
	f.run(t, f.bracketCommand(t, commissioner.BracketSeed, change{Slot: 2, Team: franchiseID(3)}, ""))
	if b := f.bracket(t); b.Started || !b.StartKnown {
		t.Fatalf("before kickoff: %v %v", b.Started, b.StartKnown)
	}
	f.kickoff(t0().Add(-time.Hour))
	for name, raw := range map[string]commissioner.RawCommand{
		"edit":  edit(false, "Fix it"),
		"seed":  f.bracketCommand(t, commissioner.BracketSeed, change{Slot: 3, Team: franchiseID(2)}, "Fix it"),
		"build": f.bracketCommand(t, commissioner.BracketBuild, change{Preset: "legacy_nfl"}, "Fix it"),
	} {
		if msg := f.refused(t, raw, "creator"); !strings.Contains(msg, "has kicked off") {
			t.Fatalf("%s after kickoff at G2: %s", name, msg)
		}
	}
	if msg := f.refused(t, edit(true, ""), "creator"); !strings.Contains(msg, "reason") {
		t.Fatalf("after kickoff a change needs a typed reason: %s", msg)
	}
	lowered := edit(true, "Games began")
	lowered.Gravity = "G2"
	f.refused(t, lowered, "creator")
	f.run(t, edit(true, "A host forfeited"))
	if b := f.bracket(t); !b.Started || !b.StartKnown || gameIn(b, "AFC/wildcard_2").Away.Franchise != franchiseID(2) {
		t.Fatalf("after kickoff: %v %v", b.Started, b.StartKnown)
	}
	f.run(t, f.bracketCommand(t, commissioner.BracketAdvance, change{Game: "AFC/wildcard_4", Team: franchiseID(8)}, "Scoring error"))
	f.reopen(t)
	if b := f.bracket(t); !b.Started || gameIn(b, "AFC/wildcard_4").Winner != franchiseID(8) {
		t.Fatal("replay lost a change made after kickoff")
	}
}

func TestBracketChangesWithoutAHeldCalendarAreTakenAsStated(t *testing.T) {
	f := bracketTest(t)
	f.do(t, commissioner.BracketBuild, rulesBuild())
	f.run(t, f.bracketCommand(t, commissioner.BracketEdit, change{Node: "AFC/wildcard_2/away", Team: franchiseID(2), Started: true}, "Games began"))
	f.run(t, f.bracketCommand(t, commissioner.BracketSeed, change{Slot: 2, Team: franchiseID(3)}, ""))
	if b := f.bracket(t); b.StartKnown || b.Started {
		t.Fatal("kickoffs that are not held are not known")
	}
}

func TestBracketCommandRefusals(t *testing.T) {
	f := bracketTest(t)
	f.grantCoCommissioner(t, "cc")
	cmd := f.bracketCommand
	for name, c := range map[string]struct {
		raw  commissioner.RawCommand
		want string
	}{
		"edit before build":    {cmd(t, commissioner.BracketEdit, change{Node: "AFC/wildcard_2/away", Team: franchiseID(2)}, ""), "build the playoff bracket"},
		"advance before build": {cmd(t, commissioner.BracketAdvance, change{Game: "AFC/wildcard_2", Team: franchiseID(2)}, "Fix"), "build the playoff bracket"},
		"unsaved shape":        {cmd(t, commissioner.BracketBuild, change{Preset: "nope"}, ""), "no longer saved"},
		"seed out of range":    {cmd(t, commissioner.BracketSeed, change{Slot: 8, Team: franchiseID(2)}, ""), "seed 8 does not exist"},
		"unknown team":         {cmd(t, commissioner.BracketSeed, change{Slot: 2, Team: "franchise.99"}, ""), "not in this app league"},
		"a shape with no sources": {cmd(t, commissioner.BracketPresetSave, change{Name: "bad",
			Shape: []commissioner.BracketNodeSpec{{ID: "final", Week: 14, HomeSeed: 1, AwaySeed: 2}, {ID: "final", Week: 15, HomeSeed: 1, AwaySeed: 2}}}, ""),
			"cannot name a game twice"},
	} {
		if msg := f.refused(t, c.raw, "creator"); !strings.Contains(msg, c.want) {
			t.Fatalf("%s: %s", name, msg)
		}
	}
	wrongSeason := cmd(t, commissioner.BracketBuild, change{Preset: "league_rules"}, "")
	wrongSeason.Args.Bracket.Season = 2025
	if msg := f.refused(t, wrongSeason, "creator"); !strings.Contains(msg, "the league is in season 2026") {
		t.Fatalf("season: %s", msg)
	}
	build := cmd(t, commissioner.BracketBuild, change{Preset: "league_rules"}, "")
	if f.refused(t, build, "owner") == "" {
		t.Fatal("an owner built the bracket")
	}
	receipt := runScoreCommand(t, f.s, build, "cc")
	if again, err := f.s.ExecuteCommissioner(t.Context(), build, "cc", nil); err != nil || again != receipt {
		t.Fatalf("a retry must return the stored receipt: %v", err)
	}
	conflict := build
	conflict.Args = commissioner.RawArgs{Bracket: &change{Season: 2026, Preset: "legacy_nfl"}}
	f.refused(t, conflict, "cc")
	if msg := f.refused(t, cmd(t, commissioner.BracketBuild, change{Preset: "league_rules"}, ""), "creator"); !strings.Contains(msg, "Nothing would change") {
		t.Fatalf("a build that changes nothing: %s", msg)
	}
	for name, c := range map[string]struct {
		raw  commissioner.RawCommand
		want string
	}{
		"not a place":      {cmd(t, commissioner.BracketEdit, change{Node: "AFC/wildcard_9/home", Team: franchiseID(2)}, ""), "is not a place"},
		"other conference": {cmd(t, commissioner.BracketEdit, change{Node: "AFC/wildcard_2/home", Team: franchiseID(20)}, ""), "plays in the NFC conference"},
		"not a game":       {cmd(t, commissioner.BracketAdvance, change{Game: "AFC/nothing", Team: franchiseID(2)}, "Fix"), "is not a game"},
	} {
		if msg := f.refused(t, c.raw, "creator"); !strings.Contains(msg, c.want) {
			t.Fatalf("%s: %s", name, msg)
		}
	}
	stale := cmd(t, commissioner.BracketSeed, change{Slot: 2, Team: franchiseID(2)}, "")
	stale.Expected--
	f.refused(t, stale, "creator")
	foreign := cmd(t, commissioner.BracketSeed, change{Slot: 2, Team: franchiseID(2)}, "")
	foreign.Args.Name = "x"
	f.refused(t, foreign, "creator")
	private := cmd(t, commissioner.BracketSeed, change{Slot: 2, Team: franchiseID(2)}, "")
	private.Visibility = "private"
	f.refused(t, private, "creator")
}

func TestTheLedgerReadsPlayoffChangesInPlainWords(t *testing.T) {
	f := bracketTest(t)
	f.do(t, commissioner.BracketBuild, rulesBuild())
	f.do(t, commissioner.BracketEdit, change{Node: "AFC/wildcard_2/away", Team: franchiseID(2)})
	ledger, err := f.s.CommissionerLedger(t.Context(), bracketLeague, "creator", nil)
	if err != nil {
		t.Fatal(err)
	}
	words := []string{}
	for _, e := range ledger.Entries {
		words = append(words, e.What)
	}
	joined := strings.Join(words, "\n")
	for _, want := range []string{"Built the season 2026 playoff bracket from the league's playoff rules",
		"as the away team of the AFC wildcard 2 game"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("missing %q in\n%s", want, joined)
		}
	}
}

func TestTheBracketWaitsForTheRegularSeasonAndSaysWhy(t *testing.T) {
	f := bracketTest(t)
	f.do(t, commissioner.BracketBuild, rulesBuild())
	more := f.setRule(t, "schedule.regularWeeks", "4")
	b := f.bracket(t)
	if b.Status != "Waiting for the regular season" || len(b.Rounds) != 0 || len(b.Fields) != 0 ||
		!strings.Contains(b.Words, "covers only 3 of 4 weeks") {
		t.Fatalf("a shorter schedule than the season: %s %q", b.Status, b.Words)
	}
	f.undo(t, more.Entry)
	f.s.SetNativeLineups(&resultLineups{without: map[string]bool{franchiseID(5): true}})
	b = f.bracket(t)
	if b.Status != "Waiting for the regular season" || !strings.Contains(b.Words, "still have no result") || len(b.Rounds) != 0 {
		t.Fatalf("a game without a result: %s %q", b.Status, b.Words)
	}
	f.install()
	if b = f.bracket(t); b.Status != "Built" {
		t.Fatalf("results back: %s %q", b.Status, b.Words)
	}
	next, err := f.s.NativeBracket(t.Context(), bracketLeague, 2027)
	if err != nil || next.Status != "Waiting for the regular season" || !strings.Contains(next.Words, "has not been played") {
		t.Fatalf("a season ahead: %+v %v", next.Status, err)
	}
}

func TestTheBracketOfALeagueWithoutAStartingSnapshotSaysSo(t *testing.T) {
	s, _, _ := newStore(t)
	runScoreCommand(t, s, createWith(commissioner.PresetLegacy, 0), "creator")
	b, err := s.NativeBracket(t.Context(), bracketLeague, 2026)
	if err != nil || b.Status != "Unavailable" || !strings.Contains(b.Words, "starting snapshot") {
		t.Fatalf("%+v %v", b, err)
	}
	if _, err := s.NativeBracket(t.Context(), bracketLeague, 1800); err == nil {
		t.Fatal("a season before 1920 was read")
	}
}

func TestNativeBracketWire(t *testing.T) {
	f := bracketTest(t)
	ready := f.bracket(t)
	f.do(t, commissioner.BracketBuild, rulesBuild())
	f.do(t, commissioner.BracketSeed, change{Slot: 1, Team: franchiseID(1)})
	f.do(t, commissioner.BracketEdit, change{Node: "NFC/wildcard_2/away", Team: franchiseID(17)})
	f.run(t, f.bracketCommand(t, commissioner.BracketAdvance, change{Game: "AFC/wildcard_3", Team: franchiseID(8)}, "Scoring error"))
	f.run(t, commissioner.RawCommand{League: bracketLeague, Command: commissioner.TitleSet, Expected: f.rev(t), Request: "wire-title",
		Args:      commissioner.RawArgs{Title: &commissioner.RawTitleChange{Season: 2026, Label: "{year} Legacy Champions"}},
		Effective: "now", Visibility: "league"})
	f.run(t, f.link(t, "champion.2025", franchiseID(9)))
	built := f.bracket(t)
	archived, err := f.s.NativeBracket(t.Context(), bracketLeague, 2025)
	if err != nil {
		t.Fatal(err)
	}
	f.setRule(t, "schedule.regularWeeks", "4")
	wire := struct {
		Ready    commissioner.NativeBracket    `json:"ready"`
		Built    commissioner.NativeBracket    `json:"built"`
		Archived commissioner.NativeBracket    `json:"archived"`
		Waiting  commissioner.NativeBracket    `json:"waiting"`
		History  commissioner.FranchiseHistory `json:"history"`
	}{ready, built, archived, f.bracket(t), f.history(t, franchiseID(9))}
	body, err := json.MarshalIndent(wire, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	body = append(body, '\n')
	path := "../../commissioner/testdata/native-bracket-wire.json"
	if os.Getenv("TWR_WRITE_WIRE") == "1" {
		if err := os.WriteFile(path, body, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	held, err := os.ReadFile(path)
	if err != nil || string(held) != string(body) {
		t.Fatal("native bracket wire stale", err)
	}
}

func TestTheBracketFollowsTheStandingsTheCommissionerSet(t *testing.T) {
	f := bracketTest(t)
	computed, err := f.s.NativeStandings(t.Context(), bracketLeague, 2026)
	if err != nil {
		t.Fatal(err)
	}
	order := make([]string, len(computed.Computed))
	for i, row := range computed.Computed {
		order[len(order)-1-i] = row.Franchise
	}
	if seedsOf(f.bracket(t), "AFC")[0].Franchise == franchiseID(1) {
		t.Fatal("the computed standings already put franchise 1 first")
	}
	rev := f.rev(t)
	override := f.run(t, standingsCommand(commissioner.StandingsOverride, commissioner.RawStandingsChange{Season: 2026, Order: order}, rev))
	for range 2 {
		top := seedsOf(f.bracket(t), "AFC")[0]
		if top.Franchise != franchiseID(1) || top.Rank != 1 {
			t.Fatalf("the playoff field must follow the order the commissioner set: %+v", top)
		}
		f.reopen(t)
	}
	f.undo(t, override.Entry)
	if top := seedsOf(f.bracket(t), "AFC")[0]; top.Franchise == franchiseID(1) {
		t.Fatalf("compensation did not return the computed order: %+v", top)
	}
}

package history

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/fantasyscore"
	"github.com/secureprospective/TheWarRoom/internal/fantasyscore/source"
	"github.com/secureprospective/TheWarRoom/internal/governance"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

type resultLineups struct {
	missing bool
	without map[string]bool
}

func (l *resultLineups) WeekLineup(_ context.Context, _ []commissioner.Record, _, franchise string, _, _ int) (commissioner.WeekLineup, error) {
	if l.missing || l.without[franchise] {
		return commissioner.WeekLineup{}, nil
	}
	return commissioner.WeekLineup{Set: true, Starters: []commissioner.Starter{{Player: "10" + strings.TrimPrefix(franchise, "franchise."), Name: franchise + " starter"}}}, nil
}

type resultsFixture struct {
	s          *Store
	pools      *db.Pools
	path       string
	lineups    *resultLineups
	a, b       string
	incomplete bool
}

func resultsTest(t *testing.T) *resultsFixture { return resultsTestSeeded(t, nil) }

// resultsTestSeeded is resultsTest with the starting snapshot changed by seed before it is recorded.
func resultsTestSeeded(t *testing.T, seed func(*commissioner.NativeAssets)) *resultsFixture {
	t.Helper()
	path := filepath.Join(t.TempDir(), "history.db")
	s, pools := statStore(t, path)
	runScoreCommand(t, s, createWith(commissioner.PresetLegacy, 0), "creator")
	held := heldSeedCommand(t)
	if seed != nil {
		assets := commissioner.NativeAssets{Snapshot: held.Args.Native}
		seed(&assets)
	}
	runScoreCommand(t, s, held, "creator")
	runScoreCommand(t, s, ruleCommand("schedule.regularWeeks", "3", "now", "Choose the season length", 2), "creator")
	// These fixtures start one player per franchise, which is not a legal lineup, so the invalid-lineup
	// penalty would zero every score. The lineup tests below keep the penalty and use legal lineups.
	runScoreCommand(t, s, ruleCommand("lineup.invalidPenalty", `"none"`, "now", "Score one-player lineups as they are", 3), "creator")
	raw := commissioner.RawCommand{League: "native.test", Command: commissioner.ScheduleGenerate,
		Args: commissioner.RawArgs{Schedule: &commissioner.RawScheduleArgs{Weeks: 3}}, Expected: 4, Request: "generate", Effective: "now", Visibility: "league"}
	runScoreCommand(t, s, raw, "creator")
	raw.Command, raw.Expected, raw.Request = commissioner.SchedulePublish, 5, "publish"
	raw.Args = commissioner.RawArgs{Schedule: &commissioner.RawScheduleArgs{}}
	runScoreCommand(t, s, raw, "creator")
	schedule, err := commissioner.ScheduleReading("native.test", scoreRecords(t, s))
	if err != nil {
		t.Fatal(err)
	}
	first := schedule.Weeks[0]
	for week := 2; week <= 3; week++ {
		games := make([]commissioner.ScheduleGame, len(first.Games))
		for i, game := range first.Games {
			games[i] = commissioner.ScheduleGame{ID: fmt.Sprintf("week.%d.game.%d", week, i+1), Home: game.Home, Away: game.Away}
		}
		raw.Command, raw.Expected, raw.Request, raw.Reason = commissioner.ScheduleEdit, int64(week+4), fmt.Sprint("week", week), "Keep these opponents"
		raw.Args = commissioner.RawArgs{Schedule: &commissioner.RawScheduleArgs{Week: week, Games: games, Published: true}}
		runScoreCommand(t, s, raw, "creator")
	}
	f := &resultsFixture{s: s, pools: pools, path: path, lineups: &resultLineups{}, a: first.Games[0].Home, b: first.Games[0].Away}
	f.install()
	return f
}

func (f *resultsFixture) install() {
	f.s.SetNativeLineups(f.lineups)
	f.s.evidence = func(_ context.Context, scope source.RawNativeScope) (source.NativeEvidence, error) {
		quantity := leaguerules.Decimal(10000)
		switch "franchise." + strings.TrimPrefix(scope.Player, "10") {
		case f.a:
			if scope.Week == 1 {
				quantity = 1000000
			}
		case f.b:
			if scope.Week > 1 {
				quantity = 20000
			}
		}
		missing := []string{}
		if f.incomplete {
			missing = append(missing, "A starter has not been reported")
		}
		return source.NativeEvidence{GSIS: "held-" + scope.Player, Missing: missing, Held: &source.HeldWeek{SHA256: "held-complete",
			FetchedAt: t0(), Week: &source.PlayerWeek{Position: "LB", Line: fantasyscore.StatLine{
				Quantities: map[leaguerules.StatID]leaguerules.Decimal{"tackle": quantity}}}}}, nil
	}
}

func (f *resultsFixture) reopen(t *testing.T) {
	t.Helper()
	if err := f.pools.Close(); err != nil {
		t.Fatal(err)
	}
	f.s, f.pools = statStore(t, f.path)
	f.install()
}

func (f *resultsFixture) standings(t *testing.T) commissioner.NativeStandings {
	t.Helper()
	out, err := f.s.NativeStandings(t.Context(), "native.test", 2026)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func TestNativeStandingsCM007GovernsAndReopens(t *testing.T) {
	t.Parallel()
	f := resultsTest(t)
	for pass := 0; pass < 2; pass++ {
		baseline := f.standings(t)
		if baseline.Computed[0].Franchise != f.b {
			t.Fatalf("winning percentage: %+v", baseline.Computed[0])
		}
		raw := ruleCommand("standings.criteria", `["points","win_percentage"]`, "now", "Rank points first", revision(t, f.s))
		receipt := runScoreCommand(t, f.s, raw, "creator")
		assertPointsFirst(t, f)
		f.reopen(t)
		assertPointsFirst(t, f)
		runScoreCommand(t, f.s, reverseCommand(receipt.Entry, revision(t, f.s), fmt.Sprint("undo-rank", pass)), "creator")
		if got := f.standings(t); !reflect.DeepEqual(got.Computed, baseline.Computed) {
			t.Fatal("compensation changed computed standings")
		}
		f.reopen(t)
		if got := f.standings(t); !reflect.DeepEqual(got.Computed, baseline.Computed) {
			t.Fatal("replay changed compensated standings")
		}
	}
}

func assertPointsFirst(t *testing.T, f *resultsFixture) {
	t.Helper()
	got := f.standings(t)
	if got.Computed[0].Franchise != f.a || !strings.Contains(strings.Join(got.Computed[0].Why, " "), "Points scored") {
		t.Fatalf("criteria order or explanation: %+v", got.Computed[:2])
	}
}

func standingsCommand(command string, change commissioner.RawStandingsChange, rev int64) commissioner.RawCommand {
	return commissioner.RawCommand{League: "native.test", Command: command, Args: commissioner.RawArgs{Standings: &change},
		Expected: rev, Request: fmt.Sprintf("standings.%d", rev), Reason: "Correct the season record", Effective: "now", Visibility: "league"}
}

func TestNativeStandingsCM033BesideComputedAndReopen(t *testing.T) {
	t.Parallel()
	f := resultsTest(t)
	for pass := 0; pass < 2; pass++ {
		baseline := f.standings(t)
		raw := standingsCommand(commissioner.StandingsRecordAdjust, commissioner.RawStandingsChange{
			Season: 2026, Franchise: f.a, Wins: 3}, revision(t, f.s))
		receipt := runScoreCommand(t, f.s, raw, "creator")
		if retry := runScoreCommand(t, f.s, raw, "creator"); retry != receipt {
			t.Fatal("retry appended")
		}
		for range 2 {
			got := f.standings(t)
			if !reflect.DeepEqual(got.Computed, baseline.Computed) || got.Override == nil ||
				got.Override.Rows[0].Franchise != f.a || got.Override.Rows[0].ComputedRecord != "1-2" ||
				got.Override.Records[f.a].Reason != raw.Reason || len(got.Override.Rows[0].Why) == 0 {
				t.Fatalf("record must stand beside computed with reason: %+v", got)
			}
			f.reopen(t)
		}
		runScoreCommand(t, f.s, reverseCommand(receipt.Entry, revision(t, f.s), fmt.Sprint("undo-record", pass)), "creator")
		if got := f.standings(t); got.Override != nil || !reflect.DeepEqual(got.Computed, baseline.Computed) {
			t.Fatal("record compensation not immediately visible")
		}
		f.reopen(t)
		got := f.standings(t)
		if got.Override != nil || !reflect.DeepEqual(got.Computed, baseline.Computed) {
			t.Fatal("record compensation not replayed")
		}
	}
}

func TestNativeStandingsOrderClearAndCompensation(t *testing.T) {
	t.Parallel()
	f := resultsTest(t)
	baseline := f.standings(t)
	order := make([]string, len(baseline.Computed))
	for i, row := range baseline.Computed {
		order[len(order)-1-i] = row.Franchise
	}
	raw := standingsCommand(commissioner.StandingsOverride, commissioner.RawStandingsChange{Season: 2026, Order: order}, revision(t, f.s))
	receipt := runScoreCommand(t, f.s, raw, "creator")
	for range 2 {
		got := f.standings(t)
		if !reflect.DeepEqual(got.Computed, baseline.Computed) || got.Override == nil ||
			got.Override.Rows[0].Franchise != order[0] || !strings.Contains(got.Override.Rows[0].Why[0], raw.Reason) {
			t.Fatal("manual order replaced computed or lost reason")
		}
		f.reopen(t)
	}
	clearCommand := standingsCommand(commissioner.StandingsOverrideClear, commissioner.RawStandingsChange{Season: 2026}, revision(t, f.s))
	cleared := runScoreCommand(t, f.s, clearCommand, "creator")
	if f.standings(t).Override != nil {
		t.Fatal("clear left override")
	}
	f.reopen(t)
	if f.standings(t).Override != nil {
		t.Fatal("replay resurrected clear")
	}
	runScoreCommand(t, f.s, reverseCommand(cleared.Entry, revision(t, f.s), "undo-clear"), "creator")
	if f.standings(t).Override == nil {
		t.Fatal("compensating clear did not restore")
	}
	runScoreCommand(t, f.s, reverseCommand(receipt.Entry, revision(t, f.s), "undo-order"), "creator")
	f.reopen(t)
	if f.standings(t).Override != nil {
		t.Fatal("order compensation not replayed")
	}
}

func TestNativeStandingsStopsAtRegularSeason(t *testing.T) {
	t.Parallel()
	f := resultsTest(t)
	runScoreCommand(t, f.s, ruleCommand("schedule.regularWeeks", "1", "now", "Finish after week one", revision(t, f.s)), "creator")
	got := f.standings(t)
	if got.Through != 1 || got.Decided != 16 || got.Computed[0].Franchise != f.a || got.Computed[0].Record != "1-0" {
		t.Fatalf("later weeks counted: %+v", got)
	}
}

func TestNativeResultsAndFlipPreview(t *testing.T) {
	t.Parallel()
	f := resultsTest(t)
	before, err := f.s.NativeResults(t.Context(), "native.test", 2026, 2)
	if err != nil || before.Games[0].Winner != f.b {
		t.Fatalf("initial game: %+v %v", before, err)
	}
	for _, kind := range []string{commissioner.ScoreFranchise, commissioner.ScorePlayer} {
		change := commissioner.RawScoreChange{Season: 2026, Week: 2, Delta: pointsDelta(15001)}
		if kind == commissioner.ScoreFranchise {
			change.Franchise = f.a
		} else {
			change.Player = "10" + strings.TrimPrefix(f.a, "franchise.")
		}
		raw := scoreCommand(t, kind, change, revision(t, f.s), "flip-"+kind)
		preview, err := f.s.NativeScorePreview(t.Context(), raw)
		if err != nil || preview.Flip.Status != commissioner.FlipYes || len(preview.Flip.Flips) != 1 ||
			preview.Flip.Flips[0].Before != before.Games[0].Words || len(preview.Warnings) != 1 {
			t.Fatalf("missed result flip: %+v %v", preview, err)
		}
		receipt := runScoreCommand(t, f.s, raw, "creator")
		after, err := f.s.NativeResults(t.Context(), "native.test", 2026, 2)
		if err != nil || after.Games[0].Winner != f.a || *after.Games[0].HomePoints-*after.Games[0].AwayPoints != 1 {
			t.Fatalf("exact changed result: %+v %v", after.Games[0], err)
		}
		runScoreCommand(t, f.s, reverseCommand(receipt.Entry, revision(t, f.s), "undo-"+kind), "creator")
	}
	f.incomplete = true
	out, err := f.s.NativeResults(t.Context(), "native.test", 2026, 2)
	if err != nil || out.Games[0].Winner != "" || out.Games[0].Status != commissioner.GameNoResult ||
		!strings.Contains(out.Games[0].Words, "not been reported") {
		t.Fatalf("incomplete winner: %+v %v", out, err)
	}
	raw := scoreCommand(t, commissioner.ScoreFranchise, commissioner.RawScoreChange{
		Season: 2026, Week: 2, Franchise: f.a, Delta: pointsDelta(1)}, revision(t, f.s), "unknown")
	preview, err := f.s.NativeScorePreview(t.Context(), raw)
	if err != nil || preview.Flip.Status != commissioner.FlipUnknown || len(preview.Warnings) == 0 {
		t.Fatalf("incomplete flip guessed: %+v %v", preview, err)
	}
	f.incomplete = false
	raw.Args.Score.Delta = pointsDelta(1)
	preview, err = f.s.NativeScorePreview(t.Context(), raw)
	if err != nil || preview.Flip.Status != commissioner.FlipNo || len(preview.Warnings) != 0 {
		t.Fatal(preview, err)
	}
	raw.Args.Score.Delta = pointsDelta(15000)
	receipt := runScoreCommand(t, f.s, raw, "creator")
	out, err = f.s.NativeResults(t.Context(), "native.test", 2026, 2)
	if err != nil || out.Games[0].Status != commissioner.GameTie || out.Games[0].Winner != "" {
		t.Fatal(out, err)
	}
	f.reopen(t)
	replay, err := f.s.NativeResults(t.Context(), "native.test", 2026, 2)
	if err != nil || !reflect.DeepEqual(out, replay) {
		t.Fatal("tie replay", err)
	}
	runScoreCommand(t, f.s, reverseCommand(receipt.Entry, revision(t, f.s), "undo-tie"), "creator")
}

// TestResultFlipWarningNamesOnlyTheGameTheChangeTouches pins the sentence an owner reads when a score change
// cannot be called: one game, named once by franchise name, and why - never every game of the week, never an id.
func TestResultFlipWarningNamesOnlyTheGameTheChangeTouches(t *testing.T) {
	t.Parallel()
	f := resultsTest(t)
	change := func(kind string) commissioner.RawCommand {
		c := commissioner.RawScoreChange{Season: 2026, Week: 2, Delta: pointsDelta(15001)}
		if kind == commissioner.ScoreFranchise {
			c.Franchise = f.a
		} else {
			c.Player = "10" + strings.TrimPrefix(f.a, "franchise.")
		}
		return scoreCommand(t, kind, c, revision(t, f.s), "words-"+kind)
	}
	notTouched := "This change does not turn over any result in week 2."
	for _, tc := range []struct {
		name   string
		setup  func()
		want   string
		player string // what a change to a player's score says, which differs when no one starts him
	}{
		{"neither score is complete", func() { f.incomplete = true },
			"We can't tell yet whether this changes Held team 1 v Held team 32: neither score is complete.", ""},
		{"neither team has a lineup", func() { f.incomplete, f.lineups.missing = false, true },
			"We can't tell yet whether this changes Held team 1 v Held team 32: neither team has a lineup set.", notTouched},
		{"one team has no lineup", func() { f.lineups.missing, f.lineups.without = false, map[string]bool{f.b: true} },
			"We can't tell yet whether this changes Held team 1 v Held team 32: Held team 32 has no lineup set.", ""},
	} {
		tc.setup()
		for _, kind := range []string{commissioner.ScoreFranchise, commissioner.ScorePlayer} {
			want := tc.want
			if kind == commissioner.ScorePlayer && tc.player != "" {
				want = tc.player
			}
			preview, err := f.s.NativeScorePreview(t.Context(), change(kind))
			if err != nil {
				t.Fatalf("%s %s: %v", tc.name, kind, err)
			}
			if preview.Flip.Words != want {
				t.Fatalf("%s %s: got %q", tc.name, kind, preview.Flip.Words)
			}
			if want == notTouched {
				continue
			}
			if preview.Flip.Status != commissioner.FlipUnknown || len(preview.Warnings) != 1 || preview.Warnings[0] != want {
				t.Fatalf("%s %s: %+v", tc.name, kind, preview)
			}
			if strings.Contains(preview.Flip.Words, "franchise.") || strings.Contains(preview.Flip.Words, "Held team 2 v") {
				t.Fatalf("%s %s: the warning names a franchise id, or a game the change does not touch: %q", tc.name, kind, preview.Flip.Words)
			}
		}
	}
}

func TestNativeStandingsPermissionsAndRefusals(t *testing.T) {
	t.Parallel()
	f := resultsTest(t)
	grant := commissioner.RawCommand{League: "native.test", Command: "commish.role.grant", Args: commissioner.RawArgs{
		People: &commissioner.RawPeople{Principal: "cc", Role: governance.CoCommissioner}}, Expected: revision(t, f.s),
		Request: "grant", Reason: "Appoint deputy", Effective: "now", Visibility: "private"}
	runScoreCommand(t, f.s, grant, "creator")
	raw := standingsCommand(commissioner.StandingsRecordAdjust, commissioner.RawStandingsChange{
		Season: 2026, Franchise: f.a, Wins: 3}, revision(t, f.s))
	for _, principal := range []string{"cc", "owner", "unknown"} {
		if _, err := f.s.ExecuteCommissioner(t.Context(), raw, principal, nil); err == nil {
			t.Fatalf("%s set standings", principal)
		}
	}
	for _, mutate := range []func(*commissioner.RawCommand){
		func(r *commissioner.RawCommand) { r.Reason = " " },
		func(r *commissioner.RawCommand) { r.Expected-- },
		func(r *commissioner.RawCommand) { r.Gravity = "G2" },
		func(r *commissioner.RawCommand) { r.Visibility = "private" },
		func(r *commissioner.RawCommand) { r.Effective = "next_rollover" },
		func(r *commissioner.RawCommand) { r.Args.Standings.Franchise = "franchise.99" },
		func(r *commissioner.RawCommand) { r.Args.Standings.Wins = -1 },
		func(r *commissioner.RawCommand) { r.Args.Name = "foreign" },
	} {
		bad := raw
		change := *raw.Args.Standings
		bad.Args.Standings = &change
		mutate(&bad)
		if _, err := f.s.ExecuteCommissioner(t.Context(), bad, "creator", nil); err == nil {
			t.Fatalf("invalid command accepted: %+v", bad)
		}
	}
	if revision(t, f.s) != raw.Expected {
		t.Fatal("refusal appended")
	}
	runScoreCommand(t, f.s, raw, "creator")
	raw.Args.Standings.Wins = 2
	if _, err := f.s.ExecuteCommissioner(t.Context(), raw, "creator", nil); err == nil {
		t.Fatal("conflicting request accepted")
	}
}

func TestNativeResultsStandingsWire(t *testing.T) {
	t.Parallel()
	f := resultsTest(t)
	results, err := f.s.NativeResults(t.Context(), "native.test", 2026, 2)
	if err != nil {
		t.Fatal(err)
	}
	computed := f.standings(t)
	runScoreCommand(t, f.s, standingsCommand(commissioner.StandingsRecordAdjust, commissioner.RawStandingsChange{
		Season: 2026, Franchise: f.a, Wins: 3}, revision(t, f.s)), "creator")
	overridden := f.standings(t)
	f.incomplete = true
	incomplete, err := f.s.NativeResults(t.Context(), "native.test", 2026, 2)
	if err != nil {
		t.Fatal(err)
	}
	preview, err := f.s.NativeScorePreview(t.Context(), scoreCommand(t, commissioner.ScoreFranchise,
		commissioner.RawScoreChange{Season: 2026, Week: 2, Franchise: f.a, Delta: pointsDelta(15001)}, revision(t, f.s), "wire-preview"))
	if err != nil {
		t.Fatal(err)
	}
	wire := struct {
		Preview    commissioner.ScorePreview    `json:"preview"`
		Results    commissioner.NativeResults   `json:"results"`
		Incomplete commissioner.NativeResults   `json:"incomplete"`
		Standings  commissioner.NativeStandings `json:"standings"`
		Override   commissioner.NativeStandings `json:"override"`
	}{preview, results, incomplete, computed, overridden}
	body, err := json.MarshalIndent(wire, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	path := "../../commissioner/testdata/native-results-standings-wire.json"
	body = append(body, '\n')
	if os.Getenv("TWR_WRITE_WIRE") == "1" {
		if err := os.WriteFile(path, body, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	held, err := os.ReadFile(path)
	if err != nil || string(held) != string(body) {
		t.Fatal("native results/standings wire stale", err)
	}
}

func TestNativeResultsPastSeasonKeepsPublishedSchedule(t *testing.T) {
	t.Parallel()
	f := resultsTest(t)
	before, err := f.s.NativeResults(t.Context(), "native.test", 2026, 1)
	if err != nil {
		t.Fatal(err)
	}
	runScoreCommand(t, f.s, ruleCommand("calendar.rollover", `"2027-03-01T00:00:00Z"`, "now", "Begin the next season", revision(t, f.s)), "creator")
	f.s.SetClock(func() time.Time { return time.Date(2027, 3, 2, 12, 0, 0, 0, time.UTC) })
	schedule, err := commissioner.ScheduleReading("native.test", scoreRecords(t, f.s))
	if err != nil {
		t.Fatal(err)
	}
	game := schedule.Weeks[0].Games[0]
	game.Home, game.Away = game.Away, game.Home
	runScoreCommand(t, f.s, commissioner.RawCommand{League: "native.test", Command: commissioner.ScheduleEdit,
		Args: commissioner.RawArgs{Schedule: &commissioner.RawScheduleArgs{Week: 1,
			Games: []commissioner.ScheduleGame{game}, Published: true}},
		Expected: revision(t, f.s), Request: "new-season-schedule", Reason: "New season home field", Effective: "now", Visibility: "league"}, "creator")
	for range 2 {
		after, err := f.s.NativeResults(t.Context(), "native.test", 2026, 1)
		if err != nil || !reflect.DeepEqual(before.Games, after.Games) {
			t.Fatal("later schedule rewrote past games", err)
		}
		f.reopen(t)
	}
	if _, err := f.s.NativeResults(t.Context(), "native.test", 2025, 1); err == nil {
		t.Fatal("invented an earlier league season")
	}
}

func TestNativeStandingsClearRecordsAndInvalidOrders(t *testing.T) {
	t.Parallel()
	f := resultsTest(t)
	for _, order := range [][]string{nil, {f.a}, {f.a, f.a}} {
		raw := standingsCommand(commissioner.StandingsOverride, commissioner.RawStandingsChange{Season: 2026, Order: order}, revision(t, f.s))
		if _, err := f.s.ExecuteCommissioner(t.Context(), raw, "creator", nil); err == nil {
			t.Fatal("incomplete order accepted")
		}
	}
	raw := standingsCommand(commissioner.StandingsOverrideClear, commissioner.RawStandingsChange{Season: 2026}, revision(t, f.s))
	if _, err := f.s.ExecuteCommissioner(t.Context(), raw, "creator", nil); err == nil {
		t.Fatal("cleared nonexistent override")
	}
	runScoreCommand(t, f.s, standingsCommand(commissioner.StandingsRecordAdjust, commissioner.RawStandingsChange{
		Season: 2026, Franchise: f.a, Wins: 3}, revision(t, f.s)), "creator")
	other, err := commissioner.StandingsAdjustmentsReading(scoreRecords(t, f.s), 2027)
	if err != nil || len(other.Records) != 0 {
		t.Fatal("override crossed season", err)
	}
	raw.Expected, raw.Request = revision(t, f.s), "clear-records"
	receipt := runScoreCommand(t, f.s, raw, "creator")
	if f.standings(t).Override != nil {
		t.Fatal("clear retained record override")
	}
	f.reopen(t)
	if f.standings(t).Override != nil {
		t.Fatal("clear records did not replay")
	}
	runScoreCommand(t, f.s, reverseCommand(receipt.Entry, revision(t, f.s), "restore-records"), "creator")
	if f.standings(t).Override == nil {
		t.Fatal("compensated clear did not restore records")
	}
}

func TestMatchedRestoreKeepsPopulatedNativeResults(t *testing.T) {
	f := resultsTest(t)
	dir := filepath.Dir(f.path)
	files := [3]db.BackupFile{{Name: "thewarroom.db", MaxVersion: db.MainVersion()}, {Name: "history.db", MaxVersion: SchemaVersion()}, {Name: "whatif.db", MaxVersion: db.MainVersion()}}
	b, err := db.NewBackups(dir, "test", files)
	if err != nil {
		t.Fatal(err)
	}
	var pools [3]*db.Pools
	pools[1] = f.pools
	for _, i := range []int{0, 2} {
		p, err := db.Open(t.Context(), filepath.Join(dir, files[i].Name))
		if err != nil {
			t.Fatal(err)
		}
		pools[i] = p
		t.Cleanup(func() {
			if err := p.Close(); err != nil {
				t.Error(err)
			}
		})
	}
	before, err := f.s.NativeResults(t.Context(), "native.test", 2026, 1)
	if err != nil || len(before.Games) == 0 || before.Games[0].Winner == "" {
		t.Fatalf("no native result: %+v %v", before, err)
	}
	roster, err := f.s.NativeAssets(t.Context(), "native.test")
	if err != nil || roster.Snapshot == nil || len(roster.Snapshot.Players) == 0 {
		t.Fatalf("no native roster: %+v %v", roster, err)
	}
	at := time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC)
	set, err := b.Capture(t.Context(), pools, "daily", at)
	if err != nil {
		t.Fatal(err)
	}
	runScoreCommand(t, f.s, ruleCommand("lineup.invalidPenalty", `"zero_top_starter"`, "now", "Change after capture", revision(t, f.s)), "creator")
	for _, p := range pools {
		if err := p.Close(); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := b.Restore(t.Context(), set, at.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	f.s, f.pools = statStore(t, f.path)
	f.install()
	after, err := f.s.NativeResults(t.Context(), "native.test", 2026, 1)
	if err != nil || !reflect.DeepEqual(before, after) {
		t.Fatalf("native results changed: %v", err)
	}
	restored, err := f.s.NativeAssets(t.Context(), "native.test")
	if err != nil || !reflect.DeepEqual(roster, restored) {
		t.Fatalf("native roster changed: %v", err)
	}
	people, err := f.s.CommissionerPeople(t.Context(), "native.test", "creator")
	if err != nil || !people.MayChangeRules {
		t.Fatalf("stored commissioner rights lost: %+v %v", people, err)
	}
}

package history

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

func TestLineupSetMustBeLegalUnderTheNativeCheck(t *testing.T) {
	t.Parallel()
	f := lineupTest(t)
	was := revision(t, f.s)
	short := f.lineup(f.a, map[leaguerules.Position]int{leaguerules.QB: 1, leaguerules.RB: 1, leaguerules.WR: 2,
		leaguerules.TE: 1, leaguerules.PK: 1, leaguerules.DT: 2, leaguerules.DE: 2, leaguerules.LB: 2, leaguerules.CB: 2,
		leaguerules.S: 2})
	_, err := f.set(t, f.a, short)
	if got := refusal(t, err); !strings.Contains(got, "Invalid lineup") || !strings.Contains(got, "needs exactly 21") {
		t.Fatalf("an illegal lineup must be turned down in plain words: %q", got)
	}
	noKicker := f.lineup(f.a, map[leaguerules.Position]int{leaguerules.QB: 1, leaguerules.RB: 2, leaguerules.WR: 4,
		leaguerules.TE: 1, leaguerules.DT: 2, leaguerules.DE: 3, leaguerules.LB: 3, leaguerules.CB: 2, leaguerules.S: 4})
	if _, err := f.set(t, f.a, noKicker); !strings.Contains(refusal(t, err), "PK: 0 starting") {
		t.Fatalf("a position limit was not held: %v", err)
	}
	if revision(t, f.s) != was {
		t.Fatalf("a refused lineup was recorded: revision %d", revision(t, f.s))
	}
	bench := append([]string{}, f.legal(f.a)[1:]...)
	bench = append(bench, f.roster[f.a][leaguerules.QB][1], f.roster[f.b][leaguerules.QB][0])
	if _, err := f.set(t, f.a, bench); !strings.Contains(refusal(t, err), "is not on this franchise's active roster") {
		t.Fatalf("another franchise's player started: %v", err)
	}
	receipt := f.mustSet(t, f.a, f.legal(f.a))
	if got := f.reading(t, f.a); !got.Set || got.Verdict.Status != nativecheck.LineupValid || len(got.Starters) != 21 {
		t.Fatalf("the recorded lineup is not read back: %+v", got)
	}
	if retry, err := f.s.ExecuteCommissioner(t.Context(), commissioner.RawCommand{League: "native.test",
		Command: commissioner.LineupSet, Effective: "now", Visibility: "private", Expected: receipt.Revision - 1,
		Request: "lineup.1", Args: commissioner.RawArgs{Lineup: &commissioner.RawLineupChange{Season: 2026,
			Franchise: f.a, Week: 1, Starters: nil}}}, f.ownerOfA, nil); err == nil {
		t.Fatalf("an empty lineup was accepted: %+v", retry)
	}
}

func TestLineupBelongsToItsOwnerAndTheAbilityToSetIt(t *testing.T) {
	t.Parallel()
	f := lineupTest(t)
	for _, principal := range []string{f.ownerOfB, "creator", "stranger"} {
		_, err := f.s.ExecuteCommissioner(t.Context(),
			f.command(t, commissioner.LineupSet, f.a, 1, f.legal(f.a)), principal, nil)
		if got := refusal(t, err); got != "You may set a lineup only for your own franchise" {
			t.Fatalf("%s set another franchise's lineup: %q", principal, got)
		}
	}
	executePeople(t, f.s, "creator", "commish.abilities.set",
		commissioner.RawPeople{Franchise: f.a, Ability: "lineups", On: false}, revision(t, f.s))
	_, err := f.set(t, f.a, f.legal(f.a))
	if got := refusal(t, err); !strings.Contains(got, "turned off lineup changes") {
		t.Fatalf("the commissioner's switch was ignored: %q", got)
	}
}

func TestLockedPlayerCannotBeMovedInOrOutAfterKickoff(t *testing.T) {
	t.Parallel()
	f := lineupTest(t)
	qb2 := f.roster[f.a][leaguerules.QB][1]
	f.mustSet(t, f.a, f.legal(f.a))
	swapQB := append([]string{qb2}, f.legal(f.a)[1:]...)
	f.at = f.dallas.Add(-time.Second)
	f.mustSet(t, f.a, swapQB)
	f.mustSet(t, f.a, f.legal(f.a))
	f.at = f.dallas
	_, err := f.set(t, f.a, swapQB)
	want := "QB1 of team 01 locked at kickoff (Oct 18, 2026 at 10:00 AM PDT); a lineup can't add or drop a player once that player's game has started"
	if got := refusal(t, err); got != want {
		t.Fatalf("locked player out:\n got %q\nwant %q", got, want)
	}
	f.at = f.dallas.Add(-time.Second)
	f.mustSet(t, f.a, swapQB)
	f.at = f.dallas
	_, err = f.set(t, f.a, f.legal(f.a))
	if got := refusal(t, err); got != want {
		t.Fatalf("locked player in:\n got %q\nwant %q", got, want)
	}
	// A player whose game is later is not locked by another game's kickoff; a bye and no team never lock.
	line := f.legal(f.a)
	line = append([]string{qb2}, line[1:]...)
	defence := f.roster[f.a][leaguerules.S]
	moved := strings.Join(line, ",")
	line[len(line)-1] = defence[3]
	if strings.Join(line, ",") == moved {
		t.Fatal("the fixture did not move a defender")
	}
	f.mustSet(t, f.a, line)
	f.at = f.nyGiants
	line[len(line)-1] = defence[2]
	_, err = f.set(t, f.a, line)
	if got := refusal(t, err); !strings.Contains(got, "S3 of team 01 locked at kickoff (Oct 18, 2026 at 5:20 PM PDT)") ||
		!strings.Contains(got, "S4 of team 01 locked at kickoff") {
		t.Fatalf("the later game must lock its own players: %q", got)
	}
}

func TestLockUnknownWhenKickoffTimesAreNotHeld(t *testing.T) {
	t.Parallel()
	f := lineupTest(t)
	raw := f.command(t, commissioner.LineupSet, f.a, 2, f.legal(f.a))
	_, err := f.s.ExecuteCommissioner(t.Context(), raw, f.ownerOfA, nil)
	if got := refusal(t, err); !strings.HasPrefix(got, "We don't hold the week 2 NFL kickoff times, so we can't tell when ") {
		t.Fatalf("a lock was guessed without kickoff times: %q", got)
	}
}

func TestByeAndTeamlessPlayersNeverLock(t *testing.T) {
	t.Parallel()
	f := lineupTest(t)
	rb3, qb2 := f.roster[f.a][leaguerules.RB][2], f.roster[f.a][leaguerules.QB][1]
	base := f.legal(f.a)
	f.mustSet(t, f.a, base)
	f.at = f.nyGiants.Add(time.Hour)
	swapped := append([]string{}, base...)
	swapped[0], swapped[1] = qb2, rb3
	// Both outgoing players are locked Dallas players, so this must still be refused.
	_, err := f.set(t, f.a, swapped)
	if got := refusal(t, err); !strings.Contains(got, "QB1 of team 01 locked") {
		t.Fatalf("an outgoing locked player moved: %q", got)
	}
	// With nobody locked in the lineup, a bye player and a teamless player move freely.
	f.at = f.dallas.Add(-time.Hour)
	free := append([]string{}, base...)
	free[0], free[1] = qb2, rb3
	f.mustSet(t, f.a, free)
}

func TestCorrectionLockedOutOwnersAndCorrectionIsFinal(t *testing.T) {
	t.Parallel()
	f := lineupTest(t)
	f.mustSet(t, f.a, f.legal(f.a))
	f.at = f.dallas.Add(time.Hour)
	own := f.correct(t, f.a, f.legal(f.a))
	if _, err := f.s.ExecuteCommissioner(t.Context(), own, f.ownerOfA, nil); err == nil {
		t.Fatal("an owner corrected their own locked lineup")
	}
	grantDeputy(t, f)
	own.Expected = revision(t, f.s)
	own.Reason = " "
	if _, err := f.s.ExecuteCommissioner(t.Context(), own, f.cc, nil); err == nil {
		t.Fatal("a correction went in without a reason")
	}
	swapped := append([]string{f.roster[f.a][leaguerules.QB][1]}, f.legal(f.a)[1:]...)
	correction := f.correct(t, f.a, swapped)
	f.s.SetClock(f.clock)
	runScoreCommand(t, f.s, correction, f.cc)
	_, err := f.set(t, f.a, f.legal(f.a))
	if got := refusal(t, err); !strings.Contains(got, "A commissioner has set Held team 1's week 1 lineup; ask them to change it") {
		t.Fatalf("the owner overwrote the commissioner's correction: %q", got)
	}
}

func TestMissedLineupIsFlaggedNeverPenalised(t *testing.T) {
	t.Parallel()
	f := lineupTest(t)
	got := f.results(t)
	flagged := map[string]bool{}
	for _, m := range got.Missed {
		flagged[m.Franchise] = m.Flagged
	}
	if len(got.Missed) != 32 || !flagged[f.a] || !flagged[f.b] {
		t.Fatalf("every missed lineup must be flagged to the commissioner: %+v", got.Missed)
	}
	for _, score := range got.Scores {
		if score.Franchise != f.a && score.Franchise != f.b {
			continue
		}
		v := score.Lineup
		if v.Status != nativecheck.LineupMissed || !v.Flagged || v.Penalty != "" || score.Points != nil || score.Zeroed != "" {
			t.Fatalf("a missed lineup must be flagged and not penalised: %+v", score)
		}
		if !strings.Contains(score.Words, "flagged to the commissioner as a missed lineup, and nothing is zeroed") {
			t.Fatalf("missed lineup words: %q", score.Words)
		}
	}
	if game := got.Games[0]; game.Status != commissioner.GameNoResult || game.Winner != "" {
		t.Fatalf("a game with a missed lineup was decided: %+v", game)
	}
	reading := f.reading(t, f.a)
	if reading.Set || reading.Verdict.Status != nativecheck.LineupMissed || !reading.Verdict.Flagged {
		t.Fatalf("the reading must say the lineup was missed: %+v", reading)
	}
	f.mustSet(t, f.a, f.legal(f.a))
	for _, m := range f.results(t).Missed {
		if m.Franchise == f.a {
			t.Fatalf("a lineup that was set is still flagged: %+v", m)
		}
	}
}

func TestRecordedLineupsDecideTheResultAndReopen(t *testing.T) {
	t.Parallel()
	f := lineupTest(t)
	f.points[f.roster[f.a][leaguerules.QB][0]] = 300000
	f.points[f.roster[f.b][leaguerules.WR][0]] = 20000
	f.mustSet(t, f.a, f.legal(f.a))
	if game := f.results(t).Games[0]; game.Status != commissioner.GameNoResult {
		t.Fatalf("one lineup cannot decide a game: %+v", game)
	}
	f.mustSet(t, f.b, f.legal(f.b))
	for range 2 {
		game := f.results(t).Games[0]
		if game.Status != commissioner.GameFinal || game.Winner != f.a ||
			game.HomePoints == nil || *game.HomePoints != 500000 || *game.AwayPoints != 220000 {
			t.Fatalf("results must read the recorded lineups: %+v", game)
		}
		f.reopen(t)
	}
	if standings := f.standings(t); standings.Decided != 1 {
		t.Fatalf("standings must read the recorded lineups too: %+v", standings)
	}
}

func TestInvalidLineupLosesItsTopStarterAndNothingElse(t *testing.T) {
	t.Parallel()
	f := lineupTest(t)
	grantDeputy(t, f)
	f.points[f.roster[f.a][leaguerules.QB][0]] = 300000
	f.mustSet(t, f.b, f.legal(f.b))
	short := f.legal(f.a)[:20]
	runScoreCommand(t, f.s, f.correct(t, f.a, short), f.cc)
	home := f.results(t).Scores
	var score commissioner.FranchiseWeekScore
	for _, s := range home {
		if s.Franchise == f.a {
			score = s
		}
	}
	if score.Lineup.Status != nativecheck.LineupInvalid || score.Zeroed != f.roster[f.a][leaguerules.QB][0] ||
		score.Points == nil || *score.Points != 190000 {
		t.Fatalf("the invalid lineup must lose only its top starter's 30 points: %+v", score)
	}
	if !strings.Contains(score.Words, "the lineup is invalid, so QB1 of team 01 counts for nothing") {
		t.Fatalf("penalty words: %q", score.Words)
	}
}

func TestLineupCorrectionGovernsAndRescoresTheWeek(t *testing.T) {
	t.Parallel()
	f := lineupTest(t)
	grantDeputy(t, f)
	qb1, qb2 := f.roster[f.a][leaguerules.QB][0], f.roster[f.a][leaguerules.QB][1]
	f.points[qb1] = 300000
	f.points[f.roster[f.b][leaguerules.WR][0]] = 20000
	f.mustSet(t, f.a, f.legal(f.a))
	f.mustSet(t, f.b, f.legal(f.b))
	f.at = f.dallas.Add(time.Hour)
	swapped := append([]string{qb2}, f.legal(f.a)[1:]...)
	for pass := range 2 {
		before := f.results(t)
		if game := before.Games[0]; game.Winner != f.a || *game.HomePoints != 500000 {
			t.Fatalf("pass %d: baseline: %+v", pass, game)
		}
		raw := f.correct(t, f.a, swapped)
		preview, err := f.s.NativeScorePreview(t.Context(), raw)
		if err != nil || preview.Flip.Status != commissioner.FlipYes || len(preview.Warnings) != 1 ||
			preview.Flip.Words != "This change turns over a result in week 1: Held team 1 beat Held team 32, 50 to 22 becomes Held team 32 beat Held team 1, 22 to 21." {
			t.Fatalf("pass %d: the correction must warn that it turns over the result: %+v %v", pass, preview, err)
		}
		if revision(t, f.s) != raw.Expected {
			t.Fatal("a preview recorded a change")
		}
		receipt := runScoreCommand(t, f.s, raw, f.cc)
		if retry := runScoreCommand(t, f.s, raw, f.cc); retry != receipt {
			t.Fatal("a retried correction appended")
		}
		for range 2 {
			after := f.results(t).Games[0]
			if after.Winner != f.b || *after.HomePoints != 210000 || *after.AwayPoints != 220000 {
				t.Fatalf("pass %d: the week was not rescored from the correction: %+v", pass, after)
			}
			if reading := f.reading(t, f.a); !reading.Corrected || reading.Starters[0].Player != qb2 {
				t.Fatalf("pass %d: the reading does not show the correction: %+v", pass, reading)
			}
			f.reopen(t)
		}
		runScoreCommand(t, f.s, reverseCommand(receipt.Entry, revision(t, f.s), fmt.Sprint("undo-lineup", pass)), f.cc)
		for range 2 {
			if game := f.results(t).Games[0]; game.Winner != f.a || *game.HomePoints != 500000 {
				t.Fatalf("pass %d: compensation did not restore the owner's lineup: %+v", pass, game)
			}
			f.reopen(t)
		}
	}
}

func TestCorrectionGivesAMissedLineupItsResult(t *testing.T) {
	t.Parallel()
	f := lineupTest(t)
	grantDeputy(t, f)
	f.mustSet(t, f.b, f.legal(f.b))
	raw := f.correct(t, f.a, f.legal(f.a))
	preview, err := f.s.NativeScorePreview(t.Context(), raw)
	if err != nil || preview.Flip.Status != commissioner.FlipYes || len(preview.Flip.Flips) != 1 || !preview.Flip.Flips[0].Decided ||
		!strings.Contains(preview.Flip.Words, "a game with no result yet becomes: Held team 1 beat Held team 32, 21 to 21") &&
			!strings.Contains(preview.Flip.Words, "a game with no result yet becomes: Held team 1 and Held team 32 tied at 21") {
		t.Fatalf("a correction that gives a game its first result must say so: %+v %v", preview, err)
	}
	if preview.Lineup == nil || preview.Lineup.Status != nativecheck.LineupValid {
		t.Fatalf("the preview must judge the corrected lineup: %+v", preview.Lineup)
	}
}

func TestCorrectionPreviewWarnsOfThePenalty(t *testing.T) {
	t.Parallel()
	f := lineupTest(t)
	grantDeputy(t, f)
	raw := f.correct(t, f.a, f.legal(f.a)[:20])
	preview, err := f.s.NativeScorePreview(t.Context(), raw)
	if err != nil || preview.Lineup == nil || preview.Lineup.Status != nativecheck.LineupInvalid {
		t.Fatalf("an illegal correction must be judged invalid: %+v %v", preview, err)
	}
	joined := strings.Join(preview.Warnings, " | ")
	if !strings.Contains(joined, "Invalid lineup") || !strings.Contains(joined, "The highest-scoring starter's points are set to zero when the week is scored.") {
		t.Fatalf("the penalty must be named before the correction is made: %q", joined)
	}
}

func TestNativeLineupReadingNamesLocksInTheOwnersTime(t *testing.T) {
	t.Parallel()
	f := lineupTest(t)
	f.mustSet(t, f.a, f.legal(f.a))
	read := f.reading(t, f.a)
	if read.Zone != "PDT" || len(read.Starters) != 21 || len(read.Bench) != 8 {
		t.Fatalf("zone or bench: %q, %d starters, %d on the bench", read.Zone, len(read.Starters), len(read.Bench))
	}
	byName := map[string]commissioner.LineupPlayer{}
	for _, p := range append(append([]commissioner.LineupPlayer{}, read.Starters...), read.Bench...) {
		byName[p.Name] = p
	}
	qb1, qb2, rb3 := byName["QB1 of team 01"], byName["QB2 of team 01"], byName["RB3 of team 01"]
	if qb1.Locked || qb1.Words != "Locks at kickoff, Oct 18, 2026 at 10:00 AM PDT" || qb1.LocksAt == nil ||
		!qb1.LocksAt.Equal(f.dallas) || qb1.LocksAt.Format("-07:00") != "-07:00" {
		t.Fatalf("a starter's lock: %+v", qb1)
	}
	if qb2.Locked || qb2.LocksAt != nil || qb2.Words != "No NFL game this week, so this player never locks" ||
		rb3.Locked || rb3.LocksAt != nil {
		t.Fatalf("teamless and bye players never lock: %+v %+v", qb2, rb3)
	}
	f.at = f.dallas
	locked := f.reading(t, f.a)
	byName = map[string]commissioner.LineupPlayer{}
	for _, p := range append(append([]commissioner.LineupPlayer{}, locked.Starters...), locked.Bench...) {
		byName[p.Name] = p
	}
	if !byName["QB1 of team 01"].Locked || byName["QB1 of team 01"].Words != "Locked at kickoff, Oct 18, 2026 at 10:00 AM PDT" ||
		byName["DT1 of team 01"].Locked {
		t.Fatalf("lock state at kickoff: %+v", byName["QB1 of team 01"])
	}
	if !strings.Contains(locked.Words, "Held team 1's week 1 lineup is set. Valid lineup.") {
		t.Fatalf("verdict words: %q", locked.Words)
	}
	if _, err := f.s.NativeLineup(t.Context(), "native.test", "franchise.99", f.creatorID, 1, f.at); err == nil {
		t.Fatal("read a franchise the league does not have")
	}
	if !reflect.DeepEqual(locked.Rules, f.reading(t, f.a).Rules) {
		t.Fatal("rules changed between readings")
	}
}

// TestNativeLineupWire pins the Go-written shape of the lineup reading, the correction preview and a week
// scored from recorded lineups, so the screens that read them cannot drift from it.
func TestNativeLineupWire(t *testing.T) {
	t.Parallel()
	f := lineupTest(t)
	grantDeputy(t, f)
	f.points[f.roster[f.a][leaguerules.QB][0]] = 300000
	missed := f.reading(t, f.a)
	f.mustSet(t, f.a, f.legal(f.a))
	f.mustSet(t, f.b, f.legal(f.b))
	f.at = f.dallas
	locked := f.reading(t, f.a)
	swapped := append([]string{f.roster[f.a][leaguerules.QB][1]}, f.legal(f.a)[1:]...)
	preview, err := f.s.NativeScorePreview(t.Context(), f.correct(t, f.a, swapped))
	if err != nil {
		t.Fatal(err)
	}
	runScoreCommand(t, f.s, f.correct(t, f.a, f.legal(f.a)[:20]), f.cc)
	wire := struct {
		Missed    commissioner.NativeLineup  `json:"missed"`
		Locked    commissioner.NativeLineup  `json:"locked"`
		Preview   commissioner.ScorePreview  `json:"preview"`
		Corrected commissioner.NativeLineup  `json:"corrected"`
		Results   commissioner.NativeResults `json:"results"`
	}{missed, locked, preview, f.reading(t, f.a), f.results(t)}
	wire.Results.Games, wire.Results.Scores = wire.Results.Games[:1], wire.Results.Scores[:2]
	wire.Results.Missed = wire.Results.Missed[:1]
	body, err := json.MarshalIndent(wire, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	body = append(body, '\n')
	path := "../../commissioner/testdata/native-lineup-wire.json"
	if os.Getenv("TWR_WRITE_WIRE") == "1" {
		if err := os.WriteFile(path, body, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	held, err := os.ReadFile(path)
	if err != nil || string(held) != string(body) {
		t.Fatal("native lineup wire stale", err)
	}
}

// TestCM004GovernsTheLineupsTheLeagueRecords changes the league's lineup rules through the real command and
// shows each one reach recorded lineups: what an owner may set, what an invalid lineup costs, whether a
// missed lineup is flagged. Each is then reversed, and the database reopened.
func TestCM004GovernsTheLineupsTheLeagueRecords(t *testing.T) {
	t.Parallel()
	f := lineupTest(t)
	f.points[f.roster[f.a][leaguerules.QB][0]] = 300000
	f.mustSet(t, f.b, f.legal(f.b))
	runScoreCommand(t, f.s, f.correct(t, f.a, f.legal(f.a)[:20]), f.creatorID)
	scoreOf := func(franchise string) commissioner.FranchiseWeekScore {
		for _, s := range f.results(t).Scores {
			if s.Franchise == franchise {
				return s
			}
		}
		t.Fatalf("no score for %s", franchise)
		return commissioner.FranchiseWeekScore{}
	}
	rule := func(field, value, reason string) commissioner.Receipt {
		return runScoreCommand(t, f.s, ruleCommand(field, value, "now", reason, revision(t, f.s)), f.creatorID)
	}
	for pass := range 2 {
		if s := scoreOf(f.a); s.Zeroed == "" || *s.Points != 190000 {
			t.Fatalf("pass %d: the Legacy penalty zeroes the top starter: %+v", pass, s)
		}
		penalty := rule("lineup.invalidPenalty", `"none"`, "No penalty for an invalid lineup")
		for range 2 {
			if s := scoreOf(f.a); s.Zeroed != "" || *s.Points != 490000 || s.Lineup.Status != nativecheck.LineupInvalid ||
				s.Lineup.Penalty != "No penalty" {
				t.Fatalf("pass %d: the league's ruling of no penalty was not followed: %+v", pass, s)
			}
			f.reopen(t)
		}
		runScoreCommand(t, f.s, reverseCommand(penalty.Entry, revision(t, f.s), fmt.Sprint("undo-penalty", pass)), f.creatorID)
		if s := scoreOf(f.a); s.Zeroed == "" || *s.Points != 190000 {
			t.Fatalf("pass %d: compensation did not restore the penalty: %+v", pass, s)
		}
		flagged := func() bool {
			for _, m := range f.results(t).Missed {
				if m.Franchise == "franchise.02" {
					return m.Flagged
				}
			}
			t.Fatal("franchise.02 should have missed its lineup")
			return false
		}
		if !flagged() {
			t.Fatalf("pass %d: a missed lineup is flagged under Legacy", pass)
		}
		unflag := rule("lineup.missedFlag", `false`, "Do not flag missed lineups")
		if flagged() || strings.Contains(scoreOf("franchise.02").Words, "flagged") {
			t.Fatalf("pass %d: the league stopped flagging missed lineups", pass)
		}
		runScoreCommand(t, f.s, reverseCommand(unflag.Entry, revision(t, f.s), fmt.Sprint("undo-flag", pass)), f.creatorID)
		f.reopen(t)
		if !flagged() {
			t.Fatalf("pass %d: compensation did not restore the flag", pass)
		}
		groups := rule("lineup.groups", `{"offense":{"min":0,"max":8},"defense":{"min":12,"max":12},"total":{"min":22,"max":22}}`,
			"Start twenty-two")
		_, err := f.set(t, f.b, f.legal(f.b))
		if got := refusal(t, err); !strings.Contains(got, "Total: 21 starting, needs exactly 22") {
			t.Fatalf("pass %d: an owner set a lineup the league's limits do not allow: %q", pass, got)
		}
		runScoreCommand(t, f.s, reverseCommand(groups.Entry, revision(t, f.s), fmt.Sprint("undo-groups", pass)), f.creatorID)
		f.mustSet(t, f.b, f.legal(f.b))
	}
}

func TestLineupsNeedTheAppsFootballFacts(t *testing.T) {
	t.Parallel()
	f := lineupTest(t)
	f.s.SetLineupEnvironment(nil)
	_, err := f.set(t, f.a, f.legal(f.a))
	if got := refusal(t, err); !strings.Contains(got, "Lineups can't be judged until the app holds the NFL players and schedule") {
		t.Fatalf("a lineup was judged without football facts: %q", got)
	}
	if _, err := f.s.NativeLineup(t.Context(), "native.test", f.a, f.ownerOfA, 1, f.at); err == nil {
		t.Fatal("a lineup was read without football facts")
	}
}

// TestOwnersLineupCanBeCompensatedPrivatelyAndNeedsTheReversalPermission reverses an owner's private lineup:
// the one before it stands again, only a commissioner may do it, and it must be sent privately.
func TestOwnersLineupCanBeCompensatedPrivatelyAndNeedsTheReversalPermission(t *testing.T) {
	t.Parallel()
	f := lineupTest(t)
	grantDeputy(t, f)
	first := f.legal(f.a)
	second := append([]string{f.roster[f.a][leaguerules.QB][1]}, first[1:]...)
	f.mustSet(t, f.a, first)
	latest := f.mustSet(t, f.a, second)
	undo := reverseCommand(latest.Entry, revision(t, f.s), "undo-owner-lineup")
	if _, err := f.s.ExecuteCommissioner(t.Context(), undo, f.ownerOfA, nil); err == nil {
		t.Fatal("an owner reversed a recorded league change")
	}
	if _, err := f.s.ExecuteCommissioner(t.Context(), undo, f.cc, nil); err == nil {
		t.Fatal("a private lineup was reversed in the open")
	}
	undo.Visibility = "private"
	runScoreCommand(t, f.s, undo, f.cc)
	for range 2 {
		if got := f.reading(t, f.a); got.Starters[0].Player != first[0] {
			t.Fatalf("the lineup before the reversed one must stand again: %+v", got.Starters[0])
		}
		f.reopen(t)
	}
}

func TestVacantCaretakerSetsLineupButCannotProxyAnOwner(t *testing.T) {
	f := lineupTest(t)
	executePeople(t, f.s, "creator", "commish.seat.remove", commissioner.RawPeople{Franchise: f.a}, revision(t, f.s))
	executePeople(t, f.s, "creator", commissioner.CaretakerSeat, commissioner.RawPeople{Franchise: f.a, Principal: "creator"}, revision(t, f.s))
	raw := f.command(t, commissioner.LineupSet, f.a, 1, f.legal(f.a))
	if _, err := f.s.ExecuteCommissioner(t.Context(), raw, "creator", nil); err != nil {
		t.Fatal(err)
	}
	got, err := f.s.NativeLineup(t.Context(), "native.test", f.a, "creator", 1, f.at)
	if err != nil || !got.Set || got.Verdict.Status != nativecheck.LineupValid {
		t.Fatalf("caretaker lineup not recorded: %v", err)
	}
	executePeople(t, f.s, "creator", "commish.seat.replace", commissioner.RawPeople{Franchise: f.a, Principal: f.ownerOfA}, revision(t, f.s))
	before := revision(t, f.s)
	if _, err := f.s.ExecuteCommissioner(t.Context(), f.command(t, commissioner.LineupSet, f.a, 1, f.legal(f.a)), "creator", nil); err == nil {
		t.Fatal("former caretaker set owner's lineup")
	}
	if revision(t, f.s) != before {
		t.Fatal("refused caretaker changed ledger")
	}
}

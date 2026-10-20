package history

import (
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

// Round one of the 2027 rookie draft follows the final standings of the 2026 season (census CM-041).

// fullDraftPicks gives every franchise its own five round 2027 picks, as an app league holds them.
func fullDraftPicks(assets *commissioner.NativeAssets) {
	snapshot := assets.Snapshot
	kept := []commissioner.NativePick{}
	for _, p := range snapshot.Picks {
		if p.Season != 2027 {
			kept = append(kept, p)
		}
	}
	for round := 1; round <= 5; round++ {
		for _, franchise := range snapshot.Franchises {
			kept = append(kept, commissioner.NativePick{Season: 2027, Round: round, Original: franchise.ID,
				Owner: franchise.ID})
		}
	}
	snapshot.Picks = kept
}

func draftStart() time.Time { return time.Date(2027, 7, 14, 16, 0, 0, 0, time.UTC) }

func draftCommand(command string, d commissioner.RawDraftArgs, expected int64) commissioner.RawCommand {
	return commissioner.RawCommand{League: "native.test", Command: command, Args: commissioner.RawArgs{Draft: &d},
		Reason: "Set up our draft", Expected: expected, Request: fmt.Sprintf("draft.%d", expected), Effective: "now",
		Visibility: "league"}
}

// draftedFixture is the results league with its draft set up: later rounds in the same order, and a start.
func draftedFixture(t *testing.T) *resultsFixture {
	t.Helper()
	f := resultsTestSeeded(t, fullDraftPicks)
	runScoreCommand(t, f.s, ruleCommand("calendar.boundary", `"start_of_day"`, "now", "Open windows at the start of a shared date",
		revision(t, f.s)), "creator")
	runScoreCommand(t, f.s, draftCommand(commissioner.DraftOrderPattern,
		commissioner.RawDraftArgs{Pattern: leaguerules.Linear}, revision(t, f.s)), "creator")
	runScoreCommand(t, f.s, draftCommand(commissioner.DraftStart,
		commissioner.RawDraftArgs{Season: 2027, At: draftStart()}, revision(t, f.s)), "creator")
	return f
}

func (f *resultsFixture) draft(t *testing.T, at time.Time) commissioner.NativeDraft {
	t.Helper()
	out, err := f.s.NativeDraft(t.Context(), commissioner.RawNativeDraftScope{League: "native.test", Season: 2027,
		Franchise: "franchise.01", At: at}, time.UTC, nil)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// roundOne is who picks first, in order, or the words that say why no one can yet.
func (f *resultsFixture) roundOne(t *testing.T, at time.Time) string {
	t.Helper()
	round := f.draft(t, at).Order
	if len(round) == 0 || len(round[0].Picks) == 0 {
		return f.draft(t, at).Words
	}
	originals := make([]string, len(round[0].Picks))
	for i, p := range round[0].Picks {
		originals[i] = strings.TrimPrefix(p.Original, "franchise.")
	}
	return strings.Join(originals, " ")
}

// worstFirst is what round one must read: the team the standings put last first, the best team last, and the
// 30 teams every tie-breaker left level in the order the draft's tie rule gives.
func (f *resultsFixture) worstFirst(worst, best string, reverse bool) string {
	level := []string{}
	for n := 1; n <= 32; n++ {
		id := fmt.Sprintf("franchise.%02d", n)
		if id != worst && id != best {
			level = append(level, strings.TrimPrefix(id, "franchise."))
		}
	}
	if reverse {
		slices.Reverse(level)
	}
	return strings.Join(append(append([]string{strings.TrimPrefix(worst, "franchise.")}, level...),
		strings.TrimPrefix(best, "franchise.")), " ")
}

func TestRoundOneIsTakenWorstFirstAndNeedsTheTieRuleOnlyForTies(t *testing.T) {
	t.Parallel()
	f := draftedFixture(t)
	before := time.Date(2027, 6, 1, 12, 0, 0, 0, time.UTC)
	// 30 teams finished level on every tie-breaker, so the draft cannot order them without its own tie rule.
	words := f.roundOne(t, before)
	if !strings.Contains(words, "Needs a ruling: how a tie in the draft order is broken") {
		t.Fatalf("round one was built with a tie nobody has ruled on: %q", words)
	}
	for _, rule := range []struct {
		name    string
		reverse bool
	}{{commissioner.TieByFranchiseOrder, false}, {commissioner.TieByReverseFranchiseOrder, true}} {
		set := runScoreCommand(t, f.s, draftCommand(commissioner.DraftOrderTiebreak,
			commissioner.RawDraftArgs{Tie: leaguerules.TieRule(rule.name)}, revision(t, f.s)), "creator")
		// f.a lost two of three games and f.b won two: worst first puts f.a first and f.b last.
		if got, want := f.roundOne(t, before), f.worstFirst(f.a, f.b, rule.reverse); got != want {
			t.Fatalf("%s: round one\n got %s\nwant %s", rule.name, got, want)
		}
		runScoreCommand(t, f.s, reverseCommand(set.Entry, revision(t, f.s), "undo-tie-"+rule.name), "creator")
		if got := f.roundOne(t, before); got != words {
			t.Fatalf("%s: reverting the tie rule: %q", rule.name, got)
		}
	}
	set := runScoreCommand(t, f.s, draftCommand(commissioner.DraftOrderTiebreak,
		commissioner.RawDraftArgs{Tie: "A coin toss"}, revision(t, f.s)), "creator")
	if got := f.roundOne(t, before); !strings.Contains(got, `cannot carry out the draft tie rule "A coin toss"`) {
		t.Fatalf("a tie rule the app cannot carry out was applied anyway: %q", got)
	}
	runScoreCommand(t, f.s, reverseCommand(set.Entry, revision(t, f.s), "undo-toss"), "creator")
}

func TestARecordedResultReordersRoundOneAndItsRevertRestoresIt(t *testing.T) {
	t.Parallel()
	f := draftedFixture(t)
	runScoreCommand(t, f.s, draftCommand(commissioner.DraftOrderTiebreak,
		commissioner.RawDraftArgs{Tie: commissioner.TieByFranchiseOrder}, revision(t, f.s)), "creator")
	before := time.Date(2027, 6, 1, 12, 0, 0, 0, time.UTC)
	original := f.worstFirst(f.a, f.b, false)
	// Giving a's week two score a point more than the gap flips that game: a then wins two of three and b
	// loses two, so the two teams trade places.
	flipped := f.worstFirst(f.b, f.a, false)
	for pass, when := range []string{"live", "after reopening the database"} {
		if pass == 1 {
			f.reopen(t)
		}
		if got := f.roundOne(t, before); got != original {
			t.Fatalf("%s: round one before the correction: %s", when, got)
		}
		change := scoreCommand(t, commissioner.ScoreFranchise, commissioner.RawScoreChange{Season: 2026, Week: 2,
			Franchise: f.a, Delta: pointsDelta(15001)}, revision(t, f.s), fmt.Sprint("raise-a-", pass))
		receipt := runScoreCommand(t, f.s, change, "creator")
		if got := f.roundOne(t, before); got != flipped {
			t.Fatalf("%s: a changed result did not reorder round one\n got %s\nwant %s", when, got, flipped)
		}
		f.reopen(t)
		if got := f.roundOne(t, before); got != flipped {
			t.Fatalf("%s: the corrected order did not survive reopening: %s", when, got)
		}
		runScoreCommand(t, f.s, reverseCommand(receipt.Entry, revision(t, f.s), fmt.Sprint("restore-a-", pass)), "creator")
		if got := f.roundOne(t, before); got != original {
			t.Fatalf("%s: the compensating revert left round one changed: %s", when, got)
		}
	}
}

func TestRoundOneWaitsForEveryRegularSeasonGame(t *testing.T) {
	t.Parallel()
	f := draftedFixture(t)
	runScoreCommand(t, f.s, draftCommand(commissioner.DraftOrderTiebreak,
		commissioner.RawDraftArgs{Tie: commissioner.TieByFranchiseOrder}, revision(t, f.s)), "creator")
	before := time.Date(2027, 6, 1, 12, 0, 0, 0, time.UTC)
	f.incomplete = true
	got := f.roundOne(t, before)
	if !strings.Contains(got, "needs the final standings of season 2026") || !strings.Contains(got, "still have no result") {
		t.Fatalf("round one was built while games were undecided: %q", got)
	}
	f.incomplete = false
	if got := f.roundOne(t, before); got != f.worstFirst(f.a, f.b, false) {
		t.Fatalf("round one once every game is decided: %s", got)
	}
}

func TestRoundOneIsFixedWhenTheDraftBeginsEvenIfAResultIsCorrectedLater(t *testing.T) {
	t.Parallel()
	f := draftedFixture(t)
	runScoreCommand(t, f.s, draftCommand(commissioner.DraftOrderTiebreak,
		commissioner.RawDraftArgs{Tie: commissioner.TieByFranchiseOrder}, revision(t, f.s)), "creator")
	original := f.worstFirst(f.a, f.b, false)
	during := draftStart().Add(2 * time.Hour)
	if got := f.roundOne(t, during); got != original {
		t.Fatalf("round one after the start: %s", got)
	}
	f.s.SetClock(func() time.Time { return draftStart().Add(time.Hour) })
	change := scoreCommand(t, commissioner.ScoreFranchise, commissioner.RawScoreChange{Season: 2026, Week: 2,
		Franchise: f.a, Delta: pointsDelta(15001)}, revision(t, f.s), "late-correction")
	runScoreCommand(t, f.s, change, "creator")
	if got := f.roundOne(t, during); got != original {
		t.Fatalf("a result corrected after the draft began moved round one: %s", got)
	}
}

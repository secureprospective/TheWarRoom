package history

import (
	"context"
	"fmt"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/fantasyscore"
	"github.com/secureprospective/TheWarRoom/internal/fantasyscore/source"
	"github.com/secureprospective/TheWarRoom/internal/governance"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

// statResults is the results league with a feed that can revise a starter's week after the league counted it.
type statResults struct {
	*resultsFixture
	// revisions are the quantities of tackles later versions of a week reported, by "player/week".
	revisions map[string][]leaguerules.Decimal
	// silent weeks are ones the feed never reported at all.
	silent map[string]bool
}

func statResultsTest(t *testing.T) *statResults {
	t.Helper()
	f := &statResults{resultsFixture: resultsTest(t), revisions: map[string][]leaguerules.Decimal{},
		silent: map[string]bool{}}
	f.installFeed()
	return f
}

func (f *statResults) starter(franchise string) string {
	return "10" + strings.TrimPrefix(franchise, "franchise.")
}

func (f *statResults) firstQuantity(player string, week int) leaguerules.Decimal {
	switch "franchise." + strings.TrimPrefix(player, "10") {
	case f.a:
		if week == 1 {
			return 1000000
		}
	case f.b:
		if week > 1 {
			return 20000
		}
	}
	return 10000
}

func tackleWeek(week int, tackles leaguerules.Decimal) source.PlayerWeek {
	return source.PlayerWeek{GSIS: "gsis", Position: "LB", Season: 2026, Week: week, SeasonType: "REG",
		Line: fantasyscore.StatLine{Quantities: map[leaguerules.StatID]leaguerules.Decimal{"tackle": tackles}}}
}

func (f *statResults) evidenceFor(player string, week int) source.NativeEvidence {
	if f.silent[fmt.Sprint(player, "/", week)] {
		return source.NativeEvidence{GSIS: "held-" + player, Missing: []string{"No NFL statistics are held for this season and week"}}
	}
	quantities := append([]leaguerules.Decimal{f.firstQuantity(player, week)}, f.revisions[fmt.Sprint(player, "/", week)]...)
	versions := make([]source.LineRevision, len(quantities))
	for i, q := range quantities {
		line := tackleWeek(week, q)
		versions[i] = source.LineRevision{Seq: i, Digest: source.LineDigest(line), Held: source.HeldWeek{
			SHA256: fmt.Sprintf("held-%d", i), FetchedAt: t0().Add(time.Duration(i) * time.Hour), Week: &line}}
	}
	newest := versions[len(versions)-1].Held
	return source.NativeEvidence{GSIS: "held-" + player, Held: &newest, Lines: versions}
}

func (f *statResults) installFeed() {
	f.s.evidence = func(_ context.Context, scope source.RawNativeScope) (source.NativeEvidence, error) {
		return f.evidenceFor(scope.Player, scope.Week), nil
	}
	f.s.statWeeks = func(context.Context, int) ([]commissioner.StatWeek, error) {
		out := []commissioner.StatWeek{}
		for _, franchise := range []string{f.a, f.b} {
			for week := 1; week <= 3; week++ {
				player := f.starter(franchise)
				if !f.silent[fmt.Sprint(player, "/", week)] {
					out = append(out, commissioner.StatWeek{Player: player, Week: week, Evidence: f.evidenceFor(player, week)})
				}
			}
		}
		return out, nil
	}
}

func (f *statResults) reopen(t *testing.T) {
	t.Helper()
	f.resultsFixture.reopen(t)
	f.installFeed()
}

func (f *statResults) queue(t *testing.T) commissioner.StatQueue {
	t.Helper()
	out, err := f.s.StatQueue(t.Context(), "native.test", 2026)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func (f *statResults) score(t *testing.T, franchise string, week int) commissioner.NativePlayerScore {
	t.Helper()
	out, err := f.s.NativePlayerScore(t.Context(), "native.test",
		source.RawNativeScope{Player: f.starter(franchise), Season: 2026, Week: week})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func (f *statResults) row(t *testing.T, franchise string) commissioner.StandingRow {
	t.Helper()
	for _, row := range f.standings(t).Computed {
		if row.Franchise == franchise {
			return row
		}
	}
	t.Fatalf("%s is not in the standings", franchise)
	return commissioner.StandingRow{}
}

func entriesOf(changes []commissioner.ScoreChange, entry string) int {
	n := 0
	for _, c := range changes {
		if c.Entry == entry {
			n++
		}
	}
	return n
}

func statCommand(command string, change commissioner.RawStatChange, expected int64, request string) commissioner.RawCommand {
	return commissioner.RawCommand{League: "native.test", Command: command, Args: commissioner.RawArgs{Stat: &change},
		Expected: expected, Request: request, Effective: "now", Visibility: "league"}
}

func (f *statResults) approve(t *testing.T, item, request string) commissioner.Receipt {
	t.Helper()
	return runScoreCommand(t, f.s, statCommand(commissioner.StatApprove, commissioner.RawStatChange{Item: item},
		revision(t, f.s), request), "creator")
}

// revise has the feed report a later version of a starter's week.
func (f *statResults) revise(franchise string, week int, tackles leaguerules.Decimal) {
	key := fmt.Sprint(f.starter(franchise), "/", week)
	f.revisions[key] = append(f.revisions[key], tackles)
}

func TestNativeStatApprovalGovernsScoreStandingsAndQueue(t *testing.T) {
	t.Parallel()
	f := statResultsTest(t)
	f.revise(f.a, 2, 30000)
	for pass := 0; pass < 2; pass++ {
		wins, losses := f.row(t, f.a), f.row(t, f.b)
		score := f.score(t, f.a, 2)
		queue := f.queue(t)
		if len(queue.Open) != 1 || *score.Points != 15000 || *queue.Open[0].Points != 30000 ||
			queue.Open[0].Flip != commissioner.FlipYes {
			t.Fatalf("pass %d before: score %+v queue %+v", pass, score, queue)
		}
		item := queue.Open[0].ID
		receipt := f.approve(t, item, fmt.Sprint("approve", pass))
		if retry := runScoreCommand(t, f.s, statCommand(commissioner.StatApprove, commissioner.RawStatChange{Item: item},
			revision(t, f.s)-1, fmt.Sprint("approve", pass)), "creator"); retry != receipt {
			t.Fatal("replaying the approval recorded it twice")
		}
		for range 2 {
			after, row := f.score(t, f.a, 2), f.row(t, f.a)
			if once := entriesOf(after.Changes, receipt.Entry); once != 1 {
				t.Fatalf("pass %d the approval is on the score %d times, not once", pass, once)
			}
			if *after.Points != *score.Points+30000 || row.PointsFor != wins.PointsFor+30000 ||
				row.Wins != wins.Wins+1 || f.row(t, f.b).Wins != losses.Wins-1 {
				t.Fatalf("pass %d approved: score %+v standing %+v", pass, after, row)
			}
			if q := f.queue(t); len(q.Open) != 0 || len(q.Decided) != pass+1 || q.Decided[0].Status != "Approved" ||
				q.Decided[0].By != "creator" || !q.Decided[0].At.Equal(t0()) {
				t.Fatalf("pass %d queue after approving: %+v", pass, q)
			}
			f.reopen(t)
		}
		runScoreCommand(t, f.s, reverseCommand(receipt.Entry, revision(t, f.s), fmt.Sprint("undo", pass)), "creator")
		for range 2 {
			if got := f.score(t, f.a, 2); *got.Points != *score.Points || !reflect.DeepEqual(f.row(t, f.a), wins) ||
				!reflect.DeepEqual(f.row(t, f.b), losses) {
				t.Fatalf("pass %d compensation did not restore the score and standings", pass)
			}
			if q := f.queue(t); len(q.Open) != 1 || len(q.Decided) != pass+1 || q.Decided[0].Status != "Approval undone" {
				t.Fatalf("pass %d queue after undo: %+v", pass, q)
			}
			f.reopen(t)
		}
	}
}

func TestNativeStatRejectionKeepsTheScoreAndStaysRejectedPerRevision(t *testing.T) {
	t.Parallel()
	f := statResultsTest(t)
	f.revise(f.a, 2, 30000)
	score, item := f.score(t, f.a, 2), f.queue(t).Open[0].ID
	reject := statCommand(commissioner.StatReject, commissioner.RawStatChange{Item: item}, revision(t, f.s), "reject")
	preview, err := f.s.NativeScorePreview(t.Context(), reject)
	if err != nil || preview.Flip.Status != commissioner.FlipNo || !strings.Contains(preview.Words, "score stays as it is") {
		t.Fatalf("rejecting moves no score: %+v %v", preview, err)
	}
	receipt := runScoreCommand(t, f.s, reject, "creator")
	for range 2 {
		if got := f.score(t, f.a, 2); *got.Points != *score.Points {
			t.Fatal("rejecting moved the score")
		}
		if q := f.queue(t); len(q.Open) != 0 || len(q.Decided) != 1 || q.Decided[0].Status != "Rejected" {
			t.Fatalf("rejected item came back: %+v", q)
		}
		f.reopen(t)
	}
	f.revise(f.a, 2, 40000)
	if q := f.queue(t); len(q.Open) != 1 || q.Open[0].ID == item || !strings.Contains(q.Open[0].Words, "tackle 1 → 4") {
		t.Fatalf("a new revision must come back as its own item: %+v", q)
	}
	runScoreCommand(t, f.s, reverseCommand(receipt.Entry, revision(t, f.s), "undo-reject"), "creator")
	if q := f.queue(t); len(q.Open) != 1 {
		t.Fatalf("an undone rejection leaves the newest revision waiting: %+v", q)
	}
}

func TestNativeStatApprovalPreviewStatesPointsAndFlip(t *testing.T) {
	t.Parallel()
	f := statResultsTest(t)
	f.revise(f.a, 2, 30000)
	item := f.queue(t).Open[0].ID
	raw := statCommand(commissioner.StatApprove, commissioner.RawStatChange{Item: item}, revision(t, f.s), "preview")
	preview, err := f.s.NativeScorePreview(t.Context(), raw)
	if err != nil || preview.Flip.Status != commissioner.FlipYes || len(preview.Flip.Flips) != 1 || len(preview.Warnings) != 1 ||
		!strings.Contains(preview.Words, "+3 points") || !strings.Contains(preview.Words, "a result changes") {
		t.Fatalf("a flip must be stated before the change fires: %+v %v", preview, err)
	}
	if got := f.score(t, f.a, 2); *got.Points != 15000 {
		t.Fatal("a preview moved the score")
	}
	f.revise(f.a, 3, 10002)
	harmless := f.queue(t).Open
	var small commissioner.StatQueueItem
	for _, open := range harmless {
		if open.Week == 3 {
			small = open
		}
	}
	if small.Flip != commissioner.FlipNo || !strings.HasSuffix(small.Words, "no result changes") {
		t.Fatalf("a change that flips nothing says so: %+v", small)
	}
}

func TestNativeStatCorrectionGovernsScoreAndStandings(t *testing.T) {
	t.Parallel()
	f := statResultsTest(t)
	for pass := 0; pass < 2; pass++ {
		wins := f.row(t, f.a)
		score := f.score(t, f.a, 2)
		raw := statCommand(commissioner.StatCorrect, commissioner.RawStatChange{Player: f.starter(f.a), Season: 2026,
			Week: 2, Stat: "tackle", Value: pointsDelta(30000), Source: "Gamebook, week 2"}, revision(t, f.s),
			fmt.Sprint("correct", pass))
		preview, err := f.s.NativeScorePreview(t.Context(), raw)
		if err != nil || preview.Flip.Status != commissioner.FlipYes || !strings.Contains(preview.Words, "tackle 1 → 3") ||
			!strings.Contains(preview.Words, "+3 points") || !strings.Contains(preview.Words, "Gamebook, week 2") {
			t.Fatalf("pass %d preview: %+v %v", pass, preview, err)
		}
		receipt := runScoreCommand(t, f.s, raw, "creator")
		for range 2 {
			after := f.score(t, f.a, 2)
			if *after.Points != *score.Points+30000 || f.row(t, f.a).Wins != wins.Wins+1 ||
				f.row(t, f.a).PointsFor != wins.PointsFor+30000 {
				t.Fatalf("pass %d corrected: %+v", pass, after)
			}
			last := after.Changes[len(after.Changes)-1]
			if last.Entry != receipt.Entry || last.Status != "Applied" || !strings.Contains(last.What, "Gamebook, week 2") {
				t.Fatalf("pass %d provenance: %+v", pass, after.Changes)
			}
			if q := f.queue(t); len(q.Decided) != pass+1 || q.Decided[0].Kind != "correct" || len(q.Open) != 0 {
				t.Fatalf("pass %d the queue must record the correction: %+v", pass, q)
			}
			f.reopen(t)
		}
		runScoreCommand(t, f.s, reverseCommand(receipt.Entry, revision(t, f.s), fmt.Sprint("undo-correct", pass)), "creator")
		for range 2 {
			if got := f.score(t, f.a, 2); *got.Points != *score.Points || !reflect.DeepEqual(f.row(t, f.a), wins) {
				t.Fatalf("pass %d compensation: %+v", pass, got)
			}
			f.reopen(t)
		}
	}
}

func TestNativeStatCorrectionWhereTheFeedIsSilent(t *testing.T) {
	t.Parallel()
	f := statResultsTest(t)
	f.silent[fmt.Sprint(f.starter(f.a), "/", 2)] = true
	if got := f.score(t, f.a, 2); got.Points != nil || got.PartialPoints != nil {
		t.Fatalf("a silent feed became a score: %+v", got)
	}
	position := scoreCommand(t, commissioner.PlayerPosition,
		commissioner.RawScoreChange{Player: f.starter(f.a), Position: leaguerules.LB}, revision(t, f.s), "position")
	runScoreCommand(t, f.s, position, "creator")
	raw := statCommand(commissioner.StatCorrect, commissioner.RawStatChange{Player: f.starter(f.a), Season: 2026,
		Week: 2, Stat: "tackle", Value: pointsDelta(20000), Source: "League office call"}, revision(t, f.s), "silent")
	if _, err := f.s.ExecuteCommissioner(t.Context(), raw, "creator", nil); err != nil {
		t.Fatal(err)
	}
	got := f.score(t, f.a, 2)
	if got.Status != "Incomplete" || got.PartialPoints == nil || *got.PartialPoints != 30000 ||
		len(got.Missing) == 0 || !strings.Contains(strings.Join(got.Missing, " "), "commissioner entered") {
		t.Fatalf("a hand-entered week is counted so far, and says so: %+v", got)
	}
}

func TestNativeStatCommandsPermissionsAndRefusals(t *testing.T) {
	t.Parallel()
	f := statResultsTest(t)
	f.revise(f.a, 2, 30000)
	item := f.queue(t).Open[0].ID
	approve := func(item, request string, expected int64) commissioner.RawCommand {
		return statCommand(commissioner.StatApprove, commissioner.RawStatChange{Item: item}, expected, request)
	}
	if _, err := f.s.ExecuteCommissioner(t.Context(), approve(item, "owner", revision(t, f.s)), "owner", nil); err == nil {
		t.Fatal("an owner decided a statistic")
	}
	role := commissioner.RawCommand{League: "native.test", Command: "commish.role.grant",
		Args:   commissioner.RawArgs{People: &commissioner.RawPeople{Principal: "cc", Role: governance.CoCommissioner}},
		Reason: "Appoint the second commissioner", Expected: revision(t, f.s), Request: "cc", Effective: "now", Visibility: "private"}
	runScoreCommand(t, f.s, role, "creator")
	if _, err := f.s.ExecuteCommissioner(t.Context(), approve(item, "stale", 1), "cc", nil); err == nil {
		t.Fatal("a stale league version was accepted")
	}
	made := approve(item, "approve-cc", revision(t, f.s))
	receipt := runScoreCommand(t, f.s, made, "cc")
	conflicting := approve(item, "approve-cc", receipt.Revision)
	conflicting.Args.Stat = &commissioner.RawStatChange{Item: strings.Replace(item, ".1.", ".9.", 1)}
	if _, err := f.s.ExecuteCommissioner(t.Context(), conflicting, "cc", nil); err == nil {
		t.Fatal("a request key was reused for a different change")
	}
	again := approve(item, "again", revision(t, f.s))
	if _, err := f.s.ExecuteCommissioner(t.Context(), again, "cc", nil); err == nil ||
		!strings.Contains(err.Error(), "already counts") {
		t.Fatalf("an item already approved was approved twice: %v", err)
	}
	f.revise(f.a, 2, 40000)
	stale := approve(f.queue(t).Open[0].ID, "newer", revision(t, f.s))
	stale.Args.Stat.Item = item
	stale.Request = "stale-item"
	if _, err := f.s.ExecuteCommissioner(t.Context(), stale, "cc", nil); err == nil {
		t.Fatal("an item the feed superseded was accepted")
	}
	unsourced := statCommand(commissioner.StatCorrect, commissioner.RawStatChange{Player: f.starter(f.a), Season: 2026,
		Week: 2, Stat: "tackle", Value: pointsDelta(50000)}, revision(t, f.s), "unsourced")
	if _, err := f.s.ExecuteCommissioner(t.Context(), unsourced, "cc", nil); err == nil {
		t.Fatal("a correction without a source was accepted")
	}
	noChange := statCommand(commissioner.StatCorrect, commissioner.RawStatChange{Player: f.starter(f.a), Season: 2026,
		Week: 2, Stat: "tackle", Value: pointsDelta(30000), Source: "Gamebook"}, revision(t, f.s), "same")
	if _, err := f.s.ExecuteCommissioner(t.Context(), noChange, "cc", nil); err == nil {
		t.Fatal("a correction that changes nothing was accepted")
	}
	unknown := statCommand(commissioner.StatCorrect, commissioner.RawStatChange{Player: f.starter(f.a), Season: 2026,
		Week: 2, Stat: "not_a_stat", Value: pointsDelta(1), Source: "Gamebook"}, revision(t, f.s), "unknown")
	if _, err := f.s.ExecuteCommissioner(t.Context(), unknown, "cc", nil); err == nil {
		t.Fatal("a statistic the league does not score was accepted")
	}
}

func TestStatQueueReadsTheArchiveTheSameWayAsASinglePlayer(t *testing.T) {
	t.Parallel()
	s, _ := statStore(t, filepath.Join(t.TempDir(), "archive.db"))
	if _, err := s.LinkPlayerIDs(t.Context(), "dynastyprocess", []PlayerIDLink{
		{IDType: "gsis", IDValue: "00-0023459", PlayerID: "0099"}}); err != nil {
		t.Fatal(err)
	}
	// The feed said 244, 245 (refetched unchanged), absent the player, then went back to 244.
	putStatBody(t, s, statBody(t, "244", true), "", t0())
	putStatBody(t, s, statBody(t, "245", true), "", t0().Add(time.Hour))
	putStatBody(t, s, statBody(t, "245", true), "", t0().Add(2*time.Hour))
	putStatBody(t, s, statBody(t, "245", false), "", t0().Add(3*time.Hour))
	putStatBody(t, s, statBody(t, "244", true), "", t0().Add(4*time.Hour))
	weeks, err := seasonStatFeed{store: s, season: 2025}.Weeks(t.Context())
	if err != nil || len(weeks) != 1 || weeks[0].Player != "0099" || weeks[0].Week != 1 {
		t.Fatalf("matched weeks: %+v %v", weeks, err)
	}
	evidence, err := source.ReadNativeEvidence(t.Context(), s, s, source.RawNativeScope{Player: "0099", Season: 2025, Week: 1})
	if err != nil {
		t.Fatal(err)
	}
	got, want := weeks[0].Evidence.Lines, evidence.Lines
	if len(got) != 3 || len(want) != 3 {
		t.Fatalf("want 244, 245, 244 again; got %d and %d versions", len(got), len(want))
	}
	for i := range got {
		if got[i].Seq != want[i].Seq || got[i].Digest != want[i].Digest || got[i].Held.SHA256 != want[i].Held.SHA256 {
			t.Fatalf("the season scan and the single-player read disagree at %d", i)
		}
	}
	if got[0].Digest != got[2].Digest || got[0].Digest == got[1].Digest {
		t.Fatal("going back to 244 is the same line as the first 244")
	}
	if weeks[0].Evidence.GSIS != "00-0023459" || len(weeks[0].Evidence.Missing) != 1 {
		t.Fatalf("evidence identity and missing play-by-play: %+v", weeks[0].Evidence)
	}
}

func TestStatQueueOnASeasonNoFileCoversIsEmpty(t *testing.T) {
	t.Parallel()
	s, _ := statStore(t, filepath.Join(t.TempDir(), "none.db"))
	weeks, err := seasonStatFeed{store: s, season: 2025}.Weeks(t.Context())
	if err != nil || len(weeks) != 0 {
		t.Fatalf("no held files: %+v %v", weeks, err)
	}
	if _, err := (seasonStatFeed{store: s, season: 1999}).Weeks(t.Context()); err != nil {
		t.Fatalf("a season before the feed begins is empty, not an error: %v", err)
	}
}

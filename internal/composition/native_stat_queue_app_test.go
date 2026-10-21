package composition

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

func readStatQueue(t *testing.T, f *legalityFixture) commissioner.StatQueue {
	t.Helper()
	out, err := newCommissionerAPI(f.app).TargetStatQueue(legalityLeague, 2026)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func statDecide(t *testing.T, f *legalityFixture, command, request string, change commissioner.RawStatChange) commissioner.Receipt {
	t.Helper()
	return directoryCommand(t, f.app, command, request, commissioner.RawArgs{Stat: &change}, "now")
}

// assertQueue reads the queue after every step, then again on a database opened from the file.
func assertQueue(t *testing.T, f *legalityFixture, open, decided int, newest string) commissioner.StatQueue {
	t.Helper()
	var first commissioner.StatQueue
	for pass := 0; pass < 2; pass++ {
		q := readStatQueue(t, f)
		if len(q.Open) != open || len(q.Decided) != decided || (decided != 0 && q.Decided[0].Status != newest) {
			t.Fatalf("queue (reopened %v): %+v", pass == 1, q)
		}
		if pass == 0 {
			first = q
			f.reopen(t)
		} else if !reflect.DeepEqual(q, first) {
			t.Fatal("the queue changed across a database reopen")
		}
	}
	return first
}

func TestNativeStatQueueCM031ApprovalGovernsScoreAndQueue(t *testing.T) {
	t.Parallel()
	f := scoreApp(t)
	holdNativeStats(t, f, true, "LB", "3", 2)
	for pass := 0; pass < 2; pass++ {
		assertNativePoints(t, f, 15000)
		q := assertQueue(t, f, 1, pass, map[bool]string{true: "Approval undone"}[pass == 1])
		item := q.Open[0]
		if item.Points == nil || *item.Points != 30000 || item.Player != "1000" || item.Week != 1 ||
			!strings.HasPrefix(item.Words, "player 1000, week 1: tackle 1 → 3, +3 points, ") {
			t.Fatalf("pass %d the item: %+v", pass, item)
		}
		receipt := statDecide(t, f, commissioner.StatApprove, fmt.Sprint("approve", pass),
			commissioner.RawStatChange{Item: item.ID})
		assertNativePoints(t, f, 45000)
		last := readNativeScore(t, f).Changes
		approvals := 0
		for _, c := range last {
			if c.Entry == receipt.Entry {
				approvals++
			}
		}
		if got := last[len(last)-1]; got.Entry != receipt.Entry || got.Status != "Applied" || approvals != 1 {
			t.Fatalf("pass %d provenance: %+v", pass, last)
		}
		assertQueue(t, f, 0, pass+1, "Approved")
		assertNativePoints(t, f, 45000)
		f.revert(t, receipt)
		assertNativePoints(t, f, 15000)
		f.reopen(t)
		assertNativePoints(t, f, 15000)
	}
}

func TestNativeStatQueueCM031CorrectionGovernsScoreAndQueue(t *testing.T) {
	t.Parallel()
	f := scoreApp(t)
	for pass := 0; pass < 2; pass++ {
		assertNativePoints(t, f, 15000)
		value := leaguerules.Decimal(40000)
		receipt := statDecide(t, f, commissioner.StatCorrect, fmt.Sprint("correct", pass), commissioner.RawStatChange{
			Player: "1000", Season: 2026, Week: 1, Stat: "tackle", Value: &value, Source: "League office, Monday call"})
		assertNativePoints(t, f, 60000)
		out := readNativeScore(t, f)
		if got := out.Changes[len(out.Changes)-1]; got.Entry != receipt.Entry || got.Status != "Applied" ||
			!strings.Contains(got.What, "League office, Monday call") {
			t.Fatalf("pass %d provenance: %+v", pass, out.Changes)
		}
		q := assertQueue(t, f, 0, pass+1, "Entered")
		if q.Decided[0].Kind != "correct" || !strings.Contains(q.Decided[0].Words, "tackle set to 4") {
			t.Fatalf("pass %d the decision reads wrong: %+v", pass, q.Decided[0])
		}
		assertNativePoints(t, f, 60000)
		f.revert(t, receipt)
		assertNativePoints(t, f, 15000)
		f.reopen(t)
		assertNativePoints(t, f, 15000)
	}
}

func TestNativeStatFeedRevisionNeverMovesTheScoreOnItsOwn(t *testing.T) {
	t.Parallel()
	f := scoreApp(t)
	holdNativeStats(t, f, true, "LB", "3", 2)
	holdNativeStats(t, f, true, "DT", "3", 3)
	for range 2 {
		assertNativePoints(t, f, 15000)
		out := readNativeScore(t, f)
		if out.Position != leaguerules.LB || out.PositionWhy != "The position reported with the NFL statistics" {
			t.Fatalf("a feed revision changed the position: %+v", out)
		}
		f.reopen(t)
	}
	q := readStatQueue(t, f)
	if len(q.Open) != 1 || !strings.Contains(q.Open[0].Words, "position LB → DT; tackle 1 → 3") {
		t.Fatalf("both changes belong to one item: %+v", q)
	}
}

func TestNativeStatRejectedItemSurvivesTheFeedReturningToWhatWasCounted(t *testing.T) {
	t.Parallel()
	f := scoreApp(t)
	holdNativeStats(t, f, true, "LB", "3", 2)
	item := readStatQueue(t, f).Open[0]
	statDecide(t, f, commissioner.StatReject, "reject", commissioner.RawStatChange{Item: item.ID})
	assertQueue(t, f, 0, 1, "Rejected")
	// The feed corrects itself back: the body is the first one again.
	holdNativeStats(t, f, true, "LB", "1", 3)
	assertQueue(t, f, 0, 1, "Rejected")
	assertNativePoints(t, f, 15000)
	// And changes once more to the same line the league turned down: it is a new revision, so it asks again.
	holdNativeStats(t, f, true, "LB", "3", 4)
	again := assertQueue(t, f, 1, 1, "Rejected")
	if again.Open[0].ID == item.ID {
		t.Fatal("a rejected revision came back under its old name")
	}
	assertNativePoints(t, f, 15000)
}

func TestNativeStatPreviewStatesPointsAndFlipBeforeItFires(t *testing.T) {
	t.Parallel()
	f := scoreApp(t)
	holdNativeStats(t, f, true, "LB", "3", 2)
	item := readStatQueue(t, f).Open[0]
	houses := readDirectoryTest(t, f.app)
	body, err := json.Marshal(commissioner.RawCommand{League: legalityLeague, Command: commissioner.StatApprove,
		Args: commissioner.RawArgs{Stat: &commissioner.RawStatChange{Item: item.ID}}, Request: "preview",
		Expected: houses.Revision, Effective: "now", Visibility: "league"})
	if err != nil {
		t.Fatal(err)
	}
	preview, err := newCommissionerAPI(f.app).TargetNativeScorePreview(string(body))
	if err != nil || !strings.Contains(preview.Words, "tackle 1 → 3, +3 points") ||
		preview.Flip.Status != commissioner.FlipUnknown || len(preview.Warnings) != 1 {
		t.Fatalf("preview: %+v %v", preview, err)
	}
	assertNativePoints(t, f, 15000)
	if q := readStatQueue(t, f); len(q.Open) != 1 || len(q.Decided) != 0 {
		t.Fatalf("a preview decided something: %+v", q)
	}
}

func TestNativeStatQueueScopeAndStartingSnapshot(t *testing.T) {
	t.Parallel()
	a := nativeSeedTestApp(t)
	createNativeSeedLeague(t, a, commissioner.PresetLegacy)
	q, err := newCommissionerAPI(a).TargetStatQueue(legalityLeague, 2026)
	if err != nil || q.Words != "Record the starting snapshot first" || len(q.Open) != 0 || len(q.Decided) != 0 {
		t.Fatalf("an unseeded league queued: %+v %v", q, err)
	}
	for _, scope := range []struct {
		league string
		season int
	}{{"native.", 2026}, {"native.missing", 2026}, {legalityLeague, 1919}} {
		if _, err := newCommissionerAPI(a).TargetStatQueue(scope.league, scope.season); err == nil {
			t.Fatalf("invalid scope accepted: %+v", scope)
		}
	}
}

func TestNativeStatQueueWireFixtureMatchesGo(t *testing.T) {
	t.Parallel()
	f := scoreApp(t)
	at := func(hour int) {
		f.app.core.History.SetClock(func() time.Time { return time.Date(2026, 10, 13, hour, 0, 0, 0, time.UTC) })
	}
	readings := []commissioner.StatQueue{readStatQueue(t, f)}
	holdNativeStats(t, f, true, "LB", "3", 2)
	readings = append(readings, readStatQueue(t, f))
	at(13)
	approval := statDecide(t, f, commissioner.StatApprove, "wire.approve",
		commissioner.RawStatChange{Item: readings[1].Open[0].ID})
	readings = append(readings, readStatQueue(t, f))
	at(14)
	f.revert(t, approval)
	holdNativeStats(t, f, true, "DT", "4", 3)
	at(15)
	statDecide(t, f, commissioner.StatReject, "wire.reject", commissioner.RawStatChange{Item: readStatQueue(t, f).Open[0].ID})
	at(16)
	value := leaguerules.Decimal(20000)
	statDecide(t, f, commissioner.StatCorrect, "wire.correct", commissioner.RawStatChange{Player: "1000", Season: 2026,
		Week: 1, Stat: "tackle", Value: &value, Source: "League office, Monday call"})
	readings = append(readings, readStatQueue(t, f))
	body, err := json.MarshalIndent(readings, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	body = append(body, '\n')
	const path = "internal/commissioner/testdata/stat-queue-wire.json"
	if os.Getenv("TWR_WRITE_WIRE") == "1" {
		if err := os.WriteFile(path, body, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	held, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(held, body) {
		t.Fatal("statistics queue wire fixture is stale")
	}
}

package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
)

// previewOf asks for the preview the tray would show for a tray command, over the same binding the screen
// uses, and returns it as it travels.
func (f *draftFixture) previewOf(t *testing.T, command string, act commissioner.RawDraftAct,
	reason string) commissioner.Preview {
	t.Helper()
	rev := f.revision(t)
	d := commissioner.RawDraftArgs{Season: draftSeason}
	if command != commissioner.DraftArchive {
		d.Act = &act
	}
	body, err := json.Marshal(commissioner.RawCommand{League: draftLeague, Command: command,
		Args: commissioner.RawArgs{Draft: &d}, Reason: reason, Expected: rev, Request: fmt.Sprintf("preview.%d", rev),
		Effective: "now", Visibility: "league"})
	if err != nil {
		t.Fatal(err)
	}
	preview, err := f.app.TargetCommissionerPreview(string(body))
	if err != nil {
		t.Fatalf("preview %s: %v", command, err)
	}
	return preview
}

// TestTheDraftRoomWireIsWhatGoSends writes the readings and previews the Draft room screen parses: the
// draft before its order is known, running with a pick made and the clock on a franchise, the same draft
// with its clock paused, and the draft closed; and one preview for every tray command.
func TestTheDraftRoomWireIsWhatGoSends(t *testing.T) {
	t.Parallel()
	f := draftApp(t)
	// The running app names rookies in previews from the same facts the draft reading uses.
	f.app.history.SetLineupEnvironment(appLineupEnv{app: f.app})
	waiting := f.read(t, "franchise.01")
	f.scheduled(t, 3, 2, 1)
	f.seated(t, 1, 3)
	f.now = draftStartAt().Add(time.Hour)
	f.asOwner(3)
	f.pick(t, "franchise.03", "5001")
	f.actAs("creator")
	point := f.checkpoint(t)
	previews := map[string]commissioner.Preview{
		"skip":   f.previewOf(t, commissioner.DraftSkip, pickOf(1, 2), "Skip a missed pick"),
		"assign": f.previewOf(t, commissioner.DraftAssign, withPlayer(pickOf(1, 2), "5002"), "Choose for them"),
		"insert": f.previewOf(t, commissioner.DraftInsert, commissioner.RawDraftAct{Round: 1, Original: draftTeam(2),
			Player: "5003", Franchise: draftTeam(5)}, "Add an extra pick"),
		"change": f.previewOf(t, commissioner.DraftChange, commissioner.RawDraftAct{Round: 1, Original: draftTeam(3),
			Player: "5004", Replaces: "5001"}, "Fix a mistake"),
		"pause":   f.previewOf(t, commissioner.DraftTimer, commissioner.RawDraftAct{Timer: commissioner.TimerPause}, "Hold"),
		"set":     f.previewOf(t, commissioner.DraftTimer, commissioner.RawDraftAct{Timer: commissioner.TimerSet, Seconds: 1800}, "Speed up"),
		"restore": f.previewOf(t, commissioner.DraftRevert, commissioner.RawDraftAct{To: point}, "Back up"),
		"close":   f.previewOf(t, commissioner.DraftArchive, commissioner.RawDraftAct{}, "Finish early"),
	}
	for name, preview := range previews {
		// The tray shows what a change affects only for a count Go knows; the warning travels in those words.
		if preview.Affects.Count == nil || preview.Affects.Words == "" {
			t.Fatalf("the %s preview carries no warning the tray can show: %+v", name, preview.Affects)
		}
	}
	for _, say := range []struct{ preview, got, want string }{
		{"pause", previews["pause"].Fields[0].Before, "Running, 6 hours left"},
		{"restore", previews["restore"].Fields[0].Before, "1 pick made"},
		{"set", previews["set"].Fields[0].After, "30 minutes left for the pick on the clock"},
	} {
		if say.got != say.want {
			t.Fatalf("the %s preview says %q, want %q", say.preview, say.got, say.want)
		}
	}
	running := f.read(t, "franchise.01")
	if words := running.Checkpoints[0].Words; !strings.Contains(words, "Picked Player, 5001 for round 1") {
		t.Fatalf("a checkpoint names a rookie by his number: %q", words)
	}
	if after := previews["restore"].Fields[0].After; !strings.Contains(after, "Picked Player, 5001") {
		t.Fatalf("the restore preview names a rookie by his number: %q", after)
	}
	for _, line := range running.Rules {
		if strings.Contains(line.Words, "franchise.") {
			t.Fatalf("the rule %q names a franchise by its code: %q", line.Label, line.Words)
		}
	}
	f.tray(t, commissioner.DraftTimer, commissioner.RawDraftAct{Timer: commissioner.TimerPause})
	paused := f.read(t, "franchise.01")
	f.tray(t, commissioner.DraftTimer, commissioner.RawDraftAct{Timer: commissioner.TimerResume})
	f.draftEnds()
	f.app.processNativeDraftsDue(t.Context())
	closed := f.read(t, "franchise.01")
	if closed.Closed == nil || !closed.Closed.Automatic {
		t.Fatalf("the draft did not close at its last pick: %+v", closed.Closed)
	}
	body, err := json.MarshalIndent(map[string]any{"waiting": waiting, "running": running, "paused": paused, "closed": closed,
		"previews": previews, "checkpoint": point}, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	checkWire(t, "frontend/src/app/draftroom/draft.wire.testdata.json", body, "draft room")
}

// seasonOfRoom is the season the Draft room opens on when it is not told one.
func (f *draftFixture) seasonOfRoom(t *testing.T) int {
	t.Helper()
	reading, err := f.app.TargetNativeDraft(commissioner.RawNativeDraftScope{League: draftLeague,
		Franchise: "franchise.01", At: f.now})
	if err != nil {
		t.Fatal(err)
	}
	return reading.Season
}

func TestTheDraftRoomOpensOnTheDraftThatIsNext(t *testing.T) {
	t.Parallel()
	f := draftApp(t)
	f.scheduled(t, 3, 2, 1)
	f.seated(t, 3)
	f.now = draftStartAt().Add(time.Hour)
	f.asOwner(3)
	f.pick(t, "franchise.03", "5001")
	f.actAs("creator")
	point := f.checkpoint(t)
	for pass, when := range []string{"live", "after reopening the database"} {
		if pass == 1 {
			f.reopen(t)
		}
		if got := f.seasonOfRoom(t); got != draftSeason {
			t.Fatalf("%s: the draft being run: season %d", when, got)
		}
		f.draftEnds()
		if got := f.seasonOfRoom(t); got != draftSeason {
			t.Fatalf("%s: a finished draft that is not yet closed: season %d", when, got)
		}
		f.app.processNativeDraftsDue(t.Context())
		if got := f.seasonOfRoom(t); got != draftSeason+1 {
			t.Fatalf("%s: after the close the next draft is the one the close issued picks for: season %d", when, got)
		}
		f.restore(t, point)
		if got := f.seasonOfRoom(t); got != draftSeason {
			t.Fatalf("%s: restoring the draft did not bring it back as the next one: season %d", when, got)
		}
	}
}

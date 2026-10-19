package main

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

func previewDraft(t *testing.T, f *draftFixture, command string, args commissioner.RawArgs) commissioner.Receipt {
	t.Helper()
	raw := commissioner.RawCommand{League: draftLeague, Command: command,
		Args: args, Expected: f.revision(t), Request: fmt.Sprintf("preview-proof.%d", f.revision(t)),
		Reason: "Make the agreed draft change", Effective: "now", Visibility: "league"}
	preview := mustPreview(t, f.app, raw)
	if raw.Args.Draft != nil && !strings.Contains(preview.Affects.Words, "32 franchises") {
		t.Fatalf("draft affected count missing: %+v", preview.Affects)
	}
	return previewAndExecute(t, f.app, raw)
}

func TestEveryBuiltDraftCommandHasARealPreview(t *testing.T) {
	t.Parallel()
	f := draftApp(t)
	for _, tc := range []struct {
		command string
		args    commissioner.RawDraftArgs
	}{
		{commissioner.DraftOrderAuto, commissioner.RawDraftArgs{Season: draftSeason, Source: leaguerules.WorstStandingsFirst}},
		{commissioner.DraftOrderPattern, commissioner.RawDraftArgs{Pattern: leaguerules.Linear}},
		{commissioner.DraftOrderTiebreak, commissioner.RawDraftArgs{Tie: commissioner.TieByFranchiseOrder}},
		{commissioner.DraftOrderSet, commissioner.RawDraftArgs{Season: draftSeason, Order: draftOrder(3, 2, 1)}},
		{commissioner.DraftStart, commissioner.RawDraftArgs{Season: draftSeason, At: draftStartAt()}},
	} {
		previewDraft(t, f, tc.command, commissioner.RawArgs{Draft: &tc.args})
	}
	f.now = draftStartAt().Add(time.Hour)
	point := f.checkpoint(t)
	apply := func(command string, act commissioner.RawDraftAct) {
		t.Helper()
		previewDraft(t, f, command, commissioner.RawArgs{Draft: &commissioner.RawDraftArgs{Season: draftSeason, Act: &act}})
	}
	apply(commissioner.DraftSkip, pickOf(1, 3))
	apply(commissioner.DraftAssign, withPlayer(pickOf(1, 3), "5001"))
	inserted := withPlayer(pickOf(1, 3), "5002")
	inserted.Franchise = draftTeam(7)
	apply(commissioner.DraftInsert, inserted)
	changed := withPlayer(pickOf(1, 3), "5003")
	changed.Replaces = "5001"
	apply(commissioner.DraftChange, changed)
	apply(commissioner.DraftTimer, commissioner.RawDraftAct{Timer: commissioner.TimerPause})
	apply(commissioner.DraftTimer, commissioner.RawDraftAct{Timer: commissioner.TimerSet, Seconds: 1800})
	apply(commissioner.DraftTimer, commissioner.RawDraftAct{Timer: commissioner.TimerResume})
	apply(commissioner.DraftRevert, commissioner.RawDraftAct{To: point})
	previewDraft(t, f, commissioner.DraftArchive, commissioner.RawArgs{Draft: &commissioner.RawDraftArgs{Season: draftSeason}})
}

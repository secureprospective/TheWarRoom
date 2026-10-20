package main

import (
	"context"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

func TestResearchColdOpenReadsArchiveBeforeFetch(t *testing.T) {
	t.Parallel()
	in := realResearch(t)
	a := targetTestApp(t)
	a.lookup, a.hasLookup = in.lookup, true
	if _, err := a.league.Replace(a.ctx, in.mirror); err != nil {
		t.Fatal(err)
	}
	at := time.Date(2026, 10, 11, 10, 0, 0, 0, time.UTC)
	recordSeasonArchive(t, a, "playerScores", "YTD", string(researchSample(t, "playerScores_WYTD.json")), at)
	recordSeasonArchive(t, a, "playerScores", "1", string(researchSample(t, "playerScores_W01.json")), at)
	recordSeasonBareArchive(t, a, "injuries", string(researchSample(t, "injuries.json")), at)
	recordSeasonArchiveForm(t, a, "playerProfile", "", string(researchSample(t, "playerProfile.json")), at, false)
	// Profiles require the exact P member; this helper's URL has no P, so it must not answer.
	a.refreshMu.Lock()
	defer a.refreshMu.Unlock()
	type answer struct {
		board ResearchBoard
		err   error
	}
	read := make(chan answer, 1)
	go func() { board, err := a.TargetResearch(RawResearchScope{}); read <- answer{board, err} }()
	var out ResearchBoard
	select {
	case result := <-read:
		if result.err != nil {
			t.Fatal(result.err)
		}
		out = result.board
	case <-time.After(15 * time.Second):
		t.Fatal("opening blocked behind launch refresh")
	}
	r := reportByID(t, out.Board, "M-037")
	if len(r.Rows) == 0 || !strings.Contains(r.Note, "Archived copy") || !strings.Contains(r.AsOf, reportTime(at, time.Local)) {
		t.Fatal(r.Note, r.AsOf, len(r.Rows))
	}
	if len(a.seasonInjuries.value) == 0 {
		t.Fatal("cold injury archive not served")
	}
	if !a.research.feeds[researchKey("playerScores", "YTD")].fetchedAt.Equal(at) {
		t.Fatal("archive time replaced")
	}
	if len(a.research.news["16162"].value) != 0 {
		t.Fatal("profile URL without exact player accepted")
	}
	recordSeasonArchive(t, a, "playerProfile&P=16162", "", string(researchSample(t, "playerProfile.json")), at)
	// An archive lookup that found nothing is not repeated on every read: the refresh fills it.
	if _, err := a.TargetResearch(RawResearchScope{}); err != nil {
		t.Fatal(err)
	}
	if len(a.research.news["16162"].value) != 0 || !a.research.archiveRead["news:16162"] {
		t.Fatal("archive looked up again on a second read")
	}
	a.research.archiveRead = nil // the next launch looks again
	if _, err := a.TargetResearch(RawResearchScope{}); err != nil {
		t.Fatal(err)
	}
	if len(a.research.news["16162"].value) != 1 {
		t.Fatal("cold archived news not served")
	}
	in.mirror.Players = slices.DeleteFunc(slices.Clone(in.mirror.Players), func(p state.MirrorPlayer) bool { return p.MFLID == "16162" })
	if _, err := a.league.Replace(a.ctx, in.mirror); err != nil {
		t.Fatal(err)
	}
	a.research.news, a.research.archiveRead = nil, nil
	player, err := a.TargetResearchPlayer("16162")
	if err != nil || player.Owner != "Free agent" || len(player.Articles) != 1 {
		t.Fatal("free agent’s archived news was not served", player.Owner, err)
	}
}

func TestResearchStatusWhileWaitingForLaunchRefresh(t *testing.T) {
	t.Parallel()
	a := targetTestApp(t)
	a.seasonPause = func(context.Context, time.Duration) error { return context.Canceled }
	a.refreshMu.Lock()
	done := make(chan error, 1)
	go func() { done <- a.TargetResearchRefresh(false) }()
	deadline := time.Now().Add(time.Second)
	for a.TargetResearchStatus() != "Waiting for the MFL refresh to finish" && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	status := a.TargetResearchStatus()
	a.refreshMu.Unlock()
	if err := <-done; err == nil {
		t.Fatal("cancelled fixture refresh passed")
	}
	if status != "Waiting for the MFL refresh to finish" {
		t.Fatal(status)
	}
	if a.TargetResearchStatus() != "" {
		t.Fatal("status retained after failure")
	}
}

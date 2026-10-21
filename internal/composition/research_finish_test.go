package composition

import (
	"context"
	"runtime"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	researchapp "github.com/secureprospective/TheWarRoom/internal/app/research"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/nflschedule"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/researchfeeds"

	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

func TestResearchColdOpenReadsArchiveBeforeFetch(t *testing.T) {
	t.Parallel()
	in := realResearch(t)
	a := targetTestApp(t)
	a.core.Lookup, a.core.HasLookup = in.lookup, true
	if _, err := a.core.League.Replace(a.core.Ctx, in.mirror); err != nil {
		t.Fatal(err)
	}
	at := time.Date(2026, 10, 11, 10, 0, 0, 0, time.UTC)
	recordSeasonArchive(t, a, "playerScores", "YTD", string(researchSample(t, "playerScores_WYTD.json")), at)
	recordSeasonArchive(t, a, "playerScores", "1", string(researchSample(t, "playerScores_W01.json")), at)
	recordSeasonBareArchive(t, a, "injuries", string(researchSample(t, "injuries.json")), at)
	recordSeasonArchiveForm(t, a, "playerProfile", "", string(researchSample(t, "playerProfile.json")), at, false)
	// Profiles require the exact P member; this helper's URL has no P, so it must not answer.
	a.core.RefreshMu.Lock()
	defer a.core.RefreshMu.Unlock()
	type answer struct {
		board ResearchBoard
		err   error
	}
	read := make(chan answer, 1)
	go func() { board, err := researchAPI(a).TargetResearch(RawResearchScope{}); read <- answer{board, err} }()
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
	r := reportByID(t, wireBoard(out.Board), "M-037")
	if len(r.Rows) == 0 || !strings.Contains(r.Note, "Archived copy") || !strings.Contains(r.AsOf, reportTime(at, time.Local)) {
		t.Fatal(r.Note, r.AsOf, len(r.Rows))
	}
	if len(a.core.SeasonInjuries.Value) == 0 {
		t.Fatal("cold injury archive not served")
	}
	if !a.snapshotResearch().feeds[researchKey("playerScores", "YTD")].FetchedAt.Equal(at) {
		t.Fatal("archive time replaced")
	}
	if len(a.snapshotResearch().news["16162"].Value) != 0 {
		t.Fatal("profile URL without exact player accepted")
	}
	recordSeasonArchive(t, a, "playerProfile&P=16162", "", string(researchSample(t, "playerProfile.json")), at)
	// An archive lookup that found nothing is not repeated on every read: the refresh fills it.
	if _, err := researchAPI(a).TargetResearch(RawResearchScope{}); err != nil {
		t.Fatal(err)
	}
	if len(a.snapshotResearch().news["16162"].Value) != 0 || !a.snapshotResearch().archiveRead["news:16162"] {
		t.Fatal("archive looked up again on a second read")
	}
	editResearch(a, func(value *researchHeld) { value.archiveRead = nil }) // the next launch looks again
	if _, err := researchAPI(a).TargetResearch(RawResearchScope{}); err != nil {
		t.Fatal(err)
	}
	if len(a.snapshotResearch().news["16162"].Value) != 1 {
		t.Fatal("cold archived news not served")
	}
	in.mirror.Players = slices.DeleteFunc(slices.Clone(in.mirror.Players), func(p state.MirrorPlayer) bool { return p.MFLID == "16162" })
	if _, err := a.core.League.Replace(a.core.Ctx, in.mirror); err != nil {
		t.Fatal(err)
	}
	editResearch(a, func(value *researchHeld) { value.news, value.archiveRead = nil, nil })
	player, err := researchAPI(a).TargetResearchPlayer("16162")
	if err != nil || player.Owner != "Free agent" || len(player.Articles) != 1 {
		t.Fatal("free agent’s archived news was not served", player.Owner, err)
	}
}

func TestResearchStatusWhileWaitingForLaunchRefresh(t *testing.T) {
	t.Parallel()
	a := targetTestApp(t)
	a.core.SeasonPause = func(context.Context, time.Duration) error { return context.Canceled }
	a.core.RefreshMu.Lock()
	done := make(chan error, 1)
	go func() { done <- researchAPI(a).TargetResearchRefresh(false) }()
	deadline := time.Now().Add(time.Second)
	for researchAPI(a).TargetResearchStatus() != "Waiting for the MFL refresh to finish" && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	status := researchAPI(a).TargetResearchStatus()
	a.core.RefreshMu.Unlock()
	if err := <-done; err == nil {
		t.Fatal("cancelled fixture refresh passed")
	}
	if status != "Waiting for the MFL refresh to finish" {
		t.Fatal(status)
	}
	if researchAPI(a).TargetResearchStatus() != "" {
		t.Fatal("status retained after failure")
	}
}

func TestResearchSnapshotDetachesMutableIndexes(t *testing.T) {
	t.Parallel()
	a := targetTestApp(t)
	fixture := realResearch(t)
	a.publishResearch(fixture.held)
	editResearch(a, func(value *researchHeld) { value.links["gsis:fixture"] = "16162" })
	a.now.schedules = map[int]heldSeasonFeed[[]nflschedule.RawWeek]{1: {FetchedAt: fixture.now}}
	in := a.heldResearchInput()
	a.core.SeasonMu.Lock()
	editResearch(a, func(value *researchHeld) {
		value.feeds[researchKey("playerScores", "YTD")] = heldSeasonFeed[[]researchfeeds.Feed]{}
	})
	editResearch(a, func(value *researchHeld) { delete(value.news, "16162") })
	editResearch(a, func(value *researchHeld) { value.links["gsis:fixture"] = "0042" })
	delete(a.now.schedules, 1)
	a.core.SeasonMu.Unlock()
	if len(in.held.feeds[researchKey("playerScores", "YTD")].Value) == 0 || len(in.held.news["16162"].Value) == 0 || in.held.links["gsis:fixture"] != "16162" || in.schedules[1].FetchedAt.IsZero() {
		t.Fatal("reading retained a mutable root index")
	}
	// Construction must not reacquire App's lock or inspect newly published state.
	a.core.SeasonMu.Lock()
	done := make(chan ResearchBoard, 1)
	go func() { done <- buildResearch(in, RawResearchScope{}) }()
	select {
	case board := <-done:
		a.core.SeasonMu.Unlock()
		if len(reportByID(t, wireBoard(board.Board), "M-037").Rows) == 0 {
			t.Fatal("detached scores lost")
		}
	case <-time.After(5 * time.Second):
		a.core.SeasonMu.Unlock()
		t.Fatal("construction waited on the root season lock")
	}
}

func TestResearchConcurrentNewsRefreshAndRead(t *testing.T) {
	t.Parallel()
	a, f, refreshed := blockedResearchNews(t)
	defer close(f.release)
	reading := make(chan struct{}, 1)
	readDone := make(chan error, 1)
	go func() {
		for i := 0; i < 40; i++ {
			_, err := researchAPI(a).TargetResearch(RawResearchScope{})
			if err != nil {
				readDone <- err
				return
			}
			if i == 0 {
				reading <- struct{}{}
			}
		}
		readDone <- nil
	}()
	select {
	case <-reading:
	case <-time.After(5 * time.Second):
		t.Fatal("Research read blocked behind news refresh")
	}
	f.release <- struct{}{}
	if err := <-refreshed; err != nil {
		t.Fatal(err)
	}
	if err := <-readDone; err != nil {
		t.Fatal(err)
	}
	in := a.heldResearchInput()
	if in.held.news["0042"].FetchedAt.IsZero() {
		t.Fatal("completed refresh was not published")
	}
}

func TestResearchJointCaptureKeepsSeasonRevision(t *testing.T) {
	t.Parallel()
	a := targetTestApp(t)
	publish := func(epoch int) {
		at := time.Unix(int64(epoch), 0)
		a.core.SeasonMu.Lock()
		a.researchHeld.Publish(researchapp.Values{News: map[string]researchapp.Feed[[]researchfeeds.News]{
			"0042": {FetchedAt: at},
		}})
		runtime.Gosched()
		a.core.SeasonInjuries.FetchedAt = at
		a.now.schedules = map[int]heldSeasonFeed[[]nflschedule.RawWeek]{1: {FetchedAt: at}}
		a.core.SeasonMu.Unlock()
	}
	publish(1)
	var workers sync.WaitGroup
	workers.Go(func() {
		for epoch := 2; epoch < 1000; epoch++ {
			publish(epoch)
		}
	})
	for range 1000 {
		in := a.heldResearchInput()
		at := in.held.news["0042"].FetchedAt
		if !at.Equal(in.injuries.FetchedAt) || !at.Equal(in.schedules[1].FetchedAt) {
			t.Error("joint capture mixed season revisions", at, in.injuries.FetchedAt, in.schedules[1].FetchedAt)
			break
		}
	}
	workers.Wait()
}

func TestResearchHolderLivesForExactlyOneApp(t *testing.T) {
	first, second := NewApp(), NewApp()
	if first.researchHeld == nil || second.researchHeld == nil || first.researchHeld == second.researchHeld {
		t.Fatal("private state must be constructed once per App")
	}
	bound := bindings(first)
	var api *researchapp.API
	for _, value := range bound {
		if candidate, ok := value.(*researchapp.API); ok {
			api = candidate
		}
	}
	first.researchStatus("held by this App")
	if api == nil || api.TargetResearchStatus() != "held by this App" {
		t.Fatal("runtime binding did not retain the App holder")
	}
	if researchAPI(second).TargetResearchStatus() != "" {
		t.Fatal("state leaked between Apps")
	}
}

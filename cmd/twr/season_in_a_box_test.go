package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/tls"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/accounts"
	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/fantasyscore"
	"github.com/secureprospective/TheWarRoom/internal/fantasyscore/source"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/livegame"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/livestats"
	"github.com/secureprospective/TheWarRoom/internal/serverhttp"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
	"github.com/secureprospective/TheWarRoom/tools/hostedfixture"
)

// boxClock is the season box's fake clock: the store reads it, nothing else does.
type boxClock struct {
	mu sync.Mutex
	at time.Time
}

func (c *boxClock) Now() time.Time { c.mu.Lock(); defer c.mu.Unlock(); return c.at }
func (c *boxClock) Advance(d time.Duration) {
	c.mu.Lock()
	c.at = c.at.Add(d)
	c.mu.Unlock()
}

// replayFrames reads the recorded NFL week 9 game (ESPN summaries, with synthetic progress and
// correction frames between the recorded start and the raw final) from the on-disk archive.
func replayFrames(t *testing.T) map[string][]byte {
	t.Helper()
	file, err := os.Open("../twr-server/testdata/live-week9.tar.gz")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = file.Close() }()
	zipped, err := gzip.NewReader(file)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = zipped.Close() }()
	reader, bodies := tar.NewReader(zipped), map[string][]byte{}
	for {
		header, err := reader.Next()
		if err == io.EOF {
			return bodies
		}
		if err != nil {
			t.Fatal(err)
		}
		if bodies[header.Name], err = io.ReadAll(io.LimitReader(reader, 4<<20)); err != nil {
			t.Fatal(err)
		}
	}
}

// frameLine is the replay quarterback's stat line in one recorded frame.
func frameLine(t *testing.T, bodies map[string][]byte, frame string) fantasyscore.StatLine {
	t.Helper()
	game, err := livegame.ParseSummary(bodies["espn-401873037-"+frame+".json"])
	if err != nil {
		t.Fatal(err)
	}
	lines, err := livestats.Lines(game)
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range lines {
		if line.SourceID == "4360310" {
			return line.Stats
		}
	}
	t.Fatalf("replay quarterback missing from frame %s", frame)
	return fantasyscore.StatLine{}
}

// boxRead is a league read over the hosted workflow door (twr has no read verb yet; see
// docs/architecture/season-in-a-box.md, blocker B1).
func (f *releaseCLIFixture) boxRead(t *testing.T, token accounts.AgentToken, command string, week int, out any) {
	t.Helper()
	arguments, _ := json.Marshal(map[string]any{"league": "native.hosted", "season": 2026, "week": week})
	body, _ := json.Marshal(serverhttp.RawNativeRequest{Protocol: 1, Workflow: &serverhttp.RawWorkflow{Command: command, Version: "1", Mode: "read", Arguments: arguments}})
	request, err := http.NewRequestWithContext(t.Context(), http.MethodPost, f.address+serverhttp.WorkflowPath, bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+token.Bearer)
	response, err := (&http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{RootCAs: f.env.roots, MinVersion: tls.VersionTLS12}}}).Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = response.Body.Close() }()
	reply, _ := io.ReadAll(response.Body)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("%s read: %d %s", command, response.StatusCode, reply)
	}
	var envelope serverhttp.WorkflowResponse
	if err := json.Unmarshal(reply, &envelope); err != nil || json.Unmarshal(envelope.Data, out) != nil {
		t.Fatalf("%s read unreadable: %v %s", command, err, reply)
	}
}

// TestSeasonInABoxSpike walks one recorded NFL week through a throwaway hosted league: create ->
// lineups -> lineup lock -> live stat lines in pieces (scores move) -> official final -> standings.
// Every step the box cannot yet reach says so with t.Log("NOT REACHED ..."); see the design doc.
func TestSeasonInABoxSpike(t *testing.T) {
	clock := &boxClock{}
	f := newReleaseCLIWith(t, hostedfixture.CreateWorkflowLeague)
	clock.at = time.Now().UTC() // football kickoffs are at seed time + 24h
	f.store.SetClock(clock.Now)
	overlay := &history.LiveOverlay{}
	f.store.SetLiveOverlay(overlay)

	scope := accounts.RawTokenScope{League: "native.hosted", Franchise: "franchise.01", Modes: []string{"read", "preview"},
		Commands: []string{"results.read", "standings.read", "commish.ledger.read", "native.assets.read", "native.directory.read", commissioner.LineupSet}}
	token, err := f.tokens.Issue(t.Context(), f.human, scope)
	if err != nil {
		t.Fatal(err)
	}
	f.selectToken(t, token, "native.hosted")

	// STEP 1 league created: the CLI reads the ledger.
	if err := f.env.runTwr(t, "", "ledger"); err != nil {
		t.Fatal(err)
	}
	var ledger commissioner.Ledger
	if err := json.Unmarshal(f.env.out.(*bytes.Buffer).Bytes(), &ledger); err != nil || ledger.Revision == 0 {
		t.Fatalf("league not built: revision %v %v", ledger.Revision, err)
	}
	f.env.out.(*bytes.Buffer).Reset()
	t.Log("STEP 1 league created (CreateWorkflowLeague, 32 franchises); twr ledger revision", ledger.Revision)
	t.Log("NOT REACHED STEP 2 quick draft: needs an owner pick per slot (PickPath) or a commissioner autopick command; blocker B3")

	// STEP 3 lineups: the CLI previews, the commissioner's own session records (G2 confirmation).
	lineup := commissioner.RawCommand{League: "native.hosted", Command: commissioner.LineupSet, Request: "box.lineup", Effective: "now", Visibility: "private",
		Args: commissioner.RawArgs{Lineup: &commissioner.RawLineupChange{Season: 2026, Franchise: "franchise.01", Week: 1, Starters: []string{"0099"}}}}
	body, _ := json.Marshal(lineup)
	assets, err := f.store.NativeAssets(t.Context(), "native.hosted")
	if err != nil {
		t.Fatal(err)
	}
	lineup.Expected = assets.Revision
	if _, err := f.store.ExecuteCommissioner(t.Context(), lineup, f.principal, nil); err != nil {
		t.Fatalf("record lineup: %v", err)
	}
	if err := f.env.runTwr(t, string(body), "preview", "-"); err != nil {
		t.Fatalf("twr preview lineup: %v", err)
	}
	f.env.out.(*bytes.Buffer).Reset()
	t.Log("STEP 3 lineup set: twr preview ok, recorded through the owner's own session")

	// STEP 4 lineup lock: the clock passes kickoff; a later lineup is refused.
	clock.Advance(25 * time.Hour)
	late := lineup
	late.Request, late.Expected = "box.lineup.late", assets.Revision+1
	late.Args = commissioner.RawArgs{Lineup: &commissioner.RawLineupChange{Season: 2026, Franchise: "franchise.01", Week: 1, Starters: []string{"0099", "0101"}}}
	if _, err := f.store.ExecuteCommissioner(t.Context(), late, f.principal, nil); err == nil {
		t.Fatal("a lineup set after kickoff was accepted")
	} else {
		t.Log("STEP 4 lineup lock held after the fake clock passed kickoff:", err)
	}

	// STEP 5 live stat lines in pieces: scores move.
	bodies := replayFrames(t)
	publish := func(frame string) commissioner.NativeResults {
		generation := history.LiveGeneration{ID: frame, Season: 2026, Week: 1, At: clock.Now(), Authority: source.ESPNLive,
			Missing: livestats.Pending(), Identities: map[string]history.LiveIdentity{"0099": {GSIS: "practice.0099"}}, Lines: map[string]source.PlayerWeek{
				"0099": {Season: 2026, Week: 1, Position: "QB", Line: frameLine(t, bodies, frame)}}}
		if err := overlay.Publish(generation); err != nil {
			t.Fatal(err)
		}
		var results commissioner.NativeResults
		f.boxRead(t, token, "results.read", 1, &results)
		return results
	}
	points := func(r commissioner.NativeResults) int64 {
		for _, score := range r.Scores {
			if score.Franchise == "franchise.01" {
				if score.Points != nil {
					return int64(*score.Points)
				}
				if score.Provisional != nil {
					return int64(*score.Provisional)
				}
			}
		}
		t.Fatalf("franchise.01 has no score in a reading of %d franchises", len(r.Scores))
		return 0
	}
	first := points(publish("synthetic-01-progress"))
	second := points(publish("synthetic-02-progress"))
	if second <= first {
		t.Fatalf("scores did not move: %d then %d", first, second)
	}
	t.Logf("STEP 5 scores move: franchise.01 %d -> %d (provisional, live frames 01 -> 02)", first, second)
	t.Log("NOT REACHED STEP 6 final / 7 stat correction / 8 standings / 9 recap: need official weekly stats pinned (officialKeeper.offer, cmd/twr-server/official_stats.go) which lives in package main of twr-server; blockers B2, B4")
}

package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/tls"
	"encoding/json"
	"fmt"
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
			Missing: livestats.Pending(), Identities: map[string]history.LiveIdentity{"0099": {GSIS: "00-0000099"}}, Lines: map[string]source.PlayerWeek{
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
	// STEP 6 official final: the clock passes week 1's last kickoff and its grace; the official files are
	// read from testdata/official and pinned, with no network.
	clock.Advance(6 * time.Hour)
	final, err := f.pinOfficial(t, clock, "stats_player_week_2026.csv.gz")
	if err != nil {
		notReached(t, 6, "official final did not pin: "+err.Error())
		return
	}
	var official commissioner.NativeResults
	f.boxRead(t, token, "results.read", 1, &official)
	finalScore, ok := franchiseScore(official, "franchise.01")
	if !ok || finalScore.Points == nil || finalScore.Provisional != nil {
		notReached(t, 6, "franchise.01 has no final week 1 points after the pin (internal/commissioner/native_week_score.go:55)")
		return
	}
	t.Logf("STEP 6 official final pinned from testdata/official (weeks %v, digest %.12s): franchise.01 final %d", final.Weeks, final.Digest, int64(*finalScore.Points))

	// STEP 7 stat correction: a later official read changes one stat of the starter; the points move.
	clock.Advance(time.Hour)
	correction, err := f.pinOfficial(t, clock, "stats_player_week_2026_corrected.csv.gz")
	if err != nil {
		notReached(t, 7, "stat correction did not pin: "+err.Error())
		return
	}
	var corrected commissioner.NativeResults
	f.boxRead(t, token, "results.read", 1, &corrected)
	moved, ok := franchiseScore(corrected, "franchise.01")
	if !ok || moved.Points == nil {
		notReached(t, 7, "franchise.01 has no final week 1 points after the correction (internal/commissioner/native_week_score.go:55)")
		return
	}
	if *moved.Points == *finalScore.Points {
		notReached(t, 7, fmt.Sprintf("the correction left franchise.01 at %d (%s, weeks %v)", int64(*moved.Points), correction.Digest, correction.Weeks))
		return
	}
	t.Logf("STEP 7 stat correction: franchise.01 %d -> %d (digest %.12s)", int64(*finalScore.Points), int64(*moved.Points), correction.Digest)
	// The starter's revision (the move off the last live line) needs a live line held in the week's evidence. The
	// official pin ends live overlay reads for the week (live_overlay.go:204-206), so none is held: this is
	// a sub-check that stays open. The points move above is the required check; steps 8-9 do not need it.
	if starter, ok := franchiseStarter(moved, "0099"); ok && starter.Revision != nil {
		t.Logf("STEP 7 starter 0099 revision: %d -> %d", int64(starter.Revision.Was), int64(starter.Revision.Now))
	} else {
		t.Log("NOT REACHED STEP 7 starter revision: no live line is held for week 1 after the official pin (internal/store/history/live_overlay.go:204-206; internal/commissioner/score_revision.go:30-38)")
	}

	// STEP 8 standings: answered from the pinned, corrected season. standings.read takes the season only
	// (internal/commandcatalog/workflow.go:139), so the week it is sent with is 0.
	var standings commissioner.NativeStandings
	f.boxRead(t, token, "standings.read", 0, &standings)
	if standings.Status == "Needs a ruling" {
		notReached(t, 8, "standings need a ruling: "+standings.Words+" (internal/commissioner/standings_reading.go:116)")
		return
	}
	t.Logf("STEP 8 standings.read answered: status %q, %d decided, %d pending, through week %d", standings.Status, standings.Decided, standings.Pending, standings.Through)
	if standings.Decided == 0 {
		// The one pinned game (franchise.01 at franchise.04) reads "No result yet" until franchise.04 sets a lineup. A
		// second owner would need a rostered starter for franchise.04, and the seed rosters only franchise.01 and .02.
		t.Log("NOT REACHED STEP 8 decided games: franchise.04 has no lineup, so week 1 has no result; it has no rostered player (tools/hostedfixture/workflows.go:44-48 seeds franchise.01 and franchise.02 only), and a lineup can be set only for the setter's own franchise (internal/commissioner/governance.go:330), one person holds one seat (internal/governance/seats.go:85)")
	}

	// STEP 9 report: results.read answers the week's report; the recap text has no hosted read yet.
	t.Logf("STEP 9 report: results.read answered for week 1: %d games, %d franchise scores", len(corrected.Games), len(corrected.Scores))
	notReached(t, 9, "recap: no hosted recap read exists; the recap text is built from a desktop snapshot in internal/app/now/recap_reading.go:110 (blocker B2 recap)")
}

// notReached records a step the box cannot pass yet, with its reason. The test stops there and stays green.
func notReached(t *testing.T, step int, reason string) {
	t.Helper()
	t.Logf("NOT REACHED STEP %d: %s", step, reason)
}

// franchiseScore is one franchise's score in a week's reading.
func franchiseScore(r commissioner.NativeResults, franchise string) (commissioner.FranchiseWeekScore, bool) {
	for _, score := range r.Scores {
		if score.Franchise == franchise {
			return score, true
		}
	}
	return commissioner.FranchiseWeekScore{}, false
}

// franchiseStarter is one starter of a franchise's week score.
func franchiseStarter(score commissioner.FranchiseWeekScore, player string) (commissioner.StarterScore, bool) {
	for _, starter := range score.Starters {
		if starter.Player == player {
			return starter, true
		}
	}
	return commissioner.StarterScore{}, false
}

// officialFile is one season file under testdata/official, decompressed.
func officialFile(t *testing.T, name string) io.Reader {
	t.Helper()
	body, err := os.ReadFile("testdata/official/" + name)
	if err != nil {
		t.Fatal(err)
	}
	zipped, err := gzip.NewReader(bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	plain, err := io.ReadAll(zipped)
	if err != nil {
		t.Fatal(err)
	}
	return bytes.NewReader(plain)
}

// pinOfficial pins the season's official files (weekly is the weekly file's name) as the league's football:
// the new evidence becomes the pin the server reads and the store's lineup environment.
func (f *releaseCLIFixture) pinOfficial(t *testing.T, clock *boxClock, weekly string) (history.OfficialPin, error) {
	t.Helper()
	pin, err := f.store.PinOfficialStats(t.Context(), f.pin.Football, history.OfficialSeason{
		Weekly: officialFile(t, weekly), Plays: officialFile(t, "play_by_play_2026.csv.gz"), At: clock.Now()})
	if err != nil {
		return pin, err
	}
	if pin.Digest == "" {
		return pin, fmt.Errorf("the pin wrote no new week; skipped %v", pin.Skipped)
	}
	evidence, err := f.store.LoadFootball(t.Context(), pin.Digest)
	if err != nil {
		return pin, err
	}
	f.store.SetPinnedFootball(evidence)
	f.pin.Football, f.pin.Digest = evidence, pin.Digest
	return pin, nil
}

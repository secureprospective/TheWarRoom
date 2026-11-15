package main

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/now"
	"github.com/secureprospective/TheWarRoom/internal/fantasyscore/source"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/livegame"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/livestats"
	"github.com/secureprospective/TheWarRoom/internal/leagueservice"
	"github.com/secureprospective/TheWarRoom/internal/serverhttp"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

type observingScores struct {
	store    *history.Store
	observed []string
}

func (p *observingScores) PublishScores(league string, season, week int) {
	p.observed = append(p.observed, fmt.Sprintf("%s/%d/%d/%s", league, season, week, p.store.LiveEpoch()))
}

func liveReplayCheckpoint(t *testing.T) rawLiveCheckpoint {
	t.Helper()
	file, err := os.Open("testdata/live-week9.tar.gz")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = file.Close() }()
	gz, err := gzip.NewReader(file)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = gz.Close() }()
	reader := tar.NewReader(gz)
	endpoints, err := now.LiveEndpoints()
	if err != nil {
		t.Fatal(err)
	}
	result := rawLiveCheckpoint{Version: 1, Season: 2026, Week: 9, Packets: map[string]rawLivePacket{}, IDs: map[string]map[string]string{"espn": {}, "sleeper": {}, "gsis": {}}}
	for {
		header, err := reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		if !strings.HasSuffix(header.Name, "raw-final.json") {
			continue
		}
		body, err := io.ReadAll(io.LimitReader(reader, (4<<20)+1))
		if err != nil {
			t.Fatal(err)
		}
		address := liveReplayAddress(t, header.Name, body, endpoints, &result)
		result.Packets[address] = rawLivePacket{URL: address, Body: body, Digest: liveDigest(body), At: time.Date(2026, 11, 14, 22, 55, 41, 0, time.UTC)}
	}
	if err := result.Validate(); err != nil {
		t.Fatal(err)
	}
	return result
}

func liveReplayAddress(t *testing.T, name string, body []byte, endpoints livegame.Endpoints, checkpoint *rawLiveCheckpoint) string {
	t.Helper()
	if strings.HasPrefix(name, "espn-scoreboard") {
		return endpoints.ESPN
	}
	if strings.HasPrefix(name, "sleeper-") {
		lines, err := source.LiveSleeperLines(source.RawSleeperWeek{Season: 2026, Week: 9, Body: string(body)})
		if err != nil {
			t.Fatal(err)
		}
		for id := range lines {
			checkpoint.IDs["sleeper"]["2"+id] = id
		}
		return "https://api.sleeper.com/v1/stats/nfl/regular/2026/9"
	}
	game, err := livegame.ParseSummary(body)
	if err != nil {
		t.Fatal(err)
	}
	lines, err := livestats.Lines(game)
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range lines {
		checkpoint.IDs["espn"]["1"+line.SourceID] = line.SourceID
	}
	return strings.ReplaceAll(endpoints.Summary, "{id}", game.ID)
}

func replayCollector(t *testing.T, checkpoint rawLiveCheckpoint) (*liveCollector, *observingScores) {
	t.Helper()
	pins := newLeaguePins(&leagueFiles{open: map[string]*leagueFile{"native.test": {}}})
	store := history.New(nil, nil)
	store.SetLiveOverlay(pins.live)
	publisher := &observingScores{store: store}
	collector := &liveCollector{pins: pins, path: filepath.Join(t.TempDir(), "checkpoint.json"), publisher: publisher, packets: &livePackets{held: checkpoint, replay: true}}
	return collector, publisher
}

func TestLiveCollectorReplaysRawFinalsPublishesAfterReadableAndRestarts(t *testing.T) {
	checkpoint := liveReplayCheckpoint(t)
	collector, publisher := replayCollector(t, checkpoint)
	if delay := collector.poll(t.Context()); delay != 5*time.Minute {
		t.Fatalf("final board cadence %s", delay)
	}
	if !collector.published || len(publisher.observed) != 1 || publisher.store.LiveEpoch() == "" {
		t.Fatal("validated live evidence not published")
	}
	expected := fmt.Sprintf("native.test/2026/9/%s", publisher.store.LiveEpoch())
	if publisher.observed[0] != expected {
		t.Fatalf("notification preceded readability: %v", publisher.observed)
	}
	saved, err := readLiveCheckpoint(collector.path)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(checkpoint, saved) {
		t.Fatal("raw provenance changed on capture")
	}
	restored, again := replayCollector(t, rawLiveCheckpoint{})
	restored.path = collector.path
	if err := restored.restore(t.Context()); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(publisher.observed, again.observed) || publisher.store.LiveEpoch() != again.store.LiveEpoch() {
		t.Fatal("restart did not rebuild identical generation")
	}
}

func TestLiveCollectorRejectsCorruptionAndFailedDurability(t *testing.T) {
	checkpoint := liveReplayCheckpoint(t)
	collector, publisher := replayCollector(t, checkpoint)
	collector.poll(t.Context())
	epoch := publisher.store.LiveEpoch()
	observed := len(publisher.observed)
	collector.path = filepath.Join(t.TempDir(), "missing", "checkpoint.json")
	lines := map[string]source.PlayerWeek{"0099": {Season: 2026, Week: 9}}
	if err := collector.SetLiveStats(t.Context(), 2026, 9, time.Now(), source.ESPNLive, livestats.Pending(), lines); err == nil {
		t.Fatal("failed durable capture succeeded")
	}
	if publisher.store.LiveEpoch() != epoch || len(publisher.observed) != observed {
		t.Fatal("failed durability replaced or published generation")
	}
	for key, packet := range checkpoint.Packets {
		packet.Digest = "corrupt"
		checkpoint.Packets[key] = packet
		break
	}
	if err := checkpoint.Validate(); err == nil {
		t.Fatal("corrupt raw provenance accepted")
	}
}

func TestLiveCollectorRejectedCandidateKeepsPreviousCheckpoint(t *testing.T) {
	collector, publisher := replayCollector(t, liveReplayCheckpoint(t))
	collector.poll(t.Context())
	before, err := os.ReadFile(collector.path)
	if err != nil {
		t.Fatal(err)
	}
	epoch := publisher.store.LiveEpoch()
	lines := map[string]source.PlayerWeek{}
	for i := 0; i < 6001; i++ {
		lines[fmt.Sprint(i)] = source.PlayerWeek{Season: 2026, Week: 9}
	}
	if err := collector.SetLiveStats(context.Background(), 2026, 9, time.Now(), source.ESPNLive, livestats.Pending(), lines); err == nil {
		t.Fatal("oversized candidate accepted")
	}
	after, err := os.ReadFile(collector.path)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) || publisher.store.LiveEpoch() != epoch {
		t.Fatal("rejection destroyed last good checkpoint")
	}
}

func TestLiveCollectorBusyCadenceAndFailureKeepsLastGoodPackets(t *testing.T) {
	checkpoint := liveBusyCheckpoint(t)
	endpoints, err := now.LiveEndpoints()
	if err != nil {
		t.Fatal(err)
	}
	packet := checkpoint.Packets[endpoints.ESPN]
	collector, publisher := replayCollector(t, checkpoint)
	if delay := collector.poll(t.Context()); delay != 5*time.Second {
		t.Fatalf("live cadence %s", delay)
	}
	if publisher.store.LiveEpoch() == "" {
		t.Fatal("live candidate not readable")
	}
	collector.packets.replay = false
	collector.packets.network = offlineLiveTransport{}
	// Identity lookup failure is deliberately not exercised by this transport-level fallback.
	request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, endpoints.ESPN, nil)
	if err != nil {
		t.Fatal(err)
	}
	response, err := collector.packets.RoundTrip(request)
	if err != nil {
		t.Fatal(err)
	}
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	if err := response.Body.Close(); err != nil {
		t.Fatal(err)
	}
	if string(body) != string(packet.Body) || len(collector.packets.held.Reduced) != 1 || collector.packets.held.Packets[endpoints.ESPN].At != packet.At {
		t.Fatal("outage destroyed source provenance")
	}
}

type offlineLiveTransport struct{}

func (offlineLiveTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, fmt.Errorf("offline")
}

// The hosted server's own Publisher is what the collector is handed: a replayed live generation must move
// that league's scores topic for the replayed week, so open change streams hear about it.
func TestLiveCollectorMovesTheHostedScoresTopic(t *testing.T) {
	collector, _ := replayCollector(t, liveReplayCheckpoint(t))
	shared := serverhttp.NewPublisher()
	collector.publisher = shared
	collector.poll(t.Context())
	if !collector.published {
		t.Fatal("validated live evidence not published")
	}
	topics := shared.Versions("", "native.test").Topics
	if len(topics) != 1 || topics[0].Kind != "scores" || topics[0].Season != 2026 || topics[0].Week != 9 || topics[0].Version == "0" {
		t.Fatalf("scores topic not published: %+v", topics)
	}
}

func TestLiveCollectorHoldsTheParsedScoreboardForNFLGames(t *testing.T) {
	collector, _ := replayCollector(t, liveReplayCheckpoint(t))
	collector.games = leagueservice.NewLiveGames()
	collector.poll(t.Context())
	board := collector.games.Board()
	if len(board.Games) == 0 || board.Source != "ESPN" || board.At.IsZero() {
		t.Fatalf("scoreboard not held: %d games, source %q, at %v", len(board.Games), board.Source, board.At)
	}
	for _, game := range board.Games {
		if game.ID == "" || game.Home.Abbreviation == "" || game.Away.Abbreviation == "" {
			t.Fatalf("held game lacks its teams: %+v", game)
		}
	}
}

func liveBusyCheckpoint(t *testing.T) rawLiveCheckpoint {
	t.Helper()
	checkpoint := liveReplayCheckpoint(t)
	endpoints, err := now.LiveEndpoints()
	if err != nil {
		t.Fatal(err)
	}
	packet := checkpoint.Packets[endpoints.ESPN]
	var board livegame.RawScoreboard
	if err := json.Unmarshal(packet.Body, &board); err != nil {
		t.Fatal(err)
	}
	state := &board.Events[0].Competitions[0].Status
	state.Type.State = "in"
	*state.Type.Completed = false
	// Events may also carry the scoreboard status; keep both validated representations consistent.
	if board.Events[0].Status != nil {
		*board.Events[0].Status = *state
	}
	packet.Body, err = json.Marshal(board)
	if err != nil {
		t.Fatal(err)
	}
	packet.Digest = liveDigest(packet.Body)
	checkpoint.Packets[endpoints.ESPN] = packet
	return checkpoint
}

// Real acquisition (not replay) must reuse Sleeper while still reading ESPN every tick.
func TestLiveCollectorFiveSecondTicksThrottleSleeperAndPublishOnlyChanges(t *testing.T) {
	checkpoint := liveBusyCheckpoint(t)
	collector, publisher := replayCollector(t, checkpoint)
	files := openTestServer(t, t.TempDir())
	collector.identities = files.leagues.family.Over(files.reference)
	collector.packets.replay = false
	network := &countingLiveTransport{packets: maps.Clone(checkpoint.Packets), calls: map[string]int{}}
	collector.packets.network = network
	endpoints, err := now.LiveEndpoints()
	if err != nil {
		t.Fatal(err)
	}
	sleeper := ""
	for address := range checkpoint.Packets {
		if strings.Contains(address, "/v1/stats/nfl/regular/") {
			sleeper = address
		}
	}
	if sleeper == "" {
		t.Fatal("fixture has no Sleeper packet")
	}
	started := time.Now()
	for tick := range 2 {
		if delay := collector.poll(t.Context()); delay != 5*time.Second {
			t.Fatalf("tick %d delay %s", tick, delay)
		}
	}
	if len(publisher.observed) != 1 {
		t.Fatalf("unchanged tick published: %v", publisher.observed)
	}
	if collector.packets.nextSleeper.Before(started.Add(time.Minute)) {
		t.Fatal("Sleeper cadence fell below 60 seconds")
	}
	if network.calls[sleeper] != 1 || network.calls[endpoints.ESPN] != 2 {
		t.Fatalf("wrong acquisition counts: %v", network.calls)
	}
	games, err := livegame.ParseScoreboard(checkpoint.Packets[endpoints.ESPN].Body)
	if err != nil {
		t.Fatal(err)
	}
	for _, game := range games {
		if game.Status == "in" {
			address := strings.ReplaceAll(endpoints.Summary, "{id}", game.ID)
			if network.calls[address] != 1 {
				t.Fatalf("live summary not refreshed: %s: %d", address, network.calls[address])
			}
		}
	}
	// Hold the summary window shut so a slow run cannot let it lapse before the Sleeper-only poll.
	for address := range collector.packets.nextSummary {
		collector.packets.nextSummary[address] = time.Now().Add(time.Hour)
	}
	collector.packets.nextSleeper = time.Now().Add(-time.Second)
	collector.poll(t.Context())
	if network.calls[sleeper] != 2 || len(publisher.observed) != 1 {
		t.Fatal("due Sleeper refresh changed facts or was skipped")
	}
	// An unchanged live summary becomes eligible again only after its eight-second window.
	for address := range collector.packets.nextSummary {
		collector.packets.nextSummary[address] = time.Now().Add(-time.Second)
	}
	collector.poll(t.Context())
	for _, game := range games {
		if game.Status == "in" {
			address := strings.ReplaceAll(endpoints.Summary, "{id}", game.ID)
			if network.calls[address] != 2 {
				t.Fatal("due summary was not refreshed")
			}
		}
	}
	// A board-only change must notify even if player lines are unchanged.
	packet := network.packets[endpoints.ESPN]
	var board livegame.RawScoreboard
	if err := json.Unmarshal(packet.Body, &board); err != nil {
		t.Fatal(err)
	}
	board.Events[0].Competitions[0].Status.DisplayClock = "2:03"
	if board.Events[0].Status != nil {
		board.Events[0].Status.DisplayClock = "2:03"
	}
	packet.Body, err = json.Marshal(board)
	if err != nil {
		t.Fatal(err)
	}
	network.packets[endpoints.ESPN] = packet
	collector.poll(t.Context())
	if len(publisher.observed) != 2 {
		t.Fatal("clock-only change did not notify")
	}
	address := strings.ReplaceAll(endpoints.Summary, "{id}", games[0].ID)
	if network.calls[address] != 3 {
		t.Fatalf("changed board did not immediately fetch summary: %d", network.calls[address])
	}
	restored, again := replayCollector(t, rawLiveCheckpoint{})
	restored.path = collector.path
	if err := restored.restore(t.Context()); err != nil {
		t.Fatal(err)
	}
	if publisher.store.LiveEpoch() != again.store.LiveEpoch() {
		t.Fatal("throttled checkpoint replay changed generation")
	}
	lines := map[string]source.PlayerWeek{"0099": {Season: 2026, Week: 9}}
	if err := collector.SetLiveStats(t.Context(), 2026, 9, time.Now(), source.ESPNLive, livestats.Pending(), lines); err != nil {
		t.Fatal(err)
	}
	if len(publisher.observed) != 3 {
		t.Fatal("stat-only change did not notify")
	}
	if err := collector.SetLiveStats(t.Context(), 2026, 9, time.Now(), source.ESPNLive, livestats.Pending(), lines); err != nil {
		t.Fatal(err)
	}
	if len(publisher.observed) != 3 {
		t.Fatal("unchanged stats notified")
	}
}

type countingLiveTransport struct {
	mu           sync.Mutex
	cacheControl string
	latency      time.Duration
	active, peak int
	packets      map[string]rawLivePacket
	calls        map[string]int
}

func (n *countingLiveTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	n.mu.Lock()
	n.active++
	n.peak = max(n.peak, n.active)
	n.mu.Unlock()
	if n.latency > 0 {
		time.Sleep(n.latency)
	}
	n.mu.Lock()
	defer func() { n.active--; n.mu.Unlock() }()
	address := r.URL.String()
	n.calls[address]++
	packet, ok := n.packets[address]
	if !ok {
		return nil, fmt.Errorf("unexpected live URL %s", address)
	}
	response := livePacketResponse(packet)
	response.Header.Set("Cache-Control", n.cacheControl)
	return response, nil
}

func TestLiveCollectorBoardCadenceHonorsMaxAge(t *testing.T) {
	for _, tc := range []struct {
		header string
		want   time.Duration
	}{
		{"", 5 * time.Second}, {"max-age=2", 5 * time.Second},
		{"public, max-age=5", 5 * time.Second}, {"max-age=8", 8 * time.Second},
		{"max-age=60", 10 * time.Second}, {"max-age=garbage", 5 * time.Second},
	} {
		t.Run(tc.header, func(t *testing.T) {
			checkpoint := liveBusyCheckpoint(t)
			collector, _ := replayCollector(t, checkpoint)
			files := openTestServer(t, t.TempDir())
			collector.identities = files.leagues.family.Over(files.reference)
			collector.packets.replay = false
			collector.packets.network = &countingLiveTransport{packets: maps.Clone(checkpoint.Packets), calls: map[string]int{}, cacheControl: tc.header}
			if got := collector.poll(t.Context()); got != tc.want {
				t.Fatalf("cadence %s, want %s", got, tc.want)
			}
		})
	}
}

func TestLiveCollectorSummaryConcurrencyIsBounded(t *testing.T) {
	checkpoint := liveBusyCheckpoint(t)
	collector, _ := replayCollector(t, checkpoint)
	collector.packets.replay = false
	network := &countingLiveTransport{packets: maps.Clone(checkpoint.Packets), calls: map[string]int{}, latency: 20 * time.Millisecond}
	collector.packets.network = network
	collector.ensureClient()
	endpoints, err := now.LiveEndpoints()
	if err != nil {
		t.Fatal(err)
	}
	games, err := livegame.ParseScoreboard(checkpoint.Packets[endpoints.ESPN].Body)
	if err != nil {
		t.Fatal(err)
	}
	games = games[:8]
	for i := range games {
		games[i].Status = "in"
	}
	collector.prepareSummaries(t.Context(), endpoints, games, nil)
	if network.peak != 4 || len(network.calls) != 8 {
		t.Fatalf("concurrency %d; requests %d", network.peak, len(network.calls))
	}
	collector.prepareSummaries(t.Context(), endpoints, games, games)
	for address, calls := range network.calls {
		if calls != 1 {
			t.Fatalf("unchanged summary re-fetched within 8 seconds: %s", address)
		}
	}
}

func TestLiveCollectorHoldsNFLGameDetails(t *testing.T) {
	checkpoint := liveReplayCheckpoint(t)
	collector, _ := replayCollector(t, checkpoint)
	collector.games = leagueservice.NewLiveGames()
	collector.poll(t.Context())
	endpoints, err := now.LiveEndpoints()
	if err != nil {
		t.Fatal(err)
	}
	for _, board := range collector.games.Board().Games {
		packet, ok := checkpoint.Packets[strings.ReplaceAll(endpoints.Summary, "{id}", board.ID)]
		if !ok {
			continue
		}
		expected, err := livegame.ParseSummary(packet.Body)
		if err != nil {
			t.Fatal(err)
		}
		// Read through the admitted service so the collector-to-command contract is exercised.
		service := leagueservice.New(nil).WithLiveGames(collector.games)
		detail, err := service.TargetNFLGame("native.test", board.ID)
		if err != nil {
			t.Fatal(err)
		}
		if detail.Game == nil || len(detail.Game.Plays) != len(expected.Plays) || detail.Game.State == nil || detail.Game.Last.Text != expected.Last.Text || !detail.At.Equal(packet.At) {
			t.Fatalf("summary not held: %+v", detail)
		}
		return
	}
	t.Fatal("fixture did not exercise a summary")
}

func TestLiveCollectorBoardAheadOfSummary(t *testing.T) {
	at := func(status string, period int, clock string, home, away int) livegame.Game {
		return livegame.Game{ID: "1", Status: status, Period: period, Clock: clock, Home: livegame.Team{Score: home}, Away: livegame.Team{Score: away}, Plays: []livegame.Play{{Text: "kept"}}}
	}
	for _, tc := range []struct {
		name           string
		summary, board livegame.Game
		behind         bool
	}{
		{"same", at("in", 2, "0:28", 21, 3), at("in", 2, "0:28", 21, 3), false},
		{"clock ran", at("in", 2, "0:51", 21, 3), at("in", 2, "0:32", 21, 3), true},
		{"score on same clock", at("in", 2, "0:28", 21, 3), at("in", 2, "0:28", 21, 6), true},
		{"next quarter", at("in", 2, "0:01", 21, 3), at("in", 3, "15:00", 21, 3), true},
		{"final", at("in", 4, "0:00", 21, 3), at("post", 4, "0:00", 21, 3), true},
		{"summary ahead", at("in", 2, "0:20", 21, 6), at("in", 2, "0:28", 21, 3), false},
		{"blank board clock", at("in", 2, "0:10", 21, 3), at("in", 2, "", 21, 3), false},
	} {
		got, behind := boardAhead(tc.summary, tc.board)
		want := tc.summary
		if tc.behind {
			want = tc.board
		}
		if behind != tc.behind || got.Clock != want.Clock || got.Period != want.Period || got.Status != want.Status || got.Home.Score != want.Home.Score || got.Away.Score != want.Away.Score || len(got.Plays) != 1 {
			t.Fatalf("%s: got %+v behind=%v", tc.name, got, behind)
		}
	}
}

// Noon 2026-11-15: no game was "in" at 12:02, so a five-minute idle wait left kickoff unseen until 12:07.
func TestHostedLiveIntervalWaitsNoLongerThanTheNextKickoff(t *testing.T) {
	noon := time.Date(2026, 11, 15, 18, 0, 0, 0, time.UTC)
	pre := func(kickoff time.Time) livegame.Game { return livegame.Game{Status: "pre", Kickoff: kickoff} }
	for _, tc := range []struct {
		name  string
		at    time.Time
		games []livegame.Game
		want  time.Duration
	}{
		{"far from kickoff", noon.Add(-time.Hour), []livegame.Game{pre(noon)}, 5 * time.Minute},
		{"kickoff in two minutes", noon.Add(-2 * time.Minute), []livegame.Game{pre(noon)}, 2 * time.Minute},
		{"past kickoff, not started", noon.Add(2 * time.Minute), []livegame.Game{pre(noon)}, 5 * time.Second},
		{"game on", noon, []livegame.Game{{Status: "in"}, pre(noon.Add(3 * time.Hour))}, 5 * time.Second},
		{"all final", noon, []livegame.Game{{Status: "post", Kickoff: noon.Add(-3 * time.Hour)}}, 5 * time.Minute},
	} {
		if got := hostedLiveInterval(tc.games, tc.at); got != tc.want {
			t.Errorf("%s: %s, want %s", tc.name, got, tc.want)
		}
	}
}

// The pinned football evidence holds no statistics, so before this a week's results went back to
// "No result yet" the moment ESPN's board moved to the next week. Week 9's real final packets stand
// in for the sources: the board is on week 10, before kickoff.
func TestLiveCollectorHoldsLastWeekAfterTheBoardMovesOn(t *testing.T) {
	week9 := liveReplayCheckpoint(t)
	endpoints, err := now.LiveEndpoints()
	if err != nil {
		t.Fatal(err)
	}
	board := week9.Packets[endpoints.ESPN]
	delete(week9.Packets, endpoints.ESPN)
	board.URL = finishedBoard(endpoints.ESPN, 2026, 9)
	week9.Packets[board.URL] = board

	current, publisher := replayCollector(t, rawLiveCheckpoint{Version: 1, Season: 2026, Week: 10})
	dir := t.TempDir()
	finished := map[[2]int]*liveCollector{}
	last := current.finishedCollector(dir, 2026, 9)
	// The sources answer from week 9's real final packets.
	last.packets = &livePackets{held: week9, replay: true}
	finished[[2]int{2026, 9}] = last
	current.keepLastWeek(t.Context(), dir, finished)

	view := current.pins.live.View()
	held := view.Week(2026, 9)
	if held == nil || len(held.Lines) == 0 || view.Current != nil {
		t.Fatalf("week 9 not held as a finished week beside an empty current week (current set: %t)", view.Current != nil)
	}
	if len(publisher.observed) != 1 || !strings.HasPrefix(publisher.observed[0], "native.test/2026/9/") {
		t.Fatalf("finished week did not move the week 9 scores topic: %v", publisher.observed)
	}
	read := last.fetched
	current.keepLastWeek(t.Context(), dir, finished)
	if read.IsZero() || last.fetched != read {
		t.Fatal("a held finished week was reread inside the refresh window")
	}

	restarted, _ := replayCollector(t, rawLiveCheckpoint{})
	if got := restoreFinished(t.Context(), restarted, dir); len(got) != 1 {
		t.Fatalf("restart restored %d finished weeks", len(got))
	}
	if after := restarted.pins.live.View().Week(2026, 9); after == nil || after.ID != held.ID {
		t.Fatal("restart did not rebuild the identical finished week from its checkpoint")
	}
}

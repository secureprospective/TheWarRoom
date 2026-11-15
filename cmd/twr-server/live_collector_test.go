package main

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
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
	collector, publisher := replayCollector(t, checkpoint)
	if delay := collector.poll(t.Context()); delay != time.Minute {
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

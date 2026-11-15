package history

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/fantasyscore"
	"github.com/secureprospective/TheWarRoom/internal/fantasyscore/source"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/livegame"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/livestats"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

func overlayTestGeneration(id string, value leaguerules.Decimal) LiveGeneration {
	return LiveGeneration{ID: id, Season: 2026, Week: 1, At: time.Date(2026, 11, 1, 19, 0, 0, 0, time.UTC), Authority: source.ESPNLive, Missing: livestats.Pending(), Lines: map[string]source.PlayerWeek{
		"0099": {Season: 2026, Week: 1, Position: "LB", Line: fantasyscore.StatLine{Quantities: map[leaguerules.StatID]leaguerules.Decimal{"tackle": value}}}}}
}

func TestLiveOverlayCapturesDetachedGenerationAndRejectsOversize(t *testing.T) {
	s := New(nil, nil)
	s.evidence = func(_ context.Context, _ source.RawNativeScope) (source.NativeEvidence, error) {
		return source.NativeEvidence{}, nil
	}
	overlay := &LiveOverlay{}
	s.SetLiveOverlay(overlay)
	first := overlayTestGeneration("first", 10000)
	if err := overlay.Publish(first); err != nil {
		t.Fatal(err)
	}
	reading := s.nativeSources("native.test", 2026)
	first.Lines["0099"].Line.Quantities["tackle"] = 990000
	if err := overlay.Publish(overlayTestGeneration("second", 20000)); err != nil {
		t.Fatal(err)
	}
	scope := source.RawNativeScope{Player: "0099", Season: 2026, Week: 1}
	held, err := s.nativeEvidence(t.Context(), reading, scope)
	if err != nil || held.Held.Week.Line.Quantities["tackle"] != 10000 {
		t.Fatalf("mixed or aliased snapshot: %+v %v", held, err)
	}
	if reading.liveKey() != "first" || s.nativeSources("native.test", 2026).liveKey() != "second" {
		t.Fatal("snapshot keys did not separate generations")
	}
	big := overlayTestGeneration("oversize", 30000)
	for i := 0; i < 6001; i++ {
		big.Lines[fmt.Sprint(i)] = big.Lines["0099"]
	}
	if err := overlay.Publish(big); err == nil || s.liveSnapshot().ID != "second" {
		t.Fatal("oversize replaced last good generation")
	}
	big = overlayTestGeneration("oversize-bytes", 30000)
	line := big.Lines["0099"]
	line.Line.Labels = map[leaguerules.StatID]string{"tackle": strings.Repeat("x", 16<<20)}
	big.Lines["0099"] = line
	if err := overlay.Publish(big); err == nil || s.liveSnapshot().ID != "second" {
		t.Fatal("byte overflow replaced last good generation")
	}
	for i := 0; i < 100; i++ {
		if err := overlay.Publish(overlayTestGeneration(fmt.Sprint(i), 10000)); err != nil {
			t.Fatal(err)
		}
	}
	if len(s.liveSnapshot().Lines) != 1 || len(s.liveHeld) != 0 || len(s.liveRevisions) != 0 {
		t.Fatal("overlay entered unbounded desktop chronology")
	}
}

func TestLiveOverlayOfficialFieldsWinAndZeroSurvives(t *testing.T) {
	s := New(nil, nil)
	live := overlayTestGeneration("tick", 10000)
	live.Lines["0099"].Line.Quantities["assist"] = 20000
	final := source.PlayerWeek{Season: 2026, Week: 1, Position: "LB", Line: fantasyscore.StatLine{Quantities: map[leaguerules.StatID]leaguerules.Decimal{"tackle": 0}}}
	evidence := source.NativeEvidence{Held: &source.HeldWeek{Authority: source.NFLVerse, Week: &final, FetchedAt: live.At}}
	out := s.overlayEvidence(evidence, source.RawNativeScope{Player: "0099", Season: 2026, Week: 1}, &live)
	selected := commissioner.FeedVersion(out.Versions())
	if selected.Held.Week.Line.Quantities["tackle"] != 0 || selected.Held.Week.Line.Quantities["assist"] != 20000 {
		t.Fatal("field precedence failed")
	}
	if _, exists := selected.Held.Week.Line.Quantities["sack"]; exists {
		t.Fatal("missing became zero")
	}
	if selected.Held.Week.FieldSources["tackle"] != source.NFLVerse || selected.Held.Week.FieldSources["assist"] != source.ESPNLive {
		t.Fatal("field attribution lost")
	}
}

func TestLiveOverlayResultsStandingsOddsCarryCapturedEvidence(t *testing.T) {
	f := resultsTest(t)
	overlay := &LiveOverlay{}
	f.s.SetLiveOverlay(overlay)
	if err := overlay.Publish(overlayTestGeneration("scores-1", 10000)); err != nil {
		t.Fatal(err)
	}
	results, err := f.s.NativeResults(t.Context(), "native.test", 2026, 1)
	if err != nil {
		t.Fatal(err)
	}
	standings, err := f.s.NativeStandings(t.Context(), "native.test", 2026)
	if err != nil {
		t.Fatal(err)
	}
	odds, err := f.s.NativeOdds(t.Context(), "native.test", 2026, commissioner.OddsValues{})
	if err != nil {
		t.Fatal(err)
	}
	for _, metadata := range []*commissioner.ScoreEvidence{results.Evidence, standings.Evidence, odds.Evidence} {
		if metadata == nil || metadata.Live != "scores-1" || !metadata.Provisional || metadata.Source != source.ESPNLive {
			t.Fatalf("missing captured source labels: %+v", metadata)
		}
	}
}

func replayWeekBodies(t *testing.T) map[string][]byte {
	t.Helper()
	file, err := os.Open("../../../cmd/twr-server/testdata/live-week9.tar.gz")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = file.Close() }()
	zipped, err := gzip.NewReader(file)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = zipped.Close() }()
	reader := tar.NewReader(zipped)
	bodies := map[string][]byte{}
	for {
		header, err := reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		body, err := io.ReadAll(io.LimitReader(reader, 4<<20))
		if err != nil {
			t.Fatal(err)
		}
		bodies[header.Name] = body
	}
	return bodies
}

func TestLiveOverlayWeek9ReplayNativeScoreParity(t *testing.T) {
	bodies := replayWeekBodies(t)
	f := resultsTest(t)
	overlay := &LiveOverlay{}
	f.s.SetLiveOverlay(overlay)
	var previous leaguerules.Decimal
	for frame, name := range []string{"synthetic-01-progress", "synthetic-02-progress", "synthetic-03-correction", "synthetic-04-progress", "synthetic-05-missing", "synthetic-06-progress", "raw-final"} {
		t.Run(name, func(t *testing.T) {
			game, err := livegame.ParseSummary(bodies["espn-401873037-"+name+".json"])
			if err != nil {
				t.Fatal(err)
			}
			lines, err := livestats.Lines(game)
			if err != nil {
				t.Fatal(err)
			}
			var line fantasyscore.StatLine
			for _, player := range lines {
				if player.SourceID == "4360310" {
					line = player.Stats
				}
			}
			if len(line.Quantities) == 0 {
				t.Fatal("replay quarterback not found")
			}
			generation := overlayTestGeneration(name, 0)
			generation.Week = 9
			generation.Lines["0099"] = source.PlayerWeek{Season: 2026, Week: 9, Position: "QB", Line: line}
			if err := overlay.Publish(generation); err != nil {
				t.Fatal(err)
			}
			score, err := f.s.NativePlayerScore(t.Context(), "native.test", source.RawNativeScope{Player: "0099", Season: 2026, Week: 9})
			if err != nil {
				t.Fatal(err)
			}
			if score.Points != nil || score.PartialPoints == nil || score.Source != "live · ESPN" || score.FetchedAt == nil {
				t.Fatalf("provisional source mislabeled: %+v", score)
			}
			raw, _, err := fantasyscore.Score(leaguerules.LegacyNFL().Scoring, leaguerules.QB, line)
			if err != nil || raw.State != leaguerules.IsKnown || raw.Value != *score.PartialPoints {
				t.Fatalf("native score differs from same raw line: %+v %v, native %+v", raw, err, score)
			}
			points := *score.PartialPoints
			if frame == 1 && points <= previous {
				t.Fatal("progress did not raise points")
			}
			if frame == 3 && points >= previous {
				t.Fatal("correction did not lower points")
			}
			if frame == 4 {
				if _, exists := score.Line["passing_yards"]; exists {
					t.Fatal("absent passing yards became zero")
				}
			}
			previous = points
			// Rebuilding a detached owner from the same validated evidence preserves native scores.
			restarted := &LiveOverlay{}
			if err := restarted.Publish(generation); err != nil {
				t.Fatal(err)
			}
			f.s.SetLiveOverlay(restarted)
			again, err := f.s.NativePlayerScore(t.Context(), "native.test", source.RawNativeScope{Player: "0099", Season: 2026, Week: 9})
			if err != nil || !reflect.DeepEqual(score, again) {
				t.Fatal("rebuild changed score", err)
			}
			f.s.SetLiveOverlay(overlay)
		})
	}
}

func TestLiveOverlayMixedOfficialAndLiveIsStillProvisional(t *testing.T) {
	f := resultsTest(t)
	f.s.evidence = func(_ context.Context, _ source.RawNativeScope) (source.NativeEvidence, error) {
		week := source.PlayerWeek{Season: 2026, Week: 1, Position: "LB", Line: fantasyscore.StatLine{Quantities: map[leaguerules.StatID]leaguerules.Decimal{"tackle": 0}}}
		return source.NativeEvidence{Held: &source.HeldWeek{Authority: source.NFLVerse, FetchedAt: t0(), Week: &week}}, nil
	}
	overlay := &LiveOverlay{}
	f.s.SetLiveOverlay(overlay)
	generation := overlayTestGeneration("tick", 990000)
	generation.Lines["0099"].Line.Quantities["assist"] = 20000
	if err := overlay.Publish(generation); err != nil {
		t.Fatal(err)
	}
	score, err := f.s.NativePlayerScore(t.Context(), "native.test", source.RawNativeScope{Player: "0099", Season: 2026, Week: 1})
	if err != nil {
		t.Fatal(err)
	}
	if score.Points != nil || score.PartialPoints == nil || score.Source != "provisional · nflverse + live fill" || score.Line["tackle"] != 0 {
		t.Fatalf("mixed evidence certified final: %+v", score)
	}
}

func TestLiveOverlayTickDoesNotJoinPreviousFlight(t *testing.T) {
	for _, kind := range []string{"results", "standings", "odds"} {
		t.Run(kind, func(t *testing.T) {
			f := resultsTest(t)
			overlay := &LiveOverlay{}
			f.s.SetLiveOverlay(overlay)
			if err := overlay.Publish(overlayTestGeneration("old", 10000)); err != nil {
				t.Fatal(err)
			}
			entered, release := make(chan struct{}), make(chan struct{})
			original := f.s.evidence
			var blocked atomic.Bool
			f.s.evidence = func(ctx context.Context, scope source.RawNativeScope) (source.NativeEvidence, error) {
				if blocked.CompareAndSwap(false, true) {
					close(entered)
					<-release
				}
				return original(ctx, scope)
			}
			read := func() (string, error) {
				switch kind {
				case "results":
					out, err := f.s.NativeResults(t.Context(), "native.test", 2026, 1)
					if out.Evidence == nil {
						return "", err
					}
					return out.Evidence.Live, err
				case "standings":
					out, err := f.s.NativeStandings(t.Context(), "native.test", 2026)
					if out.Evidence == nil {
						return "", err
					}
					return out.Evidence.Live, err
				default:
					out, err := f.s.NativeOdds(t.Context(), "native.test", 2026, commissioner.OddsValues{})
					if out.Evidence == nil {
						return "", err
					}
					return out.Evidence.Live, err
				}
			}
			old := make(chan string, 1)
			go func() {
				id, err := read()
				if err != nil {
					id = err.Error()
				}
				old <- id
			}()
			<-entered
			defer close(release)
			if err := overlay.Publish(overlayTestGeneration("new", 20000)); err != nil {
				t.Fatal(err)
			}
			newer := make(chan string, 1)
			go func() {
				id, err := read()
				if err != nil {
					id = err.Error()
				}
				newer <- id
			}()
			select {
			case id := <-newer:
				if id != "new" {
					t.Fatalf("new request joined stale flight: %s", id)
				}
			case <-time.After(5 * time.Second):
				t.Fatal("new generation joined blocked old flight")
			}
			// Unblock before observing the old reply, without closing the channel twice.
			release <- struct{}{}
			if id := <-old; id != "old" {
				t.Fatalf("old reading relabeled as %s", id)
			}
		})
	}
}

func TestLiveOverlayWeekScoreCacheUsesCapturedGeneration(t *testing.T) {
	s := New(nil, nil)
	overlay := &LiveOverlay{}
	s.SetLiveOverlay(overlay)
	records := []commissioner.Record{{}}
	calls := 0
	for i := 0; i < 25; i++ {
		if err := overlay.Publish(overlayTestGeneration(fmt.Sprint(i), leaguerules.Decimal(i))); err != nil {
			t.Fatal(err)
		}
		n := s.nativeSources("native.test", 2026)
		n.revision = &scoreRevisionKey{ledger: records, ledgerLen: len(records), key: "ledger", ok: true, boundaries: map[int]int{2026: 0}}
		calculate := func() (commissioner.FranchiseWeekScore, error) {
			calls++
			v := n.overlay.Lines["0099"].Line.Quantities["tackle"]
			return commissioner.FranchiseWeekScore{Provisional: &v}, nil
		}
		for range 2 {
			out, err := n.MemoWeekScore(t.Context(), records, leaguerules.LineupRules{}, commissioner.NativeDirectorySeat{}, 2026, 1, calculate)
			if err != nil || out.Provisional == nil || *out.Provisional != leaguerules.Decimal(i) {
				t.Fatalf("stale memo: %+v %v", out, err)
			}
		}
	}
	if calls != 25 || len(s.scores.revisions["ledger"].scores) != 1 {
		t.Fatalf("memo recomputation or retention unbounded: calls %d scores %d", calls, len(s.scores.revisions["ledger"].scores))
	}
}

func TestLiveOverlayKeepsFootballDigestAndChecksPinnedIdentity(t *testing.T) {
	s := New(nil, nil)
	pin := &PinnedFootball{digest: strings.Repeat("a", 64), raw: RawFootballEvidence{Players: map[string]RawFootballPlayer{"0099": {Identity: "00-1234567"}}}}
	s.SetPinnedFootball(pin)
	overlay := &LiveOverlay{}
	s.SetLiveOverlay(overlay)
	generation := overlayTestGeneration("first", 10000)
	generation.Identities = map[string]LiveIdentity{"0099": {GSIS: "00-1234567"}}
	if err := overlay.Publish(generation); err != nil {
		t.Fatal(err)
	}
	scope := source.RawNativeScope{Player: "0099", Season: 2026, Week: 1}
	held := s.overlayEvidence(source.NativeEvidence{}, scope, s.liveSnapshot())
	if len(held.Versions()) != 1 {
		t.Fatal("matching pinned identity did not score")
	}
	generation.ID = "second"
	generation.Identities["0099"] = LiveIdentity{GSIS: "00-7654321"}
	if err := overlay.Publish(generation); err != nil {
		t.Fatal(err)
	}
	if err := pin.CheckDigest(strings.Repeat("a", 64)); err != nil {
		t.Fatal("score tick invalidated action digest", err)
	}
	held = s.overlayEvidence(source.NativeEvidence{}, scope, s.liveSnapshot())
	if len(held.Versions()) != 0 {
		t.Fatal("another pin's identity was substituted")
	}
}

// An MFL import's football names players by MFL id alone ("mfl.<id>"), which is no NFL identity: the
// live line's own GSIS must not be compared against it (CT 257, 11-15: every live line was dropped).
func TestLiveOverlayScoresMFLImportPins(t *testing.T) {
	s := New(nil, nil)
	pin := &PinnedFootball{digest: strings.Repeat("a", 64), raw: RawFootballEvidence{Players: map[string]RawFootballPlayer{
		"17269": {Identity: MFLIdentityPrefix + "17269"}}}}
	s.SetPinnedFootball(pin)
	overlay := &LiveOverlay{}
	s.SetLiveOverlay(overlay)
	generation := overlayTestGeneration("first", 10000)
	generation.Lines = map[string]source.PlayerWeek{"17269": generation.Lines["0099"]}
	generation.Identities = map[string]LiveIdentity{"17269": {GSIS: "00-0040563"}}
	if err := overlay.Publish(generation); err != nil {
		t.Fatal(err)
	}
	scope := source.RawNativeScope{Player: "17269", Season: 2026, Week: 1}
	if held := s.overlayEvidence(source.NativeEvidence{}, scope, s.liveSnapshot()); len(held.Versions()) != 1 {
		t.Fatal("an MFL import pin dropped its player's live line")
	}
}

type overlayReplayLineups map[string][]commissioner.Starter

func (l overlayReplayLineups) WeekLineup(_ context.Context, _ []commissioner.Record, _, franchise string, _, _ int) (commissioner.WeekLineup, error) {
	starters := l[franchise]
	return commissioner.WeekLineup{Set: len(starters) > 0, Starters: starters}, nil
}

func TestLiveOverlayRawFinalLineupTotalsMatchNativeScoring(t *testing.T) {
	f := resultsTest(t)
	generate := commissioner.RawCommand{League: "native.test", Command: commissioner.ScheduleGenerate, Expected: revision(t, f.s), Request: "replay-nine", Effective: "now", Visibility: "league", Args: commissioner.RawArgs{Schedule: &commissioner.RawScheduleArgs{Weeks: 9}}}
	runScoreCommand(t, f.s, generate, "creator")
	generate.Command, generate.Expected, generate.Request = commissioner.SchedulePublish, revision(t, f.s), "publish-nine"
	generate.Args = commissioner.RawArgs{Schedule: &commissioner.RawScheduleArgs{}}
	runScoreCommand(t, f.s, generate, "creator")
	game, err := livegame.ParseSummary(replayWeekBodies(t)["espn-401873037-raw-final.json"])
	if err != nil {
		t.Fatal(err)
	}
	lines, err := livestats.Lines(game)
	if err != nil {
		t.Fatal(err)
	}
	byID := map[string]fantasyscore.StatLine{}
	for _, line := range lines {
		byID[line.SourceID] = line.Stats
	}
	generation := overlayTestGeneration("raw-final", 0)
	generation.Week = 9
	generation.Lines = map[string]source.PlayerWeek{}
	lineups := overlayReplayLineups{}
	expected := map[string]leaguerules.Decimal{}
	for _, box := range game.Box {
		if len(box.Rows) == 0 || (box.Category != "passing" && box.Category != "receiving") {
			continue
		}
		franchise := f.a
		if box.Team != game.Home.Abbreviation {
			franchise = f.b
		}
		position := leaguerules.QB
		if box.Category == "receiving" {
			position = leaguerules.WR
		}
		player := fmt.Sprint(4000 + len(generation.Lines))
		line := byID[box.Rows[0].ID]
		generation.Lines[player] = source.PlayerWeek{Season: 2026, Week: 9, Position: string(position), Line: line}
		lineups[franchise] = append(lineups[franchise], commissioner.Starter{Player: player, Name: box.Rows[0].Name})
		points, _, err := fantasyscore.Score(leaguerules.LegacyNFL().Scoring, position, line)
		if err != nil || points.State != leaguerules.IsKnown {
			t.Fatalf("raw score: %+v %v", points, err)
		}
		expected[franchise] += points.Value
	}
	if len(expected) != 2 || len(generation.Lines) != 4 {
		t.Fatal("raw finals did not provide both two-player lineups")
	}
	f.s.SetNativeLineups(lineups)
	overlay := &LiveOverlay{}
	f.s.SetLiveOverlay(overlay)
	if err := overlay.Publish(generation); err != nil {
		t.Fatal(err)
	}
	results, err := f.s.NativeResults(t.Context(), "native.test", 2026, 9)
	if err != nil {
		t.Fatal(err)
	}
	found := 0
	for _, score := range results.Scores {
		want, exists := expected[score.Franchise]
		if !exists {
			continue
		}
		found++
		if score.Points != nil || score.Provisional == nil || *score.Provisional != want {
			t.Fatalf("lineup %s: got %+v, raw native total %d", score.Franchise, score, want)
		}
	}
	if found != 2 {
		t.Fatal("native results omitted scored lineups")
	}
}

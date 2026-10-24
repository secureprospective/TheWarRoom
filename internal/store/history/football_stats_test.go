package history

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/fantasyscore/source"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

func scoringImport(t *testing.T) RawFootballEvidence {
	t.Helper()
	body, _ := footballBody(t)
	var raw RawFootballEvidence
	if err := json.Unmarshal(body, &raw); err != nil {
		t.Fatal(err)
	}
	qb := raw.Players["0099"]
	qb.Identity = "00-0023459"
	raw.Players["0099"] = qb
	pk := qb
	pk.Identity, pk.Facts.Name, pk.Facts.Position = "00-0023853", "Kicker", leaguerules.PK
	raw.Players["0100"] = pk
	weekly := strings.ReplaceAll(statBody(t, "244", true), ",2025,", ",2026,")
	plays := "game_id,play_id,season,week,season_type,passer_player_id,receiver_player_id,rusher_player_id,passing_yards,receiving_yards,rushing_yards,touchdown,td_player_id,td_team,return_team,interception,punt_attempt,kickoff_attempt,play_deleted,two_point_attempt\n" +
		"2025_01_PIT_NYJ,1,2026,1,REG,00-0023459,,,40,NA,NA,0,,,,0,0,0,0,0\n" +
		"2025_01_BAL_BUF,1,2026,1,REG,,,,NA,NA,NA,0,,,,0,0,0,0,0\n"
	raw.Stats = []RawFootballStats{{AsOf: raw.AsOf.Add(-time.Hour), Weekly: weekly, Plays: plays}}
	return raw
}

func installScoringImport(t *testing.T, s *Store, raw RawFootballEvidence) *PinnedFootball {
	t.Helper()
	body, err := json.Marshal(raw)
	if err != nil {
		t.Fatal(err)
	}
	digest := footballDigest(body)
	if err := s.ImportFootball(t.Context(), body, digest); err != nil {
		t.Fatal(err)
	}
	pinned, err := s.LoadFootball(t.Context(), digest)
	if err != nil {
		t.Fatal(err)
	}
	s.SetLineupEnvironment(pinned)
	return pinned
}

type pinnedResultLineups struct{ qb string }

func (p pinnedResultLineups) WeekLineup(_ context.Context, _ []commissioner.Record, _, franchise string, _, _ int) (commissioner.WeekLineup, error) {
	player := "0100"
	if franchise == p.qb {
		player = "0099"
	}
	return commissioner.WeekLineup{Set: true, Starters: []commissioner.Starter{{Player: player}}}, nil
}

func TestPinnedScoringResultsSurviveSourceLossAndReopen(t *testing.T) {
	f := resultsTest(t)
	raw := scoringImport(t)
	pinned := installScoringImport(t, f.s, raw)
	f.s.SetNativeLineups(pinnedResultLineups{qb: f.a})
	// Archive refetches disagree with the imported baseline and have no plays.
	putStatBody(t, f.s, statBody(t, "999", true), "", t0())
	before := pinnedResultDigest(t, f.s)
	if err := f.pools.Close(); err != nil {
		t.Fatal(err)
	}
	f.s, f.pools = statStore(t, f.path)
	restored, err := f.s.LoadFootball(t.Context(), pinned.Digest())
	if err != nil {
		t.Fatal(err)
	}
	f.s.SetLineupEnvironment(restored)
	f.s.SetNativeLineups(pinnedResultLineups{qb: f.a})
	if got := pinnedResultDigest(t, f.s); got != before {
		t.Fatal("frozen results changed after reopening")
	}
	// A fresh archive containing only the pin can reproduce the scoring evidence.
	isolated, _ := statStore(t, filepath.Join(t.TempDir(), "only-pin.db"))
	replica := installScoringImport(t, isolated, raw)
	isolated.SetNativeLineups(pinnedResultLineups{qb: f.a})
	src := isolated.nativeSources("native.test", 2026)
	result, err := commissioner.NativeResultsReading(t.Context(), "native.test", scoreRecords(t, f.s), src, 2026, 1, src.now)
	if err != nil {
		t.Fatal(err)
	}
	body, err := json.Marshal(result)
	if err != nil || footballDigest(body) != before {
		t.Fatalf("results changed without duplicate archive bodies: %v", err)
	}
	for _, player := range []string{"0099", "0100"} {
		scope := source.RawNativeScope{Player: player, Season: 2026, Week: 1}
		a, err := f.s.nativeEvidence(t.Context(), f.s, scope)
		if err != nil {
			t.Fatal(err)
		}
		b, err := isolated.nativeEvidence(t.Context(), isolated, scope)
		if err != nil || !reflect.DeepEqual(a, b) {
			t.Fatalf("archive-free evidence differs: %v", err)
		}
		// Detached evidence cannot poison subsequent readings.
		b.Lines[0].Held.Week.Line.Events["long_pass"] = nil
	}
	if replica.Digest() != pinned.Digest() {
		t.Fatal("source digest changed")
	}
}

func pinnedResultDigest(t *testing.T, s *Store) string {
	t.Helper()
	out, err := s.NativeResults(t.Context(), "native.test", 2026, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Games) == 0 {
		t.Fatal("no published games")
	}
	for _, game := range out.Games {
		if game.Status != commissioner.GameFinal && game.Status != commissioner.GameTie {
			t.Fatalf("incomplete game: %+v", game)
		}
	}
	for player, want := range map[string]leaguerules.Decimal{"0099": 332500, "0100": 120000} {
		score, err := s.NativePlayerScore(t.Context(), "native.test", source.RawNativeScope{Player: player, Season: 2026, Week: 1})
		if err != nil || score.Points == nil || *score.Points != want {
			t.Fatalf("%s: %+v %v", player, score, err)
		}
	}
	body, err := json.Marshal(out)
	if err != nil {
		t.Fatal(err)
	}
	return footballDigest(body)
}

func TestPinnedStatsRejectInvalidImportsWithoutChangingAuthority(t *testing.T) {
	s, _ := statStore(t, filepath.Join(t.TempDir(), "history.db"))
	raw := scoringImport(t)
	pinned := installScoringImport(t, s, raw)
	before := footballRows(t, s)
	for _, change := range []func(*RawFootballEvidence){
		func(r *RawFootballEvidence) { r.Stats[0].Plays = "" },
		func(r *RawFootballEvidence) {
			r.Stats[0].Plays = strings.ReplaceAll(r.Stats[0].Plays, "2025_01_PIT_NYJ", "wrong-game")
		},
		func(r *RawFootballEvidence) {
			r.Stats[0].Weekly = strings.ReplaceAll(r.Stats[0].Weekly, ",2026,", ",2027,")
		},
		func(r *RawFootballEvidence) { r.Stats[0].AsOf = r.AsOf.Add(time.Hour) },
		func(r *RawFootballEvidence) { r.Stats = append(r.Stats, r.Stats[0]) },
		func(r *RawFootballEvidence) { p := r.Players["0099"]; p.Identity = "unknown"; r.Players["0099"] = p },
	} {
		bad := scoringImport(t)
		change(&bad)
		body, err := json.Marshal(bad)
		if err != nil {
			t.Fatal(err)
		}
		if err := s.ImportFootball(t.Context(), body, footballDigest(body)); err == nil {
			t.Fatal("invalid statistics accepted")
		}
	}
	if footballRows(t, s) != before || s.lineupEnv != pinned {
		t.Fatal("invalid import changed held state")
	}
}

func TestPinnedHeldFixtureDistanceAndKickerScores(t *testing.T) {
	raw := scoringImport(t)
	raw.Season = 2025
	for _, input := range []struct {
		name string
		dest *string
	}{{"weeks.csv", &raw.Stats[0].Weekly}, {"plays.csv", &raw.Stats[0].Plays}} {
		body, err := os.ReadFile("../../fantasyscore/source/testdata/" + input.name)
		if err != nil {
			t.Fatal(err)
		}
		*input.dest = string(body)
	}
	s, _ := statStore(t, filepath.Join(t.TempDir(), "held.db"))
	p := installScoringImport(t, s, raw)
	evidence, err := source.ReadNativeEvidence(t.Context(), p, p, source.RawNativeScope{Player: "0100", Season: 2025, Week: 1})
	if err != nil || len(evidence.Missing) != 0 || len(evidence.Lines) != 1 || len(evidence.Lines[0].Held.Missing) != 0 {
		t.Fatalf("complete kicker evidence lost: %+v %v", evidence, err)
	}
}

func TestPinnedDistanceBonusesReachNativeScoresAtBoundaries(t *testing.T) {
	s, _ := statStore(t, filepath.Join(t.TempDir(), "distance.db"))
	runScoreCommand(t, s, createWith(commissioner.PresetLegacy, 0), "creator")
	runScoreCommand(t, s, heldSeedCommand(t), "creator")
	for _, tc := range []struct {
		pass, rush, receive string
		want                leaguerules.Decimal
	}{
		{"39", "19", "19", 0}, {"40", "20", "20", 30000}, {"41", "40", "40", 50000},
	} {
		raw := scoringImport(t)
		raw.Stats[0].Plays = strings.Replace(raw.Stats[0].Plays, "00-0023459,,,40,NA,NA", "00-0023459,00-0023459,00-0023459,"+tc.pass+","+tc.receive+","+tc.rush, 1)
		installScoringImport(t, s, raw)
		score, err := s.NativePlayerScore(t.Context(), "native.test", source.RawNativeScope{Player: "0099", Season: 2026, Week: 1})
		if err != nil || score.Points == nil {
			t.Fatalf("distance case %+v: %+v %v", tc, score, err)
		}
		var bonus leaguerules.Decimal
		for _, item := range score.Items {
			if strings.HasPrefix(string(item.Stat), "long_") && item.Points != nil {
				bonus += *item.Points
			}
		}
		if bonus != tc.want {
			t.Fatalf("distance case %+v: got bonus %d", tc, bonus)
		}
	}
}

func TestPinnedStatApprovalKeepsCompleteBaselineAndQueueParity(t *testing.T) {
	f := resultsTest(t)
	f.s.SetNativeLineups(pinnedResultLineups{qb: f.a})
	raw := scoringImport(t)
	older := raw.Stats[0]
	newer := older
	newer.AsOf = older.AsOf.Add(time.Minute)
	newer.Weekly = strings.ReplaceAll(statBody(t, "245", true), ",2025,", ",2026,")
	raw.Stats = []RawFootballStats{newer, older}
	pinned := installScoringImport(t, f.s, raw)
	pinnedResultDigest(t, f.s)
	queue, err := f.s.StatQueue(t.Context(), "native.test", 2026)
	if err != nil || len(queue.Open) != 1 || queue.Open[0].Points == nil || *queue.Open[0].Points != 500 {
		t.Fatalf("queue: %+v %v", queue, err)
	}
	rawCommand := statCommand(commissioner.StatApprove, commissioner.RawStatChange{Item: queue.Open[0].ID}, revision(t, f.s), "approve-pin")
	runScoreCommand(t, f.s, rawCommand, "creator")
	score, err := f.s.NativePlayerScore(t.Context(), "native.test", source.RawNativeScope{Player: "0099", Season: 2026, Week: 1})
	if err != nil || score.Points == nil || *score.Points != 333000 {
		t.Fatalf("approved: %+v %v", score, err)
	}
	before, err := json.Marshal(score)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.pools.Close(); err != nil {
		t.Fatal(err)
	}
	f.s, f.pools = statStore(t, f.path)
	restored, err := f.s.LoadFootball(t.Context(), pinned.Digest())
	if err != nil {
		t.Fatal(err)
	}
	f.s.SetLineupEnvironment(restored)
	f.s.SetNativeLineups(pinnedResultLineups{qb: f.a})
	score, err = f.s.NativePlayerScore(t.Context(), "native.test", source.RawNativeScope{Player: "0099", Season: 2026, Week: 1})
	if err != nil {
		t.Fatal(err)
	}
	after, err := json.Marshal(score)
	if err != nil || string(before) != string(after) {
		t.Fatalf("approved score digest changed: %v", err)
	}
	queue, err = f.s.StatQueue(t.Context(), "native.test", 2026)
	if err != nil || len(queue.Open) != 0 || len(queue.Decided) != 1 {
		t.Fatalf("approved queue: %+v %v", queue, err)
	}
}

func TestPinnedKickerFallsBackToBaselineWhenNewestFileOmitsHim(t *testing.T) {
	s, _ := statStore(t, filepath.Join(t.TempDir(), "kicker.db"))
	runScoreCommand(t, s, createWith(commissioner.PresetLegacy, 0), "creator")
	runScoreCommand(t, s, heldSeedCommand(t), "creator")
	raw := scoringImport(t)
	newer := raw.Stats[0]
	newer.AsOf = newer.AsOf.Add(time.Minute)
	rows, err := csv.NewReader(strings.NewReader(newer.Weekly)).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	var body strings.Builder
	writer := csv.NewWriter(&body)
	if err := writer.WriteAll(rows[:2]); err != nil {
		t.Fatal(err)
	}
	newer.Weekly = body.String()
	raw.Stats = append([]RawFootballStats{newer}, raw.Stats...)
	pinned := installScoringImport(t, s, raw)
	evidence, err := source.ReadNativeEvidence(t.Context(), pinned, pinned, source.RawNativeScope{Player: "0100", Season: 2026, Week: 1})
	if err != nil || evidence.Held.Week != nil || len(evidence.Lines) != 1 {
		t.Fatalf("kicker revisions: %+v %v", evidence, err)
	}
	score, err := s.NativePlayerScore(t.Context(), "native.test", source.RawNativeScope{Player: "0100", Season: 2026, Week: 1})
	if err != nil || score.Points == nil || *score.Points != 120000 {
		t.Fatalf("missing newest kicker erased baseline: %+v %v", score, err)
	}
}

package main

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/archive"
	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/measures"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

func scoreApp(t *testing.T) *legalityFixture {
	t.Helper()
	f := legalityApp(t)
	if _, err := f.app.history.LinkPlayerIDs(t.Context(), "dynastyprocess", []history.PlayerIDLink{
		{IDType: "gsis", IDValue: "00-0023459", PlayerID: "1000"},
	}); err != nil {
		t.Fatal(err)
	}
	holdNativeScore(t, f, true, "LB", 1)
	return f
}

// The held row records one tackle and nothing else, so exact position and rule effects are visible.
func holdNativeScore(t *testing.T, f *legalityFixture, include bool, position string, revision int) string {
	t.Helper()
	return holdNativeStats(t, f, include, position, "1", revision)
}

// holdNativeStats archives one more copy of the season's weekly file, the way a feed refresh would, with the
// player credited the given number of tackles.
func holdNativeStats(t *testing.T, f *legalityFixture, include bool, position, tackles string, revision int) string {
	t.Helper()
	body, err := os.ReadFile("internal/fantasyscore/source/testdata/weeks.csv")
	if err != nil {
		t.Fatal(err)
	}
	rows, err := csv.NewReader(bytes.NewReader(body)).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	rows = rows[:2]
	for i, key := range rows[0] {
		switch key {
		case "player_id":
			if !include {
				rows[1][i] = "00-0023853"
			}
		case "position":
			rows[1][i] = position
		case "season":
			rows[1][i] = "2026"
		case "game_id":
			rows[1][i] = "2026_01_PIT_NYJ"
		case "def_tackles_solo":
			rows[1][i] = tackles
		case "fg_made_list", "fg_missed_list", "fg_blocked_list":
			rows[1][i] = ""
		case "week", "season_type", "player_name", "player_display_name", "position_group", "headshot_url", "team", "opponent_team":
		default:
			rows[1][i] = "0"
		}
	}
	var text bytes.Buffer
	writer := csv.NewWriter(&text)
	if err := writer.WriteAll(rows); err != nil {
		t.Fatal(err)
	}
	var zipped bytes.Buffer
	gz := gzip.NewWriter(&zipped)
	if _, err := gz.Write(text.Bytes()); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	hash := sha256.Sum256(text.Bytes())
	sha := hex.EncodeToString(hash[:])
	reg, err := measures.Embedded()
	if err != nil {
		t.Fatal(err)
	}
	feed, ok := reg.Feed("stats_player_week")
	if !ok {
		t.Fatal("weekly feed missing")
	}
	at := time.Date(2026, 10, 13, 11, revision, 0, 0, time.UTC)
	if err := f.app.history.Record(t.Context(), archive.Fetch{URL: feed.URLFor(2026), Status: 200,
		SHA256: sha, Size: int64(text.Len()), Gzip: zipped.Bytes(), FetchedAt: at}); err != nil {
		t.Fatal(err)
	}
	return sha
}

func readNativeScore(t *testing.T, f *legalityFixture) commissioner.NativePlayerScore {
	t.Helper()
	out, err := f.app.TargetNativePlayerScore(legalityLeague, "1000", 2026, 1)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func assertNativePoints(t *testing.T, f *legalityFixture, want leaguerules.Decimal) {
	t.Helper()
	out := readNativeScore(t, f)
	if out.Status != "Incomplete" || out.Points != nil || out.PartialPoints == nil || *out.PartialPoints != want ||
		len(out.Missing) != 1 || !strings.Contains(out.Missing[0], "long plays and return touchdowns") {
		t.Fatalf("honest exact partial score: %+v; want %d", out, want)
	}
}

func scoreCatalogue(t *testing.T, points leaguerules.Value[leaguerules.Decimal]) string {
	t.Helper()
	type rawBonus struct{ At, Points leaguerules.Decimal }
	type rawStat struct {
		Label     string
		Group     string
		Used      bool
		Points    *leaguerules.Decimal `json:"Points,omitempty"`
		Bonuses   []rawBonus
		Positions []leaguerules.Position
		Once      bool
	}
	catalogue := map[leaguerules.StatID]rawStat{}
	for id, stat := range leaguerules.LegacyNFL().Scoring.Catalogue.Value {
		value := stat.Points.Value
		row := rawStat{Label: stat.Label.Value, Group: stat.Group.Value, Used: stat.Used.Value,
			Points: &value, Bonuses: []rawBonus{}, Positions: stat.Positions.Value, Once: stat.Once.Value}
		if row.Positions == nil {
			row.Positions = []leaguerules.Position{}
		}
		for _, bonus := range stat.Bonuses.Value {
			row.Bonuses = append(row.Bonuses, rawBonus{bonus.At.Value, bonus.Points.Value})
		}
		if id == "tackle" {
			if points.State == leaguerules.IsKnown {
				row.Points = &points.Value
			} else {
				row.Points = nil
			}
		}
		catalogue[id] = row
	}
	body, err := json.Marshal(catalogue)
	if err != nil {
		t.Fatal(err)
	}
	return string(body)
}

func TestNativeScoreCM006GovernsReading(t *testing.T) {
	f := scoreApp(t)
	for pass := 0; pass < 2; pass++ {
		if pass == 1 {
			f.reopen(t)
		}
		assertNativePoints(t, f, 15000)
		receipt := f.set(t, "scoring.catalogue", scoreCatalogue(t, leaguerules.Known[leaguerules.Decimal](30000)), "now")
		assertNativePoints(t, f, 30000)
		f.reopen(t)
		assertNativePoints(t, f, 30000)
		f.revert(t, receipt)
		assertNativePoints(t, f, 15000)
		f.reopen(t)
		assertNativePoints(t, f, 15000)
	}
}

func TestNativeScoreCM030AndCM032GovernReading(t *testing.T) {
	for _, tc := range []struct {
		name, command string
		args          commissioner.RawScoreChange
		want          leaguerules.Decimal
	}{
		{"CM-030", commissioner.PlayerPosition, commissioner.RawScoreChange{Player: "1000", Position: leaguerules.DT}, 25000},
		{"CM-032", commissioner.ScorePlayer, commissioner.RawScoreChange{Player: "1000", Season: 2026,
			Week: 1, Delta: scoreDelta(12500)}, 27500},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := scoreApp(t)
			for pass := 0; pass < 2; pass++ {
				if pass == 1 {
					f.reopen(t)
				}
				assertNativePoints(t, f, 15000)
				receipt := directoryCommand(t, f.app, tc.command, fmt.Sprintf("score.%d", pass),
					commissioner.RawArgs{Score: &tc.args}, "now")
				assertNativePoints(t, f, tc.want)
				out := readNativeScore(t, f)
				last := out.Changes[len(out.Changes)-1]
				if last.Entry != receipt.Entry || last.Status != "Applied" || last.Reason != "Record our league choice" {
					t.Fatalf("ledger provenance lost: %+v", out)
				}
				if tc.name == "CM-030" && (out.Position != leaguerules.DT ||
					!strings.Contains(out.PositionWhy, "commissioner")) {
					t.Fatalf("override lost: %+v", out)
				}
				f.reopen(t)
				assertNativePoints(t, f, tc.want)
				f.revert(t, receipt)
				assertNativePoints(t, f, 15000)
				f.reopen(t)
				assertNativePoints(t, f, 15000)
				out = readNativeScore(t, f)
				if out.Changes[len(out.Changes)-1].Status != "Reversed" {
					t.Fatal("reversal provenance lost")
				}
			}
		})
	}
}

func scoreDelta(n leaguerules.Decimal) *leaguerules.Decimal { return &n }

func TestNativeScoreHonestAbsenceAndNeedsRuling(t *testing.T) {
	f := scoreApp(t)
	assertNativePoints(t, f, 15000)
	out, err := f.app.TargetNativePlayerScore(legalityLeague, "1001", 2026, 1)
	if err != nil || out.Status != "Unavailable" || out.Points != nil || out.PartialPoints != nil ||
		!strings.Contains(out.Words, "no NFL statistics identity") {
		t.Fatalf("missing identity became zero: %+v %v", out, err)
	}
	counted := readNativeScore(t, f).SHA256
	holdNativeScore(t, f, false, "LB", 2)
	out = readNativeScore(t, f)
	if out.Status != "Incomplete" || out.SHA256 != counted {
		t.Fatalf("a refresh that dropped the player unscored him or moved the score: %+v", out)
	}
	holdNativeScore(t, f, true, "LB", 3)
	f.set(t, "scoring.catalogue", scoreCatalogue(t, leaguerules.Needs[leaguerules.Decimal]("Choose tackle points")), "now")
	out = readNativeScore(t, f)
	if out.Status != "Needs a ruling" || out.Points != nil || out.PartialPoints != nil ||
		!strings.Contains(out.Words, "Choose a scoring value") || len(out.Missing) != 1 {
		t.Fatalf("unsettled rule became zero: %+v", out)
	}
	f.reopen(t)
	if got := readNativeScore(t, f); !reflect.DeepEqual(got, out) {
		t.Fatal("absence changed on replay")
	}
}

func TestNativeScorePastSeasonKeepsItsRules(t *testing.T) {
	f := scoreApp(t)
	f.set(t, "calendar.rollover", `"2027-03-01T05:00:00Z"`, "now")
	future := time.Date(2027, 3, 2, 12, 0, 0, 0, time.UTC)
	f.app.history.SetClock(func() time.Time { return future })
	f.set(t, "scoring.catalogue", scoreCatalogue(t, leaguerules.Known[leaguerules.Decimal](90000)), "now")
	assertNativePoints(t, f, 15000)
	f.reopen(t)
	f.app.history.SetClock(func() time.Time { return future })
	assertNativePoints(t, f, 15000)
}

func TestNativeScoreScopeAndStartingSnapshot(t *testing.T) {
	a := nativeSeedTestApp(t)
	createNativeSeedLeague(t, a, commissioner.PresetLegacy)
	out, err := a.TargetNativePlayerScore(legalityLeague, "1000", 2026, 1)
	if err != nil || out.Words != "Record the starting snapshot first" || out.Points != nil || out.PartialPoints != nil {
		t.Fatalf("unbound season scored: %+v %v", out, err)
	}
	for _, scope := range []struct {
		league, player string
		season, week   int
	}{
		{"native.", "1000", 2026, 1}, {"native.missing", "1000", 2026, 1},
		{legalityLeague, "99", 2026, 1}, {legalityLeague, "1000", 1919, 1}, {legalityLeague, "1000", 2026, 23},
	} {
		if _, err := a.TargetNativePlayerScore(scope.league, scope.player, scope.season, scope.week); err == nil {
			t.Fatalf("invalid scope accepted: %+v", scope)
		}
	}
}

func TestNativeScoreWireFixtureMatchesGo(t *testing.T) {
	f := scoreApp(t)
	readings := []commissioner.NativePlayerScore{readNativeScore(t, f)}
	missing, err := f.app.TargetNativePlayerScore(legalityLeague, "1001", 2026, 1)
	if err != nil {
		t.Fatal(err)
	}
	readings = append(readings, missing)
	receipt := f.set(t, "scoring.catalogue", scoreCatalogue(t,
		leaguerules.Needs[leaguerules.Decimal]("Choose tackle points")), "now")
	readings = append(readings, readNativeScore(t, f))
	f.revert(t, receipt)
	directoryCommand(t, f.app, commissioner.PlayerPosition, "wire.position",
		commissioner.RawArgs{Score: &commissioner.RawScoreChange{Player: "1000", Position: leaguerules.DT}}, "now")
	delta := directoryCommand(t, f.app, commissioner.ScorePlayer, "wire.delta",
		commissioner.RawArgs{Score: &commissioner.RawScoreChange{Player: "1000", Season: 2026, Week: 1,
			Delta: scoreDelta(12500)}}, "now")
	readings = append(readings, readNativeScore(t, f))
	f.revert(t, delta)
	readings = append(readings, readNativeScore(t, f))
	holdNativeScore(t, f, false, "LB", 2)
	readings = append(readings, readNativeScore(t, f))
	notReported, err := f.app.TargetNativePlayerScore(legalityLeague, "1000", 2026, 2)
	if err != nil {
		t.Fatal(err)
	}
	readings = append(readings, notReported)
	body, err := json.MarshalIndent(readings, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	body = append(body, '\n')
	const path = "internal/commissioner/testdata/player-score-wire.json"
	if os.Getenv("TWR_WRITE_WIRE") == "1" {
		if err := os.WriteFile(path, body, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	held, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(held, body) {
		t.Fatal("native player score wire fixture is stale")
	}
}

func TestNativeScoreKeepsTheCountedVersionUntilTheLeagueApprovesTheNewest(t *testing.T) {
	f := scoreApp(t)
	counted := readNativeScore(t, f).SHA256
	holdNativeScore(t, f, true, "DT", 2)
	out := readNativeScore(t, f)
	if out.SHA256 != counted || out.Position != leaguerules.LB {
		t.Fatalf("a feed revision moved the score without approval: %+v", out)
	}
	assertNativePoints(t, f, 15000)
	missing, err := f.app.TargetNativePlayerScore(legalityLeague, "1000", 2026, 2)
	if err != nil || missing.Status != "Unavailable" || missing.Points != nil || missing.PartialPoints != nil ||
		!strings.Contains(missing.Words, "not reported this week") {
		t.Fatalf("absent week scored: %+v %v", missing, err)
	}
}

func TestNativeScoreNoHeldFilesAndUnboundIdentity(t *testing.T) {
	f := legalityApp(t)
	if _, err := f.app.history.LinkPlayerIDs(t.Context(), "dynastyprocess", []history.PlayerIDLink{
		{IDType: "gsis", IDValue: "00-0023459", PlayerID: "1000"},
	}); err != nil {
		t.Fatal(err)
	}
	out := readNativeScore(t, f)
	if out.Status != "Unavailable" || out.Points != nil || out.PartialPoints != nil ||
		out.Words != "No NFL statistics are held for this season and week" || len(out.Missing) != 1 {
		t.Fatalf("no held files became a zero: %+v", out)
	}
	a := nativeSeedTestApp(t)
	createNativeSeedLeague(t, a, commissioner.PresetLegacy)
	if _, err := a.history.LinkPlayerIDs(t.Context(), "dynastyprocess", []history.PlayerIDLink{
		{IDType: "gsis", IDValue: "00-0023459", PlayerID: "1000"},
	}); err != nil {
		t.Fatal(err)
	}
	out, err := a.TargetNativePlayerScore(legalityLeague, "1000", 1920, 1)
	if err != nil || out.Words != "Record the starting snapshot first" {
		t.Fatalf("unbound season consulted the stats port: %+v %v", out, err)
	}
}

package history

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/fantasyscore"
	"github.com/secureprospective/TheWarRoom/internal/fantasyscore/source"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

func sleeper2025(t *testing.T) RawFootballEvidence {
	t.Helper()
	raw := scoringImport(t)
	raw.Season = 2025
	raw.Stats[0].Weekly = strings.ReplaceAll(raw.Stats[0].Weekly, ",2026,", ",2025,")
	raw.Stats[0].Plays = strings.ReplaceAll(raw.Stats[0].Plays, ",2026,", ",2025,")
	p := raw.Players["0099"]
	p.Identity, p.Sleeper, p.Facts.Name, p.Facts.Position = "", "12944", "Dorian Mausi", leaguerules.LB
	raw.Players["17367"] = p
	p.Identity, p.Sleeper, p.Facts.Name, p.Facts.Position = "00-0038535", "10938", "DJ Johnson", leaguerules.DE
	raw.Players["16277"] = p
	return raw
}

func reopenSleeper(t *testing.T, raw RawFootballEvidence) *PinnedFootball {
	t.Helper()
	path := filepath.Join(t.TempDir(), "history.db")
	s, pools := statStore(t, path)
	pinned := installScoringImport(t, s, raw)
	if err := pools.Close(); err != nil {
		t.Fatal(err)
	}
	s, _ = statStore(t, path)
	pinned, err := s.LoadFootball(t.Context(), pinned.Digest())
	if err != nil {
		t.Fatal(err)
	}
	return pinned
}

func sleeperRead(t *testing.T, p *PinnedFootball, id string, week int) source.NativeEvidence {
	t.Helper()
	evidence, err := source.ReadNativeEvidence(t.Context(), p, p, source.RawNativeScope{Player: id, Season: 2025, Week: week})
	if err != nil {
		t.Fatal(err)
	}
	return evidence
}

func sleeperScore(t *testing.T, pos leaguerules.Position, line fantasyscore.StatLine) leaguerules.Decimal {
	t.Helper()
	rules := leaguerules.Default32Scoring()
	rules.RoundStatPoints.Value = false
	score, _, err := fantasyscore.Score(rules, pos, line)
	if err != nil || score.State != leaguerules.IsKnown {
		t.Fatalf("score: %+v %v", score, err)
	}
	return score.Value
}

func TestPinnedSleeperMausiWithoutGSIS(t *testing.T) {
	raw := sleeper2025(t)
	snaps := []int{20, 20, 14, 14, 22, 19, 17, 19, 20, 0, 18, 16, 20}
	for i, n := range snaps {
		if n == 0 {
			continue
		}
		tackles, penalty := 0, 0
		switch i + 1 {
		case 5, 6:
			tackles = 1
		case 9:
			tackles = 2
		case 12:
			penalty = 1
		}
		raw.Stats[0].Sleeper = append(raw.Stats[0].Sleeper, source.RawSleeperWeek{Season: 2025, Week: i + 1, Body: fmt.Sprintf(`{"12944":{"st_snp":%d,"st_tkl_solo":%d,"penalty":%d}}`, n, tackles, penalty)})
	}
	p := reopenSleeper(t, raw)
	var total, sn, tk, pen leaguerules.Decimal
	for w := 1; w <= 13; w++ {
		e := sleeperRead(t, p, "17367", w)
		if e.GSIS != "" || e.Held == nil {
			t.Fatalf("week %d: %+v", w, e)
		}
		if w == 10 {
			if e.Held.Week != nil {
				t.Fatal("absent week created")
			}
			continue
		}
		line := e.Held.Week.Line
		if e.Held.Week.GSIS != "" || e.Held.Week.Position != "LB" || e.Held.Week.Season != 2025 || e.Held.Week.Week != w || e.Held.Week.SeasonType != "REG" {
			t.Fatalf("week %d: %+v", w, e.Held.Week)
		}
		total += sleeperScore(t, leaguerules.LB, line)
		sn += line.Quantities["special_teams_snap"]
		tk += line.Quantities["special_teams_solo_tackle"]
		pen += line.Quantities["penalty"]
	}
	if total != 14070 || sn != 2190000 || tk != 40000 || pen != 10000 {
		t.Fatalf("Mausi points=%d snaps=%d tackles=%d penalties=%d", total, sn, tk, pen)
	}
}

func TestPinnedSleeperDJJohnsonSparseWeek(t *testing.T) {
	raw := sleeper2025(t)
	raw.Stats[0].Sleeper = []source.RawSleeperWeek{{Season: 2025, Week: 1, Body: `{"10938":{"def_snp":16,"st_snp":13}}`}}
	p := reopenSleeper(t, raw)
	e := sleeperRead(t, p, "16277", 1)
	if e.Held == nil || e.Held.Week == nil {
		t.Fatalf("missing week: %+v", e)
	}
	line := e.Held.Week.Line
	if line.Quantities["defensive_snap"] != 160000 || line.Quantities["special_teams_snap"] != 130000 || sleeperScore(t, leaguerules.DE, line) != 1670 {
		t.Fatalf("DJ week: %+v", line)
	}
	if e.Held.Week.GSIS != "" {
		t.Fatal("invented GSIS in sleeper-only week")
	}
}

func TestPinnedSleeperPreservesNFLverse(t *testing.T) {
	raw := sleeper2025(t)
	p := raw.Players["0099"]
	p.Sleeper = "1234"
	raw.Players["0099"] = p
	raw.Stats[0].Plays = strings.Replace(raw.Stats[0].Plays, "play_deleted,two_point_attempt\n", "play_deleted,two_point_attempt,third_down_converted,fourth_down_converted,rush_attempt,pass_attempt,complete_pass,yardline_100,tackled_for_loss\n", 1)
	raw.Stats[0].Plays = strings.Replace(raw.Stats[0].Plays, "0,0,0,0,0\n", "0,0,0,0,0,1,0,0,1,1,10,0\n", 1)
	raw.Stats[0].Plays = strings.TrimSuffix(raw.Stats[0].Plays, "\n") + ",0,0,0,0,0,50,0\n"
	baseline := raw
	baseline.Stats = append([]RawFootballStats(nil), raw.Stats...)
	raw.Stats[0].Sleeper = []source.RawSleeperWeek{{Season: 2025, Week: 1, Body: `{"1234":{"off_snp":5,"passing_yards":9999,"third_down_passing_conversion":999},"99999":{"st_snp":9},"DAL":{"st_snp":40},"12944":{}}`}, {Season: 2025, Week: 2, Body: `{"12944":{"st_snp":0},"DAL":{"st_snp":40},"99999":{"st_snp":1}}`}}
	old := reopenSleeper(t, baseline)
	attached := reopenSleeper(t, raw)
	a, b := sleeperRead(t, old, "0099", 1), sleeperRead(t, attached, "0099", 1)
	if a.Held.Week == nil || b.Held.Week == nil {
		t.Fatal("missing nflverse week")
	}
	for _, id := range []leaguerules.StatID{"passing_yards", "rushing_yards", "third_down_passing_conversion", "red_zone_pass_attempt"} {
		if a.Held.Week.Line.Quantities[id] != b.Held.Week.Line.Quantities[id] {
			t.Fatalf("%s changed", id)
		}
	}
	if b.Held.Week.Line.Quantities["passing_yards"] != 2440000 || b.Held.Week.Line.Quantities["rushing_yards"] != -10000 {
		t.Fatal("nflverse quantities changed")
	}
	ordinary, err := attached.PlayerWeekStats(t.Context(), source.RawStatScope{GSIS: "00-0023459", Season: 2025, Week: 1})
	if err != nil || len(ordinary) != 1 || ordinary[0].Week == nil {
		t.Fatalf("ordinary read: %+v %v", ordinary, err)
	}
	if _, ok := ordinary[0].Week.Line.Quantities["offensive_snap"]; ok {
		t.Fatal("ordinary GSIS read changed")
	}
	if b.Held.Week.Line.Quantities["third_down_passing_conversion"] != 10000 || b.Held.Week.Line.Quantities["red_zone_pass_attempt"] != 10000 || b.Held.Week.Line.Quantities["offensive_snap"] != 50000 {
		t.Fatalf("missing overlap: %+v", b.Held.Week.Line.Quantities)
	}
	if sleeperRead(t, attached, "17367", 2).Held.Week != nil || attached.stats[0].unknownSleeper[2025] != 2 {
		t.Fatalf("zero or unknown handling: %+v", attached.stats[0].unknownSleeper)
	}
}

func TestPinnedSleeperRevisionAbsence(t *testing.T) {
	raw := sleeper2025(t)
	old := raw.Stats[0]
	old.AsOf = old.AsOf.Add(-time.Hour)
	old.Sleeper = []source.RawSleeperWeek{{Season: 2025, Week: 1, Body: `{"12944":{"st_snp":20}}`}}
	raw.Stats[0].Sleeper = []source.RawSleeperWeek{{Season: 2025, Week: 1, Body: `{}`}}
	raw.Stats = append(raw.Stats, old)
	e := sleeperRead(t, reopenSleeper(t, raw), "17367", 1)
	if e.Held == nil || e.Held.Week != nil || len(e.Lines) != 1 || e.Lines[0].Held.Week == nil {
		t.Fatalf("absence overwritten: %+v", e)
	}
}

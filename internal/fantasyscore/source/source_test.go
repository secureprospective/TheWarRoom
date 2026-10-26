package source

import (
	"encoding/csv"
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/fantasyscore"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

func rawFixture(t *testing.T) RawWeek {
	t.Helper()
	body, err := os.ReadFile("testdata/weeks.csv")
	if err != nil {
		t.Fatal(err)
	}
	rows, err := csv.NewReader(strings.NewReader(string(body))).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	fields := make(map[string]string)
	for i, field := range rows[0] {
		fields[field] = rows[1][i]
	}
	return RawWeek{Fields: fields}
}

func TestWeeklyMappingPreservesDirectStatsAndFieldGoalBands(t *testing.T) {
	raw := rawFixture(t)
	for field, value := range map[string]string{
		"def_tackles_solo": "3", "def_tackles_with_assist": "2", "def_tackle_assists": "4",
		"def_tackles_for_loss": "2", "def_sacks": "1.5", "carries": "7",
		"fg_made_0_19": "1", "fg_made_20_29": "1", "fg_made_30_39": "1",
		"fg_made_40_49": "1", "fg_made_50_59": "1", "fg_made_60_": "2",
		"fg_made_list": "19;29;39;49;59;60;99", "fg_missed_0_19": "1",
		"fg_missed_20_29": "1", "fg_missed_30_39": "1", "fg_missed_40_49": "1",
		"fg_missed_50_59": "1", "fg_missed_60_": "1", "pat_made": "3",
		"fg_blocked": "2", "fg_blocked_list": "29;30", "fg_missed_list": "18;25;31;45;55;61",
	} {
		raw.Fields[field] = value
	}
	week, err := raw.Convert()
	if err != nil {
		t.Fatal(err)
	}
	for id, want := range map[leaguerules.StatID]leaguerules.Decimal{
		"tackle": 50000, "assist": 40000,
		"tackle_for_loss": 20000, "sack": 15000, "rushing_attempt": 70000,
		"fg_made_0_39": 30000, "fg_made_40_49": 10000, "fg_made_50_59": 10000,
		"fg_made_60_99": 20000, "fg_missed_0_29": 30000, "fg_missed_30_99": 50000, "pat_made": 30000,
	} {
		if week.Line.Quantities[id] != want {
			t.Errorf("%s: got %d want %d", id, week.Line.Quantities[id], want)
		}
	}
	for id, want := range map[leaguerules.StatID][]leaguerules.Decimal{
		"fg_made_distance":   {190000, 290000, 390000, 490000, 590000, 600000, 990000},
		"fg_missed_distance": {180000, 250000, 310000, 450000, 550000, 610000, 290000, 300000},
	} {
		if !reflect.DeepEqual(week.Line.Events[id], want) {
			t.Errorf("%s: got %v want %v", id, week.Line.Events[id], want)
		}
	}
}

func TestRawWeekRejectsMissingMalformedAndUnsupportedEvidence(t *testing.T) {
	for _, tc := range []struct{ field, value string }{
		{"player_id", "00-abcdefg"}, {"season", "0"}, {"week", "-1"}, {"season_type", "ALL"},
		{"game_id", ""}, {"carries", ""}, {"carries", "-1"}, {"def_sacks", "NaN"}, {"def_tackles_for_loss", "1.00001"},
		{"fg_made_list", "100"}, {"fg_made_list", "-1"}, {"fg_made_60_", "1"},
		{"fg_blocked_list", "100"}, {"fg_blocked", "1"},
	} {
		t.Run(tc.field+tc.value, func(t *testing.T) {
			raw := rawFixture(t)
			raw.Fields["fg_made_60_"] = "0"
			raw.Fields["fg_made_list"] = ""
			raw.Fields[tc.field] = tc.value
			if _, err := raw.Convert(); err == nil {
				t.Fatal("invalid evidence entered the scorer")
			}
		})
	}
}

func TestDecimalParsingIsExact(t *testing.T) {
	for raw, want := range map[string]leaguerules.Decimal{"0": 0, "1.5": 15000, "-0.025": -250,
		"922337203685477.5807": 9223372036854775807, "-922337203685477.5808": -9223372036854775808} {
		got, err := decimal(raw)
		if err != nil || got != want {
			t.Errorf("%s: got %d error %v want %d", raw, got, err, want)
		}
	}
	for _, raw := range []string{"", "NaN", "Inf", "1e2", ".5", "1.", "1..0", "1.0.0", "1.00001",
		" 1", "1 ", "+1", "1.-5", "922337203685477.5808"} {
		if _, err := decimal(raw); err == nil {
			t.Errorf("accepted %q", raw)
		}
	}
}

func TestCSVBoundaryRejectsTruncatedDuplicateAndEmptyData(t *testing.T) {
	for _, raw := range []string{"", "id,id\n1,2\n", "id\n1,2\n", "id\n"} {
		if _, err := ReadWeeks(strings.NewReader(raw)); err == nil {
			t.Errorf("accepted invalid weeks %q", raw)
		}
		if _, err := ReadPlays(strings.NewReader(raw)); err == nil {
			t.Errorf("accepted invalid plays %q", raw)
		}
	}
	body, err := os.ReadFile("testdata/weeks.csv")
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(string(body), "\n")
	if _, err := ReadWeeks(strings.NewReader(lines[0] + "\n" + lines[1] + "\n" + lines[1] + "\n")); err == nil {
		t.Fatal("duplicate player-week accepted")
	}
	duplicateHeader := lines[0] + ",def_tackles_solo\n" + lines[1] + ",0\n"
	if _, err := ReadWeeks(strings.NewReader(duplicateHeader)); err == nil {
		t.Fatal("duplicate header replaced a credited stat")
	}
}

func TestPlayDistancesTouchdownKindsAndScope(t *testing.T) {
	week, err := rawFixture(t).Convert()
	if err != nil {
		t.Fatal(err)
	}
	base := RawPlay{GameID: week.GameID, PlayID: "1", Season: "2025", Week: "1", SeasonType: "REG",
		Passer: week.GSIS, Rusher: week.GSIS, Receiver: week.GSIS, Scorer: week.GSIS,
		Passing: "41", Receiving: "41", Rushing: "25", Touchdown: "1", Interception: "1",
		ScoringTeam: "CHI", ReturnTeam: "CHI"}
	plays := []RawPlay{base, base, base, base, base, base, base}
	for i := range plays {
		plays[i].PlayID = string(rune('1' + i))
	}
	plays[1].Interception, plays[1].Punt = "0", "1"
	plays[2].Interception, plays[2].Kickoff = "0", "1"
	plays[3].Deleted = "1"
	plays[4].TwoPoint = "1"
	plays[5].GameID = "another game"
	// The kicking team recovers its own kickoff in the end zone (2025_02_SEA_PIT play 3004).
	plays[6].Interception, plays[6].Kickoff, plays[6].ReturnTeam = "0", "1", ""
	plays[6].Rushing = "NA"
	weeks := []PlayerWeek{week}
	if err := AttachPlays(weeks, plays); err != nil {
		t.Fatal(err)
	}
	for _, id := range []leaguerules.StatID{"interception_return_td", "punt_return_td", "kickoff_return_td"} {
		if weeks[0].Line.Quantities[id] != 10000 {
			t.Errorf("wrong touchdown attribution %s: %v", id, weeks[0].Line.Quantities[id])
		}
	}
	if !reflect.DeepEqual(weeks[0].Line.Events["long_rush"], []leaguerules.Decimal{250000, 250000, 250000}) {
		t.Fatal("distance lost or deleted/2-point/wrong-game play counted")
	}
	if !reflect.DeepEqual(weeks[0].Line.Events["chunk_play"],
		[]leaguerules.Decimal{250000, 410000, 250000, 410000, 250000, 410000, 410000}) {
		t.Fatalf("chunk plays are every counted rush and catch, never a pass: %v", weeks[0].Line.Events["chunk_play"])
	}
	if err := AttachPlays(weeks, plays); err != nil || len(weeks[0].Line.Events["long_rush"]) != 3 ||
		len(weeks[0].Line.Events["chunk_play"]) != 7 {
		t.Fatal("re-attaching held plays counted twice", err)
	}
}

func TestMissingOrMalformedPlayEvidenceFailsBeforeMutation(t *testing.T) {
	week, err := rawFixture(t).Convert()
	if err != nil {
		t.Fatal(err)
	}
	base := RawPlay{GameID: week.GameID, PlayID: "1", Season: "2025", Week: "1", SeasonType: "REG"}
	bad := base
	bad.Passing = "NaN"
	converted := len(week.Line.Events)
	for _, plays := range [][]RawPlay{nil, {bad}, {base, base}} {
		weeks := []PlayerWeek{week}
		if err := AttachPlays(weeks, plays); err == nil || len(weeks[0].Line.Events) != converted {
			t.Fatalf("bad evidence accepted or partially mutated: %v", err)
		}
	}
	if err := AttachPlays([]PlayerWeek{{Line: fantasyscore.StatLine{}}}, []RawPlay{base}); err == nil {
		t.Fatal("incomplete player-week accepted")
	}
}

func TestKickRecoveryTDIsNotAReturnAndAttachesOnce(t *testing.T) {
	week, err := rawFixture(t).Convert()
	if err != nil {
		t.Fatal(err)
	}
	week.RecoveryTDs = 10000
	base := RawPlay{GameID: week.GameID, PlayID: "1", Season: "2025", Week: "1", SeasonType: "REG",
		Scorer: week.GSIS, Touchdown: "1", ScoringTeam: "SEA", Kickoff: "1"}
	weeks := []PlayerWeek{week}
	for range 2 {
		if err := AttachPlays(weeks, []RawPlay{base}); err != nil {
			t.Fatal(err)
		}
		if weeks[0].Line.Quantities["fumble_recovery_tds"] != 20000 ||
			weeks[0].Line.Quantities["kickoff_return_td"] != 0 {
			t.Fatalf("credits changed: %+v", weeks[0].Line)
		}
	}
}

func TestReadWeeksLeavesOutTeamRowsThatNameNoPlayer(t *testing.T) {
	body, err := os.ReadFile("testdata/weeks.csv")
	if err != nil {
		t.Fatal(err)
	}
	rows, err := csv.NewReader(strings.NewReader(string(body))).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	held, err := ReadWeeks(strings.NewReader(string(body)))
	if err != nil {
		t.Fatal(err)
	}
	withRow := func(id string) string {
		team := slices.Clone(rows[1])
		for i, name := range rows[0] {
			switch name {
			case "player_id":
				team[i] = id
			case "player_name", "player_display_name", "position", "position_group", "headshot_url":
				team[i] = ""
			}
		}
		var out strings.Builder
		w := csv.NewWriter(&out)
		if err := w.WriteAll(append(slices.Clone(rows), team)); err != nil {
			t.Fatal(err)
		}
		return out.String()
	}
	weeks, err := ReadWeeks(strings.NewReader(withRow("")))
	if err != nil || len(weeks) != len(held) {
		t.Fatalf("a team row that names no player: %d weeks, %v", len(weeks), err)
	}
	if _, err := ReadWeeks(strings.NewReader(withRow("00-abcdefg"))); err == nil {
		t.Fatal("a malformed player id entered the scorer")
	}
}

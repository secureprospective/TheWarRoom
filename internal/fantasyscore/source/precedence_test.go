package source_test

import (
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/fantasyscore"
	"github.com/secureprospective/TheWarRoom/internal/fantasyscore/source"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

func TestScoreSourcePrecedence(t *testing.T) {
	line := func(value leaguerules.Decimal) fantasyscore.StatLine {
		return fantasyscore.StatLine{Quantities: map[leaguerules.StatID]leaguerules.Decimal{"tackle": value}}
	}
	for _, tc := range []struct {
		name   string
		inputs map[string]fantasyscore.StatLine
		want   leaguerules.Decimal
		owner  string
	}{
		{"ESPN overlap", map[string]fantasyscore.StatLine{source.ESPNLive: line(7), source.Sleeper: line(9)}, 7, source.ESPNLive},
		{"ESPN explicit zero", map[string]fantasyscore.StatLine{source.ESPNLive: line(0), source.Sleeper: line(9)}, 0, source.ESPNLive},
		{"Sleeper only", map[string]fantasyscore.StatLine{source.Sleeper: line(9)}, 9, source.Sleeper},
		{"nflverse posted", map[string]fantasyscore.StatLine{source.NFLVerse: line(5), source.ESPNLive: line(7), source.Sleeper: line(9)}, 5, source.NFLVerse},
		{"MFL ignored on outage", map[string]fantasyscore.StatLine{source.MFLLive: line(6), source.Sleeper: line(9)}, 9, source.Sleeper},
		{"MFL cannot override ESPN", map[string]fantasyscore.StatLine{source.MFLLive: line(6), source.ESPNLive: line(7)}, 7, source.ESPNLive},
		{"MFL cannot fill ESPN gaps", map[string]fantasyscore.StatLine{source.MFLLive: line(6), source.ESPNLive: {}, source.Sleeper: line(9)}, 9, source.Sleeper},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, owners := source.MergeStatSources(tc.inputs)
			if value, ok := got.Quantities["tackle"]; !ok || value != tc.want || owners["tackle"] != tc.owner {
				t.Fatalf("%+v %+v", got, owners)
			}
		})
	}
}

func TestScoreSourceEventsAndGapOwnership(t *testing.T) {
	primary := fantasyscore.StatLine{Events: map[leaguerules.StatID][]leaguerules.Decimal{"distance": {5}}}
	gap := fantasyscore.StatLine{Quantities: map[leaguerules.StatID]leaguerules.Decimal{"distance": 99, "receiving_drop": 1}}
	line, owners := source.MergeStatSources(map[string]fantasyscore.StatLine{source.ESPNLive: primary, source.Sleeper: gap})
	if _, exists := line.Quantities["distance"]; exists || line.Events["distance"][0] != 5 || owners["receiving_drop"] != source.Sleeper {
		t.Fatalf("%+v %+v", line, owners)
	}
	if source.LiveAuthority(owners) != "ESPN live + Sleeper" {
		t.Fatal(source.LiveAuthority(owners))
	}
	line.Events["distance"][0] = 8
	if primary.Events["distance"][0] != 5 {
		t.Fatal("merge mutated input")
	}
	empty, _ := source.MergeStatSources(nil)
	if len(empty.Quantities) != 0 || len(empty.Events) != 0 {
		t.Fatal("missing player fabricated zero")
	}
}

func TestLiveSleeperPresentFields(t *testing.T) {
	lines, err := source.LiveSleeperLines(source.RawSleeperWeek{Season: 2026, Week: 8, Body: `{"1":{"pass_yd":0,"rec_drop":2,"off_snp":5},"2":{}}`})
	if err != nil {
		t.Fatal(err)
	}
	line := lines["1"]
	if len(line.Quantities) != 3 || line.Quantities["receiving_drop"] != 20000 || line.Quantities["offensive_snap"] != 50000 {
		t.Fatal(line)
	}
	if value, ok := line.Quantities["passing_yards"]; !ok || value != 0 {
		t.Fatal("lost explicit zero")
	}
	if _, exists := lines["2"]; exists {
		t.Fatal("empty player became zero")
	}
}

func TestLiveSleeperRejectsInvalidCore(t *testing.T) {
	for _, body := range []string{`{"1":{"pass_att":-1}}`, `{"1":{"pass_yd":1.12345}}`, `{"1":null}`, `null`, `{"1":{"pass_yd":"NaN"}}`} {
		if _, err := source.LiveSleeperLines(source.RawSleeperWeek{Season: 2026, Week: 8, Body: body}); err == nil {
			t.Errorf("accepted %s", body)
		}
	}
}

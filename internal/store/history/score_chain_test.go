package history

import (
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/fantasyscore"
	"github.com/secureprospective/TheWarRoom/internal/fantasyscore/source"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

func TestPinnedSleeperFillsWithoutReplacingFinalField(t *testing.T) {
	canonical := map[canonicalWeek]source.PlayerWeek{{"0099", 8}: {Line: fantasyscore.StatLine{Quantities: map[leaguerules.StatID]leaguerules.Decimal{"offensive_snap": 0}}}}
	raw := RawFootballEvidence{}
	_, err := raw.mergeSleeperWeek(source.RawSleeperWeek{Season: 2026, Week: 8, Body: `{"1":{"off_snp":9,"rec_drop":2}}`}, map[string]string{"1": "0099"}, canonical)
	if err != nil {
		t.Fatal(err)
	}
	got := canonical[canonicalWeek{"0099", 8}]
	if got.Line.Quantities["offensive_snap"] != 0 || got.Line.Quantities["receiving_drop"] != 20000 {
		t.Fatal(got)
	}
}

func TestLiveGapProvenanceAndMissingFinalPlayer(t *testing.T) {
	s := &Store{}
	line := liveLine(30000)
	line.Provenance = "ESPN live + Sleeper"
	s.SetLiveStats(2026, 8, time.Now(), source.ESPNLive, nil, map[string]source.PlayerWeek{"0099": line})
	absent := source.HeldWeek{Authority: source.NFLVerse, Missing: []string{"player absent"}}
	evidence := s.withLiveStats(source.NativeEvidence{Held: &absent}, source.RawNativeScope{Player: "0099", Season: 2026, Week: 8})
	selected := countedLive(t, evidence)
	if selected.Held.Authority != line.Provenance || selected.Held.Week.Line.Quantities["receiving_yards"] != 30000 {
		t.Fatal(selected)
	}
	empty := s.withLiveStats(source.NativeEvidence{Held: &absent}, source.RawNativeScope{Player: "0100", Season: 2026, Week: 8})
	if empty.Held.Week != nil || len(empty.Versions()) != 0 {
		t.Fatal("missing player fabricated zero")
	}
}

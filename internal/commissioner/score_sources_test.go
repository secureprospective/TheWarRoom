package commissioner

import (
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/fantasyscore/source"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

func TestScoreSourceMergedRevisionAndOverride(t *testing.T) {
	b := seededLedger(t)
	live, final := tackleVersion(0, "LB", 10000, 5), tackleVersion(1, "LB", 20000, 6)
	live.Held.Authority, final.Held.Authority = source.ESPNLive, source.NFLVerse
	live.Held.Week.Line.Quantities["special_teams_solo_tackle"] = 30000
	live.Held.Week.FieldSources = map[leaguerules.StatID]string{"special_teams_solo_tackle": source.Sleeper}
	live.Digest = source.LineDigest(*live.Held.Week)
	versions := source.ConsolidateFinal([]source.LineRevision{live, final})
	scope := source.RawNativeScope{Player: "0099", Season: 2026, Week: 1}
	read := func() NativePlayerScore {
		got, err := NativePlayerScoreReading("native.test", scope, b.records, statWeek(versions...).Evidence, seasonMoment(20))
		if err != nil {
			t.Fatal(err)
		}
		return got
	}
	moves, err := FeedStatMoves("native.test", b.records, []StatWeek{statWeek(versions...)}, 2026)
	if err != nil || len(moves) != 1 {
		t.Fatalf("moves: %+v %v", moves, err)
	}
	if err := CheckFeedStatAppend(moves[0], b.records, StatFacts{Versions: versions}); err != nil {
		t.Fatal(err)
	}
	got := read()
	if got.Source != "final · nflverse" || got.FieldSources["tackle"] != "final · nflverse" || got.FieldSources["special_teams_solo_tackle"] != "filled · Sleeper" {
		t.Fatal(got)
	}
	before := *got.Points
	b.statCommand(StatCorrect, RawStatChange{Player: "0099", Season: 2026, Week: 1, Stat: "tackle", Value: decimalOf(90000), Source: "gamebook"}, 12)
	got = read()
	if got.Points == nil || *got.Points <= before || !strings.Contains(got.Source, "commish override") || got.FieldSources["tackle"] != "commish override" {
		t.Fatal(got)
	}
	b.statCommand(StatApprove, RawStatChange{Item: itemOf(live)}, 13)
	got = read()
	if got.Source != "commish override" || got.FieldSources["special_teams_solo_tackle"] != "commish override" {
		t.Fatal(got)
	}
	if selected, err := countedVersion(versions, StatBaseline{Seq: 0, Digest: live.Digest, Entry: "manual"}); err != nil || selected.Digest != live.Digest {
		t.Fatalf("pin changed: %+v %v", selected, err)
	}
}

func TestScoreSourceLabelsAndMFLExclusion(t *testing.T) {
	for authority, want := range map[string]string{source.ESPNLive: "live · ESPN", source.Sleeper: "filled · Sleeper", source.NFLVerse: "final · nflverse"} {
		t.Run(authority, func(t *testing.T) {
			if got := scoreSourceLabel(authority); got != want {
				t.Fatal(got)
			}
		})
	}
	v := tackleVersion(0, "LB", 10000, 5)
	v.Held.Authority = source.MFLLive
	if FeedVersion([]source.LineRevision{v}) != nil {
		t.Fatal("MFL counted")
	}
}

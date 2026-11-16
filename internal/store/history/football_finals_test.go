package history

import (
	"io"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/fantasyscore/source"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

// officialFixture is a league pinned the way an MFL import pins it (MFL ids alone, no statistics) and
// nflverse season files for its week 1: the quarterback and kicker have rows, the bench player none.
func officialFixture(t *testing.T, ended bool) (*Store, *PinnedFootball, func() OfficialSeason) {
	t.Helper()
	s, _ := statStore(t, filepath.Join(t.TempDir(), "history.db"))
	raw := scoringImport(t)
	weekly, plays := raw.Stats[0].Weekly, raw.Stats[0].Plays
	raw.Stats = nil
	bench := raw.Players["0099"]
	bench.Facts.Name = "Bench"
	raw.Players["0103"] = bench
	for id, p := range raw.Players {
		p.Identity = MFLIdentityPrefix + id
		raw.Players[id] = p
	}
	pinned := installScoringImport(t, s, raw)
	lines := strings.Split(strings.TrimSuffix(plays, "\n"), "\n")
	lines[0] += ",desc"
	for i := 1; i < len(lines); i++ {
		lines[i] += ","
	}
	if ended {
		lines = append(lines, "2025_01_PIT_NYJ,9,2026,1,REG,,,,NA,NA,NA,0,,,,0,0,0,0,0,END GAME",
			"2025_01_BAL_BUF,9,2026,1,REG,,,,NA,NA,NA,0,,,,0,0,0,0,0,END GAME")
	}
	after := raw.Weeks[0].Games[0].Kickoff.Add(officialGrace + time.Minute)
	if after.Before(raw.AsOf) {
		after = raw.AsOf.Add(time.Minute)
	}
	season := func() OfficialSeason {
		return OfficialSeason{Weekly: strings.NewReader(weekly), Plays: strings.NewReader(strings.Join(lines, "\n") + "\n"),
			At: after, GSIS: map[string]string{"0099": "00-0023459", "0100": "00-0023853", "0103": "00-0039999"}}
	}
	return s, pinned, season
}

// A finished week becomes official: its players are scored from nflverse's final numbers, a player the
// week does not list has a final line of no statistics, and a live line no longer reopens it.
func TestOfficialWeekPinsFinalStatistics(t *testing.T) {
	s, before, season := officialFixture(t, true)
	got, err := s.PinOfficialStats(t.Context(), before, season())
	if err != nil || got.Digest == "" || len(got.Weeks) != 1 || got.Weeks[0] != 1 {
		t.Fatalf("pin official week 1: %+v %v", got, err)
	}
	pinned, err := s.LoadFootball(t.Context(), got.Digest)
	if err != nil {
		t.Fatal(err)
	}
	s.SetPinnedFootball(pinned)
	for player, want := range map[string]bool{"0099": true, "0103": false} {
		held, err := pinned.CanonicalPlayerWeekStats(t.Context(), source.RawNativeScope{Player: player, Season: 2026, Week: 1})
		if err != nil || len(held) != 1 || held[0].Week == nil || len(held[0].Missing) != 0 {
			t.Fatalf("%s: %+v %v", player, held, err)
		}
		if listed := len(held[0].Week.Line.Quantities) != 0; listed != want {
			t.Fatalf("%s: listed %v, want %v (%+v)", player, listed, want, held[0].Week)
		}
	}
	live := &LiveGeneration{Season: 2026, Week: 1, Authority: "ESPN", At: season().At, Missing: []string{"provisional"},
		Identities: map[string]LiveIdentity{"0099": {GSIS: "00-0023459"}},
		Lines:      map[string]source.PlayerWeek{"0099": {GSIS: "00-0023459", Season: 2026, Week: 1, SeasonType: "REG"}}}
	scope := source.RawNativeScope{Player: "0099", Season: 2026, Week: 1}
	for week, reopens := range map[int]bool{1: false, 2: true} {
		live.Week, scope.Week = week, week
		if out := s.overlayEvidence(source.NativeEvidence{GSIS: "00-0023459"}, scope, live); (out.Held != nil) != reopens {
			t.Fatalf("week %d: live line applied %v, want %v", week, out.Held != nil, reopens)
		}
	}
	again, err := s.PinOfficialStats(t.Context(), pinned, season())
	if err != nil || again.Digest != "" {
		t.Fatalf("the same official files pinned again: %+v %v", again, err)
	}
	early := season()
	early.At = before.raw.Weeks[0].Games[0].Kickoff.Add(time.Hour)
	if got, err := s.PinOfficialStats(t.Context(), before, early); err != nil || got.Digest != "" {
		t.Fatalf("a week an hour after kickoff was taken as official: %+v %v", got, err)
	}
}

// A stat correction adds a revision holding only the corrected week and keeps the earlier one, so the
// version the commissioner may have approved is still held beside the corrected one.
func TestOfficialCorrectionKeepsTheEarlierVersion(t *testing.T) {
	s, before, season := officialFixture(t, true)
	first, err := s.PinOfficialStats(t.Context(), before, season())
	if err != nil {
		t.Fatal(err)
	}
	pinned, err := s.LoadFootball(t.Context(), first.Digest)
	if err != nil {
		t.Fatal(err)
	}
	corrected := season()
	weekly, _ := io.ReadAll(corrected.Weekly)
	corrected.Weekly = strings.NewReader(strings.Replace(string(weekly), ",244,", ",250,", 1))
	corrected.At = corrected.At.Add(time.Hour)
	second, err := s.PinOfficialStats(t.Context(), pinned, corrected)
	if err != nil || second.Digest == "" {
		t.Fatalf("a correction was not pinned: %+v %v", second, err)
	}
	after, err := s.LoadFootball(t.Context(), second.Digest)
	if err != nil {
		t.Fatal(err)
	}
	if len(after.raw.Stats) != 2 || len(after.raw.Stats[0].Parts) != 1 {
		t.Fatalf("revisions after a correction: %+v", after.raw.Stats)
	}
	held, err := after.CanonicalPlayerWeekStats(t.Context(), source.RawNativeScope{Player: "0099", Season: 2026, Week: 1})
	if err != nil || len(source.DistinctLines(held)) != 2 {
		t.Fatalf("versions of the corrected week: %d %v", len(source.DistinctLines(held)), err)
	}
}

// A week whose games have not all ended stays live: nothing is pinned and the reason is given.
func TestUnfinishedWeekIsNotOfficial(t *testing.T) {
	s, before, season := officialFixture(t, false)
	got, err := s.PinOfficialStats(t.Context(), before, season())
	if err != nil || got.Digest != "" || len(got.Skipped) != 1 || !strings.Contains(got.Skipped[0], "0 of 2 games have ended") {
		t.Fatalf("an unfinished week: %+v %v", got, err)
	}
}

// A player the directory cannot identify takes the one recent nflverse player of his name and position
// group; two such players, or an old namesake, identify nobody.
func TestRosterIdentifiesOnlyOneRecentNamesake(t *testing.T) {
	roster, err := readRoster(strings.NewReader("gsis_id,display_name,position_group,last_season\n"+
		"THO581952,James Thompson,WR,1978\n00-0041363,James Thompson Jr.,DL,2026\n"+
		"00-0040001,Chris Jones,DL,2026\n00-0040002,Chris Jones,DL,2025\n00-0040003,Zion Childress,DB,2026\n"), 2026)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		name     string
		position leaguerules.Position
		want     string
	}{{"Thompson Jr., James", leaguerules.DT, "00-0041363"}, {"Thompson, James", leaguerules.WR, ""},
		{"Jones, Chris", leaguerules.DT, ""}, {"Childress, Zion", leaguerules.S, "00-0040003"},
		{"Childress, Zion", leaguerules.RB, ""}} {
		if got := roster.match(c.name, c.position); got != c.want {
			t.Errorf("%s (%s): %q, want %q", c.name, c.position, got, c.want)
		}
	}
}

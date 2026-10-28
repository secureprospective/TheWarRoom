package commissioner

import (
	"strings"
	"testing"
	"time"
)

func bracketRaw(command string, change RawBracketChange) RawCommand {
	return RawCommand{League: "native.x", Command: command, Args: RawArgs{Bracket: &change}, Request: "r1",
		Effective: "now", Visibility: "league", Reason: "Set the playoffs"}
}

func TestPlayoffCommandsCarryTheCensusGravity(t *testing.T) {
	build := RawBracketChange{Season: 2026, Preset: presetLeagueRules}
	edit := RawBracketChange{Season: 2026, Node: "AFC/wildcard_2/away", Team: "franchise.01"}
	advance := RawBracketChange{Season: 2026, Game: "AFC/wildcard_2", Team: "franchise.01"}
	cases := []struct {
		name string
		raw  RawCommand
		want string
	}{
		{"build before kickoff", bracketRaw(BracketBuild, build), "G2"},
		{"edit before kickoff", bracketRaw(BracketEdit, edit), "G2"},
		{"advance", bracketRaw(BracketAdvance, advance), "G3"},
		{"title", RawCommand{Command: TitleSet}, "G1"},
		{"link", RawCommand{Command: HistoryLink}, "G2"},
	}
	started := edit
	started.Started = true
	cases = append(cases, struct {
		name string
		raw  RawCommand
		want string
	}{"edit after kickoff", bracketRaw(BracketEdit, started), "G3"})
	for _, c := range cases {
		got, err := c.raw.EffectiveDefinition()
		if err != nil || got.Gravity != c.want {
			t.Fatalf("%s: %q %v, want %s", c.name, got.Gravity, err, c.want)
		}
	}
	raw := bracketRaw(BracketEdit, started)
	raw.Reason = " "
	if err := raw.Validate(); err == nil || !strings.Contains(err.Error(), "reason") {
		t.Fatalf("a change after kickoff needs a typed reason: %v", err)
	}
	raw = bracketRaw(BracketEdit, edit)
	raw.Gravity = "G1"
	if err := raw.Validate(); err == nil {
		t.Fatal("a safeguard was lowered")
	}
}

func TestBracketCommandsRefuseWhatDoesNotBelongToThem(t *testing.T) {
	good := map[string]RawBracketChange{
		BracketBuild:      {Season: 2026, Preset: PresetLegacy},
		BracketEdit:       {Season: 2026, Node: "conference/home", Team: "franchise.01"},
		BracketSeed:       {Season: 2026, Slot: 2, Team: "franchise.01"},
		BracketAdvance:    {Season: 2026, Game: "championship", Team: "franchise.01"},
		BracketPresetSave: {Season: 2026, Name: "small", Shape: []BracketNodeSpec{{ID: "final", Week: 16, HomeSeed: 1, AwaySeed: 2}}},
	}
	for command, change := range good {
		if err := bracketRaw(command, change).Validate(); err != nil {
			t.Fatalf("%s: %v", command, err)
		}
	}
	bad := map[string]RawBracketChange{
		BracketBuild:      {Season: 2026, Preset: "Bad Name"},
		BracketEdit:       {Season: 2026, Node: "conference", Team: "franchise.01"},
		BracketSeed:       {Season: 2026, Slot: 0, Team: "franchise.01"},
		BracketAdvance:    {Season: 2026, Game: "championship", Team: "01"},
		BracketPresetSave: {Season: 2026, Name: PresetLegacy, Shape: good[BracketPresetSave].Shape},
	}
	for command, change := range bad {
		if err := bracketRaw(command, change).Validate(); err == nil {
			t.Fatalf("%s accepted %+v", command, change)
		}
	}
	foreign := good[BracketSeed]
	foreign.Game = "championship"
	both := good[BracketPresetSave]
	both.Shape = []BracketNodeSpec{{ID: "final", Week: 16, HomeSeed: 1, HomeFrom: "winner of x", AwaySeed: 2}}
	started := good[BracketAdvance]
	started.Started = true
	for name, change := range map[string]RawBracketChange{"another command's field": foreign, "a seed and a rule": both} {
		command := BracketSeed
		if name == "a seed and a rule" {
			command = BracketPresetSave
		}
		if err := bracketRaw(command, change).Validate(); err == nil {
			t.Fatalf("%s accepted", name)
		}
	}
	if err := bracketRaw(BracketAdvance, started).Validate(); err == nil {
		t.Fatal("an advancement may not claim to be a pre-kickoff change either way")
	}
	private := bracketRaw(BracketBuild, good[BracketBuild])
	private.Visibility = "private"
	if err := private.Validate(); err == nil {
		t.Fatal("a bracket change is shown to every owner")
	}
}

func TestTheTitleNeverCarriesAStaleYear(t *testing.T) {
	title := func(season int, label string) error {
		raw := RawCommand{League: "native.x", Command: TitleSet, Args: RawArgs{Title: &RawTitleChange{Season: season, Label: label}},
			Request: "t", Effective: "now", Visibility: "league"}
		return raw.Validate()
	}
	for _, label := range []string{"", "Legacy Champions", "{year} Super Bowl Champion", "2026 Super Bowl Champion"} {
		if err := title(2026, label); err != nil {
			t.Fatalf("%q: %v", label, err)
		}
	}
	for _, label := range []string{"2025 Super Bowl Champion", "Champions 2019 and 2026", " padded", strings.Repeat("x", 81)} {
		if err := title(2026, label); err == nil {
			t.Fatalf("%q accepted for 2026", label)
		}
	}
	if DefaultTitle(2031) != "2031 League Champion" {
		t.Fatalf("default title: %s", DefaultTitle(2031))
	}
}

func TestAHistoryLinkNamesARecordAndAFranchise(t *testing.T) {
	link := func(record, franchise string) error {
		raw := RawCommand{League: "native.x", Command: HistoryLink, Args: RawArgs{Link: &RawHistoryLink{Record: record, Franchise: franchise}},
			Request: "l", Effective: "now", Visibility: "league"}
		return raw.Validate()
	}
	for _, record := range []string{"champion.2019", "season.2020", "archive.super-bowl-2021"} {
		if err := link(record, "franchise.01"); err != nil {
			t.Fatalf("%s: %v", record, err)
		}
	}
	for _, record := range []string{"champion.19", "champion.x", "winner.2019", "archive.", "archive.Has Space", ""} {
		if err := link(record, "franchise.01"); err == nil {
			t.Fatalf("%q accepted", record)
		}
	}
	if err := link("champion.2019", "bob"); err == nil {
		t.Fatal("a link to something that is not a franchise was accepted")
	}
}

func TestTheFirstKickoffOfAWeekIsItsEarliestGame(t *testing.T) {
	at := time.Date(2026, 12, 6, 18, 0, 0, 0, time.UTC)
	facts := LineupFacts{ScheduleHeld: true, Locks: map[string]time.Time{"KC": at.Add(3 * time.Hour), "BUF": at}}
	if got, known := firstKickoff(facts); !known || !got.Equal(at) {
		t.Fatalf("first kickoff %v %v", got, known)
	}
	if _, known := firstKickoff(LineupFacts{Locks: facts.Locks}); known {
		t.Fatal("kickoffs that are not held are not known")
	}
	if _, known := firstKickoff(LineupFacts{ScheduleHeld: true}); known {
		t.Fatal("a week with no games has no kickoff")
	}
}

func TestBracketLabelsReadInPlainWords(t *testing.T) {
	name := func(id string) string { return "The " + id }
	got := bracketLabel(BracketEdit, RawBracketChange{Season: 2026, Node: "AFC/wildcard_2/away", Team: "a"}, name)
	want := "Put The a in as the away team of the AFC wildcard 2 game for the season 2026 playoffs"
	if got != want {
		t.Fatalf("%q", got)
	}
	got = bracketLabel(BracketAdvance, RawBracketChange{Season: 2026, Game: "championship", Team: "a"}, name)
	if strings.ContainsAny(got, "/_") || !strings.Contains(got, "championship game") {
		t.Fatalf("%q", got)
	}
}

package commissioner

import (
	"fmt"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/governance"
)

const lineupBody = `{"league":"native.test","command":"commish.lineup.set","args":{"lineup":{"season":2026,` +
	`"franchise":"franchise.01","week":1,"starters":["1001","1002"]}},"reason":"","expected":3,"request":"set-1",` +
	`"effective":"now","visibility":"private"}`

func TestLineupCommandBoundary(t *testing.T) {
	raw, err := Decode(lineupBody)
	if err != nil || raw.Command != LineupSet || raw.Args.Lineup.Starters[1] != "1002" {
		t.Fatalf("a good lineup was refused: %+v %v", raw, err)
	}
	if def, _ := DefinitionFor(LineupSet); def.Gravity != "G2" {
		t.Fatalf("an owner's lineup is G2, not %s", def.Gravity)
	}
	if def, _ := DefinitionFor(LineupCorrect); def.Gravity != "G3" {
		t.Fatalf("a correction is G3, not %s", def.Gravity)
	}
	correct := strings.Replace(strings.Replace(lineupBody, "lineup.set", "lineup.correct", 1), `"visibility":"private"`, `"visibility":"league"`, 1)
	for name, body := range map[string]string{
		"a correction without its reason":    correct,
		"a lineup shown to the league":       strings.Replace(lineupBody, `"private"`, `"league"`, 1),
		"a lineup that waits for rollover":   strings.Replace(lineupBody, `"effective":"now"`, `"effective":"next_rollover"`, 1),
		"a week out of range":                strings.Replace(lineupBody, `"week":1`, `"week":23`, 1),
		"a season out of range":              strings.Replace(lineupBody, `"season":2026`, `"season":0`, 1),
		"an app franchise that isn't one":    strings.Replace(lineupBody, `"franchise.01"`, `"team 1"`, 1),
		"the same starter twice":             strings.Replace(lineupBody, `"1002"`, `"1001"`, 1),
		"a player id without its zeros":      strings.Replace(lineupBody, `"1002"`, `"099"`, 1),
		"no starters":                        strings.Replace(lineupBody, `["1001","1002"]`, `[]`, 1),
		"a score argument beside the lineup": strings.Replace(lineupBody, `"args":{`, `"args":{"score":{"week":1},`, 1),
		"an unknown lineup field":            strings.Replace(lineupBody, `"week":1,`, `"week":1,"bench":["1"],`, 1),
	} {
		if _, err := Decode(body); err == nil {
			t.Fatalf("accepted %s", name)
		}
	}
	if _, err := Decode(strings.Replace(correct, `"reason":""`, `"reason":"The outage ate the lineup"`, 1)); err != nil {
		t.Fatalf("a correction with its reason was refused: %v", err)
	}
}

func lineupRecord(entry, command string, week int, starters ...string) Record {
	return Record{Receipt: Receipt{Entry: entry}, Principal: "someone", Command: RawCommand{Command: command,
		Reason: "because", Args: RawArgs{Lineup: &RawLineupChange{Season: 2026, Franchise: "franchise.01", Week: week,
			Starters: starters}}}}
}

func revertRecord(entry, of string) Record {
	return Record{Receipt: Receipt{Entry: entry}, Command: RawCommand{Command: Revert, Args: RawArgs{Entry: of}}}
}

func TestRecordedLineupIsTheLatestActiveOneAndCompensationRestoresTheOneBefore(t *testing.T) {
	records := []Record{lineupRecord("a", LineupSet, 1, "1001"), lineupRecord("b", LineupSet, 1, "1002"),
		lineupRecord("other-week", LineupSet, 2, "1003")}
	stands := func(want ...string) {
		t.Helper()
		got, found, err := RecordedLineupFor(records, "franchise.01", 2026, 1)
		if err != nil || !found || fmt.Sprint(got.Starters) != fmt.Sprint(want) {
			t.Fatalf("the standing lineup is %v (%v, %v), want %v", got.Starters, found, err, want)
		}
	}
	stands("1002")
	records = append(records, revertRecord("r1", "b"))
	stands("1001")
	records = append(records, revertRecord("r2", "r1"))
	stands("1002")
	records = append(records, lineupRecord("c", LineupCorrect, 1, "1004"))
	if got, _, _ := RecordedLineupFor(records, "franchise.01", 2026, 1); !got.Corrected || got.Starters[0] != "1004" {
		t.Fatalf("a correction must be marked as the commissioner's: %+v", got)
	}
	records = append(records, revertRecord("r3", "a"), revertRecord("r4", "b"), revertRecord("r5", "c"))
	if _, found, _ := RecordedLineupFor(records, "franchise.01", 2026, 1); found {
		t.Fatal("every lineup was reversed, so none should stand")
	}
	if lineup, err := LedgerWeekLineup(records, "franchise.01", 2026, 1); err != nil || lineup.Set {
		t.Fatalf("a franchise with no standing lineup has none: %+v %v", lineup, err)
	}
}

func lineupGovernance() []Record {
	create := Record{Receipt: Receipt{Entry: "create"}, Principal: "boss", Command: RawCommand{League: "native.test",
		Command: Create, Args: RawArgs{Name: "T", Franchises: []string{"franchise.01", "franchise.02"}}}}
	seat := func(entry, franchise, owner string) Record {
		return Record{Receipt: Receipt{Entry: entry}, Principal: "boss", Command: RawCommand{League: "native.test",
			Command: "commish.seat.replace", Args: RawArgs{People: &RawPeople{Franchise: franchise, Principal: owner}}}}
	}
	grant := Record{Receipt: Receipt{Entry: "grant"}, Principal: "boss", Command: RawCommand{League: "native.test",
		Command: "commish.role.grant", Args: RawArgs{People: &RawPeople{Principal: "deputy", Role: governance.CoCommissioner}}}}
	return []Record{create, seat("seat1", "franchise.01", "owner-1"), seat("seat2", "franchise.02", "owner-2"), grant}
}

func TestOnlyTheSeatedOwnerSetsALineupAndOnlyACommissionerCorrectsOne(t *testing.T) {
	records := lineupGovernance()
	set := lineupRecord("x", LineupSet, 1, "1001").Command
	set.League = "native.test"
	correct := lineupRecord("y", LineupCorrect, 1, "1001").Command
	correct.League = "native.test"
	for principal, wantSet := range map[string]bool{"owner-1": true, "owner-2": false, "boss": false, "deputy": false, "stranger": false} {
		_, err := AuthorizeRecords(set, principal, records)
		if (err == nil) != wantSet {
			t.Fatalf("%s setting franchise.01's lineup: %v", principal, err)
		}
	}
	for principal, want := range map[string]bool{"owner-1": false, "boss": true, "deputy": true, "stranger": false} {
		_, err := AuthorizeRecords(correct, principal, records)
		if (err == nil) != want {
			t.Fatalf("%s correcting a lineup: %v", principal, err)
		}
	}
	off := append(append([]Record{}, records...), Record{Receipt: Receipt{Entry: "off"}, Principal: "boss", Command: RawCommand{League: "native.test",
		Command: "commish.abilities.set", Args: RawArgs{People: &RawPeople{Franchise: "franchise.01", Ability: "lineups"}}}})
	if _, err := AuthorizeRecords(set, "owner-1", off); err == nil || !strings.Contains(err.Error(), "turned off lineup changes") {
		t.Fatalf("the commissioner's switch was ignored: %v", err)
	}
}

func TestFlipWordsNameOnlyWhoHoldsTheGameUp(t *testing.T) {
	score := func(id string, starters int, done bool) FranchiseWeekScore {
		s := FranchiseWeekScore{Franchise: id, Name: "Team " + id, Starters: make([]StarterScore, starters)}
		if done {
			points := FranchiseWeekScore{}.Adjustment + 10000
			s.Points = &points
		}
		return s
	}
	game := NativeGame{Home: "a", Away: "b", HomeName: "Team a", AwayName: "Team b"}
	for _, tc := range []struct {
		home, away FranchiseWeekScore
		want       string
	}{
		{score("a", 0, false), score("b", 0, false), "neither team has a lineup set"},
		{score("a", 3, false), score("b", 3, false), "neither score is complete"},
		{score("a", 3, false), score("b", 3, true), "Team a's score is not complete"},
		{score("a", 3, true), score("b", 0, false), "Team b has no lineup set"},
		{score("a", 3, false), score("b", 0, false), "Team a's score is not complete and Team b has no lineup set"},
	} {
		got := incompleteReason(game, map[string]FranchiseWeekScore{"a": tc.home, "b": tc.away})
		if got != tc.want {
			t.Fatalf("got %q, want %q", got, tc.want)
		}
	}
}

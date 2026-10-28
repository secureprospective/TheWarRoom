package commissioner

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/engine/competition"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

// runSideStartups opens the startup drafts of a per-side league and makes every pick on each side in turn,
// each with the first player that side has left; it returns when no side has a pick on the clock.
func runSideStartups(t *testing.T, b *ledgerBuilder) time.Time {
	t.Helper()
	facts := startupFacts()
	at := seasonMoment(3)
	b.record(startupRaw(t, DraftStartupOpen, "open"), facts, at)
	for i := 0; ; i++ {
		at = at.Add(time.Minute)
		picked := false
		for _, key := range startupKeys(b.records) {
			basis, err := readDraft(b.records, key, at, nil)
			if err != nil || basis.words != "" {
				t.Fatalf("read %s: %v %q", key.noun(), err, basis.words)
			}
			clock := basis.draft.OnClock
			if clock == nil {
				continue
			}
			taken := map[string]bool{}
			for _, p := range basis.draft.Picks {
				taken[p.Player] = true
			}
			raw, err := PlanDraftPick(DraftPickInput{League: "native.test", Season: 2026, Startup: true,
				Franchise: clock.Owner, Player: startupPool(facts, taken)[0].ID, Request: fmt.Sprintf("pick%d%s", i, key.Side),
				Expected: int64(len(b.records))}, b.records, facts, at, time.UTC, nil)
			if err != nil {
				t.Fatalf("pick %d on %s: %v", i, key.Side, err)
			}
			b.add(raw, at)
			picked = true
		}
		if !picked {
			return at
		}
	}
}

// Each side of a 16-Team Mirror drafts its own copy of the whole pool: the same players go once on each side,
// each side's draft orders only its own teams, and the draft is done when both sides close.
func TestPerSideStartupDraftEachSideDraftsItsOwnPool(t *testing.T) {
	b := mirrorLeague(t)
	b.set("draft.startupRounds", `2`, "now", seasonMoment(2))
	at := runSideStartups(t, b)
	keys := startupKeys(b.records)
	if len(keys) != 2 || keys[0].Side != "AFC" || keys[1].Side != "NFC" {
		t.Fatalf("keys: %+v", keys)
	}
	assets, err := NativeAssetsReading("native.test", b.records, at)
	if err != nil {
		t.Fatal(err)
	}
	seed := assets.Snapshot
	drafted := map[string]map[string]int{}
	for _, p := range seed.Players {
		side := seed.SideOf(p.Franchise)
		if drafted[side] == nil {
			drafted[side] = map[string]int{}
		}
		drafted[side][p.ID]++
	}
	if len(drafted["AFC"]) != 16 || len(drafted["NFC"]) != 16 {
		t.Fatalf("drafted per side: %d %d", len(drafted["AFC"]), len(drafted["NFC"]))
	}
	for id := range drafted["AFC"] {
		if drafted["NFC"][id] != 1 || drafted["AFC"][id] != 1 {
			t.Fatalf("player %s is not drafted once on each side", id)
		}
	}
	for _, key := range keys {
		basis, err := readDraft(b.records, key, at, nil)
		if err != nil {
			t.Fatal(err)
		}
		for _, r := range basis.order {
			for _, slot := range r.Picks {
				if seed.SideOf(slot.Owner) != key.Side {
					t.Fatalf("%s orders %s", key.noun(), slot.Owner)
				}
			}
		}
		if !basis.draft.Exhausted {
			t.Fatalf("%s not done", key.noun())
		}
	}
	due := DueDrafts(b.records, at, nil)
	if len(due) != 2 {
		t.Fatalf("due drafts: %+v", due)
	}
	if !hasJob(t, b.records, "startup_draft", at) {
		t.Fatal("the startup job ended before the drafts closed")
	}
	for _, key := range due[:1] {
		b.record(RawCommand{League: "native.test", Command: DraftArchive, Request: "close." + key.Side,
			Args: RawArgs{Draft: &RawDraftArgs{Season: key.Season, Startup: true, Side: key.Side}}, Effective: "now",
			Visibility: "league"}, startupFacts(), at)
	}
	if !hasJob(t, b.records, "startup_draft", at) {
		t.Fatal("the startup job ended with one side still open")
	}
	b.record(RawCommand{League: "native.test", Command: DraftArchive, Request: "close." + due[1].Side,
		Args: RawArgs{Draft: &RawDraftArgs{Season: due[1].Season, Startup: true, Side: due[1].Side}}, Effective: "now",
		Visibility: "league"}, startupFacts(), at)
	if hasJob(t, b.records, "startup_draft", at) {
		t.Fatal("the startup job outlived both drafts")
	}
	// Play has begun: the player pools can no longer change.
	if err := b.change(poolsField, `"one"`, 20); err == nil || !strings.Contains(err.Error(), "player pools can't change") {
		t.Fatalf("pools after the draft: %v", err)
	}
}

// The sides meet on the field and in one final: the 16-Team Mirror's schedule has games between the sides,
// and its bracket runs each side's playoffs and then a single championship between the side champions.
func TestPerSideScheduleCrossesSidesAndEndsInOneFinal(t *testing.T) {
	rules := rulesAt(t, mirrorLeague(t))
	sides := rules.League.Sides.Value
	weeks, err := competition.GenerateSchedule(franchiseIDs(16), rules.League.Alignment.Value, rules.Schedule)
	if err != nil {
		t.Fatal(err)
	}
	cross := 0
	for _, w := range weeks {
		for _, g := range w.Games {
			if sides[g.Home] != sides[g.Away] {
				cross++
			}
		}
	}
	if len(weeks) != 13 || cross == 0 {
		t.Fatalf("%d weeks, %d cross-side games", len(weeks), cross)
	}
	pr := ResolvePlayoffRules(rules, rules.Playoffs.Graph.Value)
	if pr.Words != "" {
		t.Fatal(pr.Words)
	}
	finals := 0
	for _, n := range pr.Shape {
		if n.LeagueWide() {
			finals++
		}
	}
	in := competition.FieldInput{Teams: map[string]competition.FieldTeam{}, Conferences: []string{"AFC", "NFC"}, Rules: pr}
	rank, names := map[string]int{}, map[string]string{}
	for i, id := range franchiseIDs(16) {
		in.Order = append(in.Order, id)
		in.Teams[id] = competition.FieldTeam{Name: id, Division: rules.League.Alignment.Value[id], Conference: sides[id]}
		rank[id], names[id] = i+1, id
	}
	fields, words := competition.SeedField(in)
	if words != "" {
		t.Fatal(words)
	}
	rounds, champion, words := competition.BuildBracket(competition.BracketInput{Rules: pr, Fields: fields, Rank: rank,
		Names: names, Points: spread(rank), Edits: map[string]string{}, Advance: map[string]string{}})
	if words != "" || champion == nil || finals != 1 {
		t.Fatalf("%q champion %+v, %d finals", words, champion, finals)
	}
	final := gameOf(rounds, "championship")
	if final.Home == nil || final.Away == nil || sides[final.Home.Franchise] == sides[final.Away.Franchise] {
		t.Fatalf("the final is not between the side champions: %+v", final)
	}
}

// spread scores each team ten points a place apart, wider than the home field points, so no game ties.
func spread(rank map[string]int) competition.PointsFunc {
	return func(franchise string, _ int) (*leaguerules.Decimal, string) {
		v := leaguerules.Decimal((1000 - 10*rank[franchise]) * 10000)
		return &v, ""
	}
}

func TestStartupDraftReadingNamesSide(t *testing.T) {
	b := mirrorLeague(t)
	at := seasonMoment(3)
	b.record(startupRaw(t, DraftStartupOpen, "open"), startupFacts(), at)
	reading, err := NativeDraftReading(RawNativeDraftScope{League: "native.test", Season: 2026,
		Franchise: "franchise.01", Startup: true, At: at}, b.records, time.UTC, nil, startupFacts())
	if err != nil {
		t.Fatal(err)
	}
	body, err := json.Marshal(reading)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), `"side":"AFC"`) {
		t.Fatalf("missing side: %s", body)
	}
}

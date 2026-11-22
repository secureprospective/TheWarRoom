package history

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/model"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

type fixedDraftRanking struct {
	order []string
}

func (f fixedDraftRanking) Rank(_ context.Context, _ string, _ int, pool []string) ([]string, error) {
	avail := make(map[string]bool, len(pool))
	for _, id := range pool {
		avail[id] = true
	}
	out := make([]string, 0, len(pool))
	for _, id := range f.order {
		if avail[id] {
			out = append(out, id)
		}
	}
	return out, nil
}

type autoPickFixture struct {
	t      *testing.T
	s      *Store
	c      *clock
	league string
	facts  nativecheck.Facts
}

func newAutoPickFixture(t *testing.T, league, preset string, teams, rounds int,
	pools string, order []string, facts nativecheck.Facts) *autoPickFixture {
	t.Helper()
	s, _ := statStore(t, filepath.Join(t.TempDir(), "history.db"))
	at := time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)
	c := &clock{now: at}
	s.SetClock(c.Now)
	f := &autoPickFixture{t: t, s: s, c: c, league: league, facts: facts}
	f.exec("creator", commissioner.Create, commissioner.RawArgs{Name: "AutoPick Test",
		Preset: preset, Source: commissioner.SourceNative, Season: 2026, FranchiseCount: teams})
	if pools != "" {
		f.setRule("players.pools", fmt.Sprintf("%q", pools))
	}
	if rounds > 0 {
		f.setRule("draft.startupRounds", fmt.Sprint(rounds))
	}
	if len(order) > 0 {
		f.setRule("draft.startupOrder", `"commissioner"`)
		b, err := json.Marshal(order)
		if err != nil {
			t.Fatal(err)
		}
		f.setRule("draft.startupList", string(b))
	}
	return f
}

func (f *autoPickFixture) exec(principal, cmd string, args commissioner.RawArgs) commissioner.Receipt {
	f.t.Helper()
	recs, err := f.s.nativeRecords(f.t.Context(), f.league)
	if err != nil {
		f.t.Fatal(err)
	}
	raw := commissioner.RawCommand{League: f.league, Command: cmd, Args: args,
		Expected: int64(len(recs)), Request: fmt.Sprintf("req.%d.%s", len(recs), cmd),
		Reason: "Test setup", Effective: "now", Visibility: "league"}
	r, err := f.s.ExecuteCommissionerWithFacts(f.t.Context(), raw, principal, nil, f.facts, time.UTC)
	if err != nil {
		f.t.Fatalf("%s: %v", cmd, err)
	}
	return r
}

func (f *autoPickFixture) setRule(field, val string) {
	f.t.Helper()
	f.exec("creator", commissioner.RulesSet,
		commissioner.RawArgs{Field: field, Value: json.RawMessage(val)})
}

func (f *autoPickFixture) seat(franchise, owner string) {
	f.t.Helper()
	f.exec("creator", "commish.seat.replace", commissioner.RawArgs{
		People: &commissioner.RawPeople{Franchise: franchise, Principal: owner}})
}

func (f *autoPickFixture) openStartup() {
	f.t.Helper()
	f.exec("creator", commissioner.DraftStartupOpen, commissioner.RawArgs{
		Draft: &commissioner.RawDraftArgs{Season: 2026, Startup: true}})
	f.c.now = f.c.now.Add(time.Second)
}

func (f *autoPickFixture) draftReading(franchise string) commissioner.NativeDraft {
	f.t.Helper()
	d, err := f.s.NativeDraft(f.t.Context(), commissioner.RawNativeDraftScope{
		League: f.league, Season: 2026, Franchise: franchise, At: f.c.now, Startup: true},
		time.UTC, f.facts)
	if err != nil {
		f.t.Fatal(err)
	}
	return d
}

func testFacts(n int) nativecheck.Facts {
	positions := []leaguerules.Position{
		leaguerules.QB, leaguerules.RB, leaguerules.WR, leaguerules.TE,
		leaguerules.DE, leaguerules.DT, leaguerules.LB, leaguerules.CB, leaguerules.S,
	}
	out := make(nativecheck.Facts, n)
	for i := 1; i <= n; i++ {
		id := fmt.Sprintf("%04d", 1000+i)
		out[id] = nativecheck.PlayerFacts{
			Name:     fmt.Sprintf("Player %03d", i),
			Position: positions[(i-1)%len(positions)],
			Team:     "KC",
		}
	}
	return out
}

func putWeeklyStatsForSeason(t *testing.T, s *Store, season int, passingByPlayer map[string]int,
	at time.Time) {
	t.Helper()
	body, err := os.ReadFile("../../fantasyscore/source/testdata/weeks.csv")
	if err != nil {
		t.Fatal(err)
	}
	rows, err := csv.NewReader(bytes.NewReader(body)).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	col := map[string]int{}
	for i, name := range rows[0] {
		col[name] = i
	}
	var links []PlayerIDLink
	out := [][]string{rows[0]}
	idx := 1
	for playerID, yards := range passingByPlayer {
		gsis := fmt.Sprintf("00-%07d", 8000000+idx)
		idx++
		links = append(links, PlayerIDLink{IDType: "gsis", IDValue: gsis, PlayerID: playerID})
		for week := 1; week <= 3; week++ {
			row := append([]string(nil), rows[1]...)
			row[col["player_id"]] = gsis
			row[col["season"]] = fmt.Sprint(season)
			row[col["week"]] = fmt.Sprint(week)
			row[col["game_id"]] = fmt.Sprintf("%d_%02d_PIT_NYJ", season, week)
			row[col["passing_yards"]] = fmt.Sprint(yards)
			out = append(out, row)
		}
	}
	if _, err := s.LinkPlayerIDs(t.Context(), "dynastyprocess", links); err != nil {
		t.Fatal(err)
	}
	var buf strings.Builder
	if err := csv.NewWriter(&buf).WriteAll(out); err != nil {
		t.Fatal(err)
	}
	feed, ok := s.reg.Feed("stats_player_week")
	if !ok {
		t.Fatal("missing stats_player_week feed")
	}
	if err := s.Record(t.Context(), archiveFetch(feed.URLFor(season), buf.String(), at)); err != nil {
		t.Fatal(err)
	}
}

func TestAutoPickStartupFourFranchisesStopsOnSeatedAndResumes(t *testing.T) {
	t.Parallel()
	order := []string{"franchise.01", "franchise.02", "franchise.03", "franchise.04"}
	facts := testFacts(12)
	f := newAutoPickFixture(t, "native.auto4", commissioner.PresetDefault32, 4, 2, "", order, facts)
	f.seat("franchise.04", "owner.four")
	f.openStartup()

	rank := fixedDraftRanking{order: []string{
		"1001", "1002", "1003", "1004", "1005", "1006", "1007", "1008",
	}}
	if err := f.s.ProcessDueAutoPicks(t.Context(), f.league, f.c.now, facts, rank); err != nil {
		t.Fatal(err)
	}

	d := f.draftReading("franchise.04")
	if d.Draft.OnClock == nil || d.Draft.OnClock.Owner != "franchise.04" {
		t.Fatalf("after pass 1 onClock = %+v, want franchise.04", d.Draft.OnClock)
	}
	got := []string{d.Draft.Picks[0].Player, d.Draft.Picks[1].Player, d.Draft.Picks[2].Player}
	if !slices.Equal(got, []string{"1001", "1002", "1003"}) {
		t.Fatalf("pass 1 picks = %v, want [1001 1002 1003]", got)
	}
	if d.Draft.Picks[3].Player != "" {
		t.Fatalf("seated franchise.04 was picked for: %s", d.Draft.Picks[3].Player)
	}

	// Seated owner makes their round 1 pick (slot 4) and round 2 snake pick (slot 1 of round 2).
	f.c.now = f.c.now.Add(time.Second)
	for _, pickPlayer := range []string{"1004", "1005"} {
		recs, err := f.s.nativeRecords(t.Context(), f.league)
		if err != nil {
			t.Fatal(err)
		}
		gate, err := commissioner.NativeActionGate(f.league, recs, "franchise.04", "draft_pick", f.c.now, time.UTC)
		if err != nil {
			t.Fatal(err)
		}
		permit, open := gate.Permit()
		if !open {
			t.Fatalf("draft_pick gate closed: %s", gate.Sentence)
		}
		if _, err := f.s.RecordDraftPick(t.Context(), permit, commissioner.DraftPickInput{
			League: f.league, Season: 2026, Franchise: "franchise.04", Player: pickPlayer,
			Request: "human." + pickPlayer, Expected: int64(len(recs)), Startup: true,
		}, "owner.four", facts, time.UTC); err != nil {
			t.Fatalf("human pick %s: %v", pickPlayer, err)
		}
		f.c.now = f.c.now.Add(time.Second)
	}

	// Next pass finishes the remaining 3 vacant picks in round 2.
	if err := f.s.ProcessDueAutoPicks(t.Context(), f.league, f.c.now, facts, rank); err != nil {
		t.Fatal(err)
	}
	d = f.draftReading("franchise.04")
	if !d.Draft.Exhausted {
		t.Fatalf("after pass 2 draft not exhausted; onClock = %+v", d.Draft.OnClock)
	}
	gotR2 := []string{d.Draft.Picks[5].Player, d.Draft.Picks[6].Player, d.Draft.Picks[7].Player}
	if !slices.Equal(gotR2, []string{"1006", "1007", "1008"}) {
		t.Fatalf("pass 2 picks = %v, want [1006 1007 1008]", gotR2)
	}
}

func TestAutoPickRankingModelScoresProjectionFallbackAndSkipRefused(t *testing.T) {
	t.Parallel()
	order := []string{"franchise.01", "franchise.02", "franchise.03", "franchise.04"}
	facts := nativecheck.Facts{
		"1001": {Name: "QB One", Position: leaguerules.QB, Team: "KC"},
		"1002": {Name: "QB Two", Position: leaguerules.QB, Team: "BUF"},
		"1003": {Name: "WR One", Position: leaguerules.WR, Team: "MIN"},
		"1004": {Name: "WR Two", Position: leaguerules.WR, Team: "CIN"},
		"1005": {Name: "RB One", Position: leaguerules.RB, Team: "SF"},
	}

	t.Run("no model run falls back to projection order", func(t *testing.T) {
		t.Parallel()
		f := newAutoPickFixture(t, "native.rankproj", commissioner.PresetDefault32, 4, 1, "", order, facts)
		putWeeklyStatsForSeason(t, f.s, 2025, map[string]int{
			"1001": 150,
			"1002": 320,
			"1003": 240,
		}, f.c.now)
		got, err := DynastyDraftRanking{Store: f.s}.Rank(t.Context(), f.league, 2026,
			[]string{"1001", "1002", "1003", "1004", "1005"})
		if err != nil {
			t.Fatal(err)
		}
		// 1002 (320 yds) > 1003 (240 yds) > 1001 (150 yds), then unscored/unprojected by ID.
		want := []string{"1002", "1003", "1001", "1004", "1005"}
		if !slices.Equal(got, want) {
			t.Fatalf("no-model-run rank = %v, want %v", got, want)
		}
	})

	t.Run("newest model run dynasty_ppg then projection then ID", func(t *testing.T) {
		t.Parallel()
		f := newAutoPickFixture(t, "native.rankmodel", commissioner.PresetDefault32, 4, 1, "", order, facts)
		putWeeklyStatsForSeason(t, f.s, 2025, map[string]int{
			"1003": 180,
			"1004": 310,
		}, f.c.now)

		// Older run gives 1001 a huge score (50.0); newer run lowers 1001 to 10.0 and puts 1002 at 22.0.
		oldRun := NewModelRun{
			Refresh: true, Season: 2026, AsOf: f.c.now.Add(-48 * time.Hour),
			Params: ParamSet{Params: map[string]float64{"a": 1}, Measures: []string{}},
			Engine: "test", InputsHash: "in-old",
			Scores: []ModelScore{
				{MFLID: "1001", Position: "QB", Value: model.Value{DynastyPPG: 50.0}},
				{MFLID: "1002", Position: "QB", Value: model.Value{DynastyPPG: 15.0}},
			},
		}
		if _, _, err := f.s.WriteModelRun(t.Context(), oldRun); err != nil {
			t.Fatal(err)
		}
		newRun := NewModelRun{
			Refresh: true, Season: 2026, AsOf: f.c.now,
			Params: ParamSet{Params: map[string]float64{"a": 2}, Measures: []string{}},
			Engine: "test", InputsHash: "in-new",
			Scores: []ModelScore{
				{MFLID: "1001", Position: "QB", Value: model.Value{DynastyPPG: 10.0}},
				{MFLID: "1002", Position: "QB", Value: model.Value{DynastyPPG: 22.0}},
			},
		}
		if _, _, err := f.s.WriteModelRun(t.Context(), newRun); err != nil {
			t.Fatal(err)
		}

		got, err := DynastyDraftRanking{Store: f.s}.Rank(t.Context(), f.league, 2026,
			[]string{"1001", "1002", "1003", "1004", "1005"})
		if err != nil {
			t.Fatal(err)
		}
		// Newest run: 1002 (22) > 1001 (10); projection: 1004 (310) > 1003 (180); unprojected: 1005.
		want := []string{"1002", "1001", "1004", "1003", "1005"}
		if !slices.Equal(got, want) {
			t.Fatalf("model+proj rank = %v, want %v", got, want)
		}
	})

	t.Run("skips refused player and takes next legal player", func(t *testing.T) {
		t.Parallel()
		f := newAutoPickFixture(t, "native.skiprefuse", commissioner.PresetDefault32, 2, 2, "",
			[]string{"franchise.01", "franchise.02"}, facts)
		// Cap QBs at 1 per roster so franchise.02 (which picks 1.02 and 2.01 in snake) must skip 1002.
		f.setRule("roster.positionBounds", `{"OL":{"min":1,"max":1},"QB":{"min":0,"max":1}}`)
		f.openStartup()

		// 1001 (QB), 1002 (QB), 1003 (WR), 1004 (WR)
		// Pick 1 (f01): 1001 (QB)
		// Pick 2 (f02): 1002 (QB)
		// Pick 3 (f02, snake): 1003 (WR) -- even if we put another QB ahead of 1003 in rank!
		factsWithExtraQB := nativecheck.Facts{
			"1001": {Name: "QB One", Position: leaguerules.QB, Team: "KC"},
			"1002": {Name: "QB Two", Position: leaguerules.QB, Team: "BUF"},
			"1006": {Name: "QB Three", Position: leaguerules.QB, Team: "BAL"},
			"1003": {Name: "WR One", Position: leaguerules.WR, Team: "MIN"},
			"1004": {Name: "WR Two", Position: leaguerules.WR, Team: "CIN"},
		}
		rank := fixedDraftRanking{order: []string{"1001", "1002", "1006", "1003", "1004"}}
		if err := f.s.ProcessDueAutoPicks(t.Context(), f.league, f.c.now, factsWithExtraQB, rank); err != nil {
			t.Fatal(err)
		}
		d := f.draftReading("franchise.01")
		got := []string{
			d.Draft.Picks[0].Player, d.Draft.Picks[1].Player,
			d.Draft.Picks[2].Player, d.Draft.Picks[3].Player,
		}
		// f01 gets 1001 (QB), f02 gets 1002 (QB), f02 skips 1006 (QB over max 1) and takes 1003 (WR),
		// f01 skips 1006 (QB over max 1) and takes 1004 (WR).
		want := []string{"1001", "1002", "1003", "1004"}
		if !slices.Equal(got, want) {
			t.Fatalf("picks with QB cap = %v, want %v", got, want)
		}
	})
}

func TestAutoPickTwinLeagueBothSidesAdvanceAndSharePool(t *testing.T) {
	t.Parallel()
	facts := testFacts(40)
	f := newAutoPickFixture(t, "native.autotwin", commissioner.PresetDefault32, 32, 1,
		string(leaguerules.PerSide), nil, facts)
	f.openStartup()

	order := make([]string, 0, 40)
	for i := 1; i <= 40; i++ {
		order = append(order, fmt.Sprintf("%04d", 1000+i))
	}
	if err := f.s.ProcessDueAutoPicks(t.Context(), f.league, f.c.now, facts,
		fixedDraftRanking{order: order}); err != nil {
		t.Fatal(err)
	}

	afc := f.draftReading("franchise.01")
	nfc := f.draftReading("franchise.17")
	if !afc.Draft.Exhausted || !nfc.Draft.Exhausted {
		t.Fatalf("twin drafts exhausted: AFC=%v NFC=%v, want both true",
			afc.Draft.Exhausted, nfc.Draft.Exhausted)
	}
	if afc.Draft.Picks[0].Player != "1001" || nfc.Draft.Picks[0].Player != "1001" {
		t.Fatalf("first picks AFC=%s NFC=%s, want 1001 on both sides",
			afc.Draft.Picks[0].Player, nfc.Draft.Picks[0].Player)
	}
	recs, err := f.s.nativeRecords(t.Context(), f.league)
	if err != nil {
		t.Fatal(err)
	}
	assets, err := commissioner.NativeAssetsReading(f.league, recs)
	if err != nil {
		t.Fatal(err)
	}
	copies := 0
	for _, p := range assets.Snapshot.Players {
		if p.ID == "1001" {
			copies++
		}
	}
	if copies != 2 {
		t.Fatalf("copies of 1001 across twin league = %d, want 2 (one per side)", copies)
	}
}

func TestAutoPickNeverPicksForSeatedFranchiseAndAuthorizerRefuses(t *testing.T) {
	t.Parallel()
	order := []string{"franchise.01", "franchise.02"}
	facts := testFacts(6)
	f := newAutoPickFixture(t, "native.autoseat", commissioner.PresetDefault32, 2, 1, "", order, facts)
	f.seat("franchise.01", "owner.one")
	// Assign a caretaker to vacant franchise.02: Seat is still "", so franchise.02 remains AUTOMATIC.
	f.exec("creator", "commish.seat.caretaker", commissioner.RawArgs{
		People: &commissioner.RawPeople{Franchise: "franchise.02", Principal: "creator"}})
	f.openStartup()

	recs, err := f.s.nativeRecords(t.Context(), f.league)
	if err != nil {
		t.Fatal(err)
	}
	// Authorizer must refuse seated franchise.01.
	seatedAuth := autoPickDraftAuthorizer{records: recs, franchise: "franchise.01"}
	if _, err := seatedAuth.Allow(f.league, commissioner.DraftAutoPickPrincipal,
		commissioner.DraftPick); err == nil {
		t.Fatal("autoPickDraftAuthorizer allowed seated franchise.01")
	}
	// Authorizer must refuse wrong command for vacant franchise.02.
	vacantAuth := autoPickDraftAuthorizer{records: recs, franchise: "franchise.02"}
	if _, err := vacantAuth.Allow(f.league, commissioner.DraftAutoPickPrincipal,
		commissioner.DraftArchive); err == nil {
		t.Fatal("autoPickDraftAuthorizer allowed draft.archive")
	}
	// Authorizer must allow vacant franchise.02 (even with caretaker) for draft.pick.
	role, err := vacantAuth.Allow(f.league, commissioner.DraftAutoPickPrincipal, commissioner.DraftPick)
	if err != nil || role != "automatic" {
		t.Fatalf("vacant caretaker franchise Allow = (%q, %v), want (automatic, nil)", role, err)
	}

	// ProcessDueAutoPicks must not pick for seated franchise.01 on the clock.
	if err := f.s.ProcessDueAutoPicks(t.Context(), f.league, f.c.now, facts,
		fixedDraftRanking{order: []string{"1001", "1002"}}); err != nil {
		t.Fatal(err)
	}
	d := f.draftReading("franchise.01")
	if d.Draft.Picks[0].Player != "" {
		t.Fatalf("auto-pick picked %s for seated franchise.01", d.Draft.Picks[0].Player)
	}
}

func TestAutoPickPausedAndExhaustedDrafts(t *testing.T) {
	t.Parallel()
	order := []string{"franchise.01", "franchise.02"}
	facts := testFacts(6)
	f := newAutoPickFixture(t, "native.autopause", commissioner.PresetDefault32, 2, 1, "", order, facts)
	f.openStartup()

	// Pause the draft clock.
	f.exec("creator", commissioner.DraftTimer, commissioner.RawArgs{
		Draft: &commissioner.RawDraftArgs{Season: 2026, Startup: true,
			Act: &commissioner.RawDraftAct{Timer: commissioner.TimerPause}}})
	f.c.now = f.c.now.Add(time.Second)

	rank := fixedDraftRanking{order: []string{"1001", "1002"}}
	if err := f.s.ProcessDueAutoPicks(t.Context(), f.league, f.c.now, facts, rank); err != nil {
		t.Fatal(err)
	}
	if d := f.draftReading("franchise.01"); d.Draft.Picks[0].Player != "" {
		t.Fatalf("paused draft picked %s", d.Draft.Picks[0].Player)
	}

	// Resume, let auto-pick exhaust the draft, verify a second pass picks nothing, then close it.
	f.exec("creator", commissioner.DraftTimer, commissioner.RawArgs{
		Draft: &commissioner.RawDraftArgs{Season: 2026, Startup: true,
			Act: &commissioner.RawDraftAct{Timer: commissioner.TimerResume}}})
	f.c.now = f.c.now.Add(time.Second)
	if err := f.s.ProcessDueAutoPicks(t.Context(), f.league, f.c.now, facts, rank); err != nil {
		t.Fatal(err)
	}
	recsAfter, err := f.s.nativeRecords(t.Context(), f.league)
	if err != nil {
		t.Fatal(err)
	}
	// Exhausted pass must be a no-op.
	if err := f.s.ProcessDueAutoPicks(t.Context(), f.league, f.c.now, facts, rank); err != nil {
		t.Fatal(err)
	}
	recsAgain, err := f.s.nativeRecords(t.Context(), f.league)
	if err != nil {
		t.Fatal(err)
	}
	if len(recsAgain) != len(recsAfter) {
		t.Fatalf("exhausted draft appended %d records", len(recsAgain)-len(recsAfter))
	}
	if err := f.s.CloseFinishedDrafts(t.Context(), f.league, facts); err != nil {
		t.Fatalf("CloseFinishedDrafts: %v", err)
	}
	if d := f.draftReading("franchise.01"); d.Closed == nil || !d.Closed.Automatic {
		t.Fatalf("exhausted draft not closed automatically: %+v", d.Closed)
	}
}

func TestAutoPickSpeed32Team3RoundPass(t *testing.T) {
	t.Parallel()
	order := make([]string, 32)
	for i := 1; i <= 32; i++ {
		order[i-1] = fmt.Sprintf("franchise.%02d", i)
	}
	facts := testFacts(120)
	f := newAutoPickFixture(t, "native.autospeed", commissioner.PresetDefault32, 32, 3, "", order, facts)
	f.seat("franchise.32", "owner.human")
	f.openStartup()

	// Seed a model run so the real DynastyDraftRanking is exercised.
	scores := make([]ModelScore, 0, 120)
	for i := 1; i <= 120; i++ {
		id := fmt.Sprintf("%04d", 1000+i)
		scores = append(scores, ModelScore{
			MFLID:    id,
			Position: string(facts[id].Position),
			Value:    model.Value{DynastyPPG: float64(130 - i)},
		})
	}
	if _, _, err := f.s.WriteModelRun(t.Context(), NewModelRun{
		Refresh: true, Season: 2026, AsOf: f.c.now,
		Params: ParamSet{Params: map[string]float64{"v": 1}, Measures: []string{}},
		Engine: "bench", InputsHash: "speed", Scores: scores,
	}); err != nil {
		t.Fatal(err)
	}

	start := time.Now()
	if err := f.s.ProcessDueAutoPicks(t.Context(), f.league, f.c.now, facts, nil); err != nil {
		t.Fatal(err)
	}
	elapsed := time.Since(start)

	d := f.draftReading("franchise.32")
	if d.Draft.OnClock == nil || d.Draft.OnClock.Owner != "franchise.32" {
		t.Fatalf("onClock = %+v, want franchise.32", d.Draft.OnClock)
	}
	picked := 0
	for _, p := range d.Draft.Picks {
		if p.Player != "" {
			picked++
		}
	}
	if picked != 31 {
		t.Fatalf("picked = %d, want 31", picked)
	}
	t.Logf("32-team 3-round startup pass (31 automatic picks to human turn): %v (%.2f ms/pick)",
		elapsed, float64(elapsed.Microseconds())/1000.0/31.0)
}

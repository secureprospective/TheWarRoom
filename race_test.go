package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/engine/race"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/league"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/leaguestandings"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/players"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/playoffs"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/weeklyresults"
	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
	"github.com/secureprospective/TheWarRoom/internal/normalize"
)

var raceHeldAt = time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC) //nolint:gochecknoglobals // a fixed test clock

func held[T any](v T) heldSeasonFeed[T] {
	return heldSeasonFeed[T]{value: v, fetchedAt: raceHeldAt}
}

// realRace is a season exactly as the league archive holds it: MFL's league, weekly results,
// brackets, standings and players directory.
func realRace(t *testing.T, year string) raceInput {
	t.Helper()
	cfg, err := league.Parse([]byte(seasonFixture(t, "league", "league-"+year+".json")),
		[]byte(seasonFixture(t, "league", "rules-2025.json")))
	if err != nil {
		t.Fatal(err)
	}
	weeks, err := weeklyresults.Parse([]byte(seasonGzFixture(t, "weeklyresults", "weeklyResults-"+year+".json.gz")))
	if err != nil {
		t.Fatal(err)
	}
	brackets, err := playoffs.ParseList([]byte(seasonFixture(t, "playoffs", "playoffBrackets-"+year+".json")))
	if err != nil {
		t.Fatal(err)
	}
	for i, b := range brackets {
		name := fmt.Sprintf("playoffBracket_%d-%s.json", b.ID, year)
		if brackets[i].Games, err = playoffs.ParseBracket(b.ID, []byte(seasonFixture(t, "playoffs", name))); err != nil {
			t.Fatal(err)
		}
	}
	standings, err := leaguestandings.Parse([]byte(seasonFixture(t, "leaguestandings", "leagueStandings-"+year+".json")))
	if err != nil {
		t.Fatal(err)
	}
	raws, err := players.Parse(context.Background(), []byte(seasonGzFixture(t, "players", "players-2025.json.gz")))
	if err != nil {
		t.Fatal(err)
	}
	lookup, err := normalize.NewLookup(raws)
	if err != nil {
		t.Fatal(err)
	}
	return raceInput{held: raceHeld{weeks: held(weeks), brackets: held(brackets), standings: held(standings)},
		config: cfg, lookup: lookup, hasLookup: true, team: "0001", mine: "0001", zone: time.UTC}
}

// mflStandings is MFL's own standings export, the numbers the app's must equal.
func mflStandings(t *testing.T, year string) []map[string]string {
	t.Helper()
	var env struct {
		LeagueStandings struct {
			Franchise []map[string]string `json:"franchise"`
		} `json:"leagueStandings"`
	}
	if err := json.Unmarshal([]byte(seasonFixture(t, "leaguestandings", "leagueStandings-"+year+".json")), &env); err != nil {
		t.Fatal(err)
	}
	return env.LeagueStandings.Franchise
}

func wlt(r race.Record) string { return fmt.Sprintf("%d-%d-%d", r.W, r.L, r.T) }

func assertStandingsMatchMFL(t *testing.T, year string) raceView {
	t.Helper()
	v, err := newRaceView(realRace(t, year))
	if err != nil {
		t.Fatal(err)
	}
	mfl := mflStandings(t, year)
	if len(mfl) != 32 {
		t.Fatal(len(mfl))
	}
	for _, s := range mfl {
		l, ok := v.table.Line(s["id"])
		if !ok {
			t.Fatal(s["id"])
		}
		got := []string{wlt(l.Overall), wlt(l.InDivision), wlt(l.InConference), wlt(l.AllPlay),
			strings.ReplaceAll(scoreText(l.For), ",", ""), strings.ReplaceAll(scoreText(l.Against), ",", ""),
			string(l.Streak.Result) + strconv.Itoa(l.Streak.Count)}
		want := []string{s["h2hw"] + "-" + s["h2hl"] + "-" + s["h2ht"], s["divw"] + "-" + s["divl"] + "-" + s["divt"],
			s["confw"] + "-" + s["confl"] + "-" + s["conft"], s["all_play_wlt"], s["pf"], s["pa"], s["strk"]}
		if !slices.Equal(got, want) {
			t.Fatalf("%s %s: app %v, MFL %v", year, s["id"], got, want)
		}
	}
	for _, d := range v.divisions {
		mflOrder := []string{}
		for _, s := range mfl {
			if l, _ := v.table.Line(s["id"]); l.Division == d.ID {
				mflOrder = append(mflOrder, s["id"])
			}
		}
		if got := v.table.Division(d.ID).Order; !slices.Equal(got, mflOrder) {
			t.Fatalf("%s %s: app %v, MFL %v", year, d.Name, got, mflOrder)
		}
	}
	return v
}

func TestRace2025StandingsMatchMFLExactly(t *testing.T) {
	v := assertStandingsMatchMFL(t, "2025")
	if v.table.RegularWeeks != 13 {
		t.Fatal(v.table.RegularWeeks)
	}
	standings := v.standings()
	first := standings.Rows[0]
	if !slices.Equal(first.Cells, []string{"Buffalo Bills", "12-1", ".923", "4,747.96", "2,400.25", "Won 11", "—"}) ||
		!first.Mine || first.Group != "AFC East" || standings.Columns[0] != "Franchise" {
		t.Fatal(first)
	}
	// The division heads each run of four rows rather than repeating in a column.
	for i, row := range standings.Rows {
		if row.Group != standings.Rows[i/4*4].Group || i%4 == 0 && i > 0 && row.Group == standings.Rows[i-1].Group {
			t.Fatal("division groups", i, row.Group)
		}
	}
	if west := standings.Rows[5]; west.Group != "AFC West" || west.Cells[0] != "Los Angeles Chargers" || west.Cells[6] != "0" {
		t.Fatal("AFC West runner-up lost the head-to-head tiebreak at 9-4", west)
	}
}

func TestRace2026StandingsInProgressMatchMFLExactly(t *testing.T) {
	v := assertStandingsMatchMFL(t, "2026")
	if v.table.RegularWeeks != 3 || v.week != 3 {
		t.Fatal(v.table.RegularWeeks, v.week)
	}
	if l, _ := v.table.Line("0001"); len(l.Remaining) != 10 {
		t.Fatal(l.Remaining)
	}
}

// bracketSeeds is MFL's seed for each franchise in a bracket, from every game side.
func bracketSeeds(b leaguefeed.Bracket) []string {
	seeds := make([]string, b.Teams)
	for _, g := range b.Games {
		for _, s := range []leaguefeed.BracketSide{g.Home, g.Away} {
			if s.Seed > 0 && s.Franchise != "" {
				seeds[s.Seed-1] = s.Franchise
			}
		}
	}
	return seeds
}

func TestRace2025SeedsAndOpeningRoundMatchMFLBrackets(t *testing.T) {
	v, err := newRaceView(realRace(t, "2025"))
	if err != nil {
		t.Fatal(err)
	}
	fields, note := v.fields()
	if len(fields) != 2 || note != "" {
		t.Fatal(len(fields), note)
	}
	for _, f := range fields {
		got := []string{}
		for _, s := range f.field.Seeds {
			got = append(got, s.Franchise)
		}
		if want := bracketSeeds(f.bracket); !slices.Equal(got, want) {
			t.Fatalf("%s seeds: app %v, MFL %v", f.conference.Name, got, want)
		}
		byes, games := race.FirstRound(f.bracket.Teams)
		mflGames := [][2]int{}
		for _, g := range f.bracket.Games {
			if g.Week == f.bracket.StartWeek {
				mflGames = append(mflGames, [2]int{g.Home.Seed, g.Away.Seed})
			}
		}
		if byes != 1 || !slices.Equal(games, mflGames) {
			t.Fatalf("%s opening round: app %v, MFL %v", f.conference.Name, games, mflGames)
		}
	}
	seeding := v.seeding()
	if r := seeding.Rows[0]; !slices.Equal(r.Cells, []string{"AFC", "1", "Buffalo Bills", "12-1", "Leads the AFC East", "Bye"}) {
		t.Fatal(r.Cells)
	}
	if r := seeding.Rows[13]; !slices.Equal(r.Cells[1:], []string{"7", "Philadelphia Eagles", "8-5", "Wild card", "At seed 2"}) {
		t.Fatal(r.Cells)
	}
}

func TestRace2025BracketReportIsMFLsBracket(t *testing.T) {
	v, err := newRaceView(realRace(t, "2025"))
	if err != nil {
		t.Fatal(err)
	}
	r := v.brackets()
	if r.Status != "ok" || len(r.Rows) != 13 {
		t.Fatal(r.Status, len(r.Rows))
	}
	if r.Rows[0].Cells[0] != "AFC Conference" || r.Rows[0].Cells[2] != "Seed 2 · Cleveland Browns" ||
		r.Rows[0].Cells[4] != "Cleveland Browns won 276.68 to 179.81" {
		t.Fatal(r.Rows[0].Cells)
	}
	last := r.Rows[12]
	if last.Cells[0] != "Legacy NFL Super Bowl" || last.Cells[3] != "Buffalo Bills" ||
		last.Cells[4] != "Buffalo Bills won 307.85 to 270.05" || !last.Mine {
		t.Fatal(last.Cells)
	}
	summary := map[string]string{}
	for _, s := range r.Summary {
		summary[s.Cells[0]] = s.Cells[1]
	}
	if summary["2025 Super Bowl Champion"] != "Buffalo Bills" ||
		summary["Legacy NFL Super Bowl"] != "Its teams are the other brackets' winners." || summary["NFC Conference Champion"] != "Minnesota Vikings" ||
		!strings.HasPrefix(summary["AFC Conference"], "Seeds: 1 Buffalo Bills · 2 Cleveland Browns · 3 Denver Broncos") {
		t.Fatal(summary)
	}
}

func TestRace2025TiebreakersShowEachStep(t *testing.T) {
	v, err := newRaceView(realRace(t, "2025"))
	if err != nil {
		t.Fatal(err)
	}
	r := v.tiebreakers()
	if r.Summary[0].Cells[1] != "1 Win percentage · 2 Head-to-head among the tied teams · 3 Division win percentage · "+
		"4 Conference win percentage · 5 All-play win percentage · 6 Points for · 7 MFL power rating" {
		t.Fatal(r.Summary)
	}
	find := func(race, step string) ReportRow {
		for _, row := range r.Rows {
			if strings.HasPrefix(row.Cells[0], race) && strings.HasSuffix(row.Cells[2], step) {
				return row
			}
		}
		t.Fatalf("no %s step in %s", step, race)
		return ReportRow{}
	}
	west := find("AFC West · 9-4", "Head-to-head among the tied teams")
	if west.Cells[3] != "Los Angeles Chargers 0-2\nDenver Broncos 2-0" || west.Cells[4] != "Order: Denver Broncos, then Los Angeles Chargers" {
		t.Fatal(west.Cells)
	}
	if south := find("AFC South · 9-4", "Division win percentage"); south.Cells[4] != "Order: Houston Texans, then Indianapolis Colts" {
		t.Fatal(south.Cells)
	}
	wild := find("NFC wild cards · 8-5", "All-play win percentage")
	if wild.Cells[4] != "Order: Detroit Lions, then Philadelphia Eagles" ||
		find("NFC wild cards · 8-5", "Division win percentage").Cells[3] != "Skipped: the tied teams are not all in one division" {
		t.Fatal(wild.Cells)
	}
}

// Every week's starters add up to the franchise's score, so starter points by position total
// MFL's points for, less the +3 home-field points MFL adds to ten 2025 playoff scores.
func TestRaceStarterPointsAddUpToPointsFor(t *testing.T) {
	v, err := newRaceView(realRace(t, "2025"))
	if err != nil {
		t.Fatal(err)
	}
	r := v.positionPoints()
	if r.Columns[2] != "QB" || r.Columns[6] != "K" || len(r.Rows) != 32 {
		t.Fatal(r.Columns, len(r.Rows))
	}
	hundredths := func(s string) int64 {
		n, err := strconv.ParseFloat(strings.ReplaceAll(s, ",", ""), 64)
		if err != nil {
			t.Fatal(s)
		}
		return int64(n*100 + 0.5)
	}
	gap := int64(0)
	for _, id := range v.table.IDs() {
		l, _ := v.table.Line(id)
		for _, row := range r.Rows {
			if row.Cells[0] == v.name(id) {
				positions := int64(0)
				for _, cell := range row.Cells[2:] {
					positions += hundredths(cell)
				}
				if positions != hundredths(row.Cells[1]) || int64(l.For)-positions < 0 || (int64(l.For)-positions)%300 != 0 {
					t.Fatal(row.Cells, scoreText(l.For))
				}
				gap += int64(l.For) - positions
			}
		}
	}
	if gap != 3000 {
		t.Fatal(gap)
	}
}

func TestRaceChoicesAreCheckedInGo(t *testing.T) {
	in := realRace(t, "2026")
	for name, change := range map[string]func(*raceInput){
		"unscored week": func(in *raceInput) { in.week = 4 },
		"unknown week":  func(in *raceInput) { in.week = 40 },
		"unknown team":  func(in *raceInput) { in.team = "0099" },
		"unknown mine":  func(in *raceInput) { in.mine = "x" },
	} {
		bad := in
		change(&bad)
		if _, err := buildRace(bad); err == nil {
			t.Errorf("%s accepted", name)
		}
	}
	in.week, in.team = 2, "0017"
	board, err := buildRace(in)
	if err != nil || board.Week != 2 || board.Team != "0017" || len(board.Weeks) != 3 || len(board.Teams) != 32 {
		t.Fatal(err, board.Week, board.Team, len(board.Weeks))
	}
	detail := reportByID(t, ReportBoard{Reports: board.Board.Reports}, "M-010")
	if detail.Summary[0].Cells[1] != "Atlanta Falcons" || detail.Summary[1].Cells[1] != "Week 2" ||
		detail.Summary[2].Cells[1] != "213.90" || detail.Summary[3].Cells[1] != "269.55 · 55.65 left on the bench" ||
		len(detail.Rows) < 21 {
		t.Fatal(detail.Summary, len(detail.Rows))
	}
	in.team = ""
	board, _ = buildRace(in)
	if r := reportByID(t, ReportBoard{Reports: board.Board.Reports}, "M-012"); r.Status != "empty" || len(r.Rows) != 0 {
		t.Fatal(r.Status)
	}
}

// A row names franchises and players; it never shows an MFL id or says a name is missing.
func TestRaceRowsNeverShowIDs(t *testing.T) {
	franchiseID := regexp.MustCompile(`(^|[^\d.,])00\d\d([^\d.,]|$)`)
	for _, year := range []string{"2025", "2026"} {
		in := realRace(t, year)
		ids := map[string]bool{}
		for _, w := range in.held.weeks.value {
			for _, g := range w.Games {
				for _, p := range append(slices.Clone(g.Home.Players), g.Away.Players...) {
					ids[p.ID.String()] = true
				}
			}
		}
		board, err := buildRace(in)
		if err != nil {
			t.Fatal(err)
		}
		for _, r := range append(board.Board.Reports, board.Standings) {
			for _, row := range append(slices.Clone(r.Rows), r.Summary...) {
				for _, cell := range row.Cells {
					for _, word := range strings.FieldsFunc(cell, func(c rune) bool { return c == ' ' || c == '\n' || c == '@' }) {
						if ids[word] || franchiseID.MatchString(word) {
							t.Fatalf("%s %s shows an id: %q", year, r.ID, cell)
						}
					}
					if strings.Contains(strings.ToLower(cell), "not held") {
						t.Fatalf("%s %s: %q", year, r.ID, cell)
					}
				}
			}
		}
	}
}

func TestRaceMissingFeedsAreLabelledNotBlank(t *testing.T) {
	in := realRace(t, "2026")
	in.held = raceHeld{}
	in.config.StandingsSort = ""
	board, err := buildRace(in)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range append(board.Board.Reports, board.Standings) {
		if r.Status != "failed" || r.Note == "" || len(r.Rows) != 0 {
			t.Fatalf("%s: %s %q", r.ID, r.Status, r.Note)
		}
	}
	in = realRace(t, "2026")
	in.held.brackets = heldSeasonFeed[[]leaguefeed.Bracket]{}
	in.config.StandingsSort = ""
	board, err = buildRace(in)
	if err != nil {
		t.Fatal(err)
	}
	if board.Standings.Status != "ok" || !strings.Contains(board.Standings.Note, "tiebreak order is not held") {
		t.Fatal(board.Standings.Note)
	}
	for _, id := range []string{"M-090", "M-170", "M-014"} {
		if r := reportByID(t, ReportBoard{Reports: board.Board.Reports}, id); r.Status != "failed" {
			t.Fatal(id, r.Status)
		}
	}
	if r := reportByID(t, ReportBoard{Reports: board.Board.Reports}, "M-161"); r.Status != "ok" || len(r.Rows) != 32 {
		t.Fatal("all-play needs only the weekly results")
	}
}

func TestRace2026BubbleAndUnseededBrackets(t *testing.T) {
	v, err := newRaceView(realRace(t, "2026"))
	if err != nil {
		t.Fatal(err)
	}
	bubble := v.bubble()
	if bubble.Status != "ok" || len(bubble.Rows) < 6 || bubble.Rows[0].Cells[3] != "Holds seed 5" {
		t.Fatal(bubble.Rows[:1])
	}
	for _, row := range bubble.Rows {
		if row.Cells[4] != "10" || len(strings.Split(row.Cells[5], " · ")) != 10 {
			t.Fatal(row.Cells)
		}
	}
	brackets := v.brackets()
	if !strings.HasPrefix(brackets.Rows[0].Cells[2], "Seed 2 · ") || !strings.Contains(brackets.Rows[0].Cells[2], "(projected)") || brackets.Rows[0].Cells[4] != "Not played yet" ||
		!strings.HasPrefix(brackets.Summary[0].Cells[1], "Seeds are set when the regular season ends") {
		t.Fatal(brackets.Rows[0].Cells, brackets.Summary)
	}
}

func TestRaceWeeklyPointsShareOneScale(t *testing.T) {
	v, err := newRaceView(realRace(t, "2025"))
	if err != nil {
		t.Fatal(err)
	}
	r := v.weeklyPoints()
	if len(r.BarLabels) != 17 || r.BarMax != 409.63 && r.BarMax < 375.15 {
		t.Fatal(len(r.BarLabels), r.BarMax)
	}
	if r.Rows[0].Cells[0] != "Buffalo Bills" || r.Rows[0].Cells[1] != "4,747.96" {
		t.Fatal("most points first", r.Rows[0].Cells)
	}
	for _, row := range r.Rows {
		if len(row.Bars) != len(r.BarLabels) || slices.Max(row.Bars) > r.BarMax {
			t.Fatal(row.Cells[0])
		}
	}
	ap := v.allPlay()
	if ap.Rows[0].Cells[0] != "Buffalo Bills" || ap.Rows[0].Cells[1] != "481-46" || ap.Rows[0].Range.Max != 1 {
		t.Fatal(ap.Rows[0])
	}
}

func testPower() PowerRankingsResult {
	return PowerRankingsResult{OK: true, View: ViewSeason, Weight: 4.0 / 7, WeeksScored: 3, ModelRunID: 3,
		AggMode: "lineup", Performance: "all-play", Rows: []PowerRow{
			{Rank: 1, FranchiseID: "0032", PowerScore: 1, RosterValue: 210.4, Results: 89.0 / 93, H2HW: 3, DeltaOK: true, RankDelta: 2},
			{Rank: 2, FranchiseID: "0001", PowerScore: 0.6, RosterValue: 205.1, Results: 0.7, H2HW: 3},
		}}
}

func TestRacePowerRankingsNameTheBlend(t *testing.T) {
	in := realRace(t, "2026")
	in.held.power = map[string]PowerRankingsResult{ViewSeason: testPower(),
		ViewFranchise: {OK: true, View: ViewFranchise, AgeWeight: 0.5, ModelRunID: 3, AggMode: "sum", Rows: []PowerRow{}}}
	v, err := newRaceView(in)
	if err != nil {
		t.Fatal(err)
	}
	season := v.power(ViewSeason)
	if !slices.Equal(season.Rows[0].Cells, []string{"1", "Washington Commanders", "Up 2", "3-0", "210.4", ".957", "1.00"}) ||
		!season.Rows[1].Mine || season.Rows[0].Range.Max != 1 || season.Rows[1].Range.Hi != 0.6 {
		t.Fatal(season.Rows)
	}
	if !strings.Contains(season.How, "the roster counts 57% and results 43%") || !strings.Contains(season.How, "2022–2025") {
		t.Fatal(season.How)
	}
	if franchise := v.power(ViewFranchise); franchise.Status != "empty" || !strings.Contains(franchise.Note, "run the app's model") {
		t.Fatal(franchise.Status, franchise.Note)
	}
	in.held.power = nil
	v, _ = newRaceView(in)
	if r := v.power(ViewSeason); r.Status != "failed" {
		t.Fatal(r.Status)
	}
}

func TestRaceBindingReadsHeldAndRefreshFallsBack(t *testing.T) {
	a, transport, _, _ := seasonTestApp(t)
	board, err := a.TargetRace(0, "", "")
	if err != nil || len(transport.requests) != 0 || board.Standings.Status != "failed" {
		t.Fatal("reading fetched or invented standings", err, transport.requests)
	}
	if _, err := a.TargetRace(3, "", ""); err == nil {
		t.Fatal("accepted a week with no held results")
	}
	recordSeasonArchive(t, a, "weeklyResults", "YTD", transport.bodies["weeklyResults"], raceHeldAt)
	transport.failure = "weeklyResults"
	if err := a.TargetRaceRefresh(false); err != nil {
		t.Fatal(err)
	}
	a.seasonMu.Lock()
	weeks := a.race.weeks
	a.seasonMu.Unlock()
	if len(weeks.value) != 17 || weeks.refreshError == "" {
		t.Fatal("archived weekly results not served as stale", weeks.refreshError)
	}
	requests := len(transport.requests)
	if err := a.TargetRaceRefresh(false); err != nil || len(transport.requests) == requests {
		t.Fatal("a stale feed must refresh again", err)
	}
	transport.failure = ""
	requests = len(transport.requests)
	_ = a.TargetRaceRefresh(true)
	if err := a.TargetRaceRefresh(false); err != nil || len(transport.requests) != requests+6 {
		t.Fatal("fresh feeds refetched", transport.requests[requests:])
	}
}

func TestRaceWireMatchesGo(t *testing.T) {
	in := realRace(t, "2026")
	in.held.power = map[string]PowerRankingsResult{ViewSeason: testPower()}
	board, err := buildRace(in)
	if err != nil {
		t.Fatal(err)
	}
	board.Standings.Rows = board.Standings.Rows[:5]
	for i := range board.Board.Reports {
		r := &board.Board.Reports[i]
		r.Rows = r.Rows[:min(len(r.Rows), 3)]
	}
	body, err := json.MarshalIndent(struct {
		Race RaceBoard `json:"race"`
	}{board}, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	file := filepath.Join("frontend", "src", "app", "race", "wire.testdata.json")
	body = append(body, '\n')
	if os.Getenv("TWR_WRITE_WIRE") == "1" {
		if err := os.WriteFile(file, body, 0o644); err != nil { //nolint:gosec // a committed test fixture
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	if string(want) != string(body) {
		t.Fatal("race wire stale; TWR_WRITE_WIRE=1 go test -run TestRaceWireMatchesGo .")
	}
}

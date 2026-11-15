package race

import (
	"encoding/json"
	"regexp"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/report"
	"github.com/secureprospective/TheWarRoom/internal/app/season"
	"github.com/secureprospective/TheWarRoom/internal/app/season/seasontest"
	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
)

var raceHeldAt = time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC) //nolint:gochecknoglobals // a fixed test clock

func held[T any](v T) season.Feed[T] {
	return season.Feed[T]{Value: v, FetchedAt: raceHeldAt}
}

func realRace(t *testing.T, year string) season.Snapshot {
	d := seasontest.Load(t, year)
	return season.Snapshot{Held: season.Held{Weeks: held(d.Weeks), Brackets: held(d.Brackets), Standings: held(d.Standings)}, Config: d.Config, Lookup: d.Lookup, HasLookup: true, Team: "0001", Mine: "0001", Zone: time.UTC}
}

// bracketSeeds is MFL's seed for each franchise in a bracket, from every game side.
func BracketSeeds(b leaguefeed.Bracket) []string {
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

func TestRaceChoicesAreCheckedInGo(t *testing.T) {
	t.Parallel()
	in := realRace(t, "2026")
	for name, change := range map[string]func(*season.Snapshot){
		"unscored week": func(in *season.Snapshot) { in.Week = 4 },
		"unknown week":  func(in *season.Snapshot) { in.Week = 40 },
		"unknown team":  func(in *season.Snapshot) { in.Team = "0099" },
		"unknown mine":  func(in *season.Snapshot) { in.Mine = "x" },
	} {
		bad := in
		change(&bad)
		if _, err := Build(bad); err == nil {
			t.Errorf("%s accepted", name)
		}
	}
	in.Week, in.Team = 2, "0017"
	board, err := Build(in)
	if err != nil || board.Week != 2 || board.Team != "0017" || len(board.Weeks) != 3 || len(board.Teams) != 32 {
		t.Fatal(err, board.Week, board.Team, len(board.Weeks))
	}
	detail := reportByID(t, report.Board{Reports: board.Board.Reports}, "M-010")
	if detail.Summary[0].Cells[1] != "Atlanta Falcons" || detail.Summary[1].Cells[1] != "Week 2" ||
		detail.Summary[2].Cells[1] != "213.90" || detail.Summary[3].Cells[1] != "269.55 · 55.65 left on the bench" ||
		len(detail.Rows) < 21 {
		t.Fatal(detail.Summary, len(detail.Rows))
	}
	in.Team = ""
	board, _ = Build(in)
	if r := reportByID(t, report.Board{Reports: board.Board.Reports}, "M-012"); r.Status != "empty" || len(r.Rows) != 0 {
		t.Fatal(r.Status)
	}
}

// A row names franchises and players; it never shows an MFL id or says a name is missing.
func TestRaceRowsNeverShowIDs(t *testing.T) {
	t.Parallel()
	franchiseID := regexp.MustCompile(`(^|[^\d.,])00\d\d([^\d.,]|$)`)
	for _, year := range []string{"2025", "2026"} {
		in := realRace(t, year)
		ids := map[string]bool{}
		for _, w := range in.Held.Weeks.Value {
			for _, g := range w.Games {
				for _, p := range append(slices.Clone(g.Home.Players), g.Away.Players...) {
					ids[p.ID.String()] = true
				}
			}
		}
		board, err := Build(in)
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
	t.Parallel()
	in := realRace(t, "2026")
	in.Held = season.Held{}
	in.Config.StandingsSort = ""
	board, err := Build(in)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range append(board.Board.Reports, board.Standings) {
		if r.Status != "failed" || r.Note == "" || len(r.Rows) != 0 {
			t.Fatalf("%s: %s %q", r.ID, r.Status, r.Note)
		}
	}
	in = realRace(t, "2026")
	in.Held.Brackets = season.Feed[[]leaguefeed.Bracket]{}
	in.Config.StandingsSort = ""
	board, err = Build(in)
	if err != nil {
		t.Fatal(err)
	}
	if board.Standings.Status != "ok" || !strings.Contains(board.Standings.Note, "tiebreak order is not held") {
		t.Fatal(board.Standings.Note)
	}
	for _, id := range []string{"M-090", "M-170", "M-014"} {
		if r := reportByID(t, report.Board{Reports: board.Board.Reports}, id); r.Status != "failed" {
			t.Fatal(id, r.Status)
		}
	}
	if r := reportByID(t, report.Board{Reports: board.Board.Reports}, "M-161"); r.Status != "ok" || len(r.Rows) != 32 {
		t.Fatal("all-play needs only the weekly results")
	}
}

func testPower() season.PowerRankingsResult { return seasontest.Power() }

func TestRacePowerRankingsNameTheBlend(t *testing.T) {
	t.Parallel()
	in := realRace(t, "2026")
	in.Held.Power = map[string]season.PowerRankingsResult{season.ViewSeason: testPower(),
		season.ViewFranchise: {OK: true, View: season.ViewFranchise, AgeWeight: 0.5, ModelRunID: 3, AggMode: "sum", Rows: []season.PowerRow{}}}
	v, err := season.NewView(in)
	if err != nil {
		t.Fatal(err)
	}
	seasonReport := v.Power(season.ViewSeason)
	if !slices.Equal(seasonReport.Rows[0].Cells, []string{"1", "Washington Commanders", "Up 2", "3-0", "210.4", ".957", "1.00"}) ||
		!seasonReport.Rows[1].Mine || seasonReport.Rows[0].Range.Max != 1 || seasonReport.Rows[1].Range.Hi != 0.6 {
		t.Fatal(seasonReport.Rows)
	}
	if !strings.Contains(seasonReport.How, "the roster counts 57% and results 43%") || !strings.Contains(seasonReport.How, "2022–2025") {
		t.Fatal(seasonReport.How)
	}
	if franchise := v.Power(season.ViewFranchise); franchise.Status != "empty" || !strings.Contains(franchise.Note, "run the app's model") {
		t.Fatal(franchise.Status, franchise.Note)
	}
	in.Held.Power = nil
	v, _ = season.NewView(in)
	if r := v.Power(season.ViewSeason); r.Status != "failed" {
		t.Fatal(r.Status)
	}
}

func reportByID(t *testing.T, board report.Board, id string) report.Report {
	t.Helper()
	for _, r := range board.Reports {
		if r.ID == id {
			return r
		}
	}
	t.Fatal(id)
	return report.Report{}
}

func TestConcurrentBuildPreservesSnapshot(t *testing.T) {
	t.Parallel()
	in := realRace(t, "2026")
	slices.Reverse(in.Held.Weeks.Value)
	in.Held.Power = map[string]season.PowerRankingsResult{season.ViewSeason: testPower()}
	before, err := json.Marshal(in)
	if err != nil {
		t.Fatal(err)
	}
	var group sync.WaitGroup
	for range 12 {
		group.Go(func() {
			board, err := Build(in)
			if err != nil || len(board.Board.Reports) != 15 {
				t.Errorf("parallel construction: reports %d, error %v", len(board.Board.Reports), err)
			}
		})
	}
	group.Wait()
	after, err := json.Marshal(in)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Fatal("construction mutated its snapshot")
	}
}

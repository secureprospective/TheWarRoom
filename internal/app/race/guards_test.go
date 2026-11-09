package race

import (
	"encoding/json"
	"fmt"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/app/report"
	"github.com/secureprospective/TheWarRoom/internal/app/season"
	"github.com/secureprospective/TheWarRoom/internal/app/season/seasontest"
	"github.com/secureprospective/TheWarRoom/internal/engine/race"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/leaguestandings"
)

func mutate(in season.Held) {
	in.Weeks.Value[0].Games[0].Home.Players[0].Points = 100
	in.Weeks.Value[0].Games[0].Away.Players[0].Points = 200
	in.Weeks.Value[0].Others[0].Players[0].Points = 300
	in.Weeks.Value[0].Games[0].Regular = true
	in.Weeks.Value[0].Week = 2
	in.Brackets.Value[0].Games[0].ID = 2
	in.Brackets.Value[0].Name = "changed"
	in.Standings.Value[0].FranchiseID = "0002"
	in.Power["season"].Rows[0].Name = "changed"
	delete(in.Power, "season")
}

func TestHeldDetachesIngressAndEgress(t *testing.T) {
	h := &Held{}
	input, want := heldFixture(), heldFixture()
	publish(h, input)
	mutate(input)
	if got := h.Snapshot(); !reflect.DeepEqual(got, want) {
		t.Fatal("publication retained caller storage", got)
	}
	mutate(h.Snapshot())
	if got := h.Snapshot(); !reflect.DeepEqual(got, want) {
		t.Fatal("snapshot exposed held storage", got)
	}
	h.PublishStandings(season.Feed[[]leaguestandings.RawStanding]{RefreshError: "failed"}, nil, false)
	if got := h.Snapshot(); !reflect.DeepEqual(got.Power, want.Power) {
		t.Fatal("failed refresh cleared last good power")
	}
}

func TestHeldPublishReadRace(t *testing.T) {
	h := &Held{}
	input := heldFixture()
	publish(h, input)
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Go(func() {
			for j := 0; j < 300; j++ {
				publish(h, input)
				mutate(h.Snapshot())
			}
		})
	}
	wg.Wait()
	if !reflect.DeepEqual(h.Snapshot(), input) {
		t.Fatal("concurrent reads mutated publication")
	}
}

func mflStandings(t *testing.T, year string) []map[string]string {
	t.Helper()
	var env struct {
		LeagueStandings struct {
			Franchise []map[string]string `json:"franchise"`
		} `json:"leagueStandings"`
	}
	if err := json.Unmarshal([]byte(seasontest.Sample(t, "leaguestandings", "leagueStandings-"+year+".json")), &env); err != nil {
		t.Fatal(err)
	}
	return env.LeagueStandings.Franchise
}

func wlt(r race.Record) string { return fmt.Sprintf("%d-%d-%d", r.W, r.L, r.T) }

func assertStandingsMatchMFL(t *testing.T, year string) season.View {
	t.Helper()
	v, err := season.NewView(realRace(t, year))
	if err != nil {
		t.Fatal(err)
	}
	mfl := mflStandings(t, year)
	if len(mfl) != 32 {
		t.Fatal(len(mfl))
	}
	for _, s := range mfl {
		l, ok := v.Table.Line(s["id"])
		if !ok {
			t.Fatal(s["id"])
		}
		got := []string{wlt(l.Overall), wlt(l.InDivision), wlt(l.InConference), wlt(l.AllPlay),
			strings.ReplaceAll(report.ScoreText(l.For), ",", ""), strings.ReplaceAll(report.ScoreText(l.Against), ",", ""),
			string(l.Streak.Result) + strconv.Itoa(l.Streak.Count)}
		want := []string{s["h2hw"] + "-" + s["h2hl"] + "-" + s["h2ht"], s["divw"] + "-" + s["divl"] + "-" + s["divt"],
			s["confw"] + "-" + s["confl"] + "-" + s["conft"], s["all_play_wlt"], s["pf"], s["pa"], s["strk"]}
		if !slices.Equal(got, want) {
			t.Fatalf("%s %s: app %v, MFL %v", year, s["id"], got, want)
		}
	}
	for _, d := range v.Divisions {
		mflOrder := []string{}
		for _, s := range mfl {
			if l, _ := v.Table.Line(s["id"]); l.Division == d.ID {
				mflOrder = append(mflOrder, s["id"])
			}
		}
		if got := v.Table.Division(d.ID).Order; !slices.Equal(got, mflOrder) {
			t.Fatalf("%s %s: app %v, MFL %v", year, d.Name, got, mflOrder)
		}
	}
	return v
}

func TestRace2025StandingsMatchMFLExactly(t *testing.T) {
	t.Parallel()
	v := assertStandingsMatchMFL(t, "2025")
	if v.Table.RegularWeeks != 13 {
		t.Fatal(v.Table.RegularWeeks)
	}
	standings := v.Standings()
	first := standings.Rows[0]
	if !slices.Equal(first.Cells, []string{"Buffalo Bills", "12-1", ".923", "4,747.96", "2,400.25", "Won 11", "—"}) ||
		!first.Mine || first.Group != "AFC East" || standings.Columns[0] != "Franchise" {
		t.Fatal(first)
	}

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
	t.Parallel()
	v := assertStandingsMatchMFL(t, "2026")
	if v.Table.RegularWeeks != 3 || v.Week != 3 {
		t.Fatal(v.Table.RegularWeeks, v.Week)
	}
	if l, _ := v.Table.Line("0001"); len(l.Remaining) != 10 {
		t.Fatal(l.Remaining)
	}
}

func TestRace2025SeedsAndOpeningRoundMatchMFLBrackets(t *testing.T) {
	t.Parallel()
	v, err := season.NewView(realRace(t, "2025"))
	if err != nil {
		t.Fatal(err)
	}
	fields, note := v.Fields()
	if len(fields) != 2 || note != "" {
		t.Fatal(len(fields), note)
	}
	for _, f := range fields {
		got := []string{}
		for _, s := range f.Seeding.Seeds {
			got = append(got, s.Franchise)
		}
		if want := BracketSeeds(f.Bracket); !slices.Equal(got, want) {
			t.Fatalf("%s seeds: app %v, MFL %v", f.Conference.Name, got, want)
		}
		byes, games := race.FirstRound(f.Bracket.Teams)
		mflGames := [][2]int{}
		for _, g := range f.Bracket.Games {
			if g.Week == f.Bracket.StartWeek {
				mflGames = append(mflGames, [2]int{g.Home.Seed, g.Away.Seed})
			}
		}
		if byes != 1 || !slices.Equal(games, mflGames) {
			t.Fatalf("%s opening round: app %v, MFL %v", f.Conference.Name, games, mflGames)
		}
	}
	seeding := v.Seeding()
	if r := seeding.Rows[0]; !slices.Equal(r.Cells, []string{"AFC", "1", "Buffalo Bills", "12-1", "Leads the AFC East", "Bye"}) {
		t.Fatal(r.Cells)
	}
	if r := seeding.Rows[13]; !slices.Equal(r.Cells[1:], []string{"7", "Philadelphia Eagles", "8-5", "Wild card", "At seed 2"}) {
		t.Fatal(r.Cells)
	}
}

func TestRace2025BracketReportIsMFLsBracket(t *testing.T) {
	t.Parallel()
	v, err := season.NewView(realRace(t, "2025"))
	if err != nil {
		t.Fatal(err)
	}
	r := v.Brackets()
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
	t.Parallel()
	v, err := season.NewView(realRace(t, "2025"))
	if err != nil {
		t.Fatal(err)
	}
	r := v.Tiebreakers()
	if r.Summary[0].Cells[1] != "1 Win percentage · 2 Head-to-head among the tied teams · 3 Division win percentage · "+
		"4 Conference win percentage · 5 All-play win percentage · 6 Points for · 7 MFL power rating" {
		t.Fatal(r.Summary)
	}
	find := func(race, step string) report.Row {
		for _, row := range r.Rows {
			if strings.HasPrefix(row.Cells[0], race) && strings.HasSuffix(row.Cells[2], step) {
				return row
			}
		}
		t.Fatalf("no %s step in %s", step, race)
		return report.Row{}
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

func TestRaceStarterPointsAddUpToPointsFor(t *testing.T) {
	t.Parallel()
	v, err := season.NewView(realRace(t, "2025"))
	if err != nil {
		t.Fatal(err)
	}
	r := v.PositionPoints()
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
	for _, id := range v.Table.IDs() {
		l, _ := v.Table.Line(id)
		for _, row := range r.Rows {
			if row.Cells[0] == v.Name(id) {
				positions := int64(0)
				for _, cell := range row.Cells[2:] {
					positions += hundredths(cell)
				}
				if positions != hundredths(row.Cells[1]) || int64(l.For)-positions < 0 || (int64(l.For)-positions)%300 != 0 {
					t.Fatal(row.Cells, report.ScoreText(l.For))
				}
				gap += int64(l.For) - positions
			}
		}
	}
	if gap != 3000 {
		t.Fatal(gap)
	}
}

func TestRace2026BubbleAndUnseededBrackets(t *testing.T) {
	t.Parallel()
	v, err := season.NewView(realRace(t, "2026"))
	if err != nil {
		t.Fatal(err)
	}
	bubble := v.Bubble()
	if bubble.Status != "ok" || len(bubble.Rows) < 6 || bubble.Rows[0].Cells[3] != "Holds seed 5" {
		t.Fatal(bubble.Rows[:1])
	}
	for _, row := range bubble.Rows {
		if row.Cells[4] != "10" || len(strings.Split(row.Cells[5], " · ")) != 10 {
			t.Fatal(row.Cells)
		}
	}
	brackets := v.Brackets()
	if !strings.HasPrefix(brackets.Rows[0].Cells[2], "Seed 2 · ") || !strings.Contains(brackets.Rows[0].Cells[2], "(projected)") || brackets.Rows[0].Cells[4] != "Not played yet" ||
		!strings.HasPrefix(brackets.Summary[0].Cells[1], "Seeds are set when the regular season ends") {
		t.Fatal(brackets.Rows[0].Cells, brackets.Summary)
	}
}

func TestRaceWeeklyPointsShareOneScale(t *testing.T) {
	t.Parallel()
	v, err := season.NewView(realRace(t, "2025"))
	if err != nil {
		t.Fatal(err)
	}
	r := v.WeeklyPoints()
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
	ap := v.AllPlay()
	if ap.Rows[0].Cells[0] != "Buffalo Bills" || ap.Rows[0].Cells[1] != "481-46" || ap.Rows[0].Range.Max != 1 {
		t.Fatal(ap.Rows[0])
	}
}

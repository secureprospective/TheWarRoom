package now

import (
	"encoding/json"
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/season"

	"github.com/secureprospective/TheWarRoom/internal/app/now/nowtest"
	"github.com/secureprospective/TheWarRoom/internal/app/report"
	"github.com/secureprospective/TheWarRoom/internal/app/season/seasontest"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/leaguestandings"

	"github.com/secureprospective/TheWarRoom/internal/ingestion/livescoring"
	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"

	"github.com/secureprospective/TheWarRoom/internal/ingestion/marketfeeds"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/nflschedule"
)

// mflWeek is one week of MFL's weeklyResults export, read straight from the JSON so the reports
// are checked against MFL's own numbers, not the app's parser.
type mflWeek struct {
	Week      string
	Games     [][]mflSide
	Franchise []mflSide
}

type mflSide struct {
	ID     string `json:"id"`
	Score  string `json:"score"`
	Result string `json:"result"`
	Player json.RawMessage
}

// mflList decodes MFL's list-or-single-object field.
func mflList[T any](t *testing.T, raw json.RawMessage) []T {
	t.Helper()
	if len(raw) == 0 {
		return nil
	}
	var many []T
	if err := json.Unmarshal(raw, &many); err == nil {
		return many
	}
	var one T
	if err := json.Unmarshal(raw, &one); err != nil {
		t.Fatal(err)
	}
	return []T{one}
}

func mflWeeks(t *testing.T, year string) map[int]mflWeek {
	t.Helper()
	var env struct {
		All struct {
			Weeks []struct {
				Week      string          `json:"week"`
				Matchup   json.RawMessage `json:"matchup"`
				Franchise json.RawMessage `json:"franchise"`
			} `json:"weeklyResults"`
		} `json:"allWeeklyResults"`
	}
	if err := json.Unmarshal([]byte(nowtest.Compressed(t, "weeklyresults", "weeklyResults-"+year+".json.gz")), &env); err != nil {
		t.Fatal(err)
	}
	out := map[int]mflWeek{}
	for _, w := range env.All.Weeks {
		n, _ := strconv.Atoi(w.Week)
		week := mflWeek{Week: w.Week, Franchise: mflList[mflSide](t, w.Franchise)}
		for _, m := range mflList[struct {
			Franchise []mflSide `json:"franchise"`
		}](t, w.Matchup) {
			week.Games = append(week.Games, m.Franchise)
		}
		out[n] = week
	}
	return out
}

// hundredthsText is MFL's score string to the hundredth, as the reports print it.

func hundredthsText(t *testing.T, score string) string {
	t.Helper()
	f, err := strconv.ParseFloat(score, 64)
	if err != nil {
		t.Fatal(err)
	}
	return scoreText(leaguefeed.Points(math.Round(f * 100)))
}

func nowBoard(t *testing.T, in Snapshot) Board {
	t.Helper()
	board, err := Build(in)
	if err != nil {
		t.Fatal(err)
	}
	return board
}

func nowReport(t *testing.T, board Board, id string) report.Report {
	t.Helper()
	for _, r := range append(board.Board.Reports, board.Matchup) {
		if r.ID == id {
			return r
		}
	}
	t.Fatalf("no report %s", id)
	return report.Report{}
}

func franchiseNames(in Snapshot) map[string]string {
	names := map[string]string{}
	for _, f := range in.Race.Config.Franchises {
		names[f.ID] = f.Name
	}
	return names
}

func TestNow2025WeeklyResultsMatchMFLExactly(t *testing.T) {
	t.Parallel()
	mfl := mflWeeks(t, "2025")
	in := realNow(t, 0)
	names := franchiseNames(in)
	for week := 1; week <= 17; week++ {
		in.Week = week
		r := nowReport(t, nowBoard(t, in), "M-003")
		if r.Status != "ok" || len(r.Rows) != len(mfl[week].Games) {
			t.Fatalf("week %d: %s %q, %d rows for %d games", week, r.Status, r.Note, len(r.Rows), len(mfl[week].Games))
		}
		want := []string{}
		for _, g := range mfl[week].Games {
			win, lose := g[0], g[1]
			if lose.Result == "W" {
				win, lose = lose, win
			}
			ws, ls := hundredthsText(t, win.Score), hundredthsText(t, lose.Score)
			want = append(want, strings.Join([]string{names[win.ID], ws, names[lose.ID], ls}, "|"))
		}
		got := []string{}
		for _, row := range r.Rows {
			got = append(got, strings.Join(row.Cells[:4], "|"))
		}
		slices.Sort(want)
		slices.Sort(got)
		if !slices.Equal(want, got) {
			t.Fatalf("week %d results differ from MFL:\n%v\n%v", week, want, got)
		}
		plays := slices.ContainsFunc(mfl[week].Games, func(g []mflSide) bool { return g[0].ID == "0001" || g[1].ID == "0001" })
		if r.Rows[0].Mine != plays {
			t.Fatalf("week %d: my game is not first", week)
		}
	}
}

func TestNow2025RecapsAreSentencesFromTheNumbers(t *testing.T) {
	t.Parallel()
	r := nowReport(t, nowBoard(t, realNow(t, 5)), "M-021")
	recaps := map[string]string{}
	for _, row := range r.Rows {
		recaps[row.Cells[0]] = row.Cells[1]
	}
	for game, want := range map[string]string{
		"Buffalo Bills at New York Jets": "Buffalo Bills beat New York Jets 277.03 to 179.98, a margin of 97.05. " +
			"Buffalo Bills' top starter was Fields, Justin (QB) with 31.80. New York Jets' top starter was Sweat, " +
			"Josh (DE) with 27.50. New York Jets left 26.05 on the bench; its best lineup, 206.03, would still have lost.",
		"Indianapolis Colts at Los Angeles Rams": "Indianapolis Colts beat Los Angeles Rams 195.05 to 193.20, a margin " +
			"of 1.85. Indianapolis Colts' top starter was Williams, Javonte (RB) with 31.30. Los Angeles Rams' top " +
			"starter was Stafford, Matthew (QB) with 32.60. Los Angeles Rams left 44.85 on the bench; its best " +
			"lineup, 238.05, would have won.",
		"Jacksonville Jaguars at Arizona Cardinals": "Arizona Cardinals beat Jacksonville Jaguars 281.60 to 270.95, a " +
			"margin of 10.65. Arizona Cardinals' top starter was Stroud, C.J. (QB) with 37.35. Jacksonville Jaguars' " +
			"top starter was Mayfield, Baker (QB) with 34.20. Jacksonville Jaguars started its best lineup.",
	} {
		if recaps[game] != want {
			t.Fatalf("%s:\n got %q\nwant %q", game, recaps[game], want)
		}
	}
	if !r.Rows[0].Mine || r.Rows[0].Cells[0] != "Buffalo Bills at New York Jets" {
		t.Fatal("my game is not first", r.Rows[0].Cells)
	}
	playoff := nowReport(t, nowBoard(t, realNow(t, 15)), "M-021")
	if !strings.HasPrefix(playoff.Rows[0].Cells[1], "Playoff game. Buffalo Bills beat Houston Texans 312.70 to 250.58") {
		t.Fatal(playoff.Rows[0].Cells[1])
	}
}

func TestNowMyMatchupPairsStartersAndTotals(t *testing.T) {
	t.Parallel()
	in := realNow(t, 5)
	r := nowReport(t, nowBoard(t, in), "X-03")
	if strings.Join(r.Columns, "|") != "Slot|Buffalo Bills|Points|New York Jets|Points" {
		t.Fatal(r.Columns)
	}
	if strings.Join(r.Rows[0].Cells, "|") != "QB|Fields, Justin · NYJ|31.80|Rush, Cooper · BAL|3.95" {
		t.Fatal(r.Rows[0].Cells)
	}
	var mine, theirs leaguefeed.Points
	for _, row := range r.Rows[:len(r.Rows)-1] {
		for i, sum := range map[int]*leaguefeed.Points{2: &mine, 4: &theirs} {
			if row.Cells[i] != "" {
				f, _ := strconv.ParseFloat(row.Cells[i], 64)
				*sum += hundredths(f)
			}
		}
	}
	total := r.Rows[len(r.Rows)-1].Cells
	if total[0] != "Total" || total[2] != "277.03" || total[4] != "179.98" ||
		scoreText(mine) != "277.03" || scoreText(theirs) != "179.98" {
		t.Fatal("starter points must add up to MFL's totals", total, mine, theirs)
	}
	in.Race.Mine = ""
	r = nowReport(t, nowBoard(t, in), "X-03")
	if r.Status != "empty" || r.Note != "Choose your franchise to see your matchup." {
		t.Fatal(r.Status, r.Note)
	}
}

func TestNowWeekIsCheckedInGo(t *testing.T) {
	t.Parallel()
	in := realNow(t, 0)
	if b := nowBoard(t, in); b.Week != 5 || len(b.Weeks) != 17 || b.Weeks[14].Label != "Week 15 · playoffs" {
		t.Fatal("the current week is the default", b.Week, b.Weeks)
	}
	in.Current = 0
	if b := nowBoard(t, in); b.Week != 17 {
		t.Fatal("without a current week the latest scored week shows", b.Week)
	}
	for _, week := range []int{-1, 18, 99} {
		in.Week = week
		if _, err := Build(in); err == nil {
			t.Fatal("accepted week", week)
		}
	}
	in.Week, in.Race.Mine = 0, "9999"
	if _, err := Build(in); err == nil {
		t.Fatal("accepted an unknown franchise")
	}
}

func TestNowUnscoredWeekShowsHeldLiveScores(t *testing.T) {
	t.Parallel()
	raw, err := livescoring.Parse([]byte(seasontest.Sample(t, "livescoring", "liveScoring-w5.json")))
	if err != nil {
		t.Fatal(err)
	}
	lineups, err := livescoring.ToLineups(raw)
	if err != nil {
		t.Fatal(err)
	}
	in := Snapshot{Race: realRace(t, "2026"), Current: 5, ReportWeek: 5, At: fixtureAt()}
	in.Race.Mine = lineups.Matchups[0].Home
	before := nowBoard(t, in)
	if r := nowReport(t, before, "X-03"); r.Rows[len(r.Rows)-1].Cells[2] != "—" {
		t.Fatal("an unscored week without live scores shows no points", r.Rows)
	}
	lineups.Franchises[0].Score = 12.345
	live := held(lineups)
	live.FetchedAt = fixtureAt().Add(time.Hour)
	in.Live = live
	board := nowBoard(t, in)
	r := nowReport(t, board, "X-03")
	if r.AsOf != "As of: Oct 11, 2026, 1:00 PM" || !strings.Contains(r.Note, "Held live scores, not final") {
		t.Fatal(r.AsOf, r.Note)
	}
	// Mine is the home side; my side still reads first, as in the matchup's columns.
	mineName := r.Columns[1] // the matchup's first franchise column is mine
	for _, row := range r.Summary {
		if row.Cells[0] == "Status" && !strings.HasPrefix(row.Cells[1], "In progress: "+mineName+" ") {
			t.Fatal("my side first", row.Cells[1])
		}
	}
	totals := strings.Join(r.Rows[len(r.Rows)-1].Cells, "|")
	if !strings.Contains(totals, "12.35") {
		t.Fatal("live total not shown", totals)
	}
	if res := nowReport(t, board, "M-003"); res.Status != "empty" || !strings.Contains(res.Note, "MFL has not scored week 5 yet.") {
		t.Fatal(res.Status, res.Note)
	}
	in.Live.Value.Week = 4
	if r := nowReport(t, nowBoard(t, in), "X-03"); strings.Contains(r.Note, "Held live") {
		t.Fatal("another week's live scores must not stand in")
	}
}

func TestPulsePlayerKickoffAndStableSlots(t *testing.T) {
	t.Parallel()
	in := realNow(t, 5)
	v, err := newNowView(in)
	if err != nil {
		t.Fatal(err)
	}
	v.live = true
	me, them, _, ok := v.myGame()
	if !ok {
		t.Fatal("fixture has no matchup")
	}
	me.Scored = false
	var checked bool
	for _, p := range me.Players {
		g, found := v.games[v.team(p)]
		if !found {
			continue
		}
		p.Points = 0
		v.now.At = g.kickoff.Add(-time.Second)
		want := "yet to play · " + g.kickoff.In(v.in.Zone).Format("Mon 3:04 PM")
		if got := v.playerPointsText(me, p); got != want {
			t.Fatalf("before kickoff: %q, want %q", got, want)
		}
		me.Players = []leaguefeed.WeekPlayer{p}
		p.Starter = true
		me.Players[0] = p
		if rows := v.classicRows(me, them); !strings.Contains(fmt.Sprint(rows), want) {
			t.Fatal("matchup bypassed the kickoff rule", rows)
		}
		if row := v.boxRow(me, p, ""); row.Cells[3] != want {
			t.Fatal("box score bypassed the kickoff rule", row)
		}
		v.now.At = g.kickoff
		if got := v.playerPointsText(me, p); got != "0.00" {
			t.Fatal("at kickoff", got)
		}
		checked = true
		break
	}
	if !checked {
		t.Fatal("no player with a held NFL kickoff")
	}
}

func TestPulseLiveWeekUnplayedFollowsKickoffs(t *testing.T) {
	t.Parallel()
	now := realNow(t, 5)
	games := NFLGames(now.Held.Schedules[5].Value, now.At)
	first := time.Time{}
	for _, g := range games {
		if first.IsZero() || g.kickoff.Before(first) {
			first = g.kickoff
		}
	}
	live := held(leaguefeed.Lineups{Week: 5})
	if !LiveWeekUnplayed(live, now.Held.Schedules, first.Add(-time.Minute)) {
		t.Fatal("before the first kickoff the live week must read unplayed")
	}
	if LiveWeekUnplayed(live, now.Held.Schedules, first) {
		t.Fatal("at the first kickoff the live week has begun")
	}
	if LiveWeekUnplayed(live, nil, first.Add(-time.Minute)) {
		t.Fatal("a week whose schedule is not held counts as begun")
	}
}

func TestPulseByeAndNotStartedWeek(t *testing.T) {
	t.Parallel()
	v, err := newNowView(realNow(t, 5))
	if err != nil {
		t.Fatal(err)
	}
	if len(v.games) == 0 {
		t.Fatal("fixture holds no NFL schedule for week 5")
	}
	v.live = true
	game, _, _, ok := v.myGame()
	if !ok {
		t.Fatal("fixture has no matchup")
	}
	first := time.Time{}
	for _, g := range v.games {
		if first.IsZero() || g.kickoff.Before(first) {
			first = g.kickoff
		}
	}
	all := leaguefeed.Game{Home: game, Away: game}
	all.Home.Scored, all.Away.Scored = false, false
	v.now.At = first.Add(-time.Minute)
	if got := v.gameState(all); got != "Not played yet" {
		t.Fatalf("before the first kickoff: %q", got)
	}
	v.now.At = first
	if got := v.gameState(all); !strings.HasPrefix(got, "In progress: ") {
		t.Fatalf("at the first kickoff: %q", got)
	}
	game.Scored = false
	p := game.Players[0]
	delete(v.games, v.team(p))
	if v.team(p) == "" || v.team(p) == "FA" {
		t.Fatal("fixture player has no NFL team", p.ID)
	}
	if got := v.playerPointsText(game, p); got != "bye" {
		t.Fatalf("team without a game: %q, want bye", got)
	}
	if noGameText("FA") != "no NFL team" || noGameText("") != "no NFL team" {
		t.Fatal("an NFL free agent must read no NFL team")
	}
}

func fixtureAt() time.Time    { return time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC) }
func held[T any](v T) Feed[T] { return Feed[T]{Value: v, FetchedAt: fixtureAt()} }
func realRace(t *testing.T, year string) season.Snapshot {
	d := seasontest.Load(t, year)
	return season.Snapshot{Held: season.Held{Weeks: season.Feed[[]leaguefeed.WeekResult]{Value: d.Weeks, FetchedAt: fixtureAt()}, Brackets: season.Feed[[]leaguefeed.Bracket]{Value: d.Brackets, FetchedAt: fixtureAt()}, Standings: season.Feed[[]leaguestandings.RawStanding]{Value: d.Standings, FetchedAt: fixtureAt()}}, Config: d.Config, Lookup: d.Lookup, HasLookup: true, Team: "0001", Mine: "0001", Zone: time.UTC}
}
func realNow(t *testing.T, week int) Snapshot {
	d := nowtest.Load(t)
	return Snapshot{Race: realRace(t, "2025"), Week: week, Current: 5, At: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), Held: Feeds{Schedules: map[int]Feed[[]nflschedule.RawWeek]{5: held([]nflschedule.RawWeek{d.Schedule})}, Byes: held(d.Byes)}, Projections: held([]marketfeeds.Feed{d.Projections})}
}

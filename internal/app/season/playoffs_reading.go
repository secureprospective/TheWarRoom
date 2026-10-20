package season

import (
	"cmp"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/app/report"

	"github.com/secureprospective/TheWarRoom/internal/engine/race"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/league"
	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
)

// raceField is one conference's playoff picture and the bracket it seeds.
type Field struct {
	Conference league.Conference
	Bracket    leaguefeed.Bracket
	Seeding    race.Field
}

// fields seeds every conference that has a bracket named for it; note says what could not be.
func (v View) Fields() ([]Field, string) {
	if len(v.Conferences) == 0 {
		return nil, "MFL's league settings list no conferences; refresh from MFL."
	}
	out, missing := []Field{}, []string{}
	for _, c := range v.Conferences {
		b, ok := v.BracketFor(c)
		if !ok {
			missing = append(missing, c.Name)
			continue
		}
		divisions := []string{}
		for _, d := range v.Divisions {
			if d.Conference == c.ID {
				divisions = append(divisions, d.ID)
			}
		}
		f, err := v.Table.Seeding(c.ID, divisions, b.Teams)
		if err != nil {
			missing = append(missing, c.Name)
			continue
		}
		out = append(out, Field{Conference: c, Bracket: b, Seeding: f})
	}
	note := ""
	if len(missing) > 0 {
		note = "No playoff bracket held for " + strings.Join(missing, " or ") + "; refresh from MFL."
	}
	return out, note
}

// bracketFor is the bracket whose name carries the conference's name, as MFL names them
// ("AFC Conference").
func (v View) BracketFor(c league.Conference) (leaguefeed.Bracket, bool) {
	for _, b := range v.in.Held.Brackets.Value {
		if c.Name != "" && slices.Contains(strings.Fields(b.Name), c.Name) {
			return b, true
		}
	}
	return leaguefeed.Bracket{}, false
}

func (v View) PlayoffReport(id, title, how string, columns ...string) report.Report {
	r := v.WeeksReport(id, title, how, columns...)
	if r.Status != "failed" && v.in.Held.Brackets.FetchedAt.IsZero() {
		r.Status, r.Note = "failed", "Playoff brackets unavailable; refresh from MFL to fill this report."
	}
	return r
}

func (v View) SeasonSummary(r *report.Report) {
	last, _ := strconv.Atoi(v.in.Config.LastRegularSeasonWeek)
	text := fmt.Sprintf("%d weeks scored", v.Table.RegularWeeks)
	if last > 0 {
		text = fmt.Sprintf("%d of %d weeks scored", v.Table.RegularWeeks, last)
	}
	r.Summary = append(r.Summary, report.Row{Cells: []string{"Regular season", text}})
}

func (v View) Seeding() report.Report {
	r := v.PlayoffReport("M-090", "Projected playoff seeding",
		"If the regular season ended today: each division leader is seeded first, in the league's tiebreak "+
			"order, then the best of the rest as wild cards, until each conference's bracket is full. The opening "+
			"round follows the bracket's size: the top seeds sit out, and the rest meet highest seed against lowest.",
		"Conference", "Seed", "Franchise", "Record", "How they get in", "Opening round")
	if r.Status == "failed" {
		return r
	}
	v.SeasonSummary(&r)
	fields, note := v.Fields()
	r.Note = strings.TrimSpace(r.Note + " " + note + " " + v.SortNote)
	for _, f := range fields {
		byes, games := race.FirstRound(len(f.Seeding.Seeds))
		opening := map[int]string{}
		for _, g := range games {
			opening[g[0]] = "Hosts seed " + strconv.Itoa(g[1])
			opening[g[1]] = "At seed " + strconv.Itoa(g[0])
		}
		for _, s := range f.Seeding.Seeds {
			l, _ := v.Table.Line(s.Franchise)
			how := "Wild card"
			if s.DivisionWinner {
				how = "Leads the " + v.DivisionName(l.Division)
			}
			first := opening[s.Number]
			if s.Number <= byes {
				first = "Bye"
			}
			r.Rows = append(r.Rows, v.Row(s.Franchise, f.Conference.Name, strconv.Itoa(s.Number), v.Name(s.Franchise),
				recordText(l.Overall), how, first))
		}
	}
	return r
}

func (v View) Bubble() report.Report {
	r := v.PlayoffReport("M-170", "On the bubble",
		"The wild-card holders and every franchise that can still catch its conference's last seed: one that "+
			"trails it by no more games than it has left. Games back: a win and a loss are one game apart. A team "+
			"level on games back trails on the tiebreak order.",
		"Conference", "Franchise", "Record", "Place", "Games left", "Still to play")
	if r.Status == "failed" {
		return r
	}
	v.SeasonSummary(&r)
	fields, note := v.Fields()
	r.Note = strings.TrimSpace(r.Note + " " + note)
	for _, f := range fields {
		for _, c := range v.Table.Bubble(f.Seeding) {
			l, _ := v.Table.Line(c.Franchise)
			place := "Holds seed " + strconv.Itoa(c.Seed)
			switch {
			case c.Seed == 0 && c.HalfGamesBack == 0:
				place = "Level with seed " + strconv.Itoa(len(f.Seeding.Seeds)) + "; behind on the tiebreak order"
			case c.Seed == 0:
				place = halfGamesText(c.HalfGamesBack) + " " + report.PluralNoun(float64(c.HalfGamesBack)/2, "game") + " back"
			}
			left, titles := []string{}, []string{}
			for _, o := range l.Remaining {
				left, titles = append(left, v.Short(o)), append(titles, v.Name(o))
			}
			row := v.Row(c.Franchise, f.Conference.Name, v.Name(c.Franchise), recordText(l.Overall), place,
				strconv.Itoa(len(l.Remaining)), strings.Join(left, " · "))
			row.Titles = []string{"", "", "", "", "", strings.Join(titles, ", ")}
			r.Rows = append(r.Rows, row)
		}
	}
	if len(r.Rows) == 0 && r.Status == "ok" {
		r.Status, r.Note = "empty", strings.TrimSpace(r.Note+" No franchise is in reach of a playoff seed.")
	}
	return r
}

func keyName(k race.Key) string {
	return map[race.Key]string{
		race.KeyPct: "Win percentage", race.KeyH2H: "Head-to-head among the tied teams",
		race.KeyDiv: "Division win percentage", race.KeyConf: "Conference win percentage",
		race.KeyAllPlay: "All-play win percentage", race.KeyPoints: "Points for", race.KeyPower: "MFL power rating",
	}[k]
}

func skipReason(k race.Key) string {
	return map[race.Key]string{
		race.KeyH2H:   "Skipped: not every tied team has played every other",
		race.KeyDiv:   "Skipped: the tied teams are not all in one division",
		race.KeyConf:  "Skipped: the tied teams are not all in one conference",
		race.KeyPower: "Skipped: MFL's power rating is not held",
	}[k]
}

func (v View) ValueText(k race.Key, val race.Value) string {
	switch {
	case k == race.KeyAllPlay:
		return pctText(val.Record) + " (" + recordText(val.Record) + ")"
	case val.IsRecord:
		return recordText(val.Record)
	}
	return scoreText(leaguefeed.Points(val.Amount))
}

type raceTie struct {
	race string
	tie  race.Tie
}

// ties are the ties that decide a place: within each division, among the division leaders, and
// among the wild-card contenders where a tie reaches into the seeds.
func (v View) ties() []raceTie {
	out := []raceTie{}
	for _, d := range v.Divisions {
		for _, t := range v.Table.Division(d.ID).Ties {
			out = append(out, raceTie{d.Name, t})
		}
	}
	fields, _ := v.Fields()
	for _, f := range fields {
		for _, t := range f.Seeding.Winners.Ties {
			out = append(out, raceTie{f.Conference.Name + " division leaders", t})
		}
		wildSeats := len(f.Seeding.Seeds) - len(f.Seeding.Winners.Order)
		for _, t := range f.Seeding.Wild.Ties {
			if slices.Index(f.Seeding.Wild.Order, t.Teams[0]) < wildSeats {
				out = append(out, raceTie{f.Conference.Name + " wild cards", t})
			}
		}
	}
	return out
}

func (v View) Tiebreakers() report.Report {
	r := v.WeeksReport("M-074", "Tiebreakers",
		"The league's own tiebreak order from MFL, applied to today's ties step by step. When a step separates "+
			"some teams but leaves others level, those still level start again from the first tiebreak. "+
			"Head-to-head counts only when every tied team has played every other; division and conference "+
			"percentage count only when all the tied teams share that division or conference. All-play and points "+
			"for count every week MFL scored.",
		"Race", "Tied teams", "Step", "What it found", "Result")
	steps := []string{}
	for i, k := range v.TableKeys() {
		steps = append(steps, strconv.Itoa(i+1)+" "+keyName(k))
	}
	r.Summary = append(r.Summary, report.Row{Cells: []string{"Tiebreak order", strings.Join(steps, " · ")}})
	r.Note = strings.TrimSpace(r.Note + " " + v.SortNote)
	if r.Status == "failed" {
		return r
	}
	for _, t := range v.ties() {
		l, _ := v.Table.Line(t.tie.Teams[0])
		label := t.race + " · " + recordText(l.Overall)
		for _, s := range t.tie.Steps {
			r.Rows = append(r.Rows, v.StepRow(label, t.tie, s))
		}
	}
	if len(r.Rows) == 0 && r.Status == "ok" {
		r.Status, r.Note = "empty", strings.TrimSpace(r.Note+" No ties decide a place right now.")
	}
	return r
}

func (v View) TableKeys() []race.Key {
	keys, err := race.ParseSort(v.in.Config.StandingsSort)
	if err != nil {
		return []race.Key{race.KeyPct}
	}
	return keys
}

func (v View) StepRow(label string, tie race.Tie, s race.Step) report.Row {
	names := []string{}
	for _, id := range s.Teams {
		names = append(names, v.Name(id))
	}
	step := keyName(s.Key)
	if i := slices.Index(v.TableKeys(), s.Key); i >= 0 {
		step = strconv.Itoa(i+1) + ". " + step
	}
	if s.Depth > 0 {
		step = "Then, among those still level: " + step
	}
	found, result := []string{}, ""
	switch s.Outcome {
	case race.Skipped:
		found, result = []string{skipReason(s.Key)}, "Skipped"
	case race.Level, race.Split:
		for i, id := range s.Teams {
			found = append(found, v.Name(id)+" "+v.ValueText(s.Key, s.Values[i]))
		}
		result = "Still level"
		if s.Outcome == race.Split {
			result = "Order: " + strings.Join(v.NamesInOrder(tie.Teams, s.Teams), ", then ")
		}
	case race.Unbroken:
		step, result = "Every step level", "MFL's franchise order stands: "+strings.Join(names, ", then ")
	}
	return report.Row{Cells: []string{label, strings.Join(names, " · "), step, strings.Join(found, "\n"), result},
		Mine: slices.Contains(s.Teams, v.in.Mine) && v.in.Mine != ""}
}

// namesInOrder names the members of group in the order the whole tie finished.
func (v View) NamesInOrder(final, group []string) []string {
	out := []string{}
	for _, id := range final {
		if slices.Contains(group, id) {
			out = append(out, v.Name(id))
		}
	}
	return out
}

func (v View) Brackets() report.Report {
	r := shell("M-014", "Playoff brackets",
		"Every bracket the league defines on MFL, game by game, with MFL's seeds and scores. A side not yet "+
			"known shows the seed or the earlier game it comes from. The winner is MFL's result for that week's game.",
		v.in.Held.Brackets, v.in.Zone, "Bracket", "Round", "Home", "Away", "Result")
	if r.Status == "empty" {
		r.Note = "The league defines no playoff brackets on MFL."
	}
	brackets := slices.Clone(v.in.Held.Brackets.Value)
	slices.SortStableFunc(brackets, func(a, b leaguefeed.Bracket) int { return cmp.Compare(a.StartWeek, b.StartWeek) })
	names := map[int]string{}
	for _, b := range brackets {
		names[b.ID] = b.Name
	}
	for _, b := range brackets {
		r.Summary = append(r.Summary, report.Row{Cells: []string{b.Name, v.BracketSeeds(b)}})
		for _, g := range b.Games {
			row := v.Row("", b.Name, "Week "+strconv.Itoa(g.Week), v.ProjectedBracketSide(b, g.Home, names), v.ProjectedBracketSide(b, g.Away, names),
				v.BracketResult(g))
			row.Mine = v.in.Mine != "" && (g.Home.Franchise == v.in.Mine || g.Away.Franchise == v.in.Mine)
			r.Rows = append(r.Rows, row)
		}
		if champ := v.BracketWinner(b); champ != "" {
			r.Summary = append(r.Summary, report.Row{Cells: []string{b.Title, v.Name(champ)}})
		}
	}
	return r
}

func (v View) BracketSeeds(b leaguefeed.Bracket) string {
	seeds := map[int]string{}
	for _, g := range b.Games {
		for _, s := range []leaguefeed.BracketSide{g.Home, g.Away} {
			if s.Seed > 0 && s.Franchise != "" {
				seeds[s.Seed] = s.Franchise
			}
		}
	}
	if len(seeds) == 0 && slices.ContainsFunc(b.Games, func(g leaguefeed.BracketGame) bool {
		return g.Home.FromBracket != 0 && g.Away.FromBracket != 0
	}) {
		return "Its teams are the other brackets' winners."
	}
	if len(seeds) == 0 {
		return "Seeds are set when the regular season ends; Projected playoff seeding shows them if it ended today."
	}
	out := []string{}
	for n := 1; n <= b.Teams; n++ {
		if id, ok := seeds[n]; ok {
			out = append(out, strconv.Itoa(n)+" "+v.Name(id))
		}
	}
	return "Seeds: " + strings.Join(out, " · ")
}

func (v View) ProjectedBracketSide(b leaguefeed.Bracket, s leaguefeed.BracketSide, names map[int]string) string {
	label := v.BracketSide(s, names)
	if s.Franchise != "" || s.Seed == 0 {
		return label
	}
	fields, _ := v.Fields()
	for _, f := range fields {
		if f.Bracket.ID != b.ID {
			continue
		}
		for _, seed := range f.Seeding.Seeds {
			if seed.Number == s.Seed {
				return label + " · " + v.Name(seed.Franchise) + " (projected)"
			}
		}
	}
	return label
}

func (v View) BracketSide(s leaguefeed.BracketSide, names map[int]string) string {
	from := ""
	switch {
	case s.FromBracket != 0:
		from = names[s.FromBracket] + " winner"
	case s.WinnerOf != 0:
		from = "Winner of game " + strconv.Itoa(s.WinnerOf)
	}
	switch {
	case s.Franchise != "" && s.Seed > 0:
		return "Seed " + strconv.Itoa(s.Seed) + " · " + v.Name(s.Franchise)
	case s.Franchise != "":
		return v.Name(s.Franchise)
	case s.Seed > 0:
		return "Seed " + strconv.Itoa(s.Seed)
	}
	return from
}

// bracketWinnerOf is MFL's winner of a bracket game: that week's result between the two teams,
// or the higher score when the week's results do not list the game.
func (v View) BracketWinnerOf(g leaguefeed.BracketGame) string {
	if !g.Home.Scored || !g.Away.Scored {
		return ""
	}
	for _, w := range v.Weeks {
		if game, ok := FindGame(w, g.Home.Franchise); ok && w.Week == g.Week && game.Home.Scored {
			for _, s := range []leaguefeed.Side{game.Home, game.Away} {
				if s.Result == "W" && (s.Franchise == g.Home.Franchise || s.Franchise == g.Away.Franchise) {
					return s.Franchise
				}
			}
		}
	}
	switch {
	case g.Home.Points > g.Away.Points:
		return g.Home.Franchise
	case g.Away.Points > g.Home.Points:
		return g.Away.Franchise
	}
	return ""
}

func (v View) BracketResult(g leaguefeed.BracketGame) string {
	if !g.Home.Scored || !g.Away.Scored {
		return "Not played yet"
	}
	win, lose := g.Home, g.Away
	if v.BracketWinnerOf(g) == g.Away.Franchise {
		win, lose = g.Away, g.Home
	}
	if v.BracketWinnerOf(g) == "" {
		return "Level " + scoreText(win.Points) + " to " + scoreText(lose.Points)
	}
	return v.Name(win.Franchise) + " won " + scoreText(win.Points) + " to " + scoreText(lose.Points)
}

// bracketWinner is the winner of the bracket's last game, once it is played.
func (v View) BracketWinner(b leaguefeed.Bracket) string {
	if len(b.Games) == 0 {
		return ""
	}
	last := b.Games[0]
	for _, g := range b.Games {
		if g.Week > last.Week || (g.Week == last.Week && g.ID > last.ID) {
			last = g
		}
	}
	return v.BracketWinnerOf(last)
}

package main

import (
	"cmp"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/engine/race"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/league"
	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
)

// raceField is one conference's playoff picture and the bracket it seeds.
type raceField struct {
	conference league.Conference
	bracket    leaguefeed.Bracket
	field      race.Field
}

// fields seeds every conference that has a bracket named for it; note says what could not be.
func (v raceView) fields() ([]raceField, string) {
	if len(v.conferences) == 0 {
		return nil, "MFL's league settings list no conferences; refresh from MFL."
	}
	out, missing := []raceField{}, []string{}
	for _, c := range v.conferences {
		b, ok := v.bracketFor(c)
		if !ok {
			missing = append(missing, c.Name)
			continue
		}
		divisions := []string{}
		for _, d := range v.divisions {
			if d.Conference == c.ID {
				divisions = append(divisions, d.ID)
			}
		}
		f, err := v.table.Seeding(c.ID, divisions, b.Teams)
		if err != nil {
			missing = append(missing, c.Name)
			continue
		}
		out = append(out, raceField{conference: c, bracket: b, field: f})
	}
	note := ""
	if len(missing) > 0 {
		note = "No playoff bracket held for " + strings.Join(missing, " or ") + "; refresh from MFL."
	}
	return out, note
}

// bracketFor is the bracket whose name carries the conference's name, as MFL names them
// ("AFC Conference").
func (v raceView) bracketFor(c league.Conference) (leaguefeed.Bracket, bool) {
	for _, b := range v.in.held.brackets.value {
		if c.Name != "" && slices.Contains(strings.Fields(b.Name), c.Name) {
			return b, true
		}
	}
	return leaguefeed.Bracket{}, false
}

func (v raceView) playoffReport(id, title, how string, columns ...string) Report {
	r := v.weeksReport(id, title, how, columns...)
	if r.Status != "failed" && v.in.held.brackets.fetchedAt.IsZero() {
		r.Status, r.Note = "failed", "Playoff brackets unavailable; refresh from MFL to fill this report."
	}
	return r
}

func (v raceView) seasonSummary(r *Report) {
	last, _ := strconv.Atoi(v.in.config.LastRegularSeasonWeek)
	text := fmt.Sprintf("%d weeks scored", v.table.RegularWeeks)
	if last > 0 {
		text = fmt.Sprintf("%d of %d weeks scored", v.table.RegularWeeks, last)
	}
	r.Summary = append(r.Summary, ReportRow{Cells: []string{"Regular season", text}})
}

func (v raceView) seeding() Report {
	r := v.playoffReport("M-090", "Projected playoff seeding",
		"If the regular season ended today: each division leader is seeded first, in the league's tiebreak "+
			"order, then the best of the rest as wild cards, until each conference's bracket is full. The opening "+
			"round follows the bracket's size: the top seeds sit out, and the rest meet highest seed against lowest.",
		"Conference", "Seed", "Franchise", "Record", "How they get in", "Opening round")
	if r.Status == "failed" {
		return r
	}
	v.seasonSummary(&r)
	fields, note := v.fields()
	r.Note = strings.TrimSpace(r.Note + " " + note + " " + v.sortNote)
	for _, f := range fields {
		byes, games := race.FirstRound(len(f.field.Seeds))
		opening := map[int]string{}
		for _, g := range games {
			opening[g[0]] = "Hosts seed " + strconv.Itoa(g[1])
			opening[g[1]] = "At seed " + strconv.Itoa(g[0])
		}
		for _, s := range f.field.Seeds {
			l, _ := v.table.Line(s.Franchise)
			how := "Wild card"
			if s.DivisionWinner {
				how = "Leads the " + v.divisionName(l.Division)
			}
			first := opening[s.Number]
			if s.Number <= byes {
				first = "Bye"
			}
			r.Rows = append(r.Rows, v.row(s.Franchise, f.conference.Name, strconv.Itoa(s.Number), v.name(s.Franchise),
				recordText(l.Overall), how, first))
		}
	}
	return r
}

func (v raceView) bubble() Report {
	r := v.playoffReport("M-170", "On the bubble",
		"The wild-card holders and every franchise that can still catch its conference's last seed: one that "+
			"trails it by no more games than it has left. Games back: a win and a loss are one game apart. A team "+
			"level on games back trails on the tiebreak order.",
		"Conference", "Franchise", "Record", "Place", "Games left", "Still to play")
	if r.Status == "failed" {
		return r
	}
	v.seasonSummary(&r)
	fields, note := v.fields()
	r.Note = strings.TrimSpace(r.Note + " " + note)
	for _, f := range fields {
		for _, c := range v.table.Bubble(f.field) {
			l, _ := v.table.Line(c.Franchise)
			place := "Holds seed " + strconv.Itoa(c.Seed)
			switch {
			case c.Seed == 0 && c.HalfGamesBack == 0:
				place = "Level with seed " + strconv.Itoa(len(f.field.Seeds)) + "; behind on the tiebreak order"
			case c.Seed == 0:
				place = halfGamesText(c.HalfGamesBack) + " games back of seed " + strconv.Itoa(len(f.field.Seeds))
			}
			left, titles := []string{}, []string{}
			for _, o := range l.Remaining {
				left, titles = append(left, v.short(o)), append(titles, v.name(o))
			}
			row := v.row(c.Franchise, f.conference.Name, v.name(c.Franchise), recordText(l.Overall), place,
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

func (v raceView) valueText(k race.Key, val race.Value) string {
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
func (v raceView) ties() []raceTie {
	out := []raceTie{}
	for _, d := range v.divisions {
		for _, t := range v.table.Division(d.ID).Ties {
			out = append(out, raceTie{d.Name, t})
		}
	}
	fields, _ := v.fields()
	for _, f := range fields {
		for _, t := range f.field.Winners.Ties {
			out = append(out, raceTie{f.conference.Name + " division leaders", t})
		}
		wildSeats := len(f.field.Seeds) - len(f.field.Winners.Order)
		for _, t := range f.field.Wild.Ties {
			if slices.Index(f.field.Wild.Order, t.Teams[0]) < wildSeats {
				out = append(out, raceTie{f.conference.Name + " wild cards", t})
			}
		}
	}
	return out
}

func (v raceView) tiebreakers() Report {
	r := v.weeksReport("M-074", "Tiebreakers",
		"The league's own tiebreak order from MFL, applied to today's ties step by step. When a step separates "+
			"some teams but leaves others level, those still level start again from the first tiebreak. "+
			"Head-to-head counts only when every tied team has played every other; division and conference "+
			"percentage count only when all the tied teams share that division or conference. All-play and points "+
			"for count every week MFL scored.",
		"Race", "Tied teams", "Step", "What it found", "Result")
	steps := []string{}
	for i, k := range v.tableKeys() {
		steps = append(steps, strconv.Itoa(i+1)+" "+keyName(k))
	}
	r.Summary = append(r.Summary, ReportRow{Cells: []string{"Tiebreak order", strings.Join(steps, " · ")}})
	r.Note = strings.TrimSpace(r.Note + " " + v.sortNote)
	if r.Status == "failed" {
		return r
	}
	for _, t := range v.ties() {
		l, _ := v.table.Line(t.tie.Teams[0])
		label := t.race + " · " + recordText(l.Overall)
		for _, s := range t.tie.Steps {
			r.Rows = append(r.Rows, v.stepRow(label, t.tie, s))
		}
	}
	if len(r.Rows) == 0 && r.Status == "ok" {
		r.Status, r.Note = "empty", strings.TrimSpace(r.Note+" No ties decide a place right now.")
	}
	return r
}

func (v raceView) tableKeys() []race.Key {
	keys, err := race.ParseSort(v.in.config.StandingsSort)
	if err != nil {
		return []race.Key{race.KeyPct}
	}
	return keys
}

func (v raceView) stepRow(label string, tie race.Tie, s race.Step) ReportRow {
	names := []string{}
	for _, id := range s.Teams {
		names = append(names, v.name(id))
	}
	step := keyName(s.Key)
	if i := slices.Index(v.tableKeys(), s.Key); i >= 0 {
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
			found = append(found, v.name(id)+" "+v.valueText(s.Key, s.Values[i]))
		}
		result = "Still level"
		if s.Outcome == race.Split {
			result = "Order: " + strings.Join(v.namesInOrder(tie.Teams, s.Teams), ", then ")
		}
	case race.Unbroken:
		step, result = "Every step level", "MFL's franchise order stands: "+strings.Join(names, ", then ")
	}
	return ReportRow{Cells: []string{label, strings.Join(names, " · "), step, strings.Join(found, "\n"), result},
		Mine: slices.Contains(s.Teams, v.in.mine) && v.in.mine != ""}
}

// namesInOrder names the members of group in the order the whole tie finished.
func (v raceView) namesInOrder(final, group []string) []string {
	out := []string{}
	for _, id := range final {
		if slices.Contains(group, id) {
			out = append(out, v.name(id))
		}
	}
	return out
}

func (v raceView) brackets() Report {
	r := reportShell("M-014", "Playoff brackets",
		"Every bracket the league defines on MFL, game by game, with MFL's seeds and scores. A side not yet "+
			"known shows the seed or the earlier game it comes from. The winner is MFL's result for that week's game.",
		v.in.held.brackets, v.in.zone, "Bracket", "Round", "Home", "Away", "Result")
	if r.Status == "empty" {
		r.Note = "The league defines no playoff brackets on MFL."
	}
	brackets := slices.Clone(v.in.held.brackets.value)
	slices.SortStableFunc(brackets, func(a, b leaguefeed.Bracket) int { return cmp.Compare(a.StartWeek, b.StartWeek) })
	names := map[int]string{}
	for _, b := range brackets {
		names[b.ID] = b.Name
	}
	for _, b := range brackets {
		r.Summary = append(r.Summary, ReportRow{Cells: []string{b.Name, v.bracketSeeds(b)}})
		for _, g := range b.Games {
			row := v.row("", b.Name, "Week "+strconv.Itoa(g.Week), v.bracketSide(g.Home, names), v.bracketSide(g.Away, names),
				v.bracketResult(g))
			row.Mine = v.in.mine != "" && (g.Home.Franchise == v.in.mine || g.Away.Franchise == v.in.mine)
			r.Rows = append(r.Rows, row)
		}
		if champ := v.bracketWinner(b); champ != "" {
			r.Summary = append(r.Summary, ReportRow{Cells: []string{b.Title, v.name(champ)}})
		}
	}
	return r
}

func (v raceView) bracketSeeds(b leaguefeed.Bracket) string {
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
			out = append(out, strconv.Itoa(n)+" "+v.name(id))
		}
	}
	return "Seeds: " + strings.Join(out, " · ")
}

func (v raceView) bracketSide(s leaguefeed.BracketSide, names map[int]string) string {
	from := ""
	switch {
	case s.FromBracket != 0:
		from = names[s.FromBracket] + " winner"
	case s.WinnerOf != 0:
		from = "Winner of game " + strconv.Itoa(s.WinnerOf)
	}
	switch {
	case s.Franchise != "" && s.Seed > 0:
		return "Seed " + strconv.Itoa(s.Seed) + " · " + v.name(s.Franchise)
	case s.Franchise != "":
		return v.name(s.Franchise)
	case s.Seed > 0:
		return "Seed " + strconv.Itoa(s.Seed)
	}
	return from
}

// bracketWinnerOf is MFL's winner of a bracket game: that week's result between the two teams,
// or the higher score when the week's results do not list the game.
func (v raceView) bracketWinnerOf(g leaguefeed.BracketGame) string {
	if !g.Home.Scored || !g.Away.Scored {
		return ""
	}
	for _, w := range v.weeks {
		if game, ok := findGame(w, g.Home.Franchise); ok && w.Week == g.Week && game.Home.Scored {
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

func (v raceView) bracketResult(g leaguefeed.BracketGame) string {
	if !g.Home.Scored || !g.Away.Scored {
		return "Not played yet"
	}
	win, lose := g.Home, g.Away
	if v.bracketWinnerOf(g) == g.Away.Franchise {
		win, lose = g.Away, g.Home
	}
	if v.bracketWinnerOf(g) == "" {
		return "Level " + scoreText(win.Points) + " to " + scoreText(lose.Points)
	}
	return v.name(win.Franchise) + " won " + scoreText(win.Points) + " to " + scoreText(lose.Points)
}

// bracketWinner is the winner of the bracket's last game, once it is played.
func (v raceView) bracketWinner(b leaguefeed.Bracket) string {
	if len(b.Games) == 0 {
		return ""
	}
	last := b.Games[0]
	for _, g := range b.Games {
		if g.Week > last.Week || (g.Week == last.Week && g.ID > last.ID) {
			last = g
		}
	}
	return v.bracketWinnerOf(last)
}

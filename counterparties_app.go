package main

import (
	"fmt"
	"math"
	"slices"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/lab"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
)

type RawCounterpartiesScope struct {
	From     int      `json:"from"`
	To       int      `json:"to"`
	Tiers    []string `json:"tiers"`
	Windows  []string `json:"windows"`
	Postures []string `json:"postures"`
	Team     string   `json:"team"`
}

func (r RawCounterpartiesScope) Validate() error {
	if r.From < 1900 || r.To > 2200 || r.From > r.To {
		return fmt.Errorf("counterparties scope: invalid season range")
	}
	sets := []struct {
		label           string
		values, allowed []string
	}{
		{"tier", r.Tiers, lab.TeamTiers()},
		{"window", r.Windows, []string{"spring", "summer", "season"}},
		{"posture", r.Postures, lab.TeamPostures()},
	}
	for _, set := range sets {
		seen := make(map[string]bool)
		for _, v := range set.values {
			if !slices.Contains(set.allowed, v) || seen[v] {
				return fmt.Errorf("counterparties scope: invalid or duplicate %s %q", set.label, v)
			}
			seen[v] = true
		}
	}
	return nil
}

// counterScope keeps an empty selection meaning "no filter", like the Lab's null.
func counterScope(raw RawCounterpartiesScope) lab.TeamScope {
	s := lab.TeamScope{From: max(2017, raw.From), To: raw.To, Team: raw.Team}
	if len(raw.Tiers) > 0 {
		s.Tiers = slices.Clone(raw.Tiers)
	}
	if len(raw.Windows) > 0 {
		s.Windows = slices.Clone(raw.Windows)
	}
	if len(raw.Postures) > 0 {
		s.Postures = slices.Clone(raw.Postures)
	}
	return s
}

type TierPick struct {
	With      string  `json:"with"`
	TradingAs string  `json:"tradingAs"`
	ID        string  `json:"id"`
	Label     string  `json:"label"`
	Hint      string  `json:"hint"`
	Count     float64 `json:"count"`
}
type TeamOption struct {
	Fid  string `json:"fid"`
	Name string `json:"name"`
}
type PostureCell struct {
	ID    string   `json:"id"`
	Label string   `json:"label"`
	P     *float64 `json:"p"`
	Tone  string   `json:"tone"`
	Title string   `json:"title"`
}
type CardRow struct {
	Label string        `json:"label"`
	Text  string        `json:"text"`
	Tone  string        `json:"tone"`
	Range *lab.TeamEdge `json:"range,omitempty"`
}
type TierCard struct {
	ID           string        `json:"id"`
	Label        string        `json:"label"`
	Hint         string        `json:"hint"`
	Excluded     bool          `json:"excluded"`
	Thin         bool          `json:"thin"`
	PerSeason    float64       `json:"perSeason"`
	PerSeasonTxt string        `json:"perSeasonText"`
	Posture      []PostureCell `json:"posture"`
	Rows         []CardRow     `json:"rows"`
}
type PairView struct {
	Other  string       `json:"other"`
	N      int          `json:"n"`
	Now    lab.TeamEdge `json:"now"`
	Real   lab.TeamEdge `json:"real"`
	Trades string       `json:"trades"`
	Title  string       `json:"title"`
}
type CounterShare struct {
	lab.TeamShare
	CountText string `json:"countText"`
}

func counterShare(s lab.TeamShare) CounterShare {
	return CounterShare{TeamShare: s, CountText: fmt.Sprintf("%s of %s", countNumber(s.K), countNumber(s.N))}
}

type CounterRow struct {
	DayText       string             `json:"dayText"`
	SinceText     string             `json:"sinceText"`
	NText         string             `json:"nText"`
	PartnersText  string             `json:"partnersText"`
	Fid           string             `json:"fid"`
	Name          string             `json:"name"`
	N             int                `json:"n"`
	PerSeason     float64            `json:"perSeason"`
	Tiers         map[string]float64 `json:"tiers"`
	ForNow        CounterShare       `json:"forNow"`
	ForLater      CounterShare       `json:"forLater"`
	Day           lab.TeamEdge       `json:"day"`
	Since         lab.TeamEdge       `json:"since"`
	Picks         lab.TeamEdge       `json:"picks"`
	Age           lab.TeamEdge       `json:"age"`
	Partner       *lab.TeamPartner   `json:"partner"`
	PartnerName   string             `json:"partnerName"`
	PartnersCount int                `json:"partnersCount"`
	StandingTitle string             `json:"standingTitle"`
	PartnerTrades string             `json:"partnerTrades"`
}

// CounterTrendRow lists the season row's fields rather than embedding lab.TierTrendRow: the
// embedded Now/Later would be shadowed here, and Wails would emit both into models.ts.
type CounterTrendRow struct {
	Season     int          `json:"season"`
	N          int          `json:"n"`
	Net        *float64     `json:"net"`
	Now        CounterShare `json:"now"`
	Later      CounterShare `json:"later"`
	ShowShares bool         `json:"showShares"`
	NText      string       `json:"nText"`
}
type CounterShift lab.TierShift
type CounterShifts struct {
	Now   *CounterShift `json:"now"`
	Later *CounterShift `json:"later"`
}
type CounterTrend struct {
	Tier  string            `json:"tier"`
	Rows  []CounterTrendRow `json:"rows"`
	Shift CounterShifts     `json:"shift"`
}
type CounterTrends struct {
	Seasons []int          `json:"seasons"`
	Tiers   []CounterTrend `json:"tiers"`
}

func counterTrends(trends lab.TeamTrends) CounterTrends {
	out := CounterTrends{Seasons: trends.Seasons, Tiers: []CounterTrend{}}
	for _, tier := range trends.Tiers {
		view := CounterTrend{Tier: tier.Tier, Rows: []CounterTrendRow{}, Shift: CounterShifts{Now: (*CounterShift)(tier.Shift["now"]), Later: (*CounterShift)(tier.Shift["later"])}}
		for _, row := range tier.Rows {
			view.Rows = append(view.Rows, CounterTrendRow{Season: row.Season, N: row.N, Net: row.Net, Now: counterShare(row.Now), Later: counterShare(row.Later), ShowShares: row.N >= 10, NText: countNumber(row.N)})
		}
		out.Tiers = append(out.Tiers, view)
	}
	return out
}

type CounterWords struct {
	Team      string `json:"team"`
	Sides     string `json:"sides"`
	PerSeason string `json:"perSeason"`
	Now       string `json:"now"`
	Later     string `json:"later"`
	Picks     string `json:"picks"`
	Age       string `json:"age"`
	Partners  string `json:"partners"`
	Partner   string `json:"partner"`
	Day       string `json:"day"`
	Since     string `json:"since"`
	Legend    string `json:"legend"`
	Up        string `json:"up"`
	Down      string `json:"down"`
}

func counterWords() CounterWords {
	return CounterWords{Team: "Team", Sides: "Sides", PerSeason: "Per season", Now: "For now", Later: "For later", Picks: "Picks in − out", Age: "Age in − out", Partners: "Partners", Partner: "Most frequent partner", Day: "On the day", Since: "Since", Legend: "Shares: ↑ (green) is above the league’s share beyond chance; ↓ (red) is below it beyond chance. Unmarked shares are within chance. Red ranges are losses beyond chance; green ranges are gains beyond chance.", Up: "↑", Down: "↓"}
}

type Counterparties struct {
	Words     CounterWords  `json:"words"`
	CountText string        `json:"countText"`
	Summary   string        `json:"summary"`
	AsOf      string        `json:"asOf"`
	Headline  string        `json:"headline"`
	Note      string        `json:"note"`
	Picks     []TierPick    `json:"picks"`
	Teams     []TeamOption  `json:"teams"`
	Cards     []TierCard    `json:"cards"`
	TrendSay  []string      `json:"trendSay"`
	Trends    CounterTrends `json:"trends"`
	Pairs     []PairView    `json:"pairs"`
	Rows      []CounterRow  `json:"rows"`
}

func (a *App) TargetCounterparties(raw RawCounterpartiesScope) Reading[Counterparties] {
	if err := raw.Validate(); err != nil {
		return failedCounterparties(err)
	}
	d, err := a.lab()
	if err != nil {
		return failedCounterparties(err)
	}
	if raw.Team != "" {
		if _, ok := d.Franchises[raw.Team]; !ok {
			return failedCounterparties(fmt.Errorf("counterparties scope: unknown team %q", raw.Team))
		}
	}
	return counterpartiesReading(d, counterScope(raw))
}
func failedCounterparties(err error) Reading[Counterparties] {
	reason := err.Error()
	return Reading[Counterparties]{Status: ReadingFailed, AbsenceReason: &reason, Threshold: 40, Method: "both sides of the valued trades, cut by standing, window and posture", InputSHA: lab.DataSHA}
}

func counterpartiesReading(d *lab.Data, s lab.TeamScope) Reading[Counterparties] {
	league := lab.SidesInScope(d, lab.TeamScope{From: s.From, To: s.To})
	tierless := lab.SidesInScope(d, lab.TeamScope{From: s.From, To: s.To, Windows: s.Windows, Postures: s.Postures, Team: s.Team})
	sides := lab.SidesInScope(d, s)
	seasons := max(1, len(lab.SideYears(league)))
	value := Counterparties{
		Words: counterWords(), CountText: countNumber(len(sides)),
		Summary: fmt.Sprintf("%s of valued trades, %d–%d. On the day is expected wins gained at the league's own prices; since is wins actually delivered, for trades with two or more seasons played.", countLabel(len(sides), "trade side"), s.From, s.To),
		AsOf:    "Standings as of the archive cutoff " + strings.Split(d.ArchiveThrough, "T")[0] + ", not today's live standings.",
		Note:    "Each trade has two sides, one per team. A team's tier is its rank that day: live once three games are final, else last season's finish, as the archive recorded it.",
		Picks:   tierPicks(tierless, seasons),
		Teams:   teamOptions(d),
	}
	r := Reading[Counterparties]{Sourced: snapshot.Sourced[Counterparties]{Value: value, Provenance: counterProvenance(d)}, Status: ReadingOK, Count: len(sides), Threshold: 40, Baseline: "The league's own posture and window shares in the same seasons; a share is coloured only where it clears chance", Cutoff: d.ArchiveThrough, Method: "both sides of every valued trade, cut by standing, window and posture; 1.96 standard-error ranges", InputSHA: lab.DataSHA, BuildStamp: buildLabel() + " " + buildDate, HowCounted: counterpartHowCounted(d)}
	if len(sides) < 40 {
		reason := "Fewer than 40 trade sides in view; widen the standing, time of year or what the trades did"
		r.Status, r.AbsenceReason = ReadingThin, &reason
		applyCounterFilters(&value, s, sides, league)
		r.Value = value
		return r
	}
	value.Headline = headline(sides, league, len(s.Postures) == 0)
	value.Cards = tierCards(sides, league)
	say, trends := trendSay(lab.TierTrends(sides))
	value.TrendSay, value.Trends = say, counterTrends(trends)
	value.Pairs = pairViews(sides)
	value.Rows = counterRows(d, sides, league)
	applyCounterFilters(&value, s, sides, league)
	r.Value = value
	return r
}
func counterProvenance(d *lab.Data) snapshot.Provenance {
	return snapshot.Provenance{Source: "Lab compiled facts sha256:" + lab.DataSHA, Kind: "archive", Freshness: domain.Freshness{State: domain.FreshStale, FetchedAt: d.Built, Note: "Frozen historical input; not current league truth"}}
}
func counterpartHowCounted(d *lab.Data) string {
	return "A team's tier is its rank on the day of the trade: its live standing once three games were final, otherwise the previous season's finish, exactly as the frozen archive recorded it. Contenders are ranked 1 to 10, teams in the hunt 11 to 22, rebuilding teams 23 to 32; the standings run through the archive cutoff " + strings.Split(d.ArchiveThrough, "T")[0] + ", never today's live table. " +
		"What a trade did is read from each side's flow: bought for now when it took in more player value than it sent and sent more pick value than it took, built for later the reverse, like for like otherwise. Value is forecast wins on the day; a flow under 0.05 forecast wins counts as even. " +
		"On the day is the expected wins a side gained at the league's own prices; since is the wins its assets actually delivered against the other side's, for trades with two or more played seasons. Every range is the 95% likely range of the average per trade; a range touching zero is within chance, not a finding."
}
func tierPicks(tierless []lab.TeamSide, seasons int) []TierPick {
	out := []TierPick{}
	for _, t := range lab.TeamTiers() {
		n := 0
		for _, s := range tierless {
			if s.Tier == t {
				n++
			}
		}
		out = append(out, TierPick{ID: t, Label: tierLabel(t), With: "With " + tierTeams(t), TradingAs: tierLabel(t), Hint: tierHint(t), Count: float64(n) / float64(seasons)})
	}
	return out
}
func teamOptions(d *lab.Data) []TeamOption {
	out := []TeamOption{}
	for fid, f := range d.Franchises {
		out = append(out, TeamOption{Fid: fid, Name: f.Name})
	}
	slices.SortFunc(out, func(a, b TeamOption) int { return strings.Compare(a.Name, b.Name) })
	return out
}
func tierLabel(tier string) string {
	return map[string]string{"contender": "Contenders", "hunt": "In the hunt", "rebuild": "Rebuilding"}[tier]
}
func tierTeams(tier string) string {
	return map[string]string{"contender": "contenders", "hunt": "teams in the hunt", "rebuild": "rebuilding teams"}[tier]
}
func tierHint(tier string) string {
	return map[string]string{"contender": "Ranked 1 to 10 on the day of the trade", "hunt": "Ranked 11 to 22 on the day of the trade", "rebuild": "Ranked 23 to 32 on the day of the trade"}[tier]
}
func tierOne(tier string) string {
	return tierLabel(tier)
}
func postureLabel(id string) string {
	return map[string]string{"now": "Bought for now", "later": "Built for later", "swap": "Like for like"}[id]
}
func wins(v float64, dp int) string {
	if math.Round(v*math.Pow10(dp)) == 0 {
		return fmt.Sprintf("%.*f", dp, 0.0)
	}
	return fmt.Sprintf("%+.*f", dp, v)
}
func edgeWords(e lab.TeamEdge, unit string) string {
	if e.Mean == nil || e.Lo == nil || e.Hi == nil {
		return "—"
	}
	if e.Tone == "even" {
		return fmt.Sprintf("%s %s, within chance of zero", wins(*e.Mean, 2), unit)
	}
	return fmt.Sprintf("%s %s, likely %s to %s", wins(*e.Mean, 2), unit, wins(*e.Lo, 2), wins(*e.Hi, 2))
}
func shareWords(s lab.TeamShare) string {
	switch s.Tone {
	case "up":
		return "above the league's share beyond chance"
	case "down":
		return "below the league's share beyond chance"
	}
	return "within chance of the league's share"
}
func postureCell(id string, s lab.TeamShare) PostureCell {
	cell := PostureCell{ID: id, Label: postureLabel(id), P: s.P, Tone: s.Tone}
	if s.P != nil {
		cell.Title = fmt.Sprintf("%s in %s of these trades, %s", cell.Label, percent(*s.P), shareWords(s))
	}
	return cell
}
func sidesOf(sides []lab.TeamSide, tier string) []lab.TeamSide {
	return lab.FilterSides(sides, func(s lab.TeamSide) bool { return s.Tier == tier })
}
func tierCards(sides, league []lab.TeamSide) []TierCard {
	out := []TierCard{}
	for _, id := range lab.TeamTiers() {
		mine := sidesOf(sides, id)
		card := TierCard{ID: id, Label: tierLabel(id), Hint: tierHint(id), Posture: []PostureCell{}}
		if len(mine) < 20 {
			card.Thin = true
			out = append(out, card)
			continue
		}
		b := lab.Behaviour(mine, league)
		card.PerSeason = b.PerSeason
		card.PerSeasonTxt = fmt.Sprintf("%.1f trade sides a season", b.PerSeason)
		for _, p := range lab.TeamPostures() {
			share := b.Posture[p]
			card.Posture = append(card.Posture, postureCell(p, share))
		}
		age := "—"
		if b.Age.N > 0 {
			age = fmt.Sprintf("%s years", wins(*b.Age.Mean, 1))
		}
		card.Rows = []CardRow{
			{Label: "On the day", Text: edgeWords(b.Now, "expected wins a trade"), Tone: b.Now.Tone, Range: &b.Now},
			{Label: "Since (2+ seasons played)", Text: edgeWords(b.Real, "wins a trade"), Tone: b.Real.Tone, Range: &b.Real},
			{Label: "Picks in minus out", Text: fmt.Sprintf("%s a trade", wins(*b.Picks.Mean, 2)), Tone: b.Picks.Tone, Range: &b.Picks},
			{Label: "Age in minus out", Text: age, Tone: b.Age.Tone, Range: &b.Age},
			{Label: "Trades with", Text: tradesWith(b)},
		}
		out = append(out, card)
	}
	return out
}
func tradesWith(b lab.TeamBehaviour) string {
	parts := []string{}
	for _, t := range lab.TeamTiers() {
		parts = append(parts, fmt.Sprintf("%s %s", strings.ToLower(tierLabel(t)), percent(*b.Partners[t].P)))
	}
	return strings.Join(parts, " · ")
}
func headline(sides, league []lab.TeamSide, showPosture bool) string {
	by := map[string]lab.TeamBehaviour{}
	for _, t := range lab.TeamTiers() {
		by[t] = lab.Behaviour(sidesOf(sides, t), league)
	}
	bits := []string{}
	c, r := by["contender"], by["rebuild"]
	if showPosture && c.N >= 20 && r.N >= 20 {
		bits = append(bits, fmt.Sprintf("Contenders buy for now in %s of their trades and rebuilding teams build for later in %s; most trades on both sides are like for like.", percent(*c.Posture["now"].P), percent(*r.Posture["later"].P)))
	}
	winners := []string{}
	for _, t := range lab.TeamTiers() {
		b := by[t]
		if b.N >= 20 && b.Now.Tone != "even" {
			word := "come out behind"
			if b.Now.Tone == "up" {
				word = "come out ahead"
			}
			winners = append(winners, fmt.Sprintf("%s %s on the day (%s a trade).", tierLabel(t), word, wins(*b.Now.Mean, 2)))
		}
	}
	if len(winners) == 0 {
		bits = append(bits, "No tier wins its trades on the day beyond chance: the edge is in the teams, not the standings.")
	} else {
		bits = append(bits, strings.Join(winners, " "))
	}
	return strings.Join(bits, " ")
}
func trendSay(trends lab.TeamTrends) ([]string, lab.TeamTrends) {
	bits := []string{}
	for _, t := range trends.Tiers {
		for _, id := range []string{"now", "later"} {
			s := t.Shift[id]
			if s == nil || !s.Clear {
				continue
			}
			bits = append(bits, fmt.Sprintf("%s %s in %s of their trades in %d–%02d and %s in %d–%02d.", tierLabel(t.Tier), strings.ToLower(postureLabel(id)), percent(s.Early), s.From[0], s.From[1]%100, percent(s.Late), s.To[0], s.To[1]%100))
		}
	}
	if len(bits) == 0 {
		bits = append(bits, "No tier's habit moved beyond chance between the earlier and later halves of these seasons.")
	} else {
		bits = append(bits, "Every other habit held within chance between the two halves.")
	}
	return bits, trends
}
func pairViews(sides []lab.TeamSide) []PairView {
	out := []PairView{}
	for _, row := range lab.TierPairs(sides) {
		for _, c := range row.Cells {
			out = append(out, PairView{Other: c.Other, N: c.N, Now: c.Now, Real: c.Real, Trades: countLabel(c.N, "trade"), Title: fmt.Sprintf("%s trading with %s: %s on the day; %s since.", tierLabel(row.Tier), tierTeams(c.Other), edgeWords(c.Now, "expected wins a trade"), edgeWords(c.Real, "wins a trade"))})
		}
	}
	return out
}
func counterRows(d *lab.Data, sides, league []lab.TeamSide) []CounterRow {
	out := []CounterRow{}
	for _, row := range lab.TeamRows(sides, league) {
		r := CounterRow{NText: countNumber(row.N), PartnersText: countNumber(row.PartnersCount), Fid: row.Fid, Name: franchiseName(d, row.Fid), N: row.N, PerSeason: row.PerSeason, Tiers: row.Tiers, ForNow: counterShare(row.Posture["now"]), ForLater: counterShare(row.Posture["later"]), Day: row.Now, Since: row.Real, Picks: row.Picks, Age: row.Age, Partner: row.Partner, PartnersCount: row.PartnersCount}
		parts := []string{}
		for _, t := range lab.TeamTiers() {
			parts = append(parts, fmt.Sprintf("%s %s", tierLabel(t), percent(row.Tiers[t])))
		}
		r.StandingTitle = strings.Join(parts, ", ")
		r.DayText, r.SinceText = counterTotalWords(row.Now), counterTotalWords(row.Real)
		if row.Partner != nil {
			r.PartnerName = franchiseName(d, row.Partner.Fid)
			r.PartnerTrades = countLabel(row.Partner.N, "trade")
		}
		out = append(out, r)
	}
	return out
}
func counterTotalWords(e lab.TeamEdge) string {
	if e.Total == nil || e.Mean == nil {
		return "—"
	}
	line := fmt.Sprintf("%s total · %s a trade", wins(*e.Total, 2), wins(*e.Mean, 2))
	if e.Lo == nil || e.Hi == nil {
		return line + " (too few for a range)"
	}
	return fmt.Sprintf("%s (%s to %s)", line, wins(*e.Lo, 2), wins(*e.Hi, 2))
}
func franchiseName(d *lab.Data, fid string) string {
	if f, ok := d.Franchises[fid]; ok {
		return f.Name
	}
	return fid
}

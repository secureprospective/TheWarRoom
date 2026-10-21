// Package market is the Market feature's service: the MFL player market reports (projections,
// ranks, ADP and AAV) and the pool's market cells built from them, the Lab's market surfaces
// (buy and sell lists, the pick clock, trade stretches, an asset's profile) read from the frozen
// Lab facts, and the asset board's worded values.
//
// It is stateless. The caller holds the feeds, the Lab decode and the value model, and passes
// them in; nothing here locks, fetches, opens a database or starts a goroutine. The values it
// returns are the service's own types, twins of the wire types in package main, which the root
// adapters convert (market_feeds_app.go, market_surfaces_app.go, asset_board.go).
package market

import (
	"cmp"
	"fmt"
	"math"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/report"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/marketfeeds"
	"github.com/secureprospective/TheWarRoom/internal/lab"
	"github.com/secureprospective/TheWarRoom/internal/normalize"
)

// HeldFeed is one market report kind as the caller holds it: the parsed feed (one, or none before
// the first read) and the feed's read state.
type HeldFeed struct {
	Value []marketfeeds.Feed
	Feed  report.Feed
}

// Fact is one market cell of a player's row; it is main.MarketFact. Value is the number behind
// the text (the projection, the rank, the average pick or dollars), nil when there is none; it is
// not on the wire.
type Fact struct {
	Value  *float64      `json:"-"`
	Label  string        `json:"label"`
	Text   string        `json:"text"`
	Reason string        `json:"reason"`
	Range  *report.Range `json:"range"`
}

// Kinds are the four MFL market reports, in the order the board shows them.
func Kinds() []string {
	return []string{marketfeeds.Projections, marketfeeds.Ranks, marketfeeds.ADP, marketfeeds.AAV}
}

// ArchiveScope says whether an archived export at src answers this kind for this week: the
// projections of that NFL week, ADP over all draft periods, any copy of the others.
func ArchiveScope(src, kind string, week int) bool {
	u, err := url.Parse(src)
	if err != nil {
		return false
	}
	switch kind {
	case marketfeeds.Projections:
		return u.Query().Get("W") == strconv.Itoa(week)
	case marketfeeds.ADP:
		return u.Query().Get("PERIOD") == "ALL"
	default:
		return true
	}
}

// Names labels every player the held reports list: name, position and team for a player,
// and the name alone for a team unit (a defense, a kicking unit) that MFL-wide reports price.
func Names(held map[string]HeldFeed, lookup normalize.Lookup) map[string]string {
	names := map[string]string{}
	for _, feed := range held {
		if len(feed.Value) != 1 {
			continue
		}
		for _, p := range feed.Value[0].Players {
			id := p.ID.String()
			if f, ok := lookup.Facts(id); ok {
				names[id] = f.Name + " · " + string(f.Position) + " · " + f.Team
			} else if name, ok := lookup.Name(id); ok {
				names[id] = name + " · team unit"
			}
		}
	}
	return names
}

type reportSpec struct{ kind, id, title, label, how string }

func reportSpecs() []reportSpec {
	return []reportSpec{
		{marketfeeds.Projections, "M-046", "Projected points this week", "Projected points", "MFL's projected fantasy points for this NFL week under this league's scoring, not the app's model. Missing players are not projected as zero."},
		{marketfeeds.Ranks, "M-047", "MFL player ranks", "MFL rank · weekly change", "MFL's player rank, not the app's model. Up means last week's rank minus this week's rank is positive; down means negative. A last-week rank of zero means MFL supplied no previous rank. New this week is MFL's new-entry flag; no earlier rank is shown."},
		{marketfeeds.ADP, "M-067", "Average draft position", "ADP · min–max pick", "MFL-wide across many leagues, not this league. All draft periods are requested, since the default period can have no picks in season. The bar runs from the lowest to highest pick reported, with the average marked; it is not a confidence interval or a grade. Every bar shares one scale, from zero to the latest pick any player went."},
		{marketfeeds.AAV, "M-069", "Average auction value", "AAV · min–max dollars on MFL’s $1,000 scale", "MFL-wide across many leagues, not this league's dollars. Every league's auction is scaled to a $1,000 pool across all its teams. No conversion to this league's money is made. The bar runs from the smallest to largest dollar value reported, with the average marked; it is not a confidence interval or a grade. Every bar shares one scale, from zero to the largest dollar value any player drew."},
	}
}

func rankChange(p marketfeeds.Player) string {
	switch {
	case p.New:
		return "New this week"
	case p.Last == 0:
		return "Previous rank not supplied"
	case p.Change > 0:
		return fmt.Sprintf("Up %d", p.Change)
	case p.Change < 0:
		return fmt.Sprintf("Down %d", -p.Change)
	default:
		return "No change"
	}
}
func fact(spec reportSpec, feed HeldFeed, p *marketfeeds.Player, zone *time.Location) Fact {
	return factOn(spec, feed, p, scale(feed), zone)
}

// scale is one axis for a whole report, so every player's bar reads against the same
// ends: the largest pick or auction value any player reached.
func scale(feed HeldFeed) float64 {
	scale := 1.0
	if len(feed.Value) == 1 {
		for _, p := range feed.Value[0].Players {
			scale = math.Max(scale, math.Max(p.High, p.Value))
		}
	}
	return scale
}

func factOn(spec reportSpec, feed HeldFeed, p *marketfeeds.Player, scale float64, zone *time.Location) Fact {
	fact := Fact{Label: spec.label, Text: "—", Reason: "MFL does not list this player in this report."}
	if feed.Feed.FetchedAt.IsZero() {
		fact.Reason = spec.title + " unavailable; refresh from MFL to check."
		return fact
	}
	fact.Reason = "Source: MFL · As of: " + feed.Feed.FetchedAt.In(zone).Format("Jan 2, 2006, 3:04 PM")
	if spec.kind == marketfeeds.ADP || spec.kind == marketfeeds.AAV {
		fact.Reason += " · MFL-wide, not this league"
	}
	if feed.Feed.RefreshError != "" {
		fact.Reason += " · Refresh failed; last known data"
	}
	if p == nil {
		fact.Reason += " · MFL does not list this player."
		return fact
	}
	value := p.Value
	if spec.kind == marketfeeds.Ranks {
		value = float64(p.Rank)
	}
	fact.Value = &value
	switch spec.kind {
	case marketfeeds.Projections:
		fact.Text = fmt.Sprintf("%.2f", p.Value)
		fact.Reason += fmt.Sprintf(" · NFL week %d · league scoring", feed.Value[0].Week)
	case marketfeeds.Ranks:
		fact.Text = fmt.Sprintf("%d · %s", p.Rank, rankChange(*p))
	default:
		// Picks are whole numbers at the ends and one decimal on average; auction values one decimal.
		fact.Text = fmt.Sprintf("Pick %.1f · picks %.0f–%.0f", p.Value, p.Low, p.High)
		if spec.kind == marketfeeds.AAV {
			fact.Text = fmt.Sprintf("$%.1f of a $1,000 league pool · $%.1f–$%.1f", p.Value, p.Low, p.High)
		}
		fact.Range = &report.Range{Interval: lab.Interval{Value: p.Value, Lo: p.Low, Hi: p.High}, Max: scale, Text: fact.Text}
	}
	return fact
}

// Facts are every listed player's four market cells, by MFL id, in report order; the "" entry
// is the cells of a player no report lists.
func Facts(held map[string]HeldFeed, zone *time.Location) map[string][]Fact {
	indexed := map[string]map[string]marketfeeds.Player{}
	ids := map[string]bool{}
	for kind, feed := range held {
		indexed[kind] = map[string]marketfeeds.Player{}
		if len(feed.Value) != 1 {
			continue
		}
		for _, p := range feed.Value[0].Players {
			indexed[kind][p.ID.String()] = p
			ids[p.ID.String()] = true
		}
	}
	out := map[string][]Fact{}
	scales := map[string]float64{}
	for _, spec := range reportSpecs() {
		scales[spec.kind] = scale(held[spec.kind])
	}
	for id := range ids {
		for _, spec := range reportSpecs() {
			var player *marketfeeds.Player
			if p, ok := indexed[spec.kind][id]; ok {
				player = &p
			}
			out[id] = append(out[id], factOn(spec, held[spec.kind], player, scales[spec.kind], zone))
		}
	}
	out[""] = []Fact{}
	for _, spec := range reportSpecs() {
		out[""] = append(out[""], fact(spec, held[spec.kind], nil, zone))
	}
	return out
}

// Reports is the player market reports board: one report per kind, its rows the held feed's
// players labelled by names, the projections titled with this NFL week.
func Reports(held map[string]HeldFeed, names map[string]string, week int, zone *time.Location) report.Board {
	board := report.Board{Title: "Player market reports", Loading: "Reading player market reports…", Refresh: "Refresh player market reports from MFL", HowLabel: "How this is counted", Reports: []report.Report{}}
	for _, spec := range reportSpecs() {
		feed := held[spec.kind]
		title := spec.title
		if spec.kind == marketfeeds.Projections {
			title = "Projected points · " + WeekText(week)
		}
		r := report.Shell(spec.id, title, spec.how, feed.Feed, len(feed.Value), zone, "Player", spec.label)
		if spec.kind == marketfeeds.Ranks {
			r.Columns = []string{"Player", "MFL rank", "Weekly change", "Last week"}
		}
		r.SortAscending, r.SortDescending = "Sort ascending", "Sort descending"
		if spec.kind == marketfeeds.Projections {
			r.Source = "Source: MFL · this league's scoring · " + WeekText(week)
		}
		if spec.kind == marketfeeds.ADP || spec.kind == marketfeeds.AAV {
			r.Source = "Source: MFL-wide · many leagues, not this league"
		}
		if len(feed.Value) == 1 {
			value := feed.Value[0]
			if value.Samples > 0 {
				r.Note = fmt.Sprintf("%d %s sampled by MFL.", value.Samples, sampleUnit(spec.kind))
			}
			if feed.Feed.RefreshError != "" {
				r.Note += " Refresh failed; showing the last known data."
			}
			if len(value.Players) == 0 {
				r.Status = "empty"
				r.Note += " MFL lists no records for this report."
			}
			if spec.kind == marketfeeds.ADP && len(value.Players) == 0 {
				r.Note = "MFL reports no draft picks for all periods; average draft position is unavailable."
			}
			r.Rows = reportRows(spec, feed, names, zone)
		}
		board.Reports = append(board.Reports, r)
	}
	return board
}
func sampleUnit(kind string) string {
	if kind == marketfeeds.AAV {
		return "auctions"
	}
	return "drafts"
}
func reportRows(spec reportSpec, feed HeldFeed, names map[string]string, zone *time.Location) []report.Row {
	players := slices.Clone(feed.Value[0].Players)
	slices.SortFunc(players, func(a, b marketfeeds.Player) int {
		if spec.kind == marketfeeds.Ranks {
			return cmp.Or(cmp.Compare(a.Rank, b.Rank), strings.Compare(a.ID.String(), b.ID.String()))
		}
		order := cmp.Compare(a.Value, b.Value)
		// Most points and the costliest auction values lead; draft position leads with pick 1.
		if spec.kind == marketfeeds.Projections || spec.kind == marketfeeds.AAV {
			order = -order
		}
		return cmp.Or(order, strings.Compare(a.ID.String(), b.ID.String()))
	})
	alphabetical := slices.Clone(players)
	slices.SortStableFunc(alphabetical, func(a, b marketfeeds.Player) int { return strings.Compare(names[a.ID.String()], names[b.ID.String()]) })
	nameKeys := map[string]float64{}
	for i, p := range alphabetical {
		nameKeys[p.ID.String()] = float64(i)
	}
	axis := scale(feed)
	rows := []report.Row{}
	for _, p := range players {
		name := names[p.ID.String()]
		if name == "" {
			name = "Player name not held"
		}
		fact := factOn(spec, feed, &p, axis, zone)
		key := p.Value
		if spec.kind == marketfeeds.Ranks {
			key = float64(p.Rank)
		}
		row := report.Row{Cells: []string{name, fact.Text}, Range: fact.Range, Sort: []float64{nameKeys[p.ID.String()], key}, Titles: []string{"", fact.Reason}}
		if spec.kind == marketfeeds.Ranks {
			last := fmt.Sprintf("%d", p.Last)
			if p.Last == 0 || p.New {
				last = "—"
			}
			row.Cells = []string{name, fmt.Sprintf("%d", p.Rank), rankChange(p), last}
			row.Sort = []float64{nameKeys[p.ID.String()], float64(p.Rank), float64(p.Change), float64(p.Last)}
			row.Titles = []string{"", fact.Reason, fact.Reason, fact.Reason}
		}
		rows = append(rows, row)
	}
	return rows
}

// WeekText names the NFL week the projections are for, or says it is not known.
func WeekText(week int) string {
	if week < 1 || week > 18 {
		return "NFL week not known"
	}
	return fmt.Sprintf("NFL week %d", week)
}

package main

import (
	"cmp"
	"fmt"
	"math"
	"slices"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/ingestion/marketfeeds"
	"github.com/secureprospective/TheWarRoom/internal/lab"
)

type MarketFact struct {
	Label  string       `json:"label"`
	Text   string       `json:"text"`
	Reason string       `json:"reason"`
	Range  *MarketRange `json:"range"`
}

type marketReportSpec struct{ kind, id, title, label, how string }

func marketReportSpecs() []marketReportSpec {
	return []marketReportSpec{
		{marketfeeds.Projections, "M-046", "Projected points this week", "Projected points", "MFL's projected fantasy points for this NFL week under this league's scoring, not the app's model. Missing players are not projected as zero."},
		{marketfeeds.Ranks, "M-047", "MFL player ranks", "MFL rank · weekly change", "MFL's player rank, not the app's model. Up means last week's rank minus this week's rank is positive; down means negative. A last-week rank of zero means MFL supplied no previous rank. New this week is MFL's new-entry flag; no earlier rank is shown."},
		{marketfeeds.ADP, "M-067", "Average draft position", "ADP · min–max pick", "MFL-wide across many leagues, not this league. All draft periods are requested, since the default period can have no picks in season. The bar runs from the lowest to highest pick reported, with the average marked; it is not a confidence interval or a grade."},
		{marketfeeds.AAV, "M-069", "Average auction value", "AAV · min–max budget share", "MFL-wide across many leagues, not this league. Values are MFL's auction-budget shares, expressed as percent, not this league's dollars. No dollar conversion is made. The bar runs from the smallest to largest share reported, with the average marked; it is not a confidence interval or a grade. Outliers above 100% are shown as supplied, not clipped."},
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
func marketFact(spec marketReportSpec, feed heldSeasonFeed[[]marketfeeds.Feed], p *marketfeeds.Player, zone *time.Location) MarketFact {
	fact := MarketFact{Label: spec.label, Text: "—", Reason: "MFL does not list this player in this report."}
	if feed.fetchedAt.IsZero() {
		fact.Reason = spec.title + " unavailable; refresh from MFL to check."
		return fact
	}
	fact.Reason = "Source: MFL · As of: " + feed.fetchedAt.In(zone).Format("Jan 2, 2006, 3:04 PM")
	if spec.kind == marketfeeds.ADP || spec.kind == marketfeeds.AAV {
		fact.Reason += " · MFL-wide, not this league"
	}
	if feed.refreshError != "" {
		fact.Reason += " · Refresh failed; last known data"
	}
	if p == nil {
		fact.Reason += " · MFL does not list this player."
		return fact
	}
	switch spec.kind {
	case marketfeeds.Projections:
		fact.Text = fmt.Sprintf("%.2f", p.Value)
		fact.Reason += fmt.Sprintf(" · NFL week %d · league scoring", feed.value[0].Week)
	case marketfeeds.Ranks:
		fact.Text = fmt.Sprintf("%d · %s", p.Rank, rankChange(*p))
	default:
		unit := "pick"
		if spec.kind == marketfeeds.AAV {
			unit = "% of auction budget"
		}
		fact.Text = fmt.Sprintf("%.2f %s · %.2f–%.2f", p.Value, unit, p.Low, p.High)
		fact.Range = &MarketRange{Interval: lab.Interval{Value: p.Value, Lo: p.Low, Hi: p.High}, Max: math.Max(1, p.High), Text: fact.Text}
	}
	return fact
}

func marketFacts(held map[string]heldSeasonFeed[[]marketfeeds.Feed]) map[string][]MarketFact {
	return marketFactsAt(held, time.Local)
}
func marketFactsAt(held map[string]heldSeasonFeed[[]marketfeeds.Feed], zone *time.Location) map[string][]MarketFact {
	indexed := map[string]map[string]marketfeeds.Player{}
	ids := map[string]bool{}
	for kind, feed := range held {
		indexed[kind] = map[string]marketfeeds.Player{}
		if len(feed.value) != 1 {
			continue
		}
		for _, p := range feed.value[0].Players {
			indexed[kind][p.ID.String()] = p
			ids[p.ID.String()] = true
		}
	}
	out := map[string][]MarketFact{}
	for id := range ids {
		for _, spec := range marketReportSpecs() {
			var player *marketfeeds.Player
			if p, ok := indexed[spec.kind][id]; ok {
				player = &p
			}
			out[id] = append(out[id], marketFact(spec, held[spec.kind], player, zone))
		}
	}
	out[""] = []MarketFact{}
	for _, spec := range marketReportSpecs() {
		out[""] = append(out[""], marketFact(spec, held[spec.kind], nil, zone))
	}
	return out
}

func buildMarketReports(held map[string]heldSeasonFeed[[]marketfeeds.Feed], names map[string]string, week int, zone *time.Location) ReportBoard {
	board := ReportBoard{Title: "Player market reports", Loading: "Reading player market reports…", Refresh: "Refresh player market reports from MFL", HowLabel: "How this is counted", Reports: []Report{}}
	for _, spec := range marketReportSpecs() {
		feed := held[spec.kind]
		title := spec.title
		if spec.kind == marketfeeds.Projections {
			title = "Projected points · " + marketWeekText(week)
		}
		report := reportShell(spec.id, title, spec.how, feed, zone, "Player", spec.label)
		if spec.kind == marketfeeds.Ranks {
			report.Columns = []string{"Player", "MFL rank", "Weekly change", "Last week"}
		}
		report.SortAscending, report.SortDescending = "Sort ascending", "Sort descending"
		if spec.kind == marketfeeds.Projections {
			report.Source = "Source: MFL · this league's scoring · " + marketWeekText(week)
		}
		if spec.kind == marketfeeds.ADP || spec.kind == marketfeeds.AAV {
			report.Source = "Source: MFL-wide · many leagues, not this league"
		}
		if len(feed.value) == 1 {
			value := feed.value[0]
			if value.Samples > 0 {
				report.Note = fmt.Sprintf("%d %s sampled by MFL.", value.Samples, marketSampleUnit(spec.kind))
			}
			if feed.refreshError != "" {
				report.Note += " Refresh failed; showing the last known data."
			}
			if len(value.Players) == 0 {
				report.Status = "empty"
				report.Note += " MFL lists no records for this report."
			}
			if spec.kind == marketfeeds.ADP && len(value.Players) == 0 {
				report.Note = "MFL reports no draft picks for all periods; average draft position is unavailable."
			}
			report.Rows = marketReportRows(spec, feed, names, zone)
		}
		board.Reports = append(board.Reports, report)
	}
	return board
}
func marketSampleUnit(kind string) string {
	if kind == marketfeeds.AAV {
		return "auctions"
	}
	return "drafts"
}
func marketReportRows(spec marketReportSpec, feed heldSeasonFeed[[]marketfeeds.Feed], names map[string]string, zone *time.Location) []ReportRow {
	players := slices.Clone(feed.value[0].Players)
	slices.SortFunc(players, func(a, b marketfeeds.Player) int {
		if spec.kind == marketfeeds.Ranks {
			return cmp.Or(cmp.Compare(a.Rank, b.Rank), strings.Compare(a.ID.String(), b.ID.String()))
		}
		order := cmp.Compare(a.Value, b.Value)
		if spec.kind == marketfeeds.Projections {
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
	rows := []ReportRow{}
	for _, p := range players {
		name := names[p.ID.String()]
		if name == "" {
			name = "Player name not held"
		}
		fact := marketFact(spec, feed, &p, zone)
		key := p.Value
		if spec.kind == marketfeeds.Ranks {
			key = float64(p.Rank)
		}
		row := ReportRow{Cells: []string{name, fact.Text}, Range: fact.Range, Sort: []float64{nameKeys[p.ID.String()], key}, Titles: []string{"", fact.Reason}}
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

func marketWeekText(week int) string {
	if week < 1 || week > 18 {
		return "NFL week not known"
	}
	return fmt.Sprintf("NFL week %d", week)
}

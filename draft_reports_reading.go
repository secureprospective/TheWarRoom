package main

import (
	"cmp"
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/draftreports"
	"github.com/secureprospective/TheWarRoom/internal/lab"
)

type DraftReportRow struct {
	Cells []string     `json:"cells"`
	Range *MarketRange `json:"range"`
}

type DraftReport struct {
	ID      string           `json:"id"`
	Title   string           `json:"title"`
	Note    string           `json:"note"`
	Status  string           `json:"status"`
	Source  string           `json:"source"`
	AsOf    string           `json:"asOf"`
	How     string           `json:"how"`
	Columns []string         `json:"columns"`
	Rows    []DraftReportRow `json:"rows"`
}

type DraftReportBoard struct {
	Title    string        `json:"title"`
	Loading  string        `json:"loading"`
	Refresh  string        `json:"refresh"`
	HowLabel string        `json:"howLabel"`
	MFLLabel string        `json:"mflLabel"`
	MFLURL   string        `json:"mflUrl"`
	Reports  []DraftReport `json:"reports"`
}

func reportShell[T any](id, title, source, how string, feed heldSeasonFeed[[]T], zone *time.Location, columns ...string) DraftReport {
	r := DraftReport{ID: id, Title: title, Source: "Source: MFL " + source, How: how,
		Columns: columns, Rows: []DraftReportRow{}, Status: "ok", AsOf: "As of: not fetched"}
	switch {
	case feed.fetchedAt.IsZero():
		r.Status, r.Note = "failed", "Export unavailable; refresh from MFL to fill this report."
	case len(feed.value) == 0:
		r.Status, r.Note = "empty", "MFL supplied an empty export; no records to show."
	}
	if !feed.fetchedAt.IsZero() {
		r.AsOf = "As of: " + feed.fetchedAt.In(zone).Format("2006-01-02 15:04:05 MST")
	}
	if feed.refreshError != "" {
		r.Note += " Refresh warning: " + feed.refreshError
		if !feed.fetchedAt.IsZero() {
			r.Status = "stale"
		}
	}
	return r
}

func draftRow(cells ...string) DraftReportRow { return DraftReportRow{Cells: cells} }
func pickLabel(p draftreports.Pick) string    { return fmt.Sprintf("%d.%02d", p.Round, p.Slot) }
func pickPlayer(p draftreports.Pick, names map[string]string) string {
	if p.Skipped {
		return "Pick skipped by MFL"
	}
	if p.Player == nil {
		return "Not made"
	}
	id := p.Player.String()
	if names[id] != "" {
		return names[id] + " · " + id
	}
	return "Player " + id + " (name not held)"
}
func pickTime(p draftreports.Pick, zone *time.Location) string {
	if p.At.IsZero() {
		return "Time not supplied"
	}
	return p.At.In(zone).Format("2006-01-02 15:04:05 MST")
}
func orderedPicks(picks []draftreports.Pick) []draftreports.Pick {
	out := slices.Clone(picks)
	slices.SortFunc(out, func(a, b draftreports.Pick) int {
		return cmp.Or(cmp.Compare(a.Round, b.Round), cmp.Compare(a.Slot, b.Slot))
	})
	return out
}

func buildDraftReports(results heldSeasonFeed[[]draftreports.Pick], future heldSeasonFeed[[]draftreports.FuturePick],
	names, franchises map[string]string, values map[string]float64, valueNote, next string, zone *time.Location) DraftReportBoard {
	picks := orderedPicks(results.value)
	how := "Picks sorted by numeric round, then pick within round, not by franchise or response order. " +
		"Owners are the franchises recorded for each pick; MFL comments retain its trade chain without inferring an original owner. " +
		"MFL timestamp strings are Unix seconds (UTC instants), displayed in this computer's local time zone. " +
		"MFL player 0000 means a skipped pick, not a player or an unmade slot. These are held exports, not a real-time clock; refresh failures retain the last good copy."
	rows := reportShell("M-063", "Draft results · pick by pick", draftreports.ResultsExport, how, results, zone,
		"Pick", "Franchise", "Player", "Local time")
	detail := reportShell("M-102", "Draft pick detail", draftreports.ResultsExport, how, results, zone,
		"Pick", "Franchise", "Player", "Local time", "From whom · MFL comments")
	for _, p := range picks {
		owner := domain.FranchiseLabel(franchises, p.Franchise)
		rows.Rows = append(rows.Rows, draftRow(pickLabel(p), owner, pickPlayer(p, names), pickTime(p, zone)))
		comments := strings.TrimSpace(p.Comments)
		if comments == "" {
			comments = "No trade provenance supplied"
		}
		detail.Rows = append(detail.Rows, draftRow(pickLabel(p), owner, pickPlayer(p, names), pickTime(p, zone), comments))
	}
	grid := draftGrid(picks, names, franchises, reportShell("M-064", "Draft results grid · round × franchise", draftreports.ResultsExport,
		how+" A cell lists every slot drafted by that franchise in that round; traded picks do not shift other cells.", results, zone))
	recap, pace := draftFranchises(picks, names, franchises, results, how, zone)
	status := draftStatus(picks, franchises, results, how, zone)
	live := status
	live.ID, live.Title = "M-183", "Live draft room · held results"
	live.Note += " " + next + " Picks happen on MFL only; this app never makes a pick."
	live.Rows = slices.Clone(rows.Rows)
	live.Columns = slices.Clone(rows.Columns)
	return DraftReportBoard{Title: "MFL draft reports", Loading: "Reading held MFL draft reports…", Refresh: "Refresh draft reports from MFL",
		HowLabel: "How this is counted", MFLLabel: "Open MFL draft results (picks happen on MFL)", Reports: []DraftReport{rows, grid,
			draftValueReport(picks, names, values, results, how, valueNote, zone), recap, draftFuture(future, franchises, zone), detail, status, pace, live}}
}

func draftGrid(picks []draftreports.Pick, names, franchises map[string]string, r DraftReport) DraftReport {
	ids := []string{}
	rounds := []int{}
	cells := map[string][]string{}
	for _, p := range picks {
		if !slices.Contains(ids, p.Franchise) {
			ids = append(ids, p.Franchise)
		}
		if !slices.Contains(rounds, p.Round) {
			rounds = append(rounds, p.Round)
		}
		key := fmt.Sprintf("%d:%s", p.Round, p.Franchise)
		cells[key] = append(cells[key], pickLabel(p)+" · "+pickPlayer(p, names))
	}
	slices.Sort(ids)
	slices.Sort(rounds)
	r.Columns = []string{"Round"}
	for _, id := range ids {
		r.Columns = append(r.Columns, domain.FranchiseLabel(franchises, id))
	}
	for _, round := range rounds {
		row := draftRow(strconv.Itoa(round))
		for _, id := range ids {
			text := strings.Join(cells[fmt.Sprintf("%d:%s", round, id)], "; ")
			if text == "" {
				text = "No recorded pick"
			}
			row.Cells = append(row.Cells, text)
		}
		r.Rows = append(r.Rows, row)
	}
	return r
}

// Elapsed time includes overnight pauses; the first pick has no preceding timestamp.
func draftElapsed(picks []draftreports.Pick) map[string][]time.Duration {
	out := map[string][]time.Duration{}
	for i := 1; i < len(picks); i++ {
		p := picks[i]
		if (p.Player == nil && !p.Skipped) || p.At.IsZero() {
			continue
		}
		prev := picks[i-1]
		if (prev.Player == nil && !prev.Skipped) || prev.At.IsZero() || p.At.Before(prev.At) {
			continue
		}
		out[p.Franchise] = append(out[p.Franchise], p.At.Sub(prev.At))
	}
	return out
}

func draftFranchises(picks []draftreports.Pick, names, franchises map[string]string, feed heldSeasonFeed[[]draftreports.Pick], how string, zone *time.Location) (DraftReport, DraftReport) {
	recap := reportShell("M-066", "Draft recap per franchise", draftreports.ResultsExport, how, feed, zone, "Franchise", "Made picks", "Players")
	pace := reportShell("M-179", "Average elapsed time per pick", draftreports.ResultsExport, how+
		" Average wall-clock seconds from the preceding pick in draft order to this franchise's pick, including overnight pauses and zero-second pre-draft picks. First pick, missing timestamps and negative intervals are excluded; no active timer or pause subtraction is available.", feed, zone,
		"Franchise", "Measured intervals", "Average elapsed time")
	players := map[string][]string{}
	ids := []string{}
	for _, p := range picks {
		if !slices.Contains(ids, p.Franchise) {
			ids = append(ids, p.Franchise)
		}
		if p.Player != nil || p.Skipped {
			players[p.Franchise] = append(players[p.Franchise], pickLabel(p)+" · "+pickPlayer(p, names))
		}
	}
	slices.Sort(ids)
	elapsed := draftElapsed(picks)
	for _, id := range ids {
		owner := domain.FranchiseLabel(franchises, id)
		recap.Rows = append(recap.Rows, draftRow(owner, strconv.Itoa(len(players[id])), strings.Join(players[id], "; ")))
		intervals := elapsed[id]
		text := "No measured interval"
		if len(intervals) > 0 {
			var total time.Duration
			for _, d := range intervals {
				total += d
			}
			text = (total / time.Duration(len(intervals))).Round(time.Second).String()
		}
		pace.Rows = append(pace.Rows, draftRow(owner, strconv.Itoa(len(intervals)), text))
	}
	return recap, pace
}

func draftStatus(picks []draftreports.Pick, franchises map[string]string, feed heldSeasonFeed[[]draftreports.Pick], how string, zone *time.Location) DraftReport {
	r := reportShell("M-176", "Draft status", draftreports.ResultsExport, how+
		" Made counts exported player selections plus explicit skipped picks. Remaining counts only explicit unmade rows; the export does not declare total rounds or a live on-the-clock state. No missing rows are invented.", feed, zone, "State", "Held evidence")
	made, unmade := 0, 0
	for _, p := range picks {
		if p.Player != nil || p.Skipped {
			made++
		} else {
			unmade++
		}
	}
	r.Rows = append(r.Rows, draftRow("Made", strconv.Itoa(made)))
	remaining := "Unknown · total scheduled picks not supplied"
	clock := "Not supplied by draftResults"
	if unmade > 0 {
		remaining = fmt.Sprintf("%d explicit unmade picks (scheduled total unknown)", unmade)
		for _, p := range picks {
			if p.Player == nil && !p.Skipped {
				clock = "Next recorded unmade pick: " + pickLabel(p) + " · " + domain.FranchiseLabel(franchises, p.Franchise) + " (live clock unverified)"
				break
			}
		}
	}
	r.Rows = append(r.Rows, draftRow("On the clock", clock), draftRow("Remaining", remaining))
	if made > 0 && unmade == 0 {
		r.Note += " Held completed-pick history; no unmade picks are listed. Live completion is not declared by this export."
	}
	return r
}

func draftFuture(feed heldSeasonFeed[[]draftreports.FuturePick], franchises map[string]string, zone *time.Location) DraftReport {
	r := reportShell("M-068", "Future draft picks · ownership", draftreports.FutureExport,
		"Every futureDraftPick under a franchise belongs to that franchise; originalPickFor identifies whose original pick it was. Sorted by year, numeric round and original franchise. Missing assets are not inferred from the league size.", feed, zone,
		"Year", "Round", "Original franchise", "Current owner")
	picks := slices.Clone(feed.value)
	slices.SortFunc(picks, func(a, b draftreports.FuturePick) int {
		return cmp.Or(cmp.Compare(a.Year, b.Year), cmp.Compare(a.Round, b.Round), strings.Compare(a.Original, b.Original))
	})
	for _, p := range picks {
		r.Rows = append(r.Rows, draftRow(strconv.Itoa(p.Year), strconv.Itoa(p.Round), domain.FranchiseLabel(franchises, p.Original), domain.FranchiseLabel(franchises, p.Owner)))
	}
	return r
}

func draftValueReport(picks []draftreports.Pick, names map[string]string, values map[string]float64, feed heldSeasonFeed[[]draftreports.Pick], how, valueNote string, zone *time.Location) DraftReport {
	r := reportShell("M-065", "Steals and busts · distance from expected", draftreports.ResultsExport,
		how+" Value is the app's latest held DynastyPPG: discounted expected league points per game across its configured dynasty horizon. Expected is the median held value of drafted players in the same round, not an ADP or draft-day valuation. A bar is the point distance from that baseline (no uncertainty estimate and no letter grade); missing values are not zero. This retrospective comparison is not proof a draft pick was good or bad.",
		feed, zone, "Pick", "Player", "Current dynasty value", "Expected · same round", "Distance from expected")
	r.Source += " + held app model (DynastyPPG)"
	r.Note += " " + valueNote
	byRound := map[int][]float64{}
	for _, p := range picks {
		if p.Player != nil {
			if v, ok := values[p.Player.String()]; ok && !math.IsNaN(v) && !math.IsInf(v, 0) && v >= 0 {
				byRound[p.Round] = append(byRound[p.Round], v)
			}
		}
	}
	for _, p := range picks {
		row := draftRow(pickLabel(p), pickPlayer(p, names), "Value unavailable", "Expected unavailable", "Not measured")
		if p.Player != nil {
			v, ok := values[p.Player.String()]
			cohort := byRound[p.Round]
			if ok && slices.Contains(cohort, v) && len(cohort) >= 2 {
				slices.Sort(cohort)
				n := len(cohort)
				expected := (cohort[(n-1)/2] + cohort[n/2]) / 2
				row.Cells[2], row.Cells[3] = fmt.Sprintf("%.2f points/game", v), fmt.Sprintf("%.2f points/game · %d players", expected, n)
				row.Cells[4] = fmt.Sprintf("%+.2f points/game", v-expected)
				if expected > 0 {
					ratio := v / expected
					row.Range = &MarketRange{Interval: lab.Interval{Value: ratio, Lo: ratio, Hi: ratio}, Text: fmt.Sprintf("%s · %.0f%% of same-round expected value", row.Cells[4], ratio*100)}
				}
			}
		}
		r.Rows = append(r.Rows, row)
	}
	return r
}

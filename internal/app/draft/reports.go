// Package draft builds the MFL draft reports: the rookie draft pick by pick, its grid, recap,
// pace, status and live room, the steals and busts against engine values, future pick ownership,
// and the NFL draft read from the held players export.
//
// It is the feature service behind the root bindings App.TargetDraftReports and
// App.TargetNFLDraft. It holds no state and starts nothing: the caller reads the held feeds
// under its own lock and passes copies in. The Draft Lab (App.TargetDraftLab) stays at root
// until the Market surface it is built on has a shared contract.
package draft

import (
	"cmp"
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/report"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/draftreports"
	"github.com/secureprospective/TheWarRoom/internal/lab"
)

// Held is a feed's records as the caller holds them, with their read state.
type Held[T any] struct {
	Value []T
	Feed  report.Feed
}

// Input is everything the draft reports read, copied by the caller from what it holds.
type Input struct {
	Season  int
	Results Held[draftreports.Pick]
	Future  Held[draftreports.FuturePick]
	// Names are player names by MFL id; Franchises are franchise names by id.
	Names, Franchises map[string]string
	// Values are engine points per game by MFL id; ValueNote says which run they come from.
	Values    map[string]float64
	ValueNote string
	// Next is the next-draft line (NextDate); Zone is the zone times are shown in.
	Next string
	Zone *time.Location
}

func shell[T any](id, title, how string, h Held[T], zone *time.Location, columns ...string) report.Report {
	return report.Shell(id, title, how, h.Feed, len(h.Value), zone, columns...)
}

// Names are the names of the drafted players that name knows, by MFL id.
func Names(picks []draftreports.Pick, name func(id string) (string, bool)) map[string]string {
	names := map[string]string{}
	for _, p := range picks {
		if p.Player != nil {
			if n, ok := name(p.Player.String()); ok {
				names[p.Player.String()] = n
			}
		}
	}
	return names
}

func pickLabel(p draftreports.Pick) string { return fmt.Sprintf("%d.%02d", p.Round, p.Slot) }
func pickPlayer(p draftreports.Pick, names map[string]string) string {
	if p.Skipped {
		return "Pick skipped by MFL"
	}
	if p.Player == nil {
		return "Not made"
	}
	id := p.Player.String()
	if names[id] != "" {
		return names[id]
	}
	return "Player " + id + " (name not held)"
}
func pickTime(p draftreports.Pick, zone *time.Location) string {
	return report.Time(p.At, zone)
}
func orderedPicks(picks []draftreports.Pick) []draftreports.Pick {
	out := slices.Clone(picks)
	slices.SortFunc(out, func(a, b draftreports.Pick) int {
		return cmp.Or(cmp.Compare(a.Round, b.Round), cmp.Compare(a.Slot, b.Slot))
	})
	return out
}

// Build is the draft reports board. It reads only what in holds; it fetches nothing.
func Build(in Input) report.Board {
	season, results, names, franchises, zone := in.Season, in.Results, in.Names, in.Franchises, in.Zone
	picks := orderedPicks(results.Value)
	how := "Picks are ordered by round, then pick number. Each pick shows the franchise MFL records, including traded picks. " +
		"Times use this computer's local time zone. Skipped picks count as made. " +
		"Refresh to check for changes; if it fails, the app keeps the last good copy."
	rows := shell("M-063", "Draft results · pick by pick", how, results, zone,
		"Pick", "Franchise", "Player", "Local time")
	detail := shell("M-102", "Draft pick detail", how, results, zone,
		"Pick", "Franchise", "Player", "Local time", "From whom · MFL comments")
	for _, p := range picks {
		owner := domain.FranchiseLabel(franchises, p.Franchise)
		rows.Rows = append(rows.Rows, report.Cells(pickLabel(p), owner, pickPlayer(p, names), pickTime(p, zone)))
		comments := strings.TrimSpace(p.Comments)
		if comments == "" {
			comments = "MFL lists no trade details"
		}
		detail.Rows = append(detail.Rows, report.Cells(pickLabel(p), owner, pickPlayer(p, names), pickTime(p, zone), comments))
	}
	grid := draftGrid(picks, names, franchises, shell("M-064", "Draft results grid · round × franchise",
		how+" A cell lists every slot drafted by that franchise in that round; traded picks do not shift other cells.", results, zone))
	recap, pace := draftFranchises(picks, names, franchises, results, how, zone)
	status := draftStatus(season, picks, franchises, results, how, zone)
	status.Note += " " + in.Next
	live := status
	live.ID, live.Title = "M-183", "Live draft room"
	live.Note += " Picks happen on MFL only; this app never makes a pick."
	live.Rows = slices.Clone(rows.Rows)
	if !slices.ContainsFunc(picks, func(p draftreports.Pick) bool { return p.Player == nil && !p.Skipped }) {
		live.Rows = live.Rows[max(0, len(live.Rows)-12):]
	}
	live.Summary = slices.Clone(status.Rows)
	live.Link = &report.Link{Label: "See every pick", Endpoint: "M-063"}
	live.Columns = slices.Clone(rows.Columns)
	return report.Board{Title: "MFL draft reports", Loading: "Reading MFL draft reports…", Refresh: "Refresh draft reports from MFL",
		HowLabel: "How this is counted", MFLLabel: "Open MFL draft results (picks happen on MFL)", Reports: []report.Report{rows, grid,
			draftValueReport(picks, names, in.Values, results, how, in.ValueNote, zone), recap, draftFuture(in.Future, franchises, zone), detail, status, pace, live}}
}

func draftGrid(picks []draftreports.Pick, names, franchises map[string]string, r report.Report) report.Report {
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
		row := report.Cells(strconv.Itoa(round))
		for _, id := range ids {
			text := strings.Join(cells[fmt.Sprintf("%d:%s", round, id)], "\n")
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

func draftFranchises(picks []draftreports.Pick, names, franchises map[string]string, feed Held[draftreports.Pick], how string, zone *time.Location) (report.Report, report.Report) {
	recap := shell("M-066", "Draft recap per franchise", how, feed, zone, "Franchise", "Made picks", "Players")
	pace := shell("M-179", "Average elapsed time per pick", how+
		" Average and median time since the preceding pick, including overnight pauses. The first pick, missing times and times earlier than the preceding pick do not count. A median under one minute is labelled mostly pre-queued; it can also reflect fast selections. MFL does not report pauses.", feed, zone,
		"Franchise", "Measured intervals", "Average elapsed time", "Median", "Pace note")
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
		recap.Rows = append(recap.Rows, report.Cells(owner, strconv.Itoa(len(players[id])), strings.Join(players[id], "\n")))
		intervals := elapsed[id]
		average, median, note := draftPace(intervals)
		pace.Rows = append(pace.Rows, report.Cells(owner, strconv.Itoa(len(intervals)), average, median, note))
	}
	return recap, pace
}

func draftStatus(season int, picks []draftreports.Pick, franchises map[string]string, feed Held[draftreports.Pick], how string, zone *time.Location) report.Report {
	r := shell("M-176", "Draft status", how+
		" Made counts player selections and skipped picks. Remaining counts only unmade picks MFL lists. MFL does not say how many rounds are scheduled or who is on the clock.", feed, zone, "State", "What MFL shows")
	made, unmade := 0, 0
	for _, p := range picks {
		if p.Player != nil || p.Skipped {
			made++
		} else {
			unmade++
		}
	}
	r.Rows = append(r.Rows, report.Cells("Made", strconv.Itoa(made)))
	remaining := "MFL does not say whether more picks are scheduled."
	clock := "MFL does not report who is on the clock."
	if unmade > 0 {
		remaining = fmt.Sprintf("%d unmade picks listed; total scheduled picks unknown", unmade)
		for _, p := range picks {
			if p.Player == nil && !p.Skipped {
				clock = "Next unmade pick listed: " + pickLabel(p) + " · " + domain.FranchiseLabel(franchises, p.Franchise) + " (not a live clock)"
				break
			}
		}
	}
	r.Rows = append(r.Rows, report.Cells("On the clock", clock), report.Cells("Remaining", remaining))
	if complete := completedDraft(season, picks, time.Now(), zone); complete != "" {
		r.Rows = []report.Row{report.Cells("Draft", complete)}
	}
	return r
}

func draftFuture(feed Held[draftreports.FuturePick], franchises map[string]string, zone *time.Location) report.Report {
	r := shell("M-068", "Future draft picks · ownership",
		"MFL lists each future pick under its current owner and names the original franchise. Ordered by year, round and original franchise. Picks MFL does not list are not added.", feed, zone,
		"Year", "Round", "Original franchise", "Current owner")
	picks := slices.Clone(feed.Value)
	slices.SortFunc(picks, func(a, b draftreports.FuturePick) int {
		return cmp.Or(cmp.Compare(a.Year, b.Year), cmp.Compare(a.Round, b.Round), strings.Compare(a.Original, b.Original))
	})
	for _, p := range picks {
		r.Rows = append(r.Rows, report.Cells(strconv.Itoa(p.Year), strconv.Itoa(p.Round), domain.FranchiseLabel(franchises, p.Original), domain.FranchiseLabel(franchises, p.Owner)))
	}
	return r
}

func draftValueReport(picks []draftreports.Pick, names map[string]string, values map[string]float64, feed Held[draftreports.Pick], how, valueNote string, zone *time.Location) report.Report {
	r := shell("M-065", "Steals and busts · distance from expected",
		how+" Engine points per game are discounted expected league points per game across the engine’s configured dynasty horizon. Expected is the median current value of drafted players in the same round, not an ADP or draft-day valuation. A bar is the point distance from that baseline (no uncertainty estimate and no letter grade); missing values are not zero. This retrospective comparison is not proof a draft pick was good or bad.",
		feed, zone, "Pick", "Player", "Engine points per game", "Expected · same round", "Distance from expected")

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
		row := report.Cells(pickLabel(p), pickPlayer(p, names), "Value unavailable", "Expected unavailable", "Not measured")
		if p.Player != nil {
			v, ok := values[p.Player.String()]
			cohort := byRound[p.Round]
			if ok && slices.Contains(cohort, v) && len(cohort) >= 2 {
				slices.Sort(cohort)
				n := len(cohort)
				expected := (cohort[(n-1)/2] + cohort[n/2]) / 2
				row.Cells[2], row.Cells[3] = fmt.Sprintf("%.2f points/game", v), fmt.Sprintf("%.2f points/game · %d players", expected, n)
				direction, distance := "above", fmt.Sprintf("+%.2f", v-expected)
				if v < expected {
					direction, distance = "below", fmt.Sprintf("%.2f", expected-v)
				}
				row.Cells[4] = fmt.Sprintf("%s points/game %s the round %d median", distance, direction, p.Round)
				if expected > 0 {
					ratio := v / expected
					row.Range = &report.Range{Interval: lab.Interval{Value: ratio, Lo: ratio, Hi: ratio}, Text: fmt.Sprintf("%s · %.0f%% of same-round expected value", row.Cells[4], ratio*100)}
				}
			}
		}
		r.Rows = append(r.Rows, row)
	}
	return r
}

func draftDuration(d time.Duration) string {
	if d < time.Minute {
		return "under 1 min"
	}
	minutes := int(d.Round(time.Minute) / time.Minute)
	if minutes < 60 {
		return fmt.Sprintf("%d min", minutes)
	}
	return fmt.Sprintf("%d h %d min", minutes/60, minutes%60)
}

func draftPace(intervals []time.Duration) (string, string, string) {
	if len(intervals) == 0 {
		return "No measured interval", "No measured interval", ""
	}
	ordered := slices.Clone(intervals)
	slices.Sort(ordered)
	var total time.Duration
	for _, d := range ordered {
		total += d
	}
	n := len(ordered)
	median := ordered[(n-1)/2] + (ordered[n/2]-ordered[(n-1)/2])/2
	note := ""
	if median < time.Minute {
		note = "Mostly pre-queued picks"
	}
	return draftDuration(total / time.Duration(n)), draftDuration(median), note
}

func completedDraft(season int, picks []draftreports.Pick, now time.Time, zone *time.Location) string {
	if len(picks) == 0 {
		return ""
	}
	first, last := picks[0].At, picks[0].At
	for _, p := range picks {
		if (p.Player == nil && !p.Skipped) || p.At.IsZero() {
			return ""
		}
		if p.At.Before(first) {
			first = p.At
		}
		if p.At.After(last) {
			last = p.At
		}
	}
	if !last.Before(now) {
		return ""
	}
	// The league season the feed belongs to, never the year it was read.
	return fmt.Sprintf("%d rookie draft complete · %d picks, %s–%s", season, len(picks),
		first.In(zone).Format("January 2, 2006"), last.In(zone).Format("January 2, 2006"))
}

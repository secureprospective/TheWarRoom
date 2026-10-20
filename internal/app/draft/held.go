package draft

import (
	"cmp"
	"fmt"
	"slices"
	"strconv"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/report"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/players"
	"github.com/secureprospective/TheWarRoom/internal/leagueclock"
)

// NFL is the NFL draft report for season, read from the held detailed MFL players export.
func NFL(season int, held Held[players.RawPlayer], zone *time.Location) report.Report {
	r := shell("M-087", fmt.Sprintf("NFL draft · %d", season),
		"NFL draft year, round and pick are read from the held detailed MFL players export, not the fantasy league's draftResults. "+
			"Only selections for this season with both round and pick are shown; undrafted and missing selections are not invented.",
		held, zone, "Player", "Position", "NFL team", "Round", "Pick")
	r.Source = "Source: held MFL players (DETAILS=1) archive"
	rows := slices.Clone(held.Value)
	slices.SortFunc(rows, func(a, b players.RawPlayer) int {
		ar, _ := strconv.Atoi(a.DraftRound)
		br, _ := strconv.Atoi(b.DraftRound)
		if order := cmp.Compare(ar, br); order != 0 {
			return order
		}
		ap, _ := strconv.Atoi(a.DraftPick)
		bp, _ := strconv.Atoi(b.DraftPick)
		if order := cmp.Compare(ap, bp); order != 0 {
			return order
		}
		return cmp.Compare(a.ID, b.ID)
	})
	for _, p := range rows {
		year, _ := strconv.Atoi(p.DraftYear)
		round, _ := strconv.Atoi(p.DraftRound)
		pick, _ := strconv.Atoi(p.DraftPick)
		if year != season || round == 0 || pick == 0 {
			continue
		}
		r.Rows = append(r.Rows, report.Cells(p.Name, p.Position, p.Team, p.DraftRound, p.DraftPick))
	}
	if r.Status == "ok" && len(r.Rows) == 0 {
		r.Status, r.Note = "empty", "No NFL selections for this season are held."
	}
	return r
}

// NextDate is the next-draft line: the first draft window after now on the league clock.
func NextDate(windows []leagueclock.Window, now time.Time, zone *time.Location) string {
	for _, w := range windows {
		if w.Kind == leagueclock.Draft && w.At != nil && w.At.After(now) {
			return "Next draft: " + w.At.In(zone).Format("January 2, 2006, 3:04 PM")
		}
	}
	return "Next draft: no date set by the commissioner."
}

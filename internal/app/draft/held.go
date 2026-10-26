package draft

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/report"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/players"
	"github.com/secureprospective/TheWarRoom/internal/leagueclock"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

// History is the part of the history store Values reads: the season's latest engine run and
// its scores.
type History interface {
	LatestRun(ctx context.Context, season int, kind history.RunKind) (history.Run, bool, error)
	ModelScores(ctx context.Context, runID int64) ([]history.ModelScore, error)
}

// Values are the latest engine run's points per game by MFL id, and the note saying which run
// they come from. describe gives the plain meaning of each missing measure, "" when unknown.
func Values(ctx context.Context, h History, season int, describe func(names []string) []string, zone *time.Location) (map[string]float64, string) {
	values := map[string]float64{}
	run, ok, err := h.LatestRun(ctx, season, history.RunModel)
	if err != nil {
		return values, "Engine points unavailable: " + err.Error()
	}
	if !ok {
		return values, "Engine points unavailable until the engine first runs."
	}
	scores, err := h.ModelScores(ctx, run.ID)
	if err != nil {
		return values, "Engine points unavailable: " + err.Error()
	}
	for _, s := range scores {
		values[s.MFLID] = s.DynastyPPG
	}
	return values, ValueNote(run, describe(run.MissingMeasures), zone)
}

// ValueNote names the engine run and, in plain words, the player information it lacked;
// meanings are the run's missing measures described, "" for one not known.
func ValueNote(run history.Run, meanings []string, zone *time.Location) string {
	note := report.EngineRun(run.AsOf, zone) + "."
	missing := []string{}
	for _, m := range meanings {
		if m != "" {
			missing = append(missing, strings.TrimSuffix(m, "."))
		} else {
			missing = append(missing, "an unknown player measure")
		}
	}
	if len(missing) > 0 {
		note += " Missing player information: " + strings.Join(missing, "; ") + "."
	}
	return note
}

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

// NextDraft only reads the clock when a local what-if store exists.
func NextDraft(available bool, read func() (snapshot.Sourced[leagueclock.Reading], error), now func() time.Time, zone *time.Location) string {
	next := "Next draft: no date set by the commissioner."
	if available {
		clock, err := read()
		if err != nil {
			next = "Next draft date unavailable: " + err.Error()
		} else {
			next = NextDate(clock.Value.Windows, now(), zone)
		}
	}
	return next
}

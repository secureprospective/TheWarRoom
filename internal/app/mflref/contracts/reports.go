package contracts

import (
	"fmt"
	"strconv"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/report"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/nflschedule"
	"github.com/secureprospective/TheWarRoom/internal/mflref/lineup"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
)

// Scores is a held playerScores period: the feed's read state and its rows.
type Scores struct {
	Feed report.Feed
	Rows []ingestion.RawPlayerScore
}

// Byes is the held NFL bye-week schedule: the feed's read state and its rows.
type Byes struct {
	Feed report.Feed
	Rows []nflschedule.RawBye
}

// RosterReports are a franchise's roster with season and last-week points (M-101) and its bye weeks
// against the league's position minima (M-154), from held feeds only; zone is the GM's time zone.
func RosterReports(snap snapshot.Snapshot, mine string, season, week Scores, byes Byes, rules lineup.Rules, zone *time.Location) (report.Report, report.Report) {
	stats := report.Shell("M-101", "Roster with stats",
		"Season points and last completed week's points come from held playerScores, not estimates. Missing rows remain unavailable.",
		season.Feed, len(season.Rows), zone, "Player", "Position", "NFL team", "Season points", "Last week points")
	bye := report.Shell("M-154", "Bye weeks",
		"Active players left after each NFL bye are compared with the league's minimum starters at that position. IR and taxi do not count. "+
			"This checks position minima only, not flexible starters, injuries or every legal-lineup constraint.",
		byes.Feed, len(byes.Rows), zone, "Player", "Position", "NFL team", "Bye week")
	stats.Source = "Source: held MFL playerScores archive"
	bye.Source = "Source: held MFL nflByeWeeks archive"
	stats.Status = "ok"
	if season.Feed.FetchedAt.IsZero() || week.Feed.FetchedAt.IsZero() {
		stats.Status = "stale"
	}
	if !week.Feed.FetchedAt.IsZero() {
		stats.Note += " Last-week scores as of " + report.Time(week.Feed.FetchedAt, zone) + "."
	}
	if week.Feed.FetchedAt.IsZero() {
		stats.Note += " Last-week points unavailable."
	}
	identities := map[string]snapshot.Player{}
	for _, p := range snap.Players.Value {
		identities[p.ID.String()] = p
	}
	byeMap := map[string]int{}
	for _, b := range byes.Rows {
		byeMap[b.Team] = b.ByeWeek()
	}
	seasonPoints, weekPoints := scoreMap(season.Rows), scoreMap(week.Rows)
	active := []lineup.ByePlayer{}
	for _, r := range snap.Rosters.Value {
		if r.FranchiseID != mine {
			continue
		}
		for _, p := range r.Players {
			id := p.ID.String()
			identity := identities[id]
			name := identity.Name
			if name == "" {
				name = "Player name unavailable"
			}
			stats.Rows = append(stats.Rows, report.Cells(name, string(identity.Position), identity.Team,
				scoreText(seasonPoints, id), scoreText(weekPoints, id)))
			w := "Unavailable"
			if week, ok := byeMap[identity.Team]; ok {
				w = strconv.Itoa(week)
			}
			bye.Rows = append(bye.Rows, report.Cells(name, string(identity.Position), identity.Team, w))
			active = append(active, lineup.ByePlayer{Position: identity.Position, Team: identity.Team, Active: p.RosterStatus == domain.RosterActive})
		}
	}
	for _, s := range lineup.ByeShortfalls(rules, active, byeMap) {
		bye.Summary = append(bye.Summary, report.Cells(fmt.Sprintf("Week %d · %s", s.Week, s.Position),
			fmt.Sprintf("%d available; needs %d", s.Available, s.Required)))
	}
	if len(bye.Summary) == 0 && len(byes.Rows) > 0 {
		bye.Note = "No position minimum runs short in the held bye-week schedule."
	}
	return stats, bye
}

func scoreMap(rows []ingestion.RawPlayerScore) map[string]string {
	out := map[string]string{}
	for _, r := range rows {
		out[r.ID] = r.Score
	}
	return out
}

func scoreText(points map[string]string, id string) string {
	if value, ok := points[id]; ok {
		return value
	}
	return "No score"
}

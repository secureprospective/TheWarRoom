package commissioner

import (
	"context"
	"fmt"

	"github.com/secureprospective/TheWarRoom/internal/engine/competition"

	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

// build reads the final standings, seeds each conference and, once the commissioner has built it, pairs every
// round from the native results.
func (b *bracketReader) build(ctx context.Context) error {
	standings, err := NativeStandingsReading(ctx, b.league, b.records, b.src, b.season, b.now)
	if err != nil {
		return err
	}
	if words := regularSeasonOpen(standings, b.rules); words != "" {
		b.stop(bracketWaiting, words)
		return nil
	}
	rows := standings.Computed
	if standings.Override != nil {
		rows = standings.Override.Rows
	}
	field, names, rank := b.fieldInput(rows)
	fields, words := competition.SeedField(field)
	if words != "" {
		b.stop(bracketNeeds, words)
		return nil
	}
	b.out.Fields = fields
	if !b.state.Built {
		b.stop(bracketReady, "The regular season is over and the playoff field is set. Build the bracket to set the games.")
		return nil
	}
	return b.pair(ctx, competition.BracketInput{Rules: b.pr, Fields: fields, Rank: rank, Names: names, Edits: b.state.Edits,
		Advance: b.state.Advances})
}

// regularSeasonOpen says why the standings are not final, or nothing when they are.
func regularSeasonOpen(s NativeStandings, rules leaguerules.RuleSet) string {
	switch {
	case s.Status != "Ranked":
		return "The playoffs can't be set yet: " + s.Words
	case s.Through < rules.Schedule.RegularWeeks.Value:
		return fmt.Sprintf("The regular season isn't over: the schedule covers only %d of %d weeks", s.Through,
			rules.Schedule.RegularWeeks.Value)
	case s.Pending > 0:
		return fmt.Sprintf("The regular season isn't over: %d of its games still have no result", s.Pending)
	}
	return ""
}

func (b *bracketReader) fieldInput(rows []StandingRow) (competition.FieldInput, map[string]string, map[string]int) {
	in := competition.FieldInput{Teams: map[string]competition.FieldTeam{}, Conferences: b.layout.Conferences, Rules: b.pr,
		Hand: b.state.handFields(b.layout)}
	names, rank := map[string]string{}, map[string]int{}
	for i, row := range rows {
		in.Order = append(in.Order, row.Franchise)
		in.Teams[row.Franchise] = competition.FieldTeam{Name: row.Name, Division: row.Division, Conference: b.layout.Of[row.Franchise]}
		names[row.Franchise], rank[row.Franchise] = row.Name, i+1
	}
	return in, names, rank
}

// pair builds the rounds, scoring each playoff week from the native results.
func (b *bracketReader) pair(ctx context.Context, in competition.BracketInput) error {
	scores, seats := map[string]FranchiseWeekScore{}, b.directory.Seats()
	var failed error
	in.Points = func(franchise string, week int) (*leaguerules.Decimal, string) {
		key := fmt.Sprintf("%s/%d", franchise, week)
		score, ok := scores[key]
		if !ok && failed == nil {
			score, failed = franchiseWeekScore(ctx, b.records, b.src, b.rules.Lineup, seats.Of(franchise),
				b.season, week)
			scores[key] = score
		}
		return score.Points, score.Words
	}
	rounds, champion, words := competition.BuildBracket(in)
	if failed != nil {
		return failed
	}
	if words != "" {
		b.stop(bracketNeeds, words)
		return nil
	}
	b.out.Rounds = rounds
	b.out.Champion = b.linkedChampion(champion)
	b.stop(bracketBuilt, roundsWords(rounds))
	return nil
}

func roundsWords(rounds []BracketRound) string {
	decided, total := 0, 0
	for _, r := range rounds {
		for _, g := range r.Games {
			total++
			if g.Winner != "" {
				decided++
			}
		}
	}
	return fmt.Sprintf("%d of %d playoff games have a winner", decided, total)
}

package commissioner

import (
	"context"
	"fmt"

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
	fields, words := seedField(field)
	if words != "" {
		b.stop(bracketNeeds, words)
		return nil
	}
	b.out.Fields = fields
	if !b.state.Built {
		b.stop(bracketReady, "The regular season is over and the playoff field is set. Build the bracket to set the games.")
		return nil
	}
	return b.pair(ctx, bracketInput{rules: b.pr, fields: fields, rank: rank, names: names, edits: b.state.Edits,
		advance: b.state.Advances})
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

func (b *bracketReader) fieldInput(rows []StandingRow) (fieldInput, map[string]string, map[string]int) {
	in := fieldInput{teams: map[string]fieldTeam{}, conferences: b.layout.Conferences, rules: b.pr,
		hand: b.state.handFields(b.layout)}
	names, rank := map[string]string{}, map[string]int{}
	for i, row := range rows {
		in.order = append(in.order, row.Franchise)
		in.teams[row.Franchise] = fieldTeam{Name: row.Name, Division: row.Division, Conference: b.layout.Of[row.Franchise]}
		names[row.Franchise], rank[row.Franchise] = row.Name, i+1
	}
	return in, names, rank
}

// pair builds the rounds, scoring each playoff week from the native results.
func (b *bracketReader) pair(ctx context.Context, in bracketInput) error {
	scores, seats := map[string]FranchiseWeekScore{}, b.directory.Seats()
	var failed error
	in.points = func(franchise string, week int) (*leaguerules.Decimal, string) {
		key := fmt.Sprintf("%s/%d", franchise, week)
		score, ok := scores[key]
		if !ok && failed == nil {
			score, failed = franchiseWeekScore(ctx, b.records, b.src, b.rules.Lineup, seats.Of(franchise),
				b.season, week)
			scores[key] = score
		}
		return score.Points, score.Words
	}
	rounds, champion, words := buildBracket(in)
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

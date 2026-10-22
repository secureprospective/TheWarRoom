package history

import (
	"context"
	"fmt"
	"slices"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
)

// previewReadingFields compares owner readings on the same held evidence, before and after one trial act.
func (s *Store) previewReadingFields(ctx context.Context, raw commissioner.RawCommand, principal string,
	records []commissioner.Record, facts appendFacts) ([]commissioner.PreviewField, error) {
	if season := previewScoreSeason(raw); season != 0 {
		recorded, err := commissioner.PreviewSeasonRecorded(records, season, s.now())
		if err != nil {
			return nil, fmt.Errorf("read preview season: %w", err)
		}
		if !recorded {
			words := "This season has no scores in the app league"
			return []commissioner.PreviewField{{Label: "Scoring history", Before: words,
				BeforeSource: "App league history", After: words, AfterSource: "App league history", When: "now"}}, nil
		}
	}
	trial := append(slices.Clone(records), commissioner.Record{Command: raw,
		Receipt: commissioner.Receipt{Entry: "proposed." + raw.Request, RecordedAt: s.now()}})
	before, err := s.previewReadings(ctx, raw, principal, records, facts)
	if err != nil {
		return nil, err
	}
	after, err := s.previewReadings(ctx, raw, principal, trial, facts)
	if err != nil {
		return nil, err
	}
	fields := commissioner.ChangedPreviewFields(before, after)
	if len(fields) == 0 && raw.Command == commissioner.ReserveException {
		fields = commissioner.UnchangedPreviewFields(raw, before, after)
	}
	return fields, nil
}

func (s *Store) previewReadings(ctx context.Context, raw commissioner.RawCommand, principal string,
	records []commissioner.Record, facts appendFacts) (map[string]string, error) {

	out := map[string]string{}
	switch {
	case raw.Command == commissioner.ReserveException:
		if err := commissioner.ReservePreviewValues(raw, records, s.now(), facts.players, out); err != nil {
			return nil, fmt.Errorf("read reserve eligibility for preview: %w", err)
		}
	case raw.Args.Bracket != nil:
		return s.previewBracketReadings(ctx, raw, records, facts)
	case raw.Args.Standings != nil:
		season := raw.Args.Standings.Season
		src := s.nativeSources(raw.League, season)
		view, err := commissioner.NativeStandingsReading(ctx, raw.League, records, src, season, s.now())
		if err != nil {
			return nil, fmt.Errorf("read standings for preview: %w", err)
		}
		rows := view.Computed
		if view.Override != nil {
			rows = view.Override.Rows
		}
		for _, r := range rows {
			out[fmt.Sprintf("%s season %d standings", r.Name, season)] =
				fmt.Sprintf("Place %d; %s; %s", r.Rank, r.Record, r.PointsWords)
		}
	case raw.Args.Stat != nil:
		return s.previewStatReadings(ctx, raw, records)
	case raw.Args.Score != nil:
		c := raw.Args.Score
		if c.Season == 0 {
			return out, nil
		}
		src := s.nativeSources(raw.League, c.Season)
		if c.Player != "" {
			view, err := src.PlayerScore(ctx, records, c.Player, c.Week)
			if err != nil {
				return nil, err
			}
			out[fmt.Sprintf("Player %s, season %d, week %d score", c.Player, c.Season, c.Week)] = view.Words
		}
		return s.previewResultReadings(ctx, raw.League, records, c.Season, c.Week, out)
	case raw.Args.Lineup != nil:
		c := raw.Args.Lineup
		view, err := commissioner.NativeLineupReading(raw.League, c.Franchise, principal, c.Week, records, s.now(), facts.lineup)
		if err != nil {
			return nil, fmt.Errorf("read lineup for preview: %w", err)
		}
		out[view.FranchiseName+" lineup verdict"] = view.Words
		return s.previewResultReadings(ctx, raw.League, records, c.Season, c.Week, out)
	}
	return out, nil
}

func (s *Store) previewResultReadings(ctx context.Context, league string, records []commissioner.Record,
	season, week int, out map[string]string) (map[string]string, error) {
	if !nativeSeeded(records) {
		return out, nil
	}

	src := s.nativeSources(league, season)
	view, err := commissioner.NativeResultsReading(ctx, league, records, src, season, week, s.now())
	if err != nil {
		return nil, fmt.Errorf("read results for preview: %w", err)
	}
	for _, score := range view.Scores {
		out[fmt.Sprintf("%s, week %d score", score.Name, week)] = score.Words
	}
	for _, game := range view.Games {
		out[fmt.Sprintf("%s v %s, week %d result", game.HomeName, game.AwayName, week)] = game.Words
	}
	return out, nil
}

func (s *Store) previewStatReadings(ctx context.Context, raw commissioner.RawCommand,
	records []commissioner.Record) (map[string]string, error) {
	c, err := commissioner.DecodeStatChange(raw.Args, raw.Command)
	if err != nil {
		return nil, fmt.Errorf("read stat preview target: %w", err)
	}
	player, season, week := c.Scope()
	src := s.nativeSources(raw.League, season)
	score, err := src.PlayerScore(ctx, records, player, week)
	if err != nil {
		return nil, err
	}
	out := map[string]string{fmt.Sprintf("Player %s, season %d, week %d score", player, season, week): score.Words}
	for _, item := range score.Items {
		out["Player "+player+" — "+string(item.Stat)] = item.Words
	}
	view, err := commissioner.StatQueueReading(ctx, raw.League, records,
		seasonStatFeed{store: s, season: season}, src, season, s.now())
	if err != nil {
		return nil, fmt.Errorf("read stat queue for preview: %w", err)
	}
	for _, item := range view.Open {
		if item.Player == player && item.Week == week {
			out["Statistic update"] = item.Words + "; waiting for a decision"
		}
	}
	for _, d := range view.Decided {
		if d.Player == player && d.Week == week {
			out["Statistic decision "+d.Entry] = d.Status + ": " + d.Words
		}
	}
	return out, nil
}

func (s *Store) previewBracketReadings(ctx context.Context, raw commissioner.RawCommand,
	records []commissioner.Record, facts appendFacts) (map[string]string, error) {
	season := raw.Args.Bracket.Season
	src := s.nativeSources(raw.League, season)
	view, err := commissioner.NativeBracketReading(ctx, raw.League, records, src, season, s.now(), facts.lineup)
	if err != nil {
		return nil, fmt.Errorf("read playoff bracket for preview: %w", err)
	}
	out := map[string]string{"Playoff bracket standing": view.Words}
	for _, conference := range view.Fields {
		for _, seed := range conference.Seeds {
			out[fmt.Sprintf("%s seed %d", conference.Conference, seed.Seed)] = seed.Name
		}
	}
	for _, round := range view.Rounds {
		for _, game := range round.Games {
			out["Playoff game "+game.ID] = game.Words
		}
	}
	return out, nil
}

func previewScoreSeason(raw commissioner.RawCommand) int {
	if raw.Args.Score != nil {
		return raw.Args.Score.Season
	}
	if raw.Args.Standings != nil {
		return raw.Args.Standings.Season
	}
	return 0
}

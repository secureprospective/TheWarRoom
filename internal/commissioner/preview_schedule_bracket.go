package commissioner

import (
	"fmt"
	"strings"
	"time"
)

func schedulePreviewValues(records []Record) (map[string]string, error) {
	draft, published, err := replaySchedule(records)
	if err != nil {
		return nil, err
	}
	out := map[string]string{}
	schedulePreviewWeeks("Proposed schedule", draft, out)
	schedulePreviewWeeks("Published schedule", published, out)
	return out, nil
}

func schedulePreviewWeeks(label string, weeks []ScheduleWeek, out map[string]string) {
	out[label] = fmt.Sprintf("%d weeks", len(weeks))
	for _, week := range weeks {
		games := []string{}
		for _, game := range week.Games {
			games = append(games, game.Home+" v "+game.Away)
		}
		out[fmt.Sprintf("%s, week %d", label, week.Week)] = strings.Join(games, "; ")
	}
}

func bracketPreviewValues(raw RawCommand, records []Record, now time.Time) (map[string]string, error) {
	directory, err := NativeDirectoryReading(raw.League, records, now)
	if err != nil {
		return nil, err
	}
	names := map[string]string{}
	for _, f := range directory.Franchises {
		names[f.Franchise] = f.Name
	}
	name := func(id string) string {
		if names[id] != "" {
			return names[id]
		}
		return id
	}
	state := foldBracket(records, raw.Args.Bracket.Season, name)
	out := map[string]string{"Playoff bracket": "Not built"}
	if state.Built {
		out["Playoff bracket"] = "Built from " + bracketPresetWords(state.Preset)
	}
	for _, seed := range state.Seeds {
		out[fmt.Sprintf("Playoff seed %d set by hand", seed.Slot)] = name(seed.Team)
	}
	for node, team := range state.Edits {
		out["Playoff place "+node] = name(team)
	}
	for game, team := range state.Advances {
		out["Winner of playoff game "+game] = name(team)
	}
	for preset, shape := range state.Presets {
		for _, game := range shape {
			out[fmt.Sprintf("%s game %s", preset, game.ID)] =
				fmt.Sprintf("Week %d; home seed %d (%s); away seed %d (%s)", game.Week,
					game.HomeSeed, game.HomeFrom, game.AwaySeed, game.AwayFrom)
		}
	}
	return out, nil
}

func bracketPresetWords(preset string) string {
	switch preset {
	case presetLeagueRules:
		return "the league playoff rules"
	case presetLegacyNFL:
		return "the Legacy NFL playoff shape"
	default:
		return preset
	}
}

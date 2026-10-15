package commissioner

import (
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

const (
	ScheduleGenerate = "schedule.generate"
	ScheduleEdit     = "schedule.edit"
	SchedulePublish  = "schedule.publish"
)

type RawScheduleArgs struct {
	Weeks     int                         `json:"weeks,omitempty"`
	Pattern   leaguerules.SchedulePattern `json:"pattern,omitempty"`
	Week      int                         `json:"week,omitempty"`
	Games     []ScheduleGame              `json:"games,omitempty"`
	Published bool                        `json:"published,omitempty"`
}

func scheduleCommand(command string) bool {
	return command == ScheduleGenerate || command == ScheduleEdit || command == SchedulePublish
}

func (a RawArgs) validateSchedule(command string) error {
	other := a
	other.Schedule = nil
	if a.Schedule == nil || !reflect.DeepEqual(other, RawArgs{}) {
		return fmt.Errorf("send only the details of this schedule change")
	}
	details := *a.Schedule
	switch command {
	case ScheduleGenerate:
		if details.Weeks < 1 || details.Weeks > 52 {
			return fmt.Errorf("choose between 1 and 52 regular-season weeks")
		}
		if details.Pattern != "" && details.Pattern != leaguerules.DivisionWeighted {
			return fmt.Errorf("choose a recognised schedule pattern")
		}
		details.Weeks, details.Pattern = 0, ""
	case ScheduleEdit:
		if err := details.validateGamePatches(); err != nil {
			return err
		}
		details.Week, details.Games, details.Published = 0, nil, false
	}
	if !reflect.DeepEqual(details, RawScheduleArgs{}) {
		return fmt.Errorf("send only the details of this schedule change")
	}
	return nil
}

type Schedule struct {
	League   string `json:"league"`
	Revision int64  `json:"revision"`
	// Season is the NFL season the published weeks are for.
	Season int            `json:"season"`
	Weeks  []ScheduleWeek `json:"weeks"`
}

// ScheduleReading exposes publications only; a later draft never replaces an owner's schedule.
func ScheduleReading(league string, records []Record) (Schedule, error) {
	_, published, err := replaySchedule(records)
	if err != nil {
		return Schedule{}, err
	}
	rules, err := resolveRecords(records)
	if err != nil {
		return Schedule{}, err
	}
	return Schedule{League: league, Revision: int64(len(records)), Season: rules.Season, Weeks: published}, nil
}

// ScheduleSeasonReading freezes ended seasons at their rollover boundary. A new
// season has no publication until its own schedule is published.
func ScheduleSeasonReading(league string, records []Record, season int, at time.Time) (Schedule, error) {
	if len(records) == 0 {
		return Schedule{}, fmt.Errorf("create this app league before reading its schedule")
	}
	starts, err := rolloverInstants(recordsThrough(records, at))
	if err != nil {
		return Schedule{}, err
	}
	first, err := firstSeason(records)
	if err != nil {
		return Schedule{}, err
	}
	index := season - first
	if index < 0 || index > len(starts) || (index > 0 && at.Before(starts[index-1])) {
		return Schedule{}, fmt.Errorf("choose a season in this league's history")
	}
	var start time.Time
	if index > 0 {
		start = starts[index-1]
	}
	if index < len(starts) && !at.Before(starts[index]) {
		at = starts[index].Add(-time.Nanosecond)
	}
	selected := append([]Record{}, recordsThrough(records, at)...)
	for i := range selected {
		if selected[i].Receipt.RecordedAt.Before(start) && scheduleCommand(selected[i].Command.Command) {
			selected[i].Command.Command = ""
		}
	}
	return ScheduleReading(league, selected)
}

func replaySchedule(records []Record) ([]ScheduleWeek, []ScheduleWeek, error) {
	active := Replay(records).Active
	draft, published := []ScheduleWeek{}, []ScheduleWeek{}
	for _, r := range records {
		if !active[r.Receipt.Entry] || (!scheduleCommand(r.Command.Command) && r.Command.Command != SeasonRollover) {
			continue
		}
		switch r.Command.Command {
		case SeasonRollover:
			// A schedule belongs to one season; the new season starts with none.
			draft, published = []ScheduleWeek{}, []ScheduleWeek{}
		case ScheduleGenerate:
			if len(r.GeneratedSchedule) == 0 {
				return nil, nil, fmt.Errorf("the recorded schedule is missing its generated weeks")
			}
			draft = cloneSchedule(r.GeneratedSchedule)
		case ScheduleEdit:
			patch := *r.Command.Args.Schedule
			if r.Command.Args.Schedule.Published {
				published = patchScheduleGames(published, patch)
			} else {
				draft = patchScheduleGames(draft, patch)
			}
		case SchedulePublish:
			published = append([]ScheduleWeek{}, draft...)
		}
	}
	return draft, published, nil
}

func cloneSchedule(weeks []ScheduleWeek) []ScheduleWeek {
	result := append([]ScheduleWeek{}, weeks...)
	for i := range result {
		result[i].Games = append([]ScheduleGame{}, result[i].Games...)
	}
	return result
}

// validateGamePatches needs a week and at least one game, each named once with both franchises. Several
// games change together, so two pairings can be swapped without a team playing twice in between.
func (r RawScheduleArgs) validateGamePatches() error {
	if r.Week < 1 || len(r.Games) == 0 {
		return fmt.Errorf("choose a week and the games to edit")
	}
	seen := map[string]bool{}
	for _, game := range r.Games {
		if strings.TrimSpace(game.ID) == "" || strings.TrimSpace(game.Home) == "" ||
			strings.TrimSpace(game.Away) == "" {
			return fmt.Errorf("choose each game and both of its franchises")
		}
		if seen[game.ID] {
			return fmt.Errorf("name each game once")
		}
		seen[game.ID] = true
	}
	return nil
}

// patchScheduleGames sets the named games' franchises in one week and leaves every other game as it was.
func patchScheduleGames(weeks []ScheduleWeek, patch RawScheduleArgs) []ScheduleWeek {
	changes := map[string]ScheduleGame{}
	for _, game := range patch.Games {
		changes[game.ID] = game
	}
	result := cloneSchedule(weeks)
	for i := range result {
		if result[i].Week != patch.Week {
			continue
		}
		for j := range result[i].Games {
			game := &result[i].Games[j]
			if change, ok := changes[game.ID]; ok {
				game.Home, game.Away = change.Home, change.Away
			}
		}
	}
	return result
}

// FreezeSchedule captures generation at append time, outside the request's retry identity.
func FreezeSchedule(raw RawCommand, records []Record) ([]ScheduleWeek, error) {
	if len(records) == 0 {
		return nil, fmt.Errorf("create the league before generating its schedule")
	}
	rules, err := resolveRecords(records)
	if err != nil {
		return nil, err
	}
	teams := records[0].Command.Args.Franchises

	pattern := rules.Schedule.Pattern
	if raw.Args.Schedule.Pattern != "" {
		pattern = leaguerules.Known(raw.Args.Schedule.Pattern)
	}
	return GenerateSchedule(teams, rules.League.Alignment.Value, leaguerules.ScheduleRules{
		RegularWeeks: leaguerules.Known(raw.Args.Schedule.Weeks),
		Pattern:      pattern,
	})
}

func CheckScheduleAppend(raw RawCommand, records []Record) error {
	if !scheduleCommand(raw.Command) {
		return nil
	}
	draft, published, err := replaySchedule(records)
	if err != nil {
		return err
	}
	switch raw.Command {
	case ScheduleGenerate:
		_, err = FreezeSchedule(raw, records)
	case SchedulePublish:
		if len(draft) == 0 {
			err = fmt.Errorf("generate a schedule before publishing it")
		}
	case ScheduleEdit:
		target := draft
		if raw.Args.Schedule.Published {
			target = published
			if strings.TrimSpace(raw.Reason) == "" {
				return Refused(fmt.Errorf("give a reason for changing a published week"))
			}
		}
		err = checkSchedulePatch(*raw.Args.Schedule, target, records)
	}
	return Refused(err)
}

func checkSchedulePatch(patch RawScheduleArgs, target []ScheduleWeek, records []Record) error {
	for _, week := range target {
		if week.Week != patch.Week {
			continue
		}
		ids := map[string]bool{}
		for _, game := range week.Games {
			ids[game.ID] = true
		}
		for _, game := range patch.Games {
			if !ids[game.ID] {
				return fmt.Errorf("choose a game in the generated or published schedule")
			}
		}
		return checkScheduleWeek(patchScheduleGames([]ScheduleWeek{week}, patch)[0], target, records)
	}
	return fmt.Errorf("choose a game in the generated or published schedule")
}

func checkScheduleWeek(week ScheduleWeek, target []ScheduleWeek, records []Record) error {
	if week.Week > len(target) || len(records) == 0 {
		return fmt.Errorf("choose a week in the generated or published schedule")
	}
	remaining := make(map[string]bool)
	for _, team := range records[0].Command.Args.Franchises {
		remaining[team] = true
	}
	for _, game := range week.Games {
		if !remaining[game.Home] || !remaining[game.Away] || game.Home == game.Away {
			return fmt.Errorf("each franchise must play exactly once per week")
		}
		delete(remaining, game.Home)
		delete(remaining, game.Away)
	}
	odd := len(records[0].Command.Args.Franchises)%2 == 1
	if odd {
		if len(remaining) != 1 || !remaining[week.Bye] {
			return fmt.Errorf("give exactly one franchise a bye")
		}
	} else if len(remaining) != 0 || week.Bye != "" {
		return fmt.Errorf("every franchise must play this week")
	}
	return nil
}

func (r RawCommand) validateSchedulePlacement() error {
	if scheduleCommand(r.Command) && (r.Visibility != "league" || r.Effective != "now") {
		return fmt.Errorf("schedule changes take effect now and are visible to every owner")
	}
	return nil
}

func (r Record) validateGeneratedSchedule() error {
	if r.Command.Command != ScheduleGenerate {
		if len(r.GeneratedSchedule) != 0 {
			return fmt.Errorf("only schedule generation may record generated weeks")
		}
		return nil
	}
	if len(r.GeneratedSchedule) != r.Command.Args.Schedule.Weeks {
		return fmt.Errorf("the recorded schedule does not contain its requested weeks")
	}
	ids := make(map[string]bool)
	for i, week := range r.GeneratedSchedule {
		if week.Week != i+1 || len(week.Games) == 0 {
			return fmt.Errorf("the recorded schedule has an invalid week")
		}
		if err := week.validateFrozenGames(ids); err != nil {
			return err
		}
	}
	return nil
}

func (w ScheduleWeek) validateFrozenGames(ids map[string]bool) error {
	teams := make(map[string]bool)
	for _, game := range w.Games {
		if strings.TrimSpace(game.ID) == "" || ids[game.ID] ||
			strings.TrimSpace(game.Home) == "" || strings.TrimSpace(game.Away) == "" ||
			game.Home == game.Away || teams[game.Home] || teams[game.Away] {
			return fmt.Errorf("the recorded schedule has an invalid or repeated game")
		}
		ids[game.ID], teams[game.Home], teams[game.Away] = true, true, true
	}
	if teams[w.Bye] {
		return fmt.Errorf("a franchise with a bye cannot also play")
	}
	return nil
}

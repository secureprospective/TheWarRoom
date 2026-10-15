package commissioner

import (
	"fmt"
	"reflect"
	"strings"
)

const (
	ScheduleGenerate = "schedule.generate"
	ScheduleEdit     = "schedule.edit"
	SchedulePublish  = "schedule.publish"
)

type RawScheduleArgs struct {
	Week      *ScheduleWeek `json:"week,omitempty"`
	Published bool          `json:"published,omitempty"`
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
	if command != ScheduleEdit {
		if a.Schedule.Week != nil || a.Schedule.Published {
			return fmt.Errorf("send no week details when generating or publishing the schedule")
		}
		return nil
	}
	if a.Schedule.Week == nil || a.Schedule.Week.Week < 1 {
		return fmt.Errorf("choose the schedule week to edit")
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

func replaySchedule(records []Record) ([]ScheduleWeek, []ScheduleWeek, error) {
	active := Replay(records).Active
	draft, published := []ScheduleWeek{}, []ScheduleWeek{}
	for i, r := range records {
		if !active[r.Receipt.Entry] || (!scheduleCommand(r.Command.Command) && r.Command.Command != SeasonRollover) {
			continue
		}
		switch r.Command.Command {
		case SeasonRollover:
			// A schedule belongs to one season; the new season starts with none.
			draft, published = []ScheduleWeek{}, []ScheduleWeek{}
		case ScheduleGenerate:
			var err error
			draft, err = generatedSchedule(records[:i])
			if err != nil {
				return nil, nil, err
			}
		case ScheduleEdit:
			week := *r.Command.Args.Schedule.Week
			if r.Command.Args.Schedule.Published {
				published = replaceScheduleWeek(published, week)
			} else {
				draft = replaceScheduleWeek(draft, week)
			}
		case SchedulePublish:
			published = append([]ScheduleWeek{}, draft...)
		}
	}
	return draft, published, nil
}

func replaceScheduleWeek(weeks []ScheduleWeek, week ScheduleWeek) []ScheduleWeek {
	result := append([]ScheduleWeek{}, weeks...)
	for i := range result {
		if result[i].Week == week.Week {
			result[i] = week
		}
	}
	return result
}

func generatedSchedule(records []Record) ([]ScheduleWeek, error) {
	if len(records) == 0 {
		return nil, fmt.Errorf("create the league before generating its schedule")
	}
	rules, err := resolveRecords(records)
	if err != nil {
		return nil, err
	}
	teams := records[0].Command.Args.Franchises

	return GenerateSchedule(teams, rules.League.Alignment.Value, rules.Schedule)
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
		_, err = generatedSchedule(records)
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
		err = checkScheduleWeek(*raw.Args.Schedule.Week, target, records)
	}
	return Refused(err)
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

package commissioner

import (
	"fmt"
	"reflect"
	"slices"
	"strings"
)

const (
	StandingsRecordAdjust  = "commish.standings.record.adjust"
	StandingsOverride      = "commish.standings.override"
	StandingsOverrideClear = "commish.standings.override.clear"

	// maxStandingsGames bounds a record a commissioner may set; no regular season is longer.
	maxStandingsGames = 60
)

// RawStandingsChange is a standings command's argument (Args.Standings). A record is the one that stands
// for the franchise; an order lists every franchise, best first.
type RawStandingsChange struct {
	Season    int      `json:"season"`
	Franchise string   `json:"franchise,omitempty"`
	Wins      int      `json:"wins,omitempty"`
	Losses    int      `json:"losses,omitempty"`
	Ties      int      `json:"ties,omitempty"`
	Order     []string `json:"order,omitempty"`
}

func isStandingsCommand(command string) bool {
	return command == StandingsRecordAdjust || command == StandingsOverride || command == StandingsOverrideClear
}

// DecodeStandingsChange reads a standings command's argument and refuses details that belong to another command.
func DecodeStandingsChange(a RawArgs, command string) (RawStandingsChange, error) {
	other := a
	other.Standings = nil
	if a.Standings == nil || !reflect.DeepEqual(other, RawArgs{}) {
		return RawStandingsChange{}, fmt.Errorf("send only the details of this standings change")
	}
	raw := *a.Standings
	if raw.Season < 1920 || raw.Season > 9999 {
		return raw, fmt.Errorf("choose the season these standings are for")
	}
	hasRecord := raw.Franchise != "" || raw.Wins != 0 || raw.Losses != 0 || raw.Ties != 0
	switch command {
	case StandingsRecordAdjust:
		return raw, raw.validateRecord(len(raw.Order) != 0)
	case StandingsOverride:
		if hasRecord || len(raw.Order) == 0 || slices.Contains(raw.Order, "") {
			return raw, fmt.Errorf("list every franchise in the order the standings must stand")
		}
	default:
		if hasRecord || len(raw.Order) != 0 {
			return raw, fmt.Errorf("clearing the override takes only the season")
		}
	}
	return raw, nil
}

func (r RawStandingsChange) validateRecord(hasOrder bool) error {
	if hasOrder || !strings.HasPrefix(r.Franchise, "franchise.") || strings.TrimSpace(r.Franchise) != r.Franchise {
		return fmt.Errorf("choose one app franchise and the record that must stand")
	}
	for _, n := range []int{r.Wins, r.Losses, r.Ties} {
		if n < 0 || n > maxStandingsGames {
			return fmt.Errorf("a record counts between 0 and %d wins, losses and ties", maxStandingsGames)
		}
	}
	return nil
}

func standingsLabel(command string, change RawStandingsChange) string {
	switch command {
	case StandingsRecordAdjust:
		return fmt.Sprintf("Set the record of %s to %s for season %d", change.Franchise,
			tally{W: change.Wins, L: change.Losses, T: change.Ties}.words(), change.Season)
	case StandingsOverride:
		return fmt.Sprintf("Set the standings order by hand for season %d", change.Season)
	default:
		return fmt.Sprintf("Cleared the commissioner's standings override for season %d", change.Season)
	}
}

// checkStandingsAppend refuses what the league cannot hold: a franchise it does not have, an order that
// leaves one out, or an override that is not there to clear.
func checkStandingsAppend(raw RawCommand, records []Record) error {
	change, err := DecodeStandingsChange(raw.Args, raw.Command)
	if err != nil {
		return err
	}
	if len(records) == 0 {
		return Refused(fmt.Errorf("this app league has not been created"))
	}
	franchises := records[0].Command.Args.Franchises
	switch raw.Command {
	case StandingsRecordAdjust:
		if !slices.Contains(franchises, change.Franchise) {
			return Refused(fmt.Errorf("that franchise is not in this app league"))
		}
	case StandingsOverride:
		if !sameMembers(franchises, change.Order) {
			return Refused(fmt.Errorf("list every franchise in this league exactly once"))
		}
	default:
		held, err := StandingsAdjustmentsReading(records, change.Season)
		if err != nil {
			return err
		}
		if len(held.Records) == 0 && held.Order == nil {
			return Refused(fmt.Errorf("there is no standings override to clear for that season"))
		}
	}
	return nil
}

func sameMembers(a, b []string) bool {
	x, y := slices.Clone(a), slices.Clone(b)
	slices.Sort(x)
	slices.Sort(y)
	return slices.Equal(x, y)
}

// AdjustedRecord is a record the commissioner set to stand for a franchise.
type AdjustedRecord struct {
	Franchise string `json:"franchise"`
	Wins      int    `json:"wins"`
	Losses    int    `json:"losses"`
	Ties      int    `json:"ties"`
	Entry     string `json:"entry"`
	Reason    string `json:"reason"`
}

// OrderOverride is a full order the commissioner set to stand.
type OrderOverride struct {
	Order  []string `json:"order"`
	Entry  string   `json:"entry"`
	Reason string   `json:"reason"`
}

// StandingsAdjustments is what overrides stand for one season after every compensation is folded.
type StandingsAdjustments struct {
	Records map[string]AdjustedRecord `json:"records"`
	Order   *OrderOverride            `json:"order,omitempty"`
	Changes []ScoreChange             `json:"changes"`
}

// StandingsAdjustmentsReading folds the active standings commands of a season in the order they were
// recorded: a later record replaces an earlier one, and a clear drops everything before it.
func StandingsAdjustmentsReading(records []Record, season int) (StandingsAdjustments, error) {
	out := StandingsAdjustments{Records: map[string]AdjustedRecord{}, Changes: []ScoreChange{}}
	active := Replay(records).Active
	for _, r := range records {
		command := r.Command.Command
		if !isStandingsCommand(command) {
			continue
		}
		change, err := DecodeStandingsChange(r.Command.Args, command)
		if err != nil {
			return StandingsAdjustments{}, err
		}
		if change.Season != season {
			continue
		}
		status := "Reversed"
		if active[r.Receipt.Entry] {
			status = "Applied"
			out.apply(command, change, r)
		}
		out.Changes = append(out.Changes, ScoreChange{Entry: r.Receipt.Entry,
			What: standingsLabel(command, change), Reason: r.Command.Reason, Status: status})
	}
	return out, nil
}

func (s *StandingsAdjustments) apply(command string, change RawStandingsChange, r Record) {
	switch command {
	case StandingsRecordAdjust:
		s.Records[change.Franchise] = AdjustedRecord{Franchise: change.Franchise, Wins: change.Wins,
			Losses: change.Losses, Ties: change.Ties, Entry: r.Receipt.Entry, Reason: r.Command.Reason}
	case StandingsOverride:
		s.Order = &OrderOverride{Order: slices.Clone(change.Order), Entry: r.Receipt.Entry, Reason: r.Command.Reason}
	default:
		s.Records, s.Order = map[string]AdjustedRecord{}, nil
	}
}

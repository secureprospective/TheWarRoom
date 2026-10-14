package commissioner

import (
	"fmt"
	"maps"
	"reflect"
	"slices"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

const (
	WindowSet    = "calendar.window.set"
	WindowMove   = "calendar.window.move"
	WindowRemove = "calendar.window.remove"

	windowsField = leaguerules.FieldPath("calendar.windows")
)

// RawDate is a month and day of the league's year; the year comes from the season being read.
type RawDate struct {
	Month int `json:"month"`
	Day   int `json:"day"`
}

// RawWindowArgs carry the census verbs: set takes id, start, optional end and opens; move takes id and
// start; remove takes id.
type RawWindowArgs struct {
	ID    string               `json:"id"`
	Start *RawDate             `json:"start,omitempty"`
	End   *RawDate             `json:"end,omitempty"`
	Opens []leaguerules.Action `json:"opens,omitempty"`
}

func windowCommand(command string) bool {
	return command == WindowSet || command == WindowMove || command == WindowRemove
}

func (d RawDate) validate() error {
	at := time.Date(2000, time.Month(d.Month), d.Day, 0, 0, 0, 0, time.UTC)
	if d.Month < 1 || d.Month > 12 || d.Day < 1 || int(at.Month()) != d.Month || at.Day() != d.Day {
		return fmt.Errorf("choose a valid calendar date")
	}
	return nil
}

func (d RawDate) monthDay() leaguerules.MonthDay {
	return leaguerules.MonthDay{Month: leaguerules.Known(d.Month), Day: leaguerules.Known(d.Day)}
}

func (a RawArgs) validateWindow(command string) error {
	other := a
	other.Window = nil
	if a.Window == nil || !reflect.DeepEqual(other, RawArgs{}) {
		return fmt.Errorf("send only the details of this window change")
	}
	w := a.Window
	if strings.TrimSpace(w.ID) == "" || strings.TrimSpace(w.ID) != w.ID {
		return fmt.Errorf("choose the window by its name")
	}
	if command == WindowRemove {
		if w.Start != nil || w.End != nil || len(w.Opens) != 0 {
			return fmt.Errorf("send only the name of the window to remove")
		}
		return nil
	}
	if w.Start == nil {
		return fmt.Errorf("choose the day the window opens")
	}
	if err := w.Start.validate(); err != nil {
		return err
	}
	if command == WindowMove {
		if w.End != nil || len(w.Opens) != 0 {
			return fmt.Errorf("send only the new opening day to move a window")
		}
		return nil
	}
	return w.validateDefinition()
}

func (w RawWindowArgs) validateDefinition() error {
	if w.End != nil {
		if err := w.End.validate(); err != nil {
			return err
		}
	}
	if len(w.Opens) == 0 {
		return fmt.Errorf("choose what the window opens")
	}
	seen := map[leaguerules.Action]bool{}
	for _, action := range w.Opens {
		if strings.TrimSpace(string(action)) == "" || seen[action] {
			return fmt.Errorf("choose each action once")
		}
		seen[action] = true
	}
	return nil
}

// applyWindow is the one place a window command becomes a calendar.windows value. Replay feeds it the
// windows in force just before the command, exactly as a rule change would have left them.
func applyWindow(command string, w RawWindowArgs, now map[string]leaguerules.Window) (map[string]leaguerules.Window, error) {
	next := maps.Clone(now)
	if next == nil {
		next = map[string]leaguerules.Window{}
	}
	current, exists := next[w.ID]
	switch {
	case command == WindowSet:
		window := leaguerules.Window{Start: leaguerules.Known(w.Start.monthDay()),
			End:        leaguerules.Needs[leaguerules.MonthDay]("Window set by the commissioner: no closing day"),
			Opens:      leaguerules.Known(slices.Clone(w.Opens)),
			DatedStart: leaguerules.Needs[time.Time]("Window set by the commissioner: no dated opening"),
			DatedEnd:   leaguerules.Needs[time.Time]("Window set by the commissioner: no dated closing")}
		if w.End != nil {
			window.End = leaguerules.Known(w.End.monthDay())
		}
		next[w.ID] = window
	case !exists:
		return nil, fmt.Errorf("the %s window does not exist; set it first", w.ID)
	case command == WindowMove:
		// A dated opening outranks the day, so a move that left it would not move anything.
		current.Start = leaguerules.Known(w.Start.monthDay())
		current.DatedStart = leaguerules.Needs[time.Time]("Window moved by the commissioner")
		next[w.ID] = current
	default:
		delete(next, w.ID)
	}
	return next, nil
}

// windowChange turns one active window command into the rule change that rules.set would have recorded.
// prior holds every change already in the ledger, each with the instant it starts.
func windowChange(prior []startedChange, r Record) (leaguerules.Change, error) {
	at := r.Receipt.RecordedAt
	var started []leaguerules.Change
	for _, t := range prior {
		if !t.start.After(at) {
			c := t.change
			c.Season = ledgerSeason
			started = append(started, c)
		}
	}
	current, err := leaguerules.Resolve(started, ledgerSeason, at)
	if err != nil {
		return leaguerules.Change{}, fmt.Errorf("read windows before change: %w", err)
	}
	var windows map[string]leaguerules.Window
	if current.Calendar.Windows.State == leaguerules.IsKnown {
		windows = current.Calendar.Windows.Value
	}
	next, err := applyWindow(r.Command.Command, *r.Command.Args.Window, windows)
	if err != nil {
		return leaguerules.Change{}, err
	}
	return leaguerules.Change{LeagueID: r.Command.League, Field: windowsField,
		New: leaguerules.Known(next), Effective: leaguerules.Effective{Kind: leaguerules.Now},
		RecordedAt: at}, nil
}

func windowLabel(r Record) string {
	w := r.Command.Args.Window
	switch r.Command.Command {
	case WindowSet:
		span := "opens " + w.Start.words()
		if w.End != nil {
			span = "runs " + w.Start.words() + " to " + w.End.words()
		}
		return fmt.Sprintf("Set the %s window: %s, for %s", w.ID, span, joinActions(w.Opens))
	case WindowMove:
		return fmt.Sprintf("Moved the %s window to open on %s", w.ID, w.Start.words())
	default:
		return fmt.Sprintf("Removed the %s window", w.ID)
	}
}

func (d RawDate) words() string {
	return time.Date(2000, time.Month(d.Month), d.Day, 0, 0, 0, 0, time.UTC).Format("Jan 2")
}

func joinActions(actions []leaguerules.Action) string {
	names := make([]string, len(actions))
	for i, a := range actions {
		names[i] = string(a)
	}
	return strings.Join(names, ", ")
}

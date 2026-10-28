package commissioner

import (
	"fmt"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/leagueclock"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

const CalendarException = "commish.calendar.exception"

type RawWindowException struct {
	Action    leaguerules.Action `json:"action"`
	Franchise string             `json:"franchise"`
	Until     time.Time          `json:"until"`
}

func (a RawArgs) validateException() error {
	if a.Exception == nil || a.Name != "" || a.Entry != "" || a.People != nil || a.Field != "" ||
		len(a.Value) != 0 || a.Preset != "" || a.FranchiseCount != 0 || len(a.Seed) != 0 || len(a.Franchises) != 0 {
		return fmt.Errorf("send only the action, franchise and expiry of this window exception")
	}
	r := a.Exception
	if strings.TrimSpace(string(r.Action)) == "" || strings.TrimSpace(r.Franchise) == "" || r.Until.IsZero() {
		return fmt.Errorf("choose one action, one franchise and an expiry")
	}
	return nil
}

func CheckCalendarAppend(raw RawCommand, records []Record, at time.Time) error {
	if raw.Command != CalendarException {
		return nil
	}
	r := raw.Args.Exception
	if !r.Until.After(at) {
		return Refused(fmt.Errorf("the window exception must expire in the future"))
	}
	if len(records) == 0 {
		return Refused(fmt.Errorf("create the league before allowing an exception"))
	}
	if !leagueFranchise(records, r.Franchise) {
		return Refused(fmt.Errorf("choose a franchise in this league"))
	}
	rules, err := resolveRecords(records)
	if err != nil {
		return err
	}
	if !calendarAction(rules, r.Action) {
		return Refused(fmt.Errorf("choose an action opened by a league window, or trade"))
	}
	return nil
}

func leagueFranchise(records []Record, franchise string) bool {
	for _, id := range LeagueFranchises(records) {
		if id == franchise {
			return true
		}
	}
	return false
}

func calendarAction(rules leaguerules.RuleSet, action leaguerules.Action) bool {
	if action == "trade" {
		return true
	}
	for _, window := range rules.Calendar.Windows.Value {
		for _, opens := range window.Opens.Value {
			if opens == action {
				return true
			}
		}
	}
	return false
}

type Calendar struct {
	League   string                     `json:"league"`
	Revision int64                      `json:"revision"`
	Clock    leagueclock.ActionCalendar `json:"clock"`
	Decision leagueclock.ActionDecision `json:"decision"`
}

// CalendarReading replays the same rule changes and compensations as the rules desk.
func CalendarReading(league string, records []Record, at time.Time, year int,
	action leaguerules.Action, franchise string, local *time.Location) (Calendar, error) {
	if len(records) == 0 {
		return Calendar{}, fmt.Errorf("create this app league before reading its calendar")
	}
	if local == nil || at.IsZero() || year < 1900 || year > 9999 {
		return Calendar{}, fmt.Errorf("choose a season, reading time and owner time zone")
	}
	if !leagueFranchise(records, franchise) {
		return Calendar{}, fmt.Errorf("choose a franchise in this league")
	}
	rules, err := resolveRecords(records)
	if err != nil {
		return Calendar{}, err
	}
	exceptions, err := readCalendarExceptions(records)
	if err != nil {
		return Calendar{}, err
	}

	clock, err := leagueclock.Calendar(at, year, rules.Calendar, exceptions)
	if err != nil {
		return Calendar{}, fmt.Errorf("read league calendar: %w", err)
	}
	projectDraftCalendar(&clock, records, year, at)
	decision := clock.Decide(at, action, franchise)
	if decision.Next != nil {
		next := decision.Next.In(local)
		decision.Next = &next
		decision.Reason = fmt.Sprintf("This action opens with the %s window on %s.", decision.Window,
			next.Format("Jan 2, 2006 at 3:04 PM MST"))
	}
	clock.Localize(local)
	return Calendar{League: league, Revision: int64(len(records)), Clock: clock, Decision: decision}, nil
}

func readCalendarExceptions(records []Record) ([]leagueclock.WindowException, error) {
	active := Replay(records).Active
	exceptions := []leagueclock.WindowException{}
	for _, r := range records {
		if !active[r.Receipt.Entry] {
			continue
		}
		if r.Command.Command == CalendarException {
			e := r.Command.Args.Exception
			exceptions = append(exceptions, leagueclock.WindowException{Action: e.Action, Franchise: e.Franchise,
				Start: r.Receipt.RecordedAt, Until: e.Until, Reason: r.Command.Reason})
		}
		if r.Command.Command != RuleException {
			continue
		}
		c, err := decodeCorrection(r.Command.Args, r.Command.Command)
		if err != nil {
			return nil, err
		}
		if !strings.HasPrefix(c.Rule, "calendar.") {
			continue
		}
		e := leagueclock.WindowException{Action: leaguerules.Action(strings.TrimPrefix(c.Rule, "calendar.")),
			Franchise: c.Scope.Franchise, Start: r.Receipt.RecordedAt, Permanent: c.Until == "", Reason: "Commissioner exception: " + r.Command.Reason}
		if c.Until != "" {
			if e.Until, err = time.Parse(time.RFC3339Nano, c.Until); err != nil {
				return nil, fmt.Errorf("read calendar exception expiry: %w", err)
			}
		}
		exceptions = append(exceptions, e)
	}
	return exceptions, nil
}

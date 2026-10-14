package commissioner

import (
	"fmt"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

// Permit is the gate's yes for one franchise and one action. Only NativeActionGate mints one, so a native
// owner action that takes a Permit cannot be started without having asked the calendar.
type Permit struct {
	league, franchise string
	action            leaguerules.Action
}

func (p Permit) League() string             { return p.league }
func (p Permit) Franchise() string          { return p.franchise }
func (p Permit) Action() leaguerules.Action { return p.action }

// Gate is the calendar's answer for one native owner action. When it is closed, Sentence says which window
// opens the action and when, in the owner's local time.
type Gate struct {
	Open     bool   `json:"open"`
	Sentence string `json:"sentence"`
	permit   Permit
}

// Permit is available only when the gate is open.
func (g Gate) Permit() (Permit, bool) { return g.permit, g.Open }

// NativeActionGate decides whether a native league's owner action is open for one franchise at one instant.
// Every native owner action asks it, through ActionCalendar.Decide, so window exceptions, the exact trade
// deadline and the commissioner's moved windows govern all of them alike.
func NativeActionGate(league string, records []Record, franchise string, action leaguerules.Action,
	at time.Time, local *time.Location) (Gate, error) {
	if len(records) == 0 {
		return Gate{}, fmt.Errorf("create this app league before acting in it")
	}
	rules, err := resolveRecords(records)
	if err != nil {
		return Gate{}, err
	}
	if rules.Calendar.Timezone.State != leaguerules.IsKnown {
		return Gate{}, fmt.Errorf("the league time zone needs a ruling before actions can be dated")
	}
	zone, err := time.LoadLocation(rules.Calendar.Timezone.Value)
	if err != nil {
		return Gate{}, fmt.Errorf("read league time zone: %w", err)
	}
	reading, err := CalendarReading(league, records, at, at.In(zone).Year(), action, franchise, local)
	if err != nil {
		return Gate{}, err
	}
	gate := Gate{Open: reading.Decision.Open, Sentence: reading.Decision.Reason}
	if gate.Open {
		gate.permit = Permit{league: league, franchise: franchise, action: action}
		return gate, nil
	}
	if deadline := reading.Clock.TradeDeadline; action == "trade" && deadline != nil && !at.Before(*deadline) {
		gate.Sentence = "The trade deadline passed on " + deadline.In(local).Format("Jan 2, 2006 at 3:04 PM MST") + "."
	}
	return gate, nil
}

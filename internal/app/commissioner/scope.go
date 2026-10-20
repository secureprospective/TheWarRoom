package commissioner

import (
	"errors"
	"fmt"
	"strings"
	"time"

	commish "github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

// CalendarScope is the twin of the root RawLeagueCalendarScope: the same fields in the same order.
type CalendarScope struct {
	League    string             `json:"league"`
	Season    int                `json:"season"`
	LocalZone string             `json:"localZone"`
	Franchise string             `json:"franchise"`
	Action    leaguerules.Action `json:"action"`
}

func (r CalendarScope) Validate() error {
	if AppLeague(r.League) != nil || r.Season < 1900 || r.Season > 9999 ||
		strings.TrimSpace(r.Franchise) == "" || strings.TrimSpace(string(r.Action)) == "" {
		return fmt.Errorf("choose an app league, season, franchise and action")
	}
	if r.LocalZone == "" || r.LocalZone == "Local" {
		return fmt.Errorf("choose your local time zone")
	}
	if _, err := time.LoadLocation(r.LocalZone); err != nil {
		return fmt.Errorf("read owner time zone: %w", err)
	}
	return nil
}

// ScheduleScope is the twin of the root RawLeagueScheduleScope.
type ScheduleScope struct {
	League string `json:"league"`
}

func (r ScheduleScope) Validate() error { return AppLeague(r.League) }

// AppLeague admits an app league id: "native." and a name, with no surrounding space.
func AppLeague(league string) error {
	if !strings.HasPrefix(league, "native.") || league == "native." || strings.TrimSpace(league) != league {
		return fmt.Errorf("choose an app league")
	}
	return nil
}

// Blocked is the preview of a body that could not be read: no fields, and the reason as its one blocker.
func Blocked(err error) commish.Preview {
	var refusal *commish.Refusal
	err = commish.Refused(err)
	words := err.Error()
	if errors.As(err, &refusal) {
		words = refusal.Reason()
	}
	return commish.Preview{Fields: []commish.PreviewField{},
		Affects:  commish.PreviewAffects{Readings: []string{}, Words: "Not known yet"},
		Blockers: []string{words}}
}

// ChangeError words a recorded change's failure: a refusal reads as its reason alone, without the
// context each layer adds for the log; anything else keeps the "league change" context.
func ChangeError(err error) error {
	var refused *commish.Refusal
	if errors.As(err, &refused) {
		return errors.New(refused.Reason())
	}
	if err != nil {
		return fmt.Errorf("league change: %w", err)
	}
	return nil
}

package accounts

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode"
)

// MaxAccountJobSnoozes bounds the number of active private reminders per account.
const MaxAccountJobSnoozes = 256

// RawJobSnoozesRead carries no identity; the hosted session supplies the account.
type RawJobSnoozesRead struct{}

func (RawJobSnoozesRead) Validate() error { return nil }

// RawJobSnoozeSet is one account's private reminder for a league job.
type RawJobSnoozeSet struct {
	League string `json:"league"`
	JobID  string `json:"jobId"`
	Until  string `json:"until"`
}

func (r RawJobSnoozeSet) Validate() error {
	if err := validateJobSnoozeKey(r.League, r.JobID); err != nil {
		return err
	}
	if _, err := time.Parse(time.RFC3339Nano, r.Until); err != nil {
		return fmt.Errorf("snooze time must be RFC 3339: %w", err)
	}
	return nil
}

// RawJobSnoozeWake removes only the signed-in account's mark for one job.
type RawJobSnoozeWake struct {
	League string `json:"league"`
	JobID  string `json:"jobId"`
}

func (r RawJobSnoozeWake) Validate() error { return validateJobSnoozeKey(r.League, r.JobID) }

func validateJobSnoozeKey(league, jobID string) error {
	if !validPreferenceKey(league, 128) || !validPreferenceKey(jobID, 256) {
		return fmt.Errorf("choose a valid league and job")
	}
	return nil
}

func validPreferenceKey(value string, maxBytes int) bool {
	if value == "" || len(value) > maxBytes || strings.TrimSpace(value) != value {
		return false
	}
	for _, r := range value {
		if unicode.IsControl(r) {
			return false
		}
	}
	return true
}

// JobSnooze is private to one account; league plus job ID avoids cross-league collisions.
type JobSnooze struct {
	League string `json:"league"`
	JobID  string `json:"jobId"`
	Until  string `json:"until"`
}

// JobSnoozesState is the signed-in account's complete snooze list.
type JobSnoozesState struct {
	Snoozes []JobSnooze `json:"snoozes"`
}

// JobSnoozeRepository is the account-scoped store seam for private job reminders.
type JobSnoozeRepository interface {
	ReadJobSnoozes(context.Context, string) (JobSnoozesState, error)
	SetJobSnooze(context.Context, string, RawJobSnoozeSet) (JobSnoozesState, error)
	WakeJobSnooze(context.Context, string, RawJobSnoozeWake) (JobSnoozesState, error)
}

// PreferenceRepository is the server-owned account settings port for hosted workflows.
type PreferenceRepository interface {
	HomeLayoutRepository
	JobSnoozeRepository
}

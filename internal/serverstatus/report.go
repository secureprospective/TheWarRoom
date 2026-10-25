// Package serverstatus is the league server's private health dashboard: a plain-English
// report printed at the CT console, which only the LAN/VPN-restricted Proxmox management
// plane reaches. Nothing here is routed through the Tunnel, and nothing sends a message:
// the commissioner or deputy looks, following deploy/server/CHECKLIST.md.
package serverstatus

import (
	"fmt"
	"io"
	"time"
)

// State is how one line reads on the dashboard.
type State string

const (
	OK      State = "ok"
	Problem State = "problem"
	// Note records something that happened, such as a restore; it needs no action.
	Note State = "note"
)

// Check is one dashboard line. Detail may carry private operator facts (backup errors,
// times, counts); that is why the report is only reachable from inside the CT.
type Check struct {
	Name   string `json:"name"`
	State  State  `json:"state"`
	Detail string `json:"detail"`
	Action string `json:"action,omitempty"`
}

// Report is the private status socket's reply.
type Report struct {
	Version int     `json:"version"`
	Checks  []Check `json:"checks"`
}

const reportVersion = 1

// Render prints the dashboard and returns the number of problems on it.
func Render(w io.Writer, at time.Time, checks []Check) (int, error) {
	problems := 0
	if _, err := fmt.Fprintf(w, "League server status, %s\n\n", at.UTC().Format("2006-01-02 15:04 UTC")); err != nil {
		return 0, fmt.Errorf("print status: %w", err)
	}
	for _, check := range checks {
		label := "OK"
		switch check.State {
		case Problem:
			label = "PROBLEM"
			problems++
		case Note:
			label = "NOTE"
		case OK:
		}
		if _, err := fmt.Fprintf(w, "%-8s %s: %s\n", label, check.Name, check.Detail); err != nil {
			return 0, fmt.Errorf("print status: %w", err)
		}
		if check.State == Problem && check.Action != "" {
			if _, err := fmt.Fprintf(w, "%-8s What to do: %s\n", "", check.Action); err != nil {
				return 0, fmt.Errorf("print status: %w", err)
			}
		}
	}
	summary := "\nEverything is working. Nothing to do.\n"
	if problems == 1 {
		summary = "\n1 problem. Do what it says under it.\n"
	} else if problems > 1 {
		summary = fmt.Sprintf("\n%d problems. Deal with them from the top down.\n", problems)
	}
	if _, err := io.WriteString(w, summary); err != nil {
		return 0, fmt.Errorf("print status: %w", err)
	}
	return problems, nil
}

// when reads "2026-10-25 14:05 UTC (5 minutes ago)".
func when(t, now time.Time) string {
	if t.IsZero() {
		return "never"
	}
	stamp := t.UTC().Format("2006-01-02 15:04 UTC")
	ago := now.Sub(t)
	switch {
	case ago < 0:
		return stamp + " (in the future: check the clock)"
	case ago < 2*time.Minute:
		return stamp + " (just now)"
	case ago < 2*time.Hour:
		return fmt.Sprintf("%s (%d minutes ago)", stamp, int(ago.Minutes()))
	case ago < 48*time.Hour:
		return fmt.Sprintf("%s (%d hours ago)", stamp, int(ago.Hours()))
	}
	return fmt.Sprintf("%s (%d days ago)", stamp, int(ago.Hours()/24))
}

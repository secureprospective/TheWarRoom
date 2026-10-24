package leagueservice

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// HostClock rereads systemd's kernel NTP status on each admission; nothing is cached.
// Hosts without timedatectl (including non-systemd hosts) remain untrusted.
type HostClock struct{ synchronized func() (bool, error) }

func NewHostClock() HostClock    { return HostClock{synchronized: hostSynchronized} }
func (HostClock) Now() time.Time { return time.Now() }
func (c HostClock) Trusted() error {
	if c.synchronized == nil {
		return fmt.Errorf("clock synchronization source unavailable")
	}
	ok, err := c.synchronized()
	if err != nil {
		return fmt.Errorf("read clock synchronization: %w", err)
	}
	if !ok {
		return fmt.Errorf("host clock is not NTP synchronized")
	}
	return nil
}

type RawClockStatus struct{ Synchronized string }

func (r RawClockStatus) Validate() error {
	switch r.Synchronized {
	case "yes", "no":
		return nil
	default:
		return fmt.Errorf("invalid NTP synchronization status")
	}
}

func hostSynchronized() (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	// No shell, caller arguments or localized human-readable status output.
	output, err := exec.CommandContext(ctx, "timedatectl", "show", "--property=NTPSynchronized", "--value").Output()
	if err != nil {
		return false, fmt.Errorf("query system clock health: %w", err)
	}
	raw := RawClockStatus{Synchronized: strings.TrimSpace(string(output))}
	if err := raw.Validate(); err != nil {
		return false, fmt.Errorf("validate system clock health: %w", err)
	}
	return raw.Synchronized == "yes", nil
}

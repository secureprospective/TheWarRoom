//go:build linux

package leagueservice

import (
	"fmt"

	"golang.org/x/sys/unix"
)

// kernelSynchronized reads adjtimex with zero Modes, which changes nothing and needs no
// privilege. It works with any NTP client (chrony, ntpd, timesyncd) and any init system.
func kernelSynchronized() (bool, error) {
	var status unix.Timex
	state, err := unix.Adjtimex(&status)
	if err != nil {
		return false, fmt.Errorf("read kernel clock status: %w", err)
	}
	return kernelClockSynchronized(state, status.Status), nil
}

func kernelClockSynchronized(state int, status int32) bool {
	return state != unix.TIME_ERROR && status&unix.STA_UNSYNC == 0
}

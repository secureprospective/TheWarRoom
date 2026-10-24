//go:build !linux

package leagueservice

import "fmt"

// kernelSynchronized fails closed: hosted mode trusts only the Linux kernel's NTP state.
func kernelSynchronized() (bool, error) {
	return false, fmt.Errorf("kernel clock status is unavailable on this platform")
}

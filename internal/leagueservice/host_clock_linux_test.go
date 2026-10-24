//go:build linux

package leagueservice

import (
	"testing"

	"golang.org/x/sys/unix"
)

func TestKernelClockSynchronized(t *testing.T) {
	for _, tc := range []struct {
		state  int
		status int32
		want   bool
	}{
		{unix.TIME_OK, 0, true},
		{unix.TIME_INS, unix.STA_PLL, true},
		{unix.TIME_OK, unix.STA_UNSYNC, false},
		{unix.TIME_OK, unix.STA_PLL | unix.STA_UNSYNC, false},
		{unix.TIME_ERROR, 0, false},
	} {
		if got := kernelClockSynchronized(tc.state, tc.status); got != tc.want {
			t.Fatalf("state %d status %#x: got %v", tc.state, tc.status, got)
		}
	}
}

// The unprivileged read-only call must succeed under the test user, or every hosted write fails.
func TestKernelSynchronizedReads(t *testing.T) {
	if _, err := kernelSynchronized(); err != nil {
		t.Fatal(err)
	}
}

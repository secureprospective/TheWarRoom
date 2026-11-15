//go:build linux

package leagueservice

import (
	"testing"
)

// The unprivileged read-only call must succeed under the test user, or every hosted write fails.
func TestKernelSynchronizedReads(t *testing.T) {
	if _, err := kernelSynchronized(); err != nil {
		t.Fatal(err)
	}
}

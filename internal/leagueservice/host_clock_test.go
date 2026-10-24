package leagueservice

import (
	"fmt"
	"testing"
)

func TestHostClockSynchronization(t *testing.T) {
	synced := false
	var failure error
	clock := HostClock{synchronized: func() (bool, error) { return synced, failure }}
	if clock.Trusted() == nil {
		t.Fatal("unsynchronized clock trusted")
	}
	synced = true
	if err := clock.Trusted(); err != nil {
		t.Fatal(err)
	}
	synced = false
	if clock.Trusted() == nil {
		t.Fatal("lost synchronization trusted")
	}
	synced, failure = true, fmt.Errorf("status unavailable")
	if clock.Trusted() == nil {
		t.Fatal("failed source trusted")
	}
	if (HostClock{}).Trusted() == nil {
		t.Fatal("missing source trusted")
	}
}

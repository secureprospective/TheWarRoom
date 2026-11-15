package main

import (
	"testing"
	"time"
)

func TestLeagueTimeZonesLoadWithoutSystemZoneinfo(t *testing.T) {
	t.Setenv("ZONEINFO", "/nonexistent")
	for _, zone := range []string{"America/New_York", "America/Chicago", "America/Los_Angeles"} {
		if _, err := time.LoadLocation(zone); err != nil {
			t.Fatalf("%s: %v", zone, err)
		}
	}
}

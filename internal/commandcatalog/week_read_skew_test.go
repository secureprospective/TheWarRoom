package catalog

import (
	"testing"
	"time"
)

// A device clock a little ahead of the server still reads the week; a time well in the future does not.
func TestWeekReadAllowsSmallDeviceClockSkew(t *testing.T) {
	scope := func(at time.Time) RawWorkflowScope {
		return RawWorkflowScope{League: "native.test", Franchise: "franchise.01", Season: 2026,
			LocalZone: "America/Chicago", At: at.UTC().Format(time.RFC3339Nano)}
	}
	if err := scope(time.Now().Add(30 * time.Second)).validateCommand("week.read"); err != nil {
		t.Fatalf("a phone 30 s fast was refused: %v", err)
	}
	if err := scope(time.Now().Add(time.Hour)).validateCommand("week.read"); err == nil {
		t.Fatal("a time an hour ahead was accepted")
	}
}

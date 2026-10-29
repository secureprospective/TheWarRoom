package research

import (
	"strings"
	"testing"
	"time"
)

func TestHeldFeedFailuresNeverExposeSourceErrors(t *testing.T) {
	for _, raw := range []string{"playerScores: week mismatch", "backoff: request cancelled", "context deadline exceeded"} {
		for _, at := range []time.Time{{}, time.Now()} {
			for _, got := range []string{HeldTime(at, raw, time.UTC), newsHeldTime(at, raw, time.UTC)} {
				if strings.Contains(got, raw) || got == "" {
					t.Fatalf("raw failure escaped: %s", got)
				}
			}
		}
	}
}

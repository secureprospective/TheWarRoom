package research

import (
	"strings"
	"sync"
	"testing"
	"time"
)

func TestHeldPublicationRacesDetachedReaders(t *testing.T) {
	h := &Held{}
	h.Publish(heldFixture())
	var workers sync.WaitGroup
	for worker := range 5 {
		workers.Go(func() {
			for range 200 {
				if worker == 0 {
					h.Publish(heldFixture())
					continue
				}
				out := h.Snapshot()
				if out.Feeds["populated"].Value[0].Players[0].Value != 1 || out.News["populated"].Value[0].Headline != "fixture" {
					t.Error("torn publication")
				}
				mutateHeld(out)
			}
		})
	}
	workers.Wait()
}

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

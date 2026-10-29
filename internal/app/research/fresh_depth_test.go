package research

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/researchfeeds"
)

func TestFreshDepthDoesNotWaitForMFLOrIdentityLinks(t *testing.T) {
	in := realResearch(t)
	rows := in.Held.Depth.Value
	in.Held = Values{}
	steps := RefreshSteps{
		Week: func() int { return 6 }, Status: func(string) {},
		Pause: func(context.Context, time.Duration) error { return context.DeadlineExceeded },
		Depth: func(ctx context.Context, force bool) {
			if deadline, ok := ctx.Deadline(); !ok || time.Until(deadline) > 30*time.Second {
				t.Fatal("unbounded depth read")
			}
			RefreshDepth(ctx, in.Held.Depth, force, func(context.Context) ([]researchfeeds.RawDepth, domain.Freshness, error) {
				return rows, domain.Freshness{State: domain.FreshLive, FetchedAt: time.Now().UTC().Format(time.RFC3339)}, nil
			}, func(context.Context) (map[string]string, error) {
				return nil, errors.New("crosswalk: context deadline exceeded")
			},
				func(feed Feed[[]researchfeeds.RawDepth], links map[string]string, err error) {
					PublishDepth(&in.Held, feed, links, err)
				})
		},
	}
	if err := steps.Run(t.Context(), true); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatal(err)
	}
	grid := reportByID(t, Build(in, Scope{Depth: "grid"}).Board, "M-048")
	if len(grid.Rows) != 32 {
		t.Fatalf("fresh grid has %d teams", len(grid.Rows))
	}
	if strings.Contains(grid.Note, "context deadline") {
		t.Fatal(grid.Note)
	}
}

func TestFreshDepthEmptyStateIsOnePlainSentence(t *testing.T) {
	in := realResearch(t)
	in.Held.Depth = Feed[[]researchfeeds.RawDepth]{RefreshError: "context deadline exceeded"}
	in.Held.LinkError = "crosswalk: request cancelled"
	for _, mode := range []string{"grid", "team"} {
		grid := reportByID(t, Build(in, Scope{Depth: mode}).Board, "M-048")
		if grid.Note != "NFL starters are not available yet; refresh from sources." || len(grid.Rows) != 0 {
			t.Fatal(grid)
		}
	}
}

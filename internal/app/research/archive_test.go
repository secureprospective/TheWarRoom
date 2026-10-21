package research

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/parquet-go/parquet-go"
	"github.com/secureprospective/TheWarRoom/internal/app/research/researchtest"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/researchfeeds"
)

func TestArchivedFeedMatchesSeasonAndPreservesProvenance(t *testing.T) {
	t.Parallel()
	at := time.Date(2026, 10, 11, 10, 0, 0, 0, time.UTC)
	body := researchtest.Sample(t, "playerScores_W01.json")
	bodies := func(_ context.Context, part string, accept func(string, []byte, time.Time) bool) (bool, error) {
		if part != "TYPE=playerScores" {
			t.Fatal(part)
		}
		if accept("/2025/export?TYPE=playerScores&L=14432", body, at) {
			t.Fatal("wrong season accepted")
		}
		if accept("/2026/export?TYPE=playerScores&L=14432", []byte(`{}`), at) {
			t.Fatal("invalid body accepted")
		}
		return accept("/2026/export?TYPE=playerScores&L=14432", body, at), nil
	}
	held := ReadArchivedFeed(t.Context(), bodies, 2026, "playerScores", "1")
	if !held.FetchedAt.Equal(at) || held.RefreshError != "Archived copy; refresh pending" || len(held.Value) != 1 {
		t.Fatal(held)
	}
	held = ReadArchivedFeed(t.Context(), noNewsArchive, 2026, "playerScores", "1")
	if held.RefreshError != "No archived copy held" {
		t.Fatal(held)
	}
	broken := func(context.Context, string, func(string, []byte, time.Time) bool) (bool, error) {
		return false, errors.New("fixture unreadable")
	}
	if got := ReadArchivedFeed(t.Context(), broken, 2026, "playerScores", "1"); got.RefreshError != "Archive unreadable: fixture unreadable" {
		t.Fatal(got)
	}
}

func TestDepthArchiveExactURLAndFallbackErrors(t *testing.T) {
	t.Parallel()
	in := realResearch(t)
	var body bytes.Buffer
	if err := parquet.Write(&body, in.Held.Depth.Value); err != nil {
		t.Fatal(err)
	}
	offline := errors.New("fixture offline")
	bodies := func(ctx context.Context, part string, accept func(string, []byte, time.Time) bool) (bool, error) {
		deadline, ok := ctx.Deadline()
		if !ok || time.Until(deadline) > 10*time.Second || part != researchfeeds.DepthURL(2026) {
			t.Fatal("fallback scope or timeout changed", part, deadline)
		}
		if accept(researchfeeds.DepthURL(2025), body.Bytes(), in.Now) {
			t.Fatal("wrong depth season accepted")
		}
		if accept(part, []byte("invalid parquet"), in.Now) {
			t.Fatal("invalid depth accepted")
		}
		return accept(part, body.Bytes(), in.Now), nil
	}
	rows, fresh, err := ReadDepthArchive(t.Context(), 2026, bodies, offline)
	if err != nil || len(rows) != len(in.Held.Depth.Value) || fresh.State != domain.FreshStale {
		t.Fatal(len(rows), fresh, err)
	}
	if _, _, err := ReadDepthArchive(t.Context(), 2026, noNewsArchive, offline); !errors.Is(err, offline) {
		t.Fatal("missing archive lost original cause", err)
	}
	broken := func(context.Context, string, func(string, []byte, time.Time) bool) (bool, error) {
		return false, offline
	}
	if _, _, err := ReadDepthArchive(t.Context(), 2026, broken, offline); !errors.Is(err, offline) || err.Error() != "depth chart archive: fixture offline" {
		t.Fatal(err)
	}
}

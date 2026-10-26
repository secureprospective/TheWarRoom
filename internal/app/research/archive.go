package research

import (
	"context"
	"fmt"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/report"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/researchfeeds"
)

func ReadArchivedFeed(
	ctx context.Context, bodies ArchivedBodies, season int, kind, week string,
) Feed[[]researchfeeds.Feed] {
	h := Feed[[]researchfeeds.Feed]{}
	found, err := bodies(ctx, "TYPE="+kind, func(src string, body []byte, at time.Time) bool {
		if !researchExport(src, kind, season) {
			return false
		}
		feed, err := researchfeeds.Parse(body, kind, week)
		if err != nil {
			return false
		}
		h.Value, h.FetchedAt, h.RefreshError = []researchfeeds.Feed{feed}, at, "Archived copy; refresh pending"
		return true
	})
	if err != nil {
		h.RefreshError = "Archive unreadable: " + err.Error()
	}
	if !found && err == nil {
		h.RefreshError = "No archived copy held"
	}
	return h
}

func researchExport(src, kind string, season int) bool {
	if researchfeeds.IsTrend(kind) {
		return ingestion.SameLeaguelessExport(src, kind, season)
	}
	return ingestion.SameExport(src, kind, season)
}

func ReadDepthArchive(ctx context.Context, season int, bodies ArchivedBodies, fetchErr error) ([]researchfeeds.RawDepth, domain.Freshness, error) {
	rctx, cancel := context.WithTimeout(ctx, report.FallbackTimeout)
	defer cancel()
	var fresh domain.Freshness
	var rows []researchfeeds.RawDepth
	found, aerr := bodies(rctx, researchfeeds.DepthURL(season), func(src string, body []byte, at time.Time) bool {
		if src != researchfeeds.DepthURL(season) {
			return false
		}
		parsed, perr := researchfeeds.ParseDepth(body)
		if perr != nil {
			return false
		}
		rows = parsed
		fresh = report.StaleFreshness(at, fetchErr)
		return true
	})
	if aerr != nil {
		return nil, domain.Freshness{}, fmt.Errorf("depth chart archive: %w", aerr)
	}
	if !found {
		return nil, domain.Freshness{}, fmt.Errorf("depth charts: no archived copy: %w", fetchErr)
	}
	return rows, fresh, nil
}

// FirstArchiveRead marks a key before attempting its read, including absent archive copies.
// The caller owns serialization and publication of the map.
func FirstArchiveRead(reads *map[string]bool, key string) bool {
	if (*reads)[key] {
		return false
	}
	if *reads == nil {
		*reads = map[string]bool{}
	}
	(*reads)[key] = true
	return true
}

func HydrateFeed(feeds map[string]Feed[[]researchfeeds.Feed], kind, week string, first func(string) bool, read func(string, string) Feed[[]researchfeeds.Feed]) {
	key := Key(kind, week)
	held := feeds[key]
	if held.FetchedAt.IsZero() && first("feed:"+key) {
		archived := read(kind, week)
		if held.RefreshError != "" {
			archived.RefreshError = held.RefreshError
		}
		feeds[key] = archived
	}
}

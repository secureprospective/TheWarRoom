package report

import (
	"fmt"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
)

// FallbackTimeout bounds a fallback read of the fetch archive.
const FallbackTimeout = 10 * time.Second

// HeldFeed is a feed's last good copy and why the latest refresh failed, if it did.
type HeldFeed[T any] struct {
	Value        T
	FetchedAt    time.Time
	RefreshError string
	ReadPending  bool
}

// state is the feed's held read state, without its value.
func (h HeldFeed[T]) FeedState() Feed {
	return Feed{FetchedAt: h.FetchedAt, RefreshError: h.RefreshError, ReadPending: h.ReadPending}
}

func (h HeldFeed[T]) Provenance(source, feed string) snapshot.Provenance {
	fresh := domain.Freshness{State: domain.FreshFail, Note: feed + " not fetched yet"}
	if !h.FetchedAt.IsZero() {
		fresh = domain.Freshness{State: domain.FreshLive, FetchedAt: h.FetchedAt.UTC().Format(time.RFC3339)}
	}
	if h.ReadPending {
		fresh.State, fresh.Note = domain.FreshStale, PendingRead(h.FetchedAt, time.Local)
	}
	if h.RefreshError != "" {
		fresh.Note = h.RefreshError
		if !h.FetchedAt.IsZero() {
			fresh.State = domain.FreshStale
		}
	}
	return snapshot.Provenance{Kind: "live", Source: source, Freshness: fresh}
}

func HoldFeed[T any](h *HeldFeed[T], value T, fresh domain.Freshness, err error) {
	if err != nil {
		h.RefreshError = err.Error()
		return
	}
	at, parseErr := time.Parse(time.RFC3339, fresh.FetchedAt)
	if parseErr != nil {
		h.RefreshError = "season refresh time: " + parseErr.Error()
		return
	}
	h.Value, h.FetchedAt, h.RefreshError, h.ReadPending = value, at, fresh.Note, false
}

// LiveFreshness marks a result as served from a successful fetch at time t.
func LiveFreshness(t time.Time) domain.Freshness {
	return domain.Freshness{State: domain.FreshLive, FetchedAt: t.UTC().Format(time.RFC3339)}
}

// StaleFreshness marks a cached result after a failed fetch. The cause is shown so an MFL
// outage reads differently from a local network fault.
func StaleFreshness(fetchedAt time.Time, cause error) domain.Freshness {
	return domain.Freshness{
		State:     domain.FreshStale,
		FetchedAt: fetchedAt.UTC().Format(time.RFC3339),
		Note:      fmt.Sprintf("live fetch failed, showing last known data: %v", cause),
	}
}

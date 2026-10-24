package report

import (
	"errors"
	"fmt"
	"log"
	"net"
	"net/url"
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
		h.RefreshError = FailureText(err)
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

// Offline is what a person sees when a fetch never reached the site: no connection, no DNS, or a
// refused or timed-out connection. The error chain behind it goes to the log, not the screen.
const Offline = "Can't reach the league site. Check your internet connection, then try again."

// offlineError keeps the network cause for errors.Is and the log while reading as Offline.
type offlineError struct{ cause error }

func (e offlineError) Error() string { return Offline }
func (e offlineError) Unwrap() error { return e.cause }

// PlainFailure returns a network failure as an error reading Offline alone, logging its detail
// once; any other error is returned unchanged.
func PlainFailure(err error) error {
	var plain offlineError
	if err == nil || errors.As(err, &plain) || !unreachable(err) {
		return err
	}
	log.Printf("the war room: offline: %v", err)
	return offlineError{cause: err}
}

// FailureText is PlainFailure's text, for the held-feed notes a screen shows.
func FailureText(err error) string { return PlainFailure(err).Error() }

func unreachable(err error) bool {
	var dns *net.DNSError
	var op *net.OpError
	var fetch *url.Error
	return errors.As(err, &dns) || errors.As(err, &op) || (errors.As(err, &fetch) && fetch.Timeout())
}

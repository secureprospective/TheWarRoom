// Package sources builds Sources readings and runs synchronous imports with explicit dependencies.
// The caller owns held values, locking, scheduling and database lifetimes.
package sources

import (
	"context"
	"net/http"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/archive"
	"github.com/secureprospective/TheWarRoom/internal/lab"
	"github.com/secureprospective/TheWarRoom/internal/manifest"
	"github.com/secureprospective/TheWarRoom/internal/measures"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

// HealthHistory reads source standing and the latest engine timestamp.
type HealthHistory interface {
	Registry() *measures.Registry
	SourceHealth(context.Context) ([]history.SourceHealth, error)
	FetchHealth(context.Context, string) ([]history.URLHealth, error)
	LatestRun(context.Context, int, history.RunKind) (history.Run, bool, error)
}

// FactsHistory reads pinned inputs and records compilation attempts.
type FactsHistory interface {
	manifest.Bodies
	LatestManifest(context.Context, string) (history.ManifestRecord, bool, error)
	Snapshots(context.Context, string) ([]history.SnapshotRecord, error)
	HeldBodies(context.Context, []string) (map[string]bool, error)
	RecordSnapshot(context.Context, history.SnapshotRecord) error
}

// ArchiveHistory records imported files and publishes their pinned manifests.
type ArchiveHistory interface {
	archive.Sink
	SourceFetches(context.Context, string) ([]history.FetchRecord, error)
	RecordImport(context.Context, string, int, int) error
	PublishManifest(context.Context, history.ManifestRecord) (bool, error)
	LoadFailed(context.Context, string, error) error
}

// CrosswalkHistory links player identifiers and reads the resulting coverage.
type CrosswalkHistory interface {
	LinkPlayerIDs(context.Context, string, []history.PlayerIDLink) (int, error)
	LinkedPlayers(context.Context, string) (map[string]bool, error)
	LoadFailed(context.Context, string, error) error
}

// SignalsHistory reads coverage and ingests one mapped signal batch at a time.
type SignalsHistory interface {
	Registry() *measures.Registry
	LastLoadOf(context.Context, string) (time.Time, bool, error)
	KnownIDs(context.Context, string) (map[string]bool, error)
	MeasuresHeld(context.Context, string, int) (map[string]bool, error)
	ArchivedBodies(context.Context, string, func(string, []byte, time.Time) bool) (bool, error)
	Ingest(context.Context, measures.Batch) (history.LoadReport, error)
	LoadFailed(context.Context, string, error) error
	PlayersWithData(context.Context, string, int, []string) (map[string]bool, int, error)
}

// Service borrows stores and immutable inputs; it never owns App state.
type Service struct {
	Health     HealthHistory
	Facts      FactsHistory
	Archive    ArchiveHistory
	Crosswalk  CrosswalkHistory
	Signals    SignalsHistory
	Season     int
	CollegeKey string
	Transport  http.RoundTripper
	LabData    func() (*lab.Data, error)
}

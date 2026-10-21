package sources_test

import (
	"context"
	"errors"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/report"
	sourcesapp "github.com/secureprospective/TheWarRoom/internal/app/sources"
	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/headshot"
	"github.com/secureprospective/TheWarRoom/internal/lab"
	"github.com/secureprospective/TheWarRoom/internal/measures"
	"github.com/secureprospective/TheWarRoom/internal/normalize"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

type fakeHost struct {
	hist      *history.Store
	held      *sourcesapp.Held
	dir       string
	off       bool
	startup   error
	loading   sync.Mutex
	compiling sync.Mutex
	transport http.RoundTripper
	changed   func()
	cleared   []string
	clearErr  error
}

func testHost(t *testing.T) *fakeHost {
	t.Helper()
	ctx := context.Background()
	pools, err := db.Open(ctx, filepath.Join(t.TempDir(), "history.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := pools.Close(); err != nil {
			t.Error(err)
		}
	})
	reg, err := measures.Embedded()
	if err != nil {
		t.Fatal(err)
	}
	hist := history.New(pools, reg)
	if err := hist.Initialize(ctx); err != nil {
		t.Fatal(err)
	}
	return &fakeHost{hist: hist, held: &sourcesapp.Held{}}
}

func (h *fakeHost) api() *sourcesapp.API { return sourcesapp.New(h, h.held) }
func (h *fakeHost) Ready() error         { return h.startup }
func (h *fakeHost) CacheScopes() []sourcesapp.DataScope {
	return []sourcesapp.DataScope{{ID: "pool", Label: "Free agents"}}
}
func (h *fakeHost) ClearCache(scope string) (string, bool, error) {
	if scope != "pool" {
		return "", false, nil
	}
	h.cleared = append(h.cleared, scope)
	return "Free agents", true, h.clearErr
}
func (h *fakeHost) SourcesContext() context.Context { return context.Background() }
func (h *fakeHost) SourcesInputs() (*history.Store, int, http.RoundTripper, func() (*lab.Data, error)) {
	return h.hist, 2026, h.transport, lab.Load
}
func (h *fakeHost) SourcesSettings() (string, bool, error) { return h.dir, h.off, nil }
func (h *fakeHost) SaveSourcesSettings(dir string, off bool) error {
	h.dir, h.off = dir, off
	return nil
}
func (h *fakeHost) SourcesDatabaseChecks() *report.Report { return nil }
func (h *fakeHost) SourcesPhotos() (headshot.Stats, int, int64, bool, bool) {
	return headshot.Stats{}, 0, 0, false, false
}
func (h *fakeHost) SourcesPositions() map[string]string { return map[string]string{} }
func (h *fakeHost) SourcesDirectory(context.Context) (normalize.Lookup, error) {
	return normalize.Lookup{}, nil
}
func (h *fakeHost) SourcesRostered() map[string]string { return map[string]string{} }
func (h *fakeHost) SourcesNames() map[string]string    { return map[string]string{} }
func (h *fakeHost) SourcesZone() *time.Location        { return time.Local }
func (h *fakeHost) RefreshSourcesLeague(context.Context) (int, bool, bool, error) {
	return 2026, false, false, nil
}
func (h *fakeHost) SerializeSourcesLoad(_ context.Context, load func()) {
	defer func() {
		if h.changed != nil {
			h.changed()
		}
	}()
	h.loading.Lock()
	defer h.loading.Unlock()
	load()
}
func (h *fakeHost) SerializeSourcesCompile(compile func() (string, error)) (string, error) {
	if !h.compiling.TryLock() {
		return "The checked facts are already compiling.", nil
	}
	defer h.compiling.Unlock()
	return compile()
}

func TestCrosswalkReportKeepsLeadingZeroIDsAndGoWording(t *testing.T) {
	t.Parallel()
	h := testHost(t)
	if r := h.api().TargetCrosswalk(); r.Status != "empty" || !strings.Contains(r.Note, "No crosswalk loaded") {
		t.Fatalf("empty crosswalk = %+v", r)
	}
	h.held.PublishCrosswalk(sourcesapp.CrosswalkReport{OK: true, LoadedAt: "2026-10-11T18:00:00Z", Links: 1000, Promoted: 12,
		Rates:     []sourcesapp.CrosswalkRate{{Position: "QB", Rostered: 4, RosteredMatched: 3, FreeAgents: 0}},
		Total:     sourcesapp.CrosswalkRate{Position: "ALL", Rostered: 4, RosteredMatched: 3},
		Unmatched: []sourcesapp.CrosswalkMiss{{MFLID: "0042", Name: "Waiting", Team: "BUF", Position: "QB", Franchise: "0001", Reason: "no NFL id yet"}}})
	r := h.api().TargetCrosswalk()
	if r.Note != "1,000 links; 12 waiting facts resolved." || r.Rows[0].Cells[1] != "3 of 4 · 75.0%" ||
		r.Rows[0].Cells[2] != "None" || r.Rows[2].Cells[0] != "Waiting" || r.Rows[2].Cells[1] != "QB · BUF" {
		t.Fatalf("crosswalk = %+v", r)
	}
}

func TestEveryAPIDoorRefusesStartupBeforeEffects(t *testing.T) {
	h := &fakeHost{startup: errors.New("startup refused"), held: &sourcesapp.Held{}}
	api := h.api()
	if r := api.LoadCrosswalk(); r.Error != "startup refused" {
		t.Fatal(r)
	}
	if r := api.GetCrosswalkReport(); r.Error != "startup refused" {
		t.Fatal(r)
	}
	if r := api.TargetCrosswalk(); r.Note != "startup refused" {
		t.Fatal(r)
	}
	if r := api.SetLeagueArchiveDir("relative"); r.Error != "startup refused" {
		t.Fatal(r)
	}
	if r := api.TargetDataRefresh("unknown"); r.Error != "startup refused" {
		t.Fatal(r)
	}
	if r := api.TargetCacheClear("pool"); r.Error != "startup refused" || len(h.cleared) != 0 {
		t.Fatal(r, h.cleared)
	}
	if _, err := api.TargetDataSources(); err == nil || err.Error() != "data and sources: startup: startup refused" {
		t.Fatal(err)
	}
}

func TestCacheClearKeepsItsWordsAndRunsOnlyTheNamedCopy(t *testing.T) {
	h := &fakeHost{held: &sourcesapp.Held{}}
	api := h.api()
	if got := api.TargetCacheScopes(); len(got) != 1 || got[0].ID != "pool" {
		t.Fatal(got)
	}
	if r := api.TargetCacheClear("nope"); r.OK || r.Error != `unknown held copy "nope"` || len(h.cleared) != 0 {
		t.Fatal(r, h.cleared)
	}
	h.clearErr = errors.New("MFL did not answer")
	r := api.TargetCacheClear("pool")
	if r.OK || r.Error != "Free agents could not be read again: MFL did not answer. The copy held stays." {
		t.Fatal(r)
	}
	h.clearErr = nil
	r = api.TargetCacheClear("pool")
	if !r.OK || r.Summary != "Asked for a fresh copy of Free agents. Each screen shows how fresh what it holds is." {
		t.Fatal(r)
	}
	if strings.Join(h.cleared, ",") != "pool,pool" {
		t.Fatal(h.cleared)
	}
}

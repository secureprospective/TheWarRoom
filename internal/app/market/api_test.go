package market

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/market/markettest"
	"github.com/secureprospective/TheWarRoom/internal/app/report"
	"github.com/secureprospective/TheWarRoom/internal/app/season"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/marketfeeds"
	"github.com/secureprospective/TheWarRoom/internal/lab"
	"github.com/secureprospective/TheWarRoom/internal/normalize"
)

// testStamp is the stamp a test binary's root build label and date produce, as the goldens hold.
const testStamp = "dev "

var loadLab = sync.OnceValues(lab.Load) //nolint:gochecknoglobals // decode the frozen Lab facts once per test binary

type fakeHost struct {
	mu         sync.Mutex
	calls      []string
	readyErr   error
	labErr     error
	refreshErr error
	week       int
	now        OnField
	rostered   []string
	lookup     normalize.Lookup
	inputCtx   context.Context //nolint:containedctx // records the context the API derived
}

func (h *fakeHost) call(name string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.calls = append(h.calls, name)
}
func (h *fakeHost) Ready() error                   { h.call("ready"); return h.readyErr }
func (h *fakeHost) MarketContext() context.Context { h.call("context"); return context.Background() }
func (h *fakeHost) LabFacts() (*lab.Data, error) {
	h.call("lab")
	if h.labErr != nil {
		return nil, h.labErr
	}
	return loadLab()
}
func (h *fakeHost) LabStamp() string { h.call("stamp"); return testStamp }
func (h *fakeHost) AssetInput(ctx context.Context) (OnField, []string) {
	h.call("assets")
	h.inputCtx = ctx
	return h.now, h.rostered
}
func (h *fakeHost) MarketWeek() int { h.call("week"); return h.week }
func (h *fakeHost) CaptureMarket(snapshot func() Feeds) Feeds {
	h.call("capture")
	return snapshot()
}
func (h *fakeHost) MarketLookup() normalize.Lookup { h.call("lookup"); return h.lookup }
func (h *fakeHost) MarketZone() *time.Location     { h.call("zone"); return time.UTC }
func (h *fakeHost) RefreshMarket(bool) error       { h.call("refresh"); return h.refreshErr }

func unheld() OnField { return OnFieldNow(nil, 0, 0, time.Time{}, time.UTC) }

func checkGolden(t *testing.T, name string, v any) {
	t.Helper()
	got, err := json.MarshalIndent(v, "", " ")
	if err != nil {
		t.Fatal(err)
	}
	got = append(got, '\n')
	path := markettest.RepoPath("testdata", "wire", "market", name+".json")
	if os.Getenv("TWR_WRITE_WIRE") == "1" {
		if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, got, 0o644); err != nil { //nolint:gosec // a test fixture
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(want, got) {
		t.Errorf("%s differs from the market binding's output; the wire contract changed", path)
	}
}

// heldSample is the fixture feeds as the root publishes them, sampled to a dozen players.
func heldSample(t *testing.T) Feeds {
	t.Helper()
	feeds := Feeds{}
	for kind, feed := range markettest.Feeds(t) {
		feed.Players = feed.Players[:12]
		feeds[kind] = season.Feed[[]marketfeeds.Feed]{Value: []marketfeeds.Feed{feed}, FetchedAt: markettest.Read()}
	}
	return feeds
}

// TestMarketWireMatchesGolden freezes every Market binding against testdata/wire/market. The Lab
// surfaces read the frozen Lab facts; the held reports build in UTC.
func TestMarketWireMatchesGolden(t *testing.T) {
	t.Parallel()
	fresh := New(&fakeHost{now: unheld(), rostered: []string{"0042"}}, &Held{})
	reports, err := fresh.TargetMarketReports()
	if err != nil {
		t.Fatal(err)
	}
	checkGolden(t, "reports-unfetched", reports)
	held := HeldFeeds(heldSample(t))
	names := map[string]string{"16162": "Bijan Robinson", "16161": "Jahmyr Gibbs", "13589": "Josh Allen"}
	checkGolden(t, "reports-held", Reports(held, names, 6, time.UTC))
	empty, err := marketfeeds.Parse(marketfeeds.ADP, []byte(`{"adp":{"totalDrafts":"0","totalPicks":"0"}}`))
	if err != nil {
		t.Fatal(err)
	}
	adp := held[marketfeeds.ADP]
	adp.Value, adp.Feed.RefreshError = []marketfeeds.Feed{empty}, "offline"
	held[marketfeeds.ADP] = adp
	held[marketfeeds.Ranks] = HeldFeed{Feed: report.Feed{RefreshError: "offline"}}
	checkGolden(t, "reports-failed", Reports(held, nil, 0, time.UTC))
	board, err := fresh.TargetAssetBoard()
	if err != nil {
		t.Fatal(err)
	}
	checkGolden(t, "assets-unheld", board)

	full := RawMarketScope{From: 2017, To: 2026}
	thin := RawMarketScope{From: 2100, To: 2101}
	invalid := RawMarketScope{From: 2026, To: 2017}
	filtered := RawMarketScope{From: 2017, To: 2026, Phases: []string{"winter", "deadline"}, Assets: []string{"LB", "pick1:next"}, ByAge: true}
	for name, scope := range map[string]RawMarketScope{"full": full, "thin": thin, "invalid": invalid, "filtered": filtered} {
		checkGolden(t, "market-"+name, fresh.TargetMarket(scope))
		checkGolden(t, "lists-"+name, fresh.TargetMarketLists(scope))
		checkGolden(t, "clock-"+name, fresh.TargetPickClock(scope))
		checkGolden(t, "stretches-"+name, fresh.TargetMarketStretches(scope))
	}
	for name, raw := range map[string]RawMarketAsset{
		"position": {Scope: full, Kind: "LB"},
		"anchor":   {Scope: full, Kind: lab.Anchor},
		"age":      {Scope: RawMarketScope{From: 2017, To: 2026, ByAge: true}, Kind: "WR:prime"},
		"thin":     {Scope: thin, Kind: "LB"},
		"invalid":  {Scope: full, Kind: "LB:prime"},
	} {
		checkGolden(t, "profile-"+name, fresh.TargetAssetProfile(raw))
	}
	broken := New(&fakeHost{labErr: errors.New("fingerprint mismatch")}, &Held{})
	checkGolden(t, "lists-unloaded", broken.TargetMarketLists(full))
	checkGolden(t, "profile-unloaded", broken.TargetAssetProfile(RawMarketAsset{Scope: full, Kind: "LB"}))
}

func TestTargetMarketReading(t *testing.T) {
	t.Parallel()
	a := New(&fakeHost{}, &Held{})
	r := a.TargetMarket(RawMarketScope{From: 2017, To: 2026})
	if r.Status != report.ReadingOK || r.Count < 80 || r.Threshold != 80 || r.Value.Going == nil {
		t.Fatal("missing Market reading")
	}
	if r.InputSHA != lab.DataSHA || r.BuildStamp != testStamp || r.Cutoff != "2026-10-06T01:42:57+00:00" {
		t.Fatal("missing provenance")
	}
	if r.Provenance.Freshness.State == "live" || r.Provenance.Freshness.FetchedAt != "2026-10-06T15:57:20+00:00" || !strings.Contains(r.Provenance.Source, r.InputSHA) {
		t.Fatal("archive labelled live")
	}
	b, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	var wire map[string]json.RawMessage
	if err := json.Unmarshal(b, &wire); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"value", "provenance", "status", "absenceReason", "count", "threshold", "baseline", "cutoff", "method", "howCounted", "inputSHA", "buildStamp"} {
		if _, ok := wire[key]; !ok {
			t.Errorf("missing %s", key)
		}
	}
	for _, grade := range []string{"\"evidence\"", "\"solid\"", "\"rough\"", "\"thin\""} {
		if strings.Contains(string(b), grade) {
			t.Errorf("spread grade leaked: %s", grade)
		}
	}
	if r.HowCounted == "" || r.AbsenceReason != nil {
		t.Fatal("shell disclosure")
	}
	thin := a.TargetMarket(RawMarketScope{From: 2017, To: 2026, Phases: []string{"winter"}})
	if thin.Status != report.ReadingThin || thin.Count == 0 || thin.Value.Rows != nil || thin.AbsenceReason == nil {
		t.Fatal("thin reading fabricated data")
	}
	for _, raw := range []RawMarketScope{{From: 2026, To: 2017}, {From: 2017, To: 2026, Phases: []string{"bogus"}}, {From: 2017, To: 2026, Phases: []string{"winter", "winter"}}} {
		failed := a.TargetMarket(raw)
		if failed.Status != report.ReadingFailed || failed.AbsenceReason == nil {
			t.Fatal("invalid IPC scope accepted")
		}
	}
}

func TestMarketSurfaceMethods(t *testing.T) {
	t.Parallel()
	a := New(&fakeHost{}, &Held{})
	full := RawMarketScope{From: 2017, To: 2026}
	methods := []struct {
		name string
		read func(RawMarketScope) report.Reading[MarketSurface]
	}{
		{"lists", a.TargetMarketLists}, {"clock", a.TargetPickClock}, {"stretches", a.TargetMarketStretches},
	}
	for _, method := range methods {
		t.Run(method.name, func(t *testing.T) {
			ok := method.read(full)
			if ok.Status != report.ReadingOK || ok.Count == 0 || ok.HowCounted == "" || ok.Cutoff == "" || ok.InputSHA != lab.DataSHA || len(ok.Value.Sections) == 0 {
				t.Fatalf("incomplete reading: %+v", ok)
			}
			wire, err := json.Marshal(ok)
			if err != nil {
				t.Fatal(err)
			}
			for _, grade := range []string{"\"solid\"", "\"rough\"", "\"evidence\""} {
				if strings.Contains(string(wire), grade) {
					t.Fatal("grade leaked")
				}
			}
			thin := method.read(RawMarketScope{From: 2100, To: 2101})
			if thin.Status != report.ReadingThin || thin.AbsenceReason == nil || thin.Count != 0 {
				t.Fatal("absent scope fabricated data")
			}
			failed := method.read(RawMarketScope{From: 2026, To: 2017})
			if failed.Status != report.ReadingFailed || failed.AbsenceReason == nil {
				t.Fatal("invalid IPC accepted")
			}
		})
	}
}

func TestMarketInputValidationAndLoadFailure(t *testing.T) {
	t.Parallel()
	for _, raw := range []RawMarketScope{{From: 2017, To: 2026, Assets: []string{"bogus"}}, {From: 2017, To: 2026, Assets: []string{"LB", "LB"}}} {
		if raw.Validate() == nil {
			t.Fatal("invalid asset selection accepted")
		}
	}
	host := &fakeHost{labErr: errors.New("fingerprint mismatch")}
	a := New(host, &Held{})
	for _, r := range []report.Reading[MarketSurface]{a.TargetMarketLists(RawMarketScope{From: 2017, To: 2026}), a.TargetPickClock(RawMarketScope{From: 2017, To: 2026}), a.TargetMarketStretches(RawMarketScope{From: 2017, To: 2026}), a.TargetAssetProfile(RawMarketAsset{Scope: RawMarketScope{From: 2017, To: 2026}, Kind: "LB"})} {
		if r.Status != report.ReadingFailed || r.AbsenceReason == nil || !strings.Contains(*r.AbsenceReason, "fingerprint mismatch") {
			t.Fatal("load failure hidden")
		}
	}
	// An invalid scope is refused before the Lab is read.
	host.calls = nil
	_ = a.TargetMarketLists(RawMarketScope{From: 2026, To: 2017})
	_ = a.TargetAssetProfile(RawMarketAsset{Scope: RawMarketScope{From: 2017, To: 2026}, Kind: "bogus"})
	_ = a.TargetMarket(RawMarketScope{From: 2026, To: 2017})
	if len(host.calls) != 0 {
		t.Fatal("invalid scope reached the host", host.calls)
	}
}

func TestMarketReportsReadOrderAndWeek(t *testing.T) {
	t.Parallel()
	feeds := heldSample(t)
	host := &fakeHost{week: 7}
	held := &Held{}
	for kind, feed := range feeds {
		held.PublishFeed(kind, feed) // the projection is week 6's
	}
	board, err := New(host, held).TargetMarketReports()
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"ready", "week", "capture", "lookup", "zone"}; !reflect.DeepEqual(host.calls, want) {
		t.Fatal("read order", host.calls)
	}
	for _, r := range board.Reports {
		if r.ID == "M-046" && (r.Status != "failed" || len(r.Rows) != 0) {
			t.Fatal("another week's projection read as this week's", r.Status)
		}
	}
	if got := held.SnapshotFeeds()[marketfeeds.Projections]; len(got.Value) != 1 || got.Value[0].Week != 6 {
		t.Fatal("a read changed the held projection")
	}
	stopped := &fakeHost{readyErr: errors.New("fixture startup failed")}
	if _, err := New(stopped, held).TargetMarketReports(); err == nil || err.Error() != "market reports: startup: fixture startup failed" || len(stopped.calls) != 1 {
		t.Fatal("startup refusal", err, stopped.calls)
	}
	if _, err := New(stopped, held).TargetAssetBoard(); err == nil || err.Error() != "asset board: startup: fixture startup failed" {
		t.Fatal("startup refusal", err)
	}
	refusing := &fakeHost{refreshErr: errors.New("market refresh: startup: fixture startup failed")}
	if err := New(refusing, held).TargetMarketReportsRefresh(true); err == nil || err.Error() != "market refresh: startup: fixture startup failed" {
		t.Fatal("refresh error rewritten", err)
	}
}

type fakeModel map[string]Dynasty

func (f fakeModel) Player(pid string, _ time.Time) Dynasty { return f[pid] }
func (fakeModel) Pick(int, int, string, time.Time) Dynasty { return Dynasty{} }

func TestAssetBoardReadsHeldModelWithBound(t *testing.T) {
	t.Parallel()
	host := &fakeHost{now: unheld(), rostered: []string{"0042"}}
	held := &Held{}
	a := New(host, held)
	before, err := a.TargetAssetBoard()
	if err != nil || before.Values[0].Dynasty.Held {
		t.Fatal("valued before the model landed", before, err)
	}
	if _, ok := host.inputCtx.Deadline(); !ok {
		t.Fatal("asset read has no deadline")
	}
	price := 3.1
	held.PublishModel(fakeModel{"0042": {Valued: true, Discounted: 2.3, Seasons: 5, ClassLabel: "Linebacker", Price: &price}})
	after, err := a.TargetAssetBoard()
	if err != nil || !after.Values[0].Dynasty.Held || !after.Values[0].Market.Held {
		t.Fatal("held model not read", after, err)
	}
	if want := []string{"ready", "context", "assets"}; !reflect.DeepEqual(host.calls[:3], want) {
		t.Fatal("read order", host.calls)
	}
}

func fillWire(v reflect.Value, n *int) {
	*n++
	kind := v.Kind()
	if kind == reflect.Struct {
		for i := range v.NumField() {
			fillWire(v.Field(i), n)
		}
		return
	}
	if kind == reflect.Pointer {
		v.Set(reflect.New(v.Type().Elem()))
		fillWire(v.Elem(), n)
		return
	}
	if kind == reflect.Slice {
		v.Set(reflect.MakeSlice(v.Type(), 2, 2))
		for i := range v.Len() {
			fillWire(v.Index(i), n)
		}
		return
	}
	scalars := map[reflect.Kind]func(){
		reflect.String:  func() { v.SetString(fmt.Sprint("text ", *n)) },
		reflect.Float64: func() { v.SetFloat(float64(*n) + 0.25) },
		reflect.Int:     func() { v.SetInt(int64(*n)) },
		reflect.Int64:   func() { v.SetInt(int64(*n)) },
		reflect.Bool:    func() { v.SetBool(true) },
	}
	set, ok := scalars[kind]
	if !ok {
		panic("fillWire: add a case for " + kind.String())
	}
	set()
}

// TestMarketWireTwinsMatch proves WireSurface, WireReading and WireAssetBoard lose nothing: a
// reading and an asset board with every field set, and empty ones, encode the same before and
// after conversion.
func TestMarketWireTwinsMatch(t *testing.T) {
	t.Parallel()
	var full report.Reading[Surface]
	n := 0
	fillWire(reflect.ValueOf(&full).Elem(), &n)
	var assets Board
	fillWire(reflect.ValueOf(&assets).Elem(), &n)
	for _, c := range []struct {
		name       string
		built, got any
	}{
		{"full reading", full, WireReading(full, WireSurface)},
		{"empty reading", report.Reading[Surface]{}, WireReading(report.Reading[Surface]{}, WireSurface)},
		{"empty section", Surface{Sections: []Section{{}}}, WireSurface(Surface{Sections: []Section{{}}})},
		{"full asset board", assets, WireAssetBoard(assets)},
		{"empty asset board", Board{}, WireAssetBoard(Board{})},
	} {
		want, err := json.Marshal(c.built)
		if err != nil {
			t.Fatal(err)
		}
		got, err := json.Marshal(c.got)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(want, got) {
			t.Errorf("%s: wire twin encodes differently\n built %s\n wire  %s", c.name, want, got)
		}
	}
}

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
	"sync"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/market/markettest"
	"github.com/secureprospective/TheWarRoom/internal/app/report"
	"github.com/secureprospective/TheWarRoom/internal/app/season"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/marketfeeds"
	"github.com/secureprospective/TheWarRoom/internal/normalize"
)

type fakeHost struct {
	mu         sync.Mutex
	calls      []string
	readyErr   error
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

// TestMarketWireTwinsMatch checks the asset board wire conversion.
func TestMarketWireTwinsMatch(t *testing.T) {
	t.Parallel()
	var assets Board
	n := 0
	fillWire(reflect.ValueOf(&assets).Elem(), &n)
	for _, c := range []struct {
		name       string
		built, got any
	}{
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

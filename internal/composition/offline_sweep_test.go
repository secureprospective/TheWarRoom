package composition

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/app/catalog"
	"github.com/secureprospective/TheWarRoom/internal/archive"
	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/mfl"
	"github.com/secureprospective/TheWarRoom/internal/mflkey"
	"github.com/secureprospective/TheWarRoom/internal/offline"
)

// offlineApp is the season fixture after one good refresh, then with every connection refused
// the way a computer with no network refuses them. The refused address is returned so a test
// can look for it.
func offlineApp(t *testing.T) (*App, string) {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := listener.Addr().String()
	if err := listener.Close(); err != nil {
		t.Fatal(err)
	}
	a, _, _, _ := isolatedSeasonTestApp(t)
	a.core.MovesChanged, a.core.ClockChanged = func(context.Context) {}, func(context.Context) {}
	a.core.OpenURL = func(context.Context, string) {}
	a.core.KeyStore = mflkey.NewMemory(ingestion.LeagueID, "2026")
	a.refreshSeason(t.Context())
	a.core.Network = refusedTransport{addr: addr}
	a.core.Fetches = &archive.Transport{Base: a.core.Network, Sink: a.core.History, Judge: ingestion.MFLRefusal}
	a.core.MFLClient, err = mfl.New("api", 10000, mfl.WithTransport(a.core.Fetches),
		mfl.WithKeySource(func(context.Context) (mflkey.Key, error) { return "", mflkey.ErrNoKey }))
	if err != nil {
		t.Fatal(err)
	}
	return a, addr
}

// Every feed refreshed with the network refused, then every bound reading: no text a screen
// receives carries the network chain, and the feeds that failed say the offline sentence.
func TestOfflineFeedsSayPlainWords(t *testing.T) {
	a, addr := offlineApp(t)
	ctx := t.Context()
	var said []string
	keep := func(name string, v any) { said = append(said, name+": "+wireText(t, v)) }
	// The workers' own refreshes fill the held feeds' notes; their errors reach no screen.
	_, _ = a.refreshLeague(ctx, a.core.Rulebook, a.core.League)
	a.refreshSeason(ctx)
	a.refreshShellFeeds(ctx)
	a.refreshMarketFeeds(ctx, true)
	_ = a.refreshResearch(true)
	_ = a.refreshResearchNews("0025")
	a.refreshFreeAgents(ctx)
	a.refreshWeek(ctx)
	a.refreshNowWeek(ctx, 6, true)
	a.refreshScoreFirsts(ctx)
	a.refreshFinishedScores(ctx)
	_ = a.core.RefreshMarket(true)
	// What a person starts: every refresh button, every held copy read afresh, the key check.
	keep("loadSignals", a.loadSignals(ctx))
	keep("SetKey", must(controlHost{State: a.core, app: a}.SetKey("ABCDEF0123456789abcdef0123456789")))
	src := a.sourcesAPI()
	keep("LoadCrosswalk", src.LoadCrosswalk())
	for _, scope := range []string{"league", "signals", "crosswalk", "archive"} {
		keep("TargetDataRefresh "+scope, src.TargetDataRefresh(scope))
	}
	for _, scope := range src.TargetCacheScopes() {
		keep("TargetCacheClear "+scope.ID, src.TargetCacheClear(scope.ID))
	}
	for _, bound := range bindings(a) {
		said = append(said, refreshAll(t, bound)...)
	}
	a.core.WeekWorkers.Wait()
	for _, bound := range bindings(a) {
		said = append(said, readAll(t, bound)...)
	}
	for _, text := range heldText(reflect.ValueOf(a).Elem(), map[uintptr]bool{}) {
		if text != addr { // the test's own refusing transport holds the address
			said = append(said, "held: "+text)
		}
	}

	all := strings.Join(said, "\n")
	networkDetail := []string{"dial tcp", "connection refused", "no such host", "i/o timeout", "127.0.0.1", "Get \"", "Post \""}
	for _, line := range said {
		for _, detail := range append(networkDetail, addr) {
			if strings.Contains(line, detail) {
				t.Errorf("network detail %q reaches the screen:\n%.600s", detail, line)
				break
			}
		}
	}
	if !strings.Contains(all, offline.League) {
		t.Fatal("no feed said the offline sentence")
	}
}

func must[T any](v T, err error) []any { return []any{v, errText(err)} }

func wireText(t *testing.T, v any) string {
	t.Helper()
	body, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprint(v)
	}
	return string(body)
}

// refreshAll calls every catalog method named for a refresh, with zero arguments.
func refreshAll(t *testing.T, bound any) []string {
	t.Helper()
	return callAll(t, bound, func(c catalog.Command) bool { return strings.Contains(c.Method, "Refresh") })
}

// readAll calls every read-mode catalog method of a bound API with zero arguments and words what
// it returns.
func readAll(t *testing.T, bound any) []string {
	t.Helper()
	return callAll(t, bound, func(c catalog.Command) bool { return c.Gravity == "G0" || slices.Contains(c.Modes, catalog.Read) })
}

func callAll(t *testing.T, bound any, pick func(catalog.Command) bool) []string {
	t.Helper()
	v := reflect.ValueOf(bound)
	commands := boundCatalogs()[reflect.TypeOf(bound)]
	var out []string
	for _, c := range commands {
		// These local settings readers need state absent from the season fixture; neither fetches.
		if c.Method == "GetParams" || c.Method == "TargetClock" || !pick(c) {
			continue
		}
		m := v.MethodByName(c.Method)
		if !m.IsValid() {
			continue
		}
		args := make([]reflect.Value, m.Type().NumIn())
		for i := range args {
			args[i] = reflect.Zero(m.Type().In(i))
		}
		results, ok := callRecovered(m, args)
		if !ok {
			t.Errorf("%s: fixture panicked", c.Method)
			continue
		}
		var words []string
		for _, r := range results {
			if err, ok := r.Interface().(error); ok {
				words = append(words, errText(err))
				continue
			}
			words = append(words, wireText(t, r.Interface()))
		}
		out = append(out, c.Method+": "+strings.Join(words, " | "))
	}
	return out
}

func callRecovered(m reflect.Value, args []reflect.Value) (out []reflect.Value, ok bool) {
	defer func() { ok = recover() == nil && ok }()
	return m.Call(args), true
}

// heldText gathers every string the app holds (feeds' notes and failures, exported or not),
// following pointers once; locks, pools and stores hold no screen text and are walked harmlessly.
func heldText(v reflect.Value, seen map[uintptr]bool) []string {
	var out []string
	switch v.Kind() { //nolint:exhaustive // every other kind holds no text
	case reflect.String:
		if v.String() != "" {
			out = append(out, v.String())
		}
	case reflect.Pointer, reflect.Interface:
		if v.IsNil() {
			return nil
		}
		if v.Kind() == reflect.Pointer {
			if seen[v.Pointer()] {
				return nil
			}
			seen[v.Pointer()] = true
		}
		return heldText(v.Elem(), seen)
	case reflect.Struct:
		for i := range v.NumField() {
			out = append(out, heldText(v.Field(i), seen)...)
		}
	case reflect.Slice, reflect.Array:
		for i := range min(v.Len(), 50) {
			out = append(out, heldText(v.Index(i), seen)...)
		}
	case reflect.Map:
		for _, k := range v.MapKeys() {
			out = append(out, heldText(v.MapIndex(k), seen)...)
		}
	default:
	}
	return out
}

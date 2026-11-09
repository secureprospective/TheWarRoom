package commissioner

import (
	"encoding/json"
	"reflect"
	"sync"
	"testing"
	"time"
)

// assetsCacheLedger mixes every way the reading instant matters: a backdated trade, a correction
// learned later and then compensated, a reserve move and a correction that lapses.
func assetsCacheLedger(t *testing.T) []Record {
	t.Helper()
	b := rolloverLedger(t)
	offer := TradeState{Status: "offered", Assets: []TradeAsset{{Player: "0003", From: "franchise.02", To: "franchise.01"}}}
	offer.ID = b.add(RawCommand{League: rolloverLeague, Command: TradeOffer, Args: RawArgs{TradeOutcome: &offer}, Effective: "now"}, seasonMoment(4))
	approved := offer.Clone()
	approved.Status, approved.Previous = "approved", TradeDigest(offer)
	b.add(RawCommand{League: rolloverLeague, Command: TradeReview, Args: RawArgs{TradeOutcome: &approved},
		Effective: seasonMoment(7).Format(time.RFC3339Nano)}, seasonMoment(12))
	player := b.records[1].Command.Args.Native.Players[2]
	b.add(RawCommand{League: rolloverLeague, Command: RosterReserve,
		Args:      RawArgs{Waiver: &RawWaiverArgs{Player: player.ID, Franchise: player.Franchise, Bucket: "ir"}, WaiverOutcome: &WaiverOutcome{Player: &player}},
		Effective: seasonMoment(5).Format(time.RFC3339Nano)}, seasonMoment(13))
	entry := b.add(RawCommand{League: rolloverLeague, Command: CorrectRoster,
		Args:      RawArgs{Correction: &RawCorrection{Player: "0005", Franchise: "franchise.02", Action: "remove"}},
		Effective: seasonMoment(6).Format(time.RFC3339Nano), Reason: "Repair week membership"}, seasonMoment(14))
	b.revert(entry, seasonMoment(16))
	b.add(RawCommand{League: rolloverLeague, Command: RuleException, Reason: "Eligibility only",
		Args: RawArgs{Correction: &RawCorrection{Rule: "roster.limits", Scope: &ExceptionScope{Franchise: "franchise.01", Player: "0003"},
			Until: seasonMoment(18).Format(time.RFC3339Nano)}}}, seasonMoment(15))
	return WithReplayCache(b.records)
}

// readingInstants is every fixture day each side of noon, where all its times fall, plus no instant.
// It is built apart from clockThresholds so a boundary that code forgets is still swept.
func readingInstants(t *testing.T, records []Record) [][]time.Time {
	t.Helper()
	if _, ok := clockThresholds(records); !ok {
		t.Fatal("ledger times must parse")
	}
	out := [][]time.Time{nil, {time.Unix(1, 0).UTC()}}
	for day := 1; day <= 31; day++ {
		at := seasonMoment(day)
		out = append(out, []time.Time{at.Add(-time.Nanosecond)}, []time.Time{at}, []time.Time{at.Add(time.Nanosecond)})
	}
	return out
}

func assetsJSON(t *testing.T, assets NativeAssets, book MarketBook, err error) string {
	t.Helper()
	body, jerr := json.Marshal(struct {
		A   NativeAssets
		B   MarketBook
		Err string
	}{assets, book, errText(err)})
	if jerr != nil {
		t.Fatal(jerr)
	}
	return string(body)
}

func errText(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

// TestAssetsCacheMatchesFold: every reading instant gives the uncached fold's exact bytes, and a caller
// scribbling over every field of its copy never reaches the next caller.
func TestAssetsCacheMatchesFold(t *testing.T) {
	records := assetsCacheLedger(t)
	if memoFor(records) == nil {
		t.Fatal("fixture must carry the revision cache")
	}
	for _, at := range readingInstants(t, records) {
		wantA, wantB, wantErr := foldNativeAssets(rolloverLeague, records, at...)
		want := assetsJSON(t, wantA, wantB, wantErr)
		for range 2 {
			gotA, gotB, gotErr := nativeAssetsState(rolloverLeague, records, at...)
			if got := assetsJSON(t, gotA, gotB, gotErr); got != want {
				t.Fatalf("at %v: cached fold differs\n got %s\nwant %s", at, got, want)
			}
			if !reflect.DeepEqual(gotA, wantA) || !reflect.DeepEqual(gotB, wantB) {
				t.Fatalf("at %v: cached fold differs in shape (nil vs empty)", at)
			}
			scribble(reflect.ValueOf(&gotA).Elem())
			scribble(reflect.ValueOf(&gotB).Elem())
		}
	}
	if c := &memoFor(records).assets; !c.timed || len(c.views) == 0 {
		t.Fatal("timed readings must be served from the cache")
	}
	assets, _, err := nativeAssetsState(rolloverLeague, records, seasonMoment(17))
	if err != nil || len(assets.Corrections) != 1 {
		t.Fatalf("fixture must fold with the live exception: %d %v", len(assets.Corrections), err)
	}
}

// TestAssetsCacheParallelReaders runs under -race: readers at mixed instants share each fold.
func TestAssetsCacheParallelReaders(t *testing.T) {
	records := assetsCacheLedger(t)
	instants := readingInstants(t, records)
	want := make([]string, len(instants))
	for i, at := range instants {
		a, b, err := foldNativeAssets(rolloverLeague, records, at...)
		want[i] = assetsJSON(t, a, b, err)
	}
	var wg sync.WaitGroup
	for g := range 8 {
		wg.Go(func() {
			for i := range instants {
				k := (i + g) % len(instants)
				a, b, err := nativeAssetsState(rolloverLeague, records, instants[k]...)
				if got := assetsJSON(t, a, b, err); got != want[k] {
					t.Errorf("reader %d at %v: differs", g, instants[k])
					return
				}
				scribble(reflect.ValueOf(&a).Elem())
				scribble(reflect.ValueOf(&b).Elem())
			}
		})
	}
	wg.Wait()
}

// TestAssetsCacheTypesHaveNoHiddenReferences: the copy follows exported fields only, so a pointer,
// slice or map behind an unexported field would be shared between callers.
func TestAssetsCacheTypesHaveNoHiddenReferences(t *testing.T) {
	seen := map[reflect.Type]bool{reflect.TypeFor[time.Time](): true}
	var walk func(reflect.Type, string)
	walk = func(ty reflect.Type, path string) {
		if seen[ty] {
			return
		}
		seen[ty] = true
		switch ty.Kind() { //nolint:exhaustive // scalars hold no references
		case reflect.Pointer, reflect.Slice, reflect.Array:
			walk(ty.Elem(), path+"[]")
		case reflect.Map:
			walk(ty.Key(), path+"{key}")
			walk(ty.Elem(), path+"{}")
		case reflect.Interface, reflect.Func, reflect.Chan, reflect.UnsafePointer:
			t.Errorf("%s: %s cannot be copied for each caller", path, ty)
		case reflect.Struct:
			for i := range ty.NumField() {
				f := ty.Field(i)
				if !f.IsExported() && (!reflect.Zero(f.Type).Comparable() || f.Type.Kind() == reflect.Pointer) {
					t.Errorf("%s.%s: unexported %s would be shared", path, f.Name, f.Type)
				}
				walk(f.Type, path+"."+f.Name)
			}
		}
	}
	walk(reflect.TypeFor[NativeAssets](), "NativeAssets")
	walk(reflect.TypeFor[MarketBook](), "MarketBook")
}

// scribble overwrites every reachable value, as a careless caller projecting in place might.
func scribble(v reflect.Value) {
	switch v.Kind() { //nolint:exhaustive // the remaining kinds are absent from the asset types
	case reflect.Pointer:
		if !v.IsNil() {
			scribble(v.Elem())
		}
	case reflect.Slice, reflect.Array:
		for i := range v.Len() {
			scribble(v.Index(i))
		}
	case reflect.Map:
		if v.IsNil() || !v.CanSet() {
			return
		}
		for _, k := range v.MapKeys() {
			e := reflect.New(v.Type().Elem()).Elem()
			e.Set(v.MapIndex(k))
			scribble(e)
			v.SetMapIndex(k, e)
		}
	case reflect.Struct:
		if v.Type() == reflect.TypeFor[time.Time]() {
			v.Set(reflect.ValueOf(time.Unix(7, 0)))
			return
		}
		for i := range v.NumField() {
			if v.Type().Field(i).IsExported() {
				scribble(v.Field(i))
			}
		}
	case reflect.String:
		v.SetString(v.String() + "!")
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		v.SetInt(v.Int() + 1)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		v.SetUint(v.Uint() + 1)
	case reflect.Bool:
		v.SetBool(!v.Bool())
	}
}

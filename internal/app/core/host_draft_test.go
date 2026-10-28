package core_test

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/app/core"
	"github.com/secureprospective/TheWarRoom/internal/app/core/coretest"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/draftreports"
)

func draftState(t *testing.T) (*core.State, *coretest.Transport) {
	t.Helper()
	bodies := map[string]string{}
	for export, name := range map[string]string{"draftResults": "results-2026.json", "futureDraftPicks": "future-2026.json"} {
		body, err := os.ReadFile(filepath.Join("..", "..", "ingestion", "draftreports", "testdata", name))
		if err != nil {
			t.Fatal(err)
		}
		bodies[export] = string(body)
	}
	s := coretest.New(t).Build()
	transport, client := coretest.NewTransport(t, bodies)
	s.MFLClient = client
	return s, transport
}

func TestDraftFeedsDetachedAndNilDistinct(t *testing.T) {
	s, _ := draftState(t)
	s.RefreshDraftResults(s.Ctx)
	s.RefreshFuturePicks(s.Ctx)
	results, future := s.DraftFeeds()
	assertCounterpartiesDetached(t, reflect.ValueOf(results.Value), reflect.ValueOf(s.SeasonDraftResults.Value), "results")
	assertCounterpartiesDetached(t, reflect.ValueOf(future.Value), reflect.ValueOf(s.SeasonFuturePicks.Value), "future")
	results.Value[0].Franchise = "changed"
	*results.Value[0].Player = *results.Value[1].Player
	future.Value[0].Owner = "changed"
	again, next := s.DraftFeeds()
	if again.Value[0].Franchise == "changed" || next.Value[0].Owner == "changed" {
		t.Fatal("snapshot aliases shared state")
	}
	s.SeasonDraftResults.Value, s.SeasonFuturePicks.Value = nil, []draftreports.FuturePick{}
	results, future = s.DraftFeeds()
	if results.Value != nil || future.Value == nil {
		t.Fatal("nil and empty collapsed")
	}
}

func assertCounterpartiesDetached(t *testing.T, left, right reflect.Value, path string) {
	t.Helper()
	kind := left.Kind()
	if kind == reflect.Pointer || kind == reflect.Slice || kind == reflect.Map {
		if !left.IsNil() && left.Pointer() == right.Pointer() && (kind != reflect.Slice || left.Len() > 0) {
			t.Fatalf("%s shares mutable storage", path)
		}
	}
	if kind == reflect.Pointer && !left.IsNil() {
		assertCounterpartiesDetached(t, left.Elem(), right.Elem(), path)
	}
	if kind == reflect.Struct {
		for i := range left.NumField() {
			if left.Type().Field(i).IsExported() {
				assertCounterpartiesDetached(t, left.Field(i), right.Field(i), path+"."+left.Type().Field(i).Name)
			}
		}
	}
	if kind == reflect.Slice {
		for i := range left.Len() {
			assertCounterpartiesDetached(t, left.Index(i), right.Index(i), path)
		}
	}
	if kind == reflect.Map {
		for _, key := range left.MapKeys() {
			assertCounterpartiesDetached(t, left.MapIndex(key), right.MapIndex(key), path)
		}
	}
}

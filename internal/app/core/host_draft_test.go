package core_test

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/app/core"
	"github.com/secureprospective/TheWarRoom/internal/app/core/coretest"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/draftreports"
	"github.com/secureprospective/TheWarRoom/internal/lab"
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

func TestDraftFactsDetached(t *testing.T) {
	s := coretest.New(t).Build()
	d, err := lab.Load()
	if err != nil {
		t.Fatal(err)
	}
	s.LabData = func() (*lab.Data, error) { return d, nil }
	got, _, err := s.DraftFacts()
	if err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"Draft", "Trades", "Delivered"} {
		left, right := reflect.ValueOf(got).Elem().FieldByName(field), reflect.ValueOf(d).Elem().FieldByName(field)
		if !reflect.DeepEqual(left.Interface(), right.Interface()) {
			t.Fatal("facts lost", field)
		}
		assertCounterpartiesDetached(t, left, right, field)
	}
	*got.Draft[0].Wins[0] = -9
	got.Draft[0].SlotWins[0] = -9
	if *d.Draft[0].Wins[0] == -9 || d.Draft[0].SlotWins[0] == -9 {
		t.Fatal("nested draft storage shared")
	}
}

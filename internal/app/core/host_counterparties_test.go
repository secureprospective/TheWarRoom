package core_test

import (
	"errors"
	"reflect"
	"sync"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/app/core/coretest"
	"github.com/secureprospective/TheWarRoom/internal/lab"
)

func TestCounterpartiesFactsDetached(t *testing.T) {
	t.Parallel()
	s := coretest.New(t).Build()
	d, err := lab.Load()
	if err != nil {
		t.Fatal(err)
	}
	s.LabData = func() (*lab.Data, error) { return d, nil }
	s.BuildStamp = "stamp"
	got, err := s.CounterpartiesFacts()
	if err != nil {
		t.Fatal(err)
	}
	if s.CounterpartiesBuildStamp() != "stamp" || got.Built != d.Built || got.ArchiveThrough != d.ArchiveThrough {
		t.Fatal("Lab metadata changed")
	}
	for _, field := range []string{"Trades", "TeamSeasons", "Franchises"} {
		left := reflect.ValueOf(got).Elem().FieldByName(field)
		right := reflect.ValueOf(d).Elem().FieldByName(field)
		if !reflect.DeepEqual(left.Interface(), right.Interface()) {
			t.Fatalf("%s lost facts", field)
		}
		assertCounterpartiesDetached(t, left, right, field)
	}
	got.Trades[0].Legs[0].From = "changed"
	f := got.Franchises["0001"]
	f.Name = "changed"
	got.Franchises["0001"] = f
	if d.Trades[0].Legs[0].From == "changed" || d.Franchises["0001"].Name == "changed" {
		t.Fatal("mutating the capture changed the held Lab")
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

func TestCounterpartiesFactsFallbackFailureAndConcurrentReads(t *testing.T) {
	t.Parallel()
	s := coretest.New(t).Build()
	d, err := s.CounterpartiesFacts()
	if err != nil || len(d.Trades) == 0 {
		t.Fatal("zero-state Lab fallback changed", err)
	}
	failure := errors.New("load lab facts: unavailable")
	s.LabData = func() (*lab.Data, error) { return nil, failure }
	if _, err := s.CounterpartiesFacts(); !errors.Is(err, failure) {
		t.Fatal("loader error identity changed", err)
	}
	s.LabData = sync.OnceValues(lab.Load)
	var workers sync.WaitGroup
	for range 4 {
		workers.Go(func() {
			got, readErr := s.CounterpartiesFacts()
			if readErr != nil {
				t.Error(readErr)
				return
			}
			got.Trades[0].Legs[0].From = "local mutation"
		})
	}
	workers.Wait()
	got, err := s.CounterpartiesFacts()
	if err != nil || got.Trades[0].Legs[0].From == "local mutation" {
		t.Fatal("concurrent capture leaked mutation", err)
	}
}

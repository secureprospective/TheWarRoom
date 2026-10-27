package core_test

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/core"
	"github.com/secureprospective/TheWarRoom/internal/app/core/coretest"
	"github.com/secureprospective/TheWarRoom/internal/app/report"
	"github.com/secureprospective/TheWarRoom/internal/facts"
	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
)

func TestArchiveFactsDetached(t *testing.T) {
	s := coretest.New(t).Build()
	var held report.ArchiveSnapshot
	seedArchiveColumns(reflect.ValueOf(&held.Facts).Elem())
	held.SHA, held.Cutoff = "verified", "fixture cutoff"
	held.Through = time.Date(2026, 10, 11, 0, 0, 0, 0, time.UTC)
	s.ReadArchive = func(context.Context) (report.ArchiveSnapshot, bool, error) {
		if s.FactsMu.TryLock() {
			s.FactsMu.Unlock()
			t.Error("archive callback ran without FactsMu")
		}
		return held, true, nil
	}
	got, stale, err := s.ArchiveFacts(s.Ctx)
	if err != nil || !stale || !reflect.DeepEqual(got, held) {
		t.Fatal(stale, err)
	}
	before, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	// Change every column element, including nullable scalars, in the held snapshot.
	seedArchiveColumns(reflect.ValueOf(&held.Facts).Elem())
	after, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Fatal("archive facts alias held columns")
	}
	if !s.FactsMu.TryLock() {
		t.Fatal("archive read retained FactsMu")
	}
	s.FactsMu.Unlock()
}

// Existing values are changed in place to expose both slice and nullable-scalar aliases.
func seedArchiveColumns(v reflect.Value) {
	if v.Kind() == reflect.Struct {
		for i := range v.NumField() {
			seedArchiveColumns(v.Field(i))
		}
		return
	}
	if v.Kind() == reflect.Slice {
		if v.IsNil() {
			v.Set(reflect.MakeSlice(v.Type(), 1, 1))
		}
		seedArchiveColumns(v.Index(0))
		return
	}
	if v.Kind() == reflect.Pointer {
		if v.IsNil() {
			v.Set(reflect.New(v.Type().Elem()))
		}
		seedArchiveColumns(v.Elem())
		return
	}
	if v.Kind() == reflect.String {
		v.SetString(v.String() + "value")
		return
	}
	if v.Kind() == reflect.Int64 {
		v.SetInt(v.Int() + 1)
		return
	}
	if v.Kind() == reflect.Float64 {
		v.SetFloat(v.Float() + 1)
		return
	}
	panic("unsupported archive column: " + v.Kind().String())
}

func TestArchiveFactsZeroAndError(t *testing.T) {
	s := coretest.New(t).Build()
	want := errors.New("archive: fixture failure")
	s.ReadArchive = func(context.Context) (report.ArchiveSnapshot, bool, error) {
		return report.ArchiveSnapshot{}, false, want
	}
	got, stale, err := s.ArchiveFacts(s.Ctx)
	if err != want || stale || !reflect.DeepEqual(got, report.ArchiveSnapshot{}) {
		t.Fatal(got, stale, err)
	}
	if s.ArchiveContext() != s.Ctx {
		t.Fatal("archive context changed")
	}
}

func TestArchiveOpponentLockOrderAndFallback(t *testing.T) {
	s := coretest.New(t).Build()
	s.Week.Number = 5
	s.SeasonLineups.Value = leaguefeed.Lineups{Week: 5, Matchups: []leaguefeed.Matchup{{Home: "0001", Away: "0002"}}}
	s.ArchiveOpponentFallback = func(week int, team string) string {
		if week != 5 || team != "0001" {
			t.Error(week, team)
		}
		if s.SeasonMu.TryLock() {
			s.SeasonMu.Unlock()
			t.Error("fallback ran without SeasonMu")
		}
		if !s.WeekMu.TryLock() {
			t.Error("WeekMu retained while reading fallback")
		} else {
			s.WeekMu.Unlock()
		}
		return "0003"
	}
	if got := s.ArchiveOpponent("0001"); got != "0002" {
		t.Fatal(got)
	}
	if got := s.ArchiveOpponent("0002"); got != "0001" {
		t.Fatal(got)
	}
	s.SeasonLineups.Value.Week = 4
	if got := s.ArchiveOpponent("0001"); got != "0003" {
		t.Fatal(got)
	}
	s.ArchiveOpponentFallback = nil
	if got := s.ArchiveOpponent("0001"); got != "" {
		t.Fatal(got)
	}
}

func TestArchiveConcurrentCachePublicationAndRead(t *testing.T) {
	s := coretest.New(t).Build()
	held := report.ArchiveSnapshot{SHA: "0", Facts: facts.Facts{Season: facts.SeasonColumns{Year: []int64{2026}}}}
	s.ReadArchive = func(context.Context) (report.ArchiveSnapshot, bool, error) { return held, false, nil }
	var workers sync.WaitGroup
	workers.Go(func() {
		for range 100 {
			s.FactsMu.Lock()
			held.Facts.Season.Year[0]++
			s.FactsMu.Unlock()
		}
	})
	for range 4 {
		workers.Go(func() {
			for range 100 {
				got, _, err := s.ArchiveFacts(s.Ctx)
				if err != nil || len(got.Facts.Season.Year) != 1 {
					t.Error(err)
				}
				got.Facts.Season.Year[0] = -1
			}
		})
	}
	workers.Wait()
	if held.Facts.Season.Year[0] != 2126 {
		t.Fatal("reader mutated held facts")
	}
}

func TestArchivedOnlyWithoutHistoryStaysUnavailable(t *testing.T) {
	s := &core.State{}
	got := core.ArchivedOnly(context.Background(), s, "injuries", true, func([]byte) ([]string, error) { t.Fatal("parsed absent history"); return nil, nil })
	if got.RefreshError != "No held archive" || got.Value != nil {
		t.Fatal(got)
	}
}

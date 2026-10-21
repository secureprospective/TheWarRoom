package counterparties

import (
	"encoding/json"
	"errors"
	"reflect"
	"sync"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/app/report"
	"github.com/secureprospective/TheWarRoom/internal/lab"
)

type fakeHost struct {
	data  *lab.Data
	err   error
	reads int
}

func (h *fakeHost) CounterpartiesFacts() (*lab.Data, error) {
	h.reads++
	return h.data, h.err
}
func (h *fakeHost) CounterpartiesBuildStamp() string { return "dev " }

func TestCounterpartiesDenialBeforeLabRead(t *testing.T) {
	t.Parallel()
	h := &fakeHost{err: errors.New("load lab facts: unavailable")}
	a := New(h)
	bad := RawCounterpartiesScope{From: 2026, To: 2017}
	board := a.TargetCounterparties(bad)
	team := a.TargetCounterpartTeam(RawCounterpartTeam{Scope: bad})
	if h.reads != 0 || *board.AbsenceReason != "counterparties scope: invalid season range" || *team.AbsenceReason != *board.AbsenceReason {
		t.Fatal("invalid scope reached Lab or changed its refusal")
	}
	good := RawCounterpartiesScope{From: 2017, To: 2026}
	board = a.TargetCounterparties(good)
	team = a.TargetCounterpartTeam(RawCounterpartTeam{Scope: good, Fid: "0001"})
	if h.reads != 2 || *board.AbsenceReason != "load lab facts: unavailable" || *team.AbsenceReason != *board.AbsenceReason {
		t.Fatal("Lab failure text or read order changed")
	}
	if board.Status != report.ReadingFailed || board.Threshold != 40 || team.Status != report.ReadingFailed || team.Threshold != 1 {
		t.Fatal("failure envelope changed")
	}
}

func TestAPIUsesSharedPresentationAndService(t *testing.T) {
	t.Parallel()
	d, err := lab.Load()
	if err != nil {
		t.Fatal(err)
	}
	h := &fakeHost{data: d}
	a, svc := New(h), testService()
	for _, scope := range []RawCounterpartiesScope{
		{From: 2017, To: 2026},
		{From: 2017, To: 2026, Tiers: []string{"contender"}, Postures: []string{"now"}},
		{From: 2017, To: 2017, Windows: []string{"summer"}, Postures: []string{"later"}},
	} {
		if !reflect.DeepEqual(report.Reading[Counterparties](a.TargetCounterparties(scope)), svc.TargetCounterparties(scope)) {
			t.Fatal("API changed the populated or empty board")
		}
		team := RawCounterpartTeam{Scope: scope, Fid: "0001"}
		if !reflect.DeepEqual(report.Reading[TeamCounterpart](a.TargetCounterpartTeam(team)), svc.TargetCounterpartTeam(team)) {
			t.Fatal("API changed the team panel")
		}
	}
}

func TestParallelAPIReadsDoNotMutateLab(t *testing.T) {
	t.Parallel()
	d, err := lab.Load()
	if err != nil {
		t.Fatal(err)
	}
	before, err := json.Marshal(d)
	if err != nil {
		t.Fatal(err)
	}
	var workers sync.WaitGroup
	for range 4 {
		workers.Go(func() {
			a := New(&fakeHost{data: d})
			scope := RawCounterpartiesScope{From: 2017, To: 2026}
			a.TargetCounterparties(scope)
			a.TargetCounterpartTeam(RawCounterpartTeam{Scope: scope, Fid: "0001"})
		})
	}
	workers.Wait()
	after, err := json.Marshal(d)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Fatal("API mutated the frozen facts")
	}
}

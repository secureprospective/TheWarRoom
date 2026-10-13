package history

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/db"
)

func createCommand() commissioner.RawCommand {
	return commissioner.RawCommand{League: "native.test", Command: commissioner.Create,
		Args: commissioner.RawArgs{Name: "Test league"}, Reason: "  start our league  ",
		Request: "create", Effective: "now", Visibility: "league"}
}

func reverseCommand(entry string, revision int64, request string) commissioner.RawCommand {
	return commissioner.RawCommand{League: "native.test", Command: commissioner.Revert,
		Args: commissioner.RawArgs{Entry: entry}, Reason: "correct our recorded change",
		Expected: revision, Request: request, Effective: "now", Visibility: "league"}
}

type refusingAuthorizer struct{ calls int }

func (a *refusingAuthorizer) Allow(_, _, _ string) (string, error) {
	a.calls++
	return "", errors.New("only the commissioner may make this change")
}

func TestCommissionerDirectValidation(t *testing.T) {
	s, _, _ := newStore(t)
	for _, reason := range []string{"", " \t\n"} {
		raw := createCommand()
		raw.Reason = reason
		if _, err := s.ExecuteCommissioner(t.Context(), raw, "creator", nil); err == nil {
			t.Fatal("blank direct Go reason accepted")
		}
	}
	raw := createCommand()
	raw.Gravity = "G1"
	if _, err := s.ExecuteCommissioner(t.Context(), raw, "creator", nil); err == nil {
		t.Fatal("caller lowered gravity")
	}
	events, err := s.Events(t.Context(), raw.League, commissioner.Stream, 0)
	if err != nil || len(events) != 0 {
		t.Fatalf("refusal mutated: %v %v", events, err)
	}
	receipt, err := s.ExecuteCommissioner(t.Context(), createCommand(), "creator", nil)
	if err != nil {
		t.Fatal(err)
	}
	events, err = s.Events(t.Context(), raw.League, commissioner.Stream, 0)
	if err != nil || len(events) != 1 {
		t.Fatalf("append: %v %v", events, err)
	}
	record := events[0].Payload.Commissioner
	if receipt.Revision != 1 || record.Role != "commissioner" || record.Gravity != "G3" ||
		record.Command.Reason != "  start our league  " || receipt.Message != "Recorded in the league ledger." {
		t.Fatalf("wrong landed receipt/metadata: %+v %+v", receipt, record)
	}
}

func TestCommissionerRetryConflictAndAuthorization(t *testing.T) {
	s, _, _ := newStore(t)
	raw := createCommand()
	first, err := s.ExecuteCommissioner(t.Context(), raw, "creator", nil)
	if err != nil {
		t.Fatal(err)
	}
	raw.Gravity = "G3"
	retry, err := s.ExecuteCommissioner(t.Context(), raw, "creator", nil)
	if err != nil || first != retry {
		t.Fatalf("retry: %+v %v", retry, err)
	}
	raw.Reason = "different"
	if _, err := s.ExecuteCommissioner(t.Context(), raw, "creator", nil); err == nil {
		t.Fatal("request key reused with different body")
	}
	revert := reverseCommand(first.Entry, 0, "stale")
	if _, err := s.ExecuteCommissioner(t.Context(), revert, "creator", nil); !errors.Is(err, ErrRevisionConflict) || !strings.Contains(err.Error(), "version is 1") {
		t.Fatalf("stale error does not carry current revision: %v", err)
	}
	revert.Expected = 1
	deny := &refusingAuthorizer{}
	if _, err := s.ExecuteCommissioner(t.Context(), revert, "creator", deny); err == nil || !strings.Contains(err.Error(), "only the commissioner") || deny.calls != 1 {
		t.Fatalf("fake authorizer bypassed: %v calls=%d", err, deny.calls)
	}
	if _, err := s.ExecuteCommissioner(t.Context(), revert, "owner", nil); err == nil {
		t.Fatal("noncreator authorized")
	}
	events, err := s.Events(t.Context(), raw.League, commissioner.Stream, 0)
	if err != nil || len(events) != 1 {
		t.Fatalf("refusals appended: %v %v", events, err)
	}
}

func TestCommissionerCompensationAndRestart(t *testing.T) {
	s, _, pools := newStore(t)
	first, err := s.ExecuteCommissioner(t.Context(), createCommand(), "creator", nil)
	if err != nil {
		t.Fatal(err)
	}
	raw := reverseCommand(first.Entry, 1, "undo")
	raw.Visibility = "private"
	undo, err := s.ExecuteCommissioner(t.Context(), raw, "creator", nil)
	if err != nil {
		t.Fatal(err)
	}
	events, err := s.Events(t.Context(), raw.League, commissioner.Stream, 0)
	if err != nil || len(events) != 2 || events[0].ID != first.Entry || events[1].Supersedes != first.Entry {
		t.Fatalf("not compensation: %+v %v", events, err)
	}
	records, err := commissionerRecords(events)
	if err != nil {
		t.Fatal(err)
	}
	if commissioner.Replay(records).Active[first.Entry] {
		t.Fatal("revert did not compensate original")
	}
	public := reverseCommand(undo.Entry, 2, "undo-undo")
	if _, err := s.ExecuteCommissioner(t.Context(), public, "creator", nil); err == nil {
		t.Fatal("public reversal leaked private target")
	}
	public.Visibility = "private"
	if _, err := s.ExecuteCommissioner(t.Context(), public, "creator", nil); err != nil {
		t.Fatal(err)
	}
	before, err := s.CommissionerLedger(t.Context(), raw.League, "creator", nil)
	if err != nil {
		t.Fatal(err)
	}
	// Close and reopen the file, not just the Store value.
	var path string
	if err := pools.Write().QueryRowContext(t.Context(), `SELECT file FROM pragma_database_list WHERE name='main'`).Scan(&path); err != nil {
		t.Fatal(err)
	}
	if err := pools.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := db.Open(t.Context(), path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = reopened.Close() })
	fresh := New(reopened, s.Registry())
	if err := fresh.Initialize(t.Context()); err != nil {
		t.Fatal(err)
	}
	after, err := fresh.CommissionerLedger(t.Context(), raw.League, "creator", nil)
	if err != nil || !reflect.DeepEqual(before, after) {
		t.Fatalf("restart ledger: %+v %v", after, err)
	}
	retry, err := fresh.ExecuteCommissioner(t.Context(), raw, "creator", nil)
	if err != nil || retry != undo {
		t.Fatalf("restart receipt: %+v %v", retry, err)
	}
	events, err = fresh.Events(t.Context(), raw.League, commissioner.Stream, 0)
	if err != nil {
		t.Fatal(err)
	}
	records, err = commissionerRecords(events)
	if err != nil {
		t.Fatal(err)
	}
	projection := commissioner.Replay(records)
	if !projection.Active[first.Entry] || projection.Active[undo.Entry] || len(records) != 3 {
		t.Fatalf("reversal of reversal did not replay: %+v", projection)
	}
	owner, err := fresh.CommissionerLedger(t.Context(), raw.League, "owner", nil)
	if err != nil {
		t.Fatal(err)
	}
	wire, err := json.Marshal(owner)
	if err != nil {
		t.Fatal(err)
	}
	if len(owner.Entries) != 1 || strings.Contains(string(wire), undo.Entry) ||
		strings.Contains(string(wire), "correct our recorded change") {
		t.Fatalf("private wire leak: %s", wire)
	}
}

func TestCommissionerConcurrentRetryAcrossStores(t *testing.T) {
	s, _, pools := newStore(t)
	second := New(pools, s.Registry())
	var wg sync.WaitGroup
	receipts := make(chan commissioner.Receipt, 2)
	failures := make(chan error, 2)
	for _, store := range []*Store{s, second} {
		wg.Go(func() {
			r, err := store.ExecuteCommissioner(t.Context(), createCommand(), "creator", nil)
			receipts <- r
			failures <- err
		})
	}
	wg.Wait()
	one, two := <-receipts, <-receipts
	if err := <-failures; err != nil {
		t.Fatal(err)
	}
	if err := <-failures; err != nil {
		t.Fatal(err)
	}
	if one != two {
		t.Fatalf("concurrent retries differ: %+v %+v", one, two)
	}
	events, err := s.Events(t.Context(), "native.test", commissioner.Stream, 0)
	if err != nil || len(events) != 1 {
		t.Fatalf("duplicate append: %v %v", events, err)
	}
}

func TestCommissionerLedgerRejectsDamagedRecord(t *testing.T) {
	s, _, _ := newStore(t)
	if _, err := s.ExecuteCommissioner(t.Context(), createCommand(), "creator", nil); err != nil {
		t.Fatal(err)
	}
	events, err := s.Events(t.Context(), "native.test", commissioner.Stream, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, damage := range []func(*LeagueEvent){
		func(e *LeagueEvent) { e.Payload.Commissioner.Role = " " },
		func(e *LeagueEvent) { e.Payload.Commissioner.Gravity = "G0" },
		func(e *LeagueEvent) { e.Payload.Commissioner.Canonical = "different body" },
		func(e *LeagueEvent) { e.Scope = "private" },
		func(e *LeagueEvent) { e.Payload.Commissioner.Receipt.RecordedAt = t0().Add(-time.Hour) },
	} {
		event := events[0]
		record := *event.Payload.Commissioner
		event.Payload.Commissioner = &record
		damage(&event)
		if _, err := commissionerRecords([]LeagueEvent{event}); err == nil {
			t.Fatal("damaged ledger record was read")
		}
	}
}

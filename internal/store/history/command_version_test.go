package history

import (
	"bytes"
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
)

func legacyVersionRecords(t *testing.T) []commissioner.Record {
	t.Helper()
	body, err := os.ReadFile("testdata/commissioner-v0.json")
	if err != nil {
		t.Fatal(err)
	}
	var records []commissioner.Record
	if err := json.Unmarshal(body, &records); err != nil {
		t.Fatal(err)
	}
	return records
}

func TestLegacyCommandVersionsReplayAndRetry(t *testing.T) {
	t.Parallel()
	s, _, pools := newStore(t)
	legacy := legacyVersionRecords(t)
	state, err := commissioner.GovernanceState(legacy)
	if err != nil {
		t.Fatal(err)
	}
	auth := commissioner.GovernanceAuthorizer{State: state}
	want, err := commissioner.Reading("native.test", "creator", legacy, auth)
	if err != nil {
		t.Fatal(err)
	}
	for _, record := range legacy {
		if err := record.Validate(); err != nil {
			t.Fatal(err)
		}
		event := eventFixture(record.Receipt.Entry)
		event.League, event.Stream, event.Kind = record.Command.League, commissioner.Stream, record.Command.Command
		event.Actor, event.Scope, event.CorrelationID = record.Principal, record.Command.Visibility, record.Command.Request
		event.Payload = EventPayload{Version: 1, Commissioner: &record}
		if _, err := s.AppendEvent(t.Context(), event, record.Receipt.Revision-1); err != nil {
			t.Fatal(err)
		}
	}
	before := ledgerFileBytes(t, s)
	s = New(pools, s.Registry())
	records, err := s.leagueRecords(t.Context(), "native.test")
	if err != nil {
		t.Fatal(err)
	}
	for i, record := range records {
		if record.CommandVersion != commissioner.DurableCommandVersion {
			t.Fatalf("not upcast: %+v", record)
		}
		retry, err := s.ExecuteCommissioner(t.Context(), legacy[i].Command, "creator", nil)
		if err != nil || !reflect.DeepEqual(retry, legacy[i].Receipt) {
			t.Fatalf("legacy retry: %+v %v", retry, err)
		}
		again, err := record.Upcast()
		if err != nil || !reflect.DeepEqual(again, record) {
			t.Fatalf("non-idempotent upcast: %v", err)
		}
	}
	got, err := commissioner.Reading("native.test", "creator", records, auth)
	if err != nil || !reflect.DeepEqual(got, want) || !reflect.DeepEqual(commissioner.Replay(records), commissioner.Replay(legacy)) {
		t.Fatalf("replay changed: %+v want %+v: %v", got, want, err)
	}
	assertLedgerBytes(t, s, before)
}

func TestNewCommandsCarryDurableVersion(t *testing.T) {
	t.Parallel()
	s, _, _ := newStore(t)
	if _, err := s.ExecuteCommissioner(t.Context(), createCommand(), "creator", nil); err != nil {
		t.Fatal(err)
	}
	var payload string
	if err := s.pools.Read().QueryRowContext(t.Context(), `SELECT payload FROM league_event`).Scan(&payload); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(payload, `"commandVersion":1`) {
		t.Fatalf("missing durable envelope: %s", payload)
	}
}

func TestUnsupportedDurableVersionsLeaveFilesUnchanged(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name           string
		event, command int
	}{
		{"newer event", 2, 1}, {"newer command", 1, 2}, {"negative command", 1, -1},
	} {
		t.Run(test.name, func(t *testing.T) {
			s, _, _ := newStore(t)
			record := legacyVersionRecords(t)[0]
			record.CommandVersion = test.command
			event := eventFixture(record.Receipt.Entry)
			event.League, event.Stream, event.Kind = "native.test", commissioner.Stream, record.Command.Command
			event.Actor, event.Scope, event.CorrelationID = "creator", "league", record.Command.Request
			event.Payload = EventPayload{Version: test.event, Commissioner: &record}
			beforeAppend := ledgerFileBytes(t, s)
			if _, err := s.AppendEvent(t.Context(), event, 0); err == nil || !strings.Contains(err.Error(), "version") {
				t.Fatalf("unsupported append: %v", err)
			}
			assertLedgerBytes(t, s, beforeAppend)
			// Model a file written by another binary, bypassing this binary's append validation.
			tx, err := s.pools.Write().BeginTx(t.Context(), nil)
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = tx.Rollback() }()
			if _, err := s.insertEvent(t.Context(), tx, event, 0); err != nil {
				t.Fatal(err)
			}
			if err := tx.Commit(); err != nil {
				t.Fatal(err)
			}
			before := ledgerFileBytes(t, s)
			if _, err := s.Events(t.Context(), "native.test", commissioner.Stream, 0); err == nil || !strings.Contains(err.Error(), "version") {
				t.Fatalf("unsupported version decoded: %v", err)
			}
			if _, err := s.CommissionerLedger(t.Context(), "native.test", "creator", nil); err == nil || !strings.Contains(err.Error(), "version") {
				t.Fatalf("unsupported version replayed: %v", err)
			}
			if _, err := s.ExecuteCommissioner(t.Context(), createCommand(), "creator", nil); err == nil || !strings.Contains(err.Error(), "version") {
				t.Fatalf("unsupported version retried: %v", err)
			}
			raw := reverseCommand(record.Receipt.Entry, 1, "new")
			if _, err := s.ExecuteCommissioner(t.Context(), raw, "creator", nil); err == nil || !strings.Contains(err.Error(), "version") {
				t.Fatalf("appended over unsupported version: %v", err)
			}
			assertLedgerBytes(t, s, before)
		})
	}
}

func ledgerFileBytes(t *testing.T, s *Store) [][]byte {
	t.Helper()
	var sequence int
	var name, path string
	if err := s.pools.Read().QueryRowContext(t.Context(), `PRAGMA database_list`).Scan(&sequence, &name, &path); err != nil {
		t.Fatal(err)
	}
	out := make([][]byte, 2)
	for i, suffix := range []string{"", "-wal"} {
		body, err := os.ReadFile(path + suffix)
		if err != nil && !os.IsNotExist(err) {
			t.Fatal(err)
		}
		out[i] = body
	}
	return out
}

func assertLedgerBytes(t *testing.T, s *Store, before [][]byte) {
	t.Helper()
	after := ledgerFileBytes(t, s)
	for i := range before {
		if !bytes.Equal(before[i], after[i]) {
			t.Fatalf("database/WAL file %d changed", i)
		}
	}
}

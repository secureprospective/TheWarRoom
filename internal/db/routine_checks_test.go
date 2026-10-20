package db

import (
	"path/filepath"
	"testing"
	"time"
)

func TestRoutineChecksAreWeeklyAndRetainFindings(t *testing.T) {
	p, err := Open(t.Context(), filepath.Join(t.TempDir(), "checks.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = p.Close() }()
	at := time.Date(2026, 10, 20, 0, 0, 0, 0, time.UTC)
	if !p.LastCheck().At.IsZero() {
		t.Fatal("unchecked file reported a result")
	}
	p.CheckDue(t.Context(), at)
	first := p.LastCheck()
	if !first.At.Equal(at) || first.Error != "" || len(first.Findings.Integrity) != 1 || first.Findings.Integrity[0] != "ok" {
		t.Fatalf("first check: %+v", first)
	}
	if _, err := p.Write().ExecContext(t.Context(), `CREATE TABLE parent(id INTEGER PRIMARY KEY); CREATE TABLE child(parent INTEGER REFERENCES parent(id)); PRAGMA foreign_keys=OFF; INSERT INTO child VALUES (123); PRAGMA foreign_keys=ON;`); err != nil {
		t.Fatal(err)
	}
	p.CheckDue(t.Context(), at.Add(7*24*time.Hour-time.Nanosecond))
	if got := p.LastCheck(); !got.At.Equal(at) || len(got.Findings.ForeignKeys) != 0 {
		t.Fatalf("ran early: %+v", got)
	}
	p.CheckDue(t.Context(), at.Add(7*24*time.Hour))
	got := p.LastCheck()
	if got.At.Equal(at) || len(got.Findings.ForeignKeys) != 1 {
		t.Fatalf("missed weekly violation: %+v", got)
	}
	got.Findings.ForeignKeys[0].Table = "mutated"
	got.Findings.Integrity[0] = "mutated"
	if held := p.LastCheck(); held.Findings.ForeignKeys[0].Table != "child" || held.Findings.Integrity[0] != "ok" {
		t.Fatalf("caller changed findings: %+v", held)
	}
}

func TestRoutineCheckRetainsExecutionFailure(t *testing.T) {
	p, err := Open(t.Context(), filepath.Join(t.TempDir(), "closed.db"))
	if err != nil {
		t.Fatal(err)
	}
	if err := p.Close(); err != nil {
		t.Fatal(err)
	}
	at := time.Now().UTC()
	p.CheckDue(t.Context(), at)
	if got := p.LastCheck(); !got.At.Equal(at) || got.Error == "" {
		t.Fatalf("failure hidden: %+v", got)
	}
}

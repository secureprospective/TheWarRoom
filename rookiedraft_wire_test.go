package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/engine/rookiedraft"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

// This is the engine's wire shape, not a claim that the native draft IPC is wired.
func TestNativeDraftEngineWire(t *testing.T) {
	t.Parallel()
	rules := leaguerules.LegacyNFL().Draft
	rules.Rounds = leaguerules.Known(2)
	rules.Order = leaguerules.Known(map[int][]string{1: {"a", "b"}, 2: {"b", "a"}})
	owned := []rookiedraft.OwnedPick{
		{Round: 1, Original: "a", Owner: "b"}, {Round: 1, Original: "b", Owner: "b"},
		{Round: 2, Original: "a", Owner: "a"}, {Round: 2, Original: "b", Owner: "a"},
	}
	order, err := rookiedraft.Order(rules, []string{"a", "b"}, owned, nil)
	if err != nil {
		t.Fatal(err)
	}
	start := time.Date(2031, 7, 14, 10, 0, 0, 0, time.UTC)
	draft, err := rookiedraft.Replay(rules, order, start, start.Add(8*time.Hour), []rookiedraft.Selection{
		{Round: 1, Original: "a", Owner: "b", Player: "0001", At: start.Add(time.Hour)},
	})
	if err != nil {
		t.Fatal(err)
	}
	next, hasNext := draft.NextPick("a")
	body, err := json.MarshalIndent(struct {
		Order   []rookiedraft.RoundOrder `json:"order"`
		Draft   rookiedraft.Draft        `json:"draft"`
		Next    rookiedraft.Pick         `json:"nextOwnedPick"`
		HasNext bool                     `json:"hasNextOwnedPick"`
	}{order, draft, next, hasNext}, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	body = append(body, '\n')
	path := filepath.Join("internal", "engine", "rookiedraft", "testdata", "draft-engine.wire.json")
	if os.Getenv("TWR_WRITE_WIRE") == "1" {
		if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, body, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	held, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(held, body) {
		t.Fatalf("native draft engine fixture differs; regenerate with TWR_WRITE_WIRE=1\n%s", body)
	}
}

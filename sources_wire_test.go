package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestSourcesBindingGolden(t *testing.T) {
	a := targetTestApp(t)
	a.dataDir = t.TempDir()
	a.crosswalk = CrosswalkReport{OK: true, LoadedAt: "2026-10-11T18:00:00Z", Links: 1000, Promoted: 12,
		Rates: []CrosswalkRate{{Position: "QB", Rostered: 4, RosteredMatched: 3}}, Total: CrosswalkRate{Position: "ALL", Rostered: 4, RosteredMatched: 3},
		Unmatched: []CrosswalkMiss{{MFLID: "0042", Name: "Waiting", Position: "QB", Reason: "no NFL id yet"}}}
	ds, err := a.TargetDataSources()
	if err != nil {
		t.Fatal(err)
	}
	ds.Health.AsOf = "As of: fixture clock"
	// The not-ready load exercises the binding without a network request.
	stopped := &App{started: make(chan struct{}), startupErr: errors.New("fixture startup failed")}
	close(stopped.started)
	values := map[string][]byte{
		"LoadCrosswalk":       sourcesWire(t, stopped.LoadCrosswalk()),
		"GetCrosswalkReport":  sourcesWire(t, a.GetCrosswalkReport()),
		"TargetCrosswalk":     sourcesWire(t, a.TargetCrosswalk()),
		"TargetDataSources":   sourcesWire(t, ds),
		"TargetDataRefresh":   sourcesWire(t, a.TargetDataRefresh("unknown")),
		"SetLeagueArchiveDir": sourcesWire(t, a.SetLeagueArchiveDir("relative")),
	}
	for name, body := range values {
		path := filepath.Join("testdata", "wire", "sources", name+".json")
		if os.Getenv("TWR_WRITE_WIRE") == "1" {
			if e := os.MkdirAll(filepath.Dir(path), 0o750); e != nil {
				t.Fatal(e)
			}
			if e := os.WriteFile(path, body, 0o600); e != nil {
				t.Fatal(e)
			}
		}
		held, e := os.ReadFile(path)
		if e != nil {
			t.Fatal(e)
		}
		if string(held) != string(body) {
			t.Errorf("%s binding wire changed", name)
		}
	}
}

func sourcesWire[T any](t *testing.T, value T) []byte {
	t.Helper()
	b, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	return append(b, '\n')
}

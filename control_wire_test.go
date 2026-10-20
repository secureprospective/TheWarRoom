package main

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

type controlWireResult[T any] struct {
	Value T      `json:"value"`
	Error string `json:"error"`
}

func controlResult[T any](value T, err error) controlWireResult[T] {
	text := ""
	if err != nil {
		text = err.Error()
	}
	return controlWireResult[T]{Value: value, Error: text}
}

func TestControlBindingGolden(t *testing.T) {
	a := &App{started: make(chan struct{}), startupErr: errors.New("fixture startup failed")}
	close(a.started)
	values := map[string][]byte{
		"GetParams":               sourcesWire(t, a.GetParams()),
		"SetParam":                sourcesWire(t, a.SetParam("model.k_now", "", 4)),
		"ResetParam":              sourcesWire(t, a.ResetParam("model.k_now", "")),
		"TargetLeagueSettings":    sourcesWire(t, controlResult(a.TargetLeagueSettings())),
		"TargetSiteNews":          sourcesWire(t, controlResult(a.TargetSiteNews())),
		"TargetAccounting":        sourcesWire(t, a.TargetAccounting()),
		"TargetBylaws":            sourcesWire(t, a.TargetBylaws()),
		"TargetOpenMFLAccounting": sourcesWire(t, controlResult(struct{}{}, a.TargetOpenMFLAccounting())),
		"TargetAlerts":            sourcesWire(t, controlResult(a.TargetAlerts("0001"))),
		"TargetSeason":            sourcesWire(t, controlResult(a.TargetSeason())),
		"TargetShellStatus":       sourcesWire(t, controlResult(a.TargetShellStatus())),
		"TargetSetWindow":         sourcesWire(t, controlResult(struct{}{}, a.TargetSetWindow("unknown", ""))),
		"MFLKeyStatus":            sourcesWire(t, controlResult(a.MFLKeyStatus())),
		"SetMFLKey":               sourcesWire(t, controlResult(a.SetMFLKey(""))),
		"DeleteMFLKey":            sourcesWire(t, controlResult(a.DeleteMFLKey())),
	}
	for name, body := range values {
		path := filepath.Join("testdata", "wire", "control", name+".json")
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
		if string(held) != string(body) {
			t.Errorf("%s binding wire changed", name)
		}
	}
}

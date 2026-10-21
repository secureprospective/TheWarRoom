package comms_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	commsapp "github.com/secureprospective/TheWarRoom/internal/app/comms"
	"github.com/secureprospective/TheWarRoom/internal/app/comms/commstest"
	"github.com/secureprospective/TheWarRoom/internal/app/report"
	"github.com/secureprospective/TheWarRoom/internal/facts"
	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
	"github.com/secureprospective/TheWarRoom/internal/normalize"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

type fakeHost struct {
	in         commsapp.Input
	startup    error
	archiveErr error
	open       func(string)
	refresh    func(bool) error
	reads      int
}

func (h *fakeHost) Ready() error                  { return h.startup }
func (h *fakeHost) CommsContext() context.Context { return context.Background() }
func (h *fakeHost) CommsFeeds() (report.Held[leaguefeed.Transaction], [2]report.Held[ingestion.RawLeaguePost], *time.Location) {
	h.reads++
	return h.in.Transactions, h.in.Boards, h.in.Zone
}
func (h *fakeHost) CommsLeague() (normalize.Lookup, map[string]string, state.MirrorSnapshot) {
	return h.in.Lookup, h.in.Names, h.in.Mirror
}
func (h *fakeHost) CommsArchive(context.Context) (facts.RosterWeekColumns, facts.NflGameColumns, error) {
	return facts.RosterWeekColumns{}, facts.NflGameColumns{}, h.archiveErr
}
func (h *fakeHost) LogFallback(string) {}
func (h *fakeHost) RefreshComms(force bool) error {
	if h.startup != nil {
		return fmt.Errorf("comms refresh: %w", h.startup)
	}
	return h.refresh(force)
}
func (h *fakeHost) CommsDestination() (string, int) { return "", 2026 }
func (h *fakeHost) OpenBrowser(url string)          { h.open(url) }

func TestAPIValidatesBeforeHostRead(t *testing.T) {
	host := &fakeHost{startup: errors.New("startup failed")}
	api := commsapp.New(host)
	if _, err := api.TargetComms(commsapp.RawCommsScope{Kind: "post"}); err == nil || strings.Contains(err.Error(), "startup") {
		t.Fatal(err)
	}
	if _, err := api.TargetComms(commsapp.RawCommsScope{}); err == nil || err.Error() != "comms: startup: startup failed" {
		t.Fatal(err)
	}
	if err := api.TargetOpenComms("league-board"); err == nil || err.Error() != "open comms: startup failed" {
		t.Fatal(err)
	}
	if err := api.TargetCommsRefresh(true); err == nil || err.Error() != "comms refresh: startup failed" {
		t.Fatal(err)
	}
	if host.reads != 0 {
		t.Fatal("read before ready")
	}
}
func TestAPIRefreshForwardsForce(t *testing.T) {
	for _, force := range []bool{false, true} {
		called := false
		api := commsapp.New(&fakeHost{refresh: func(got bool) error {
			called = true
			if got != force {
				t.Fatal(got)
			}
			return nil
		}})
		if err := api.TargetCommsRefresh(force); err != nil || !called {
			t.Fatal(err, called)
		}
	}
}
func TestCommsHeldBindingAndWire(t *testing.T) {
	in := commstest.Real(t)
	a := commsapp.New(&fakeHost{in: in})
	out, err := a.TargetComms(commsapp.RawCommsScope{})
	if err != nil || len(out.Events) != len(in.Transactions.Value) {
		t.Fatal(err)
	}

	events, rows := out.Events[:1:1], out.Transactions.Rows[:1:1]
	for _, phrase := range []string{" traded ", " added "} {
		for i, event := range out.Events {
			if !strings.Contains(event.Sentence, phrase) {
				continue
			}
			events = append(events, event)
			rows = append(rows, out.Transactions.Rows[i])
			break
		}
	}
	out.Events, out.Transactions.Rows = events, rows
	body, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	path := commstest.RepoPath("frontend", "src", "app", "comms", "wire.testdata.json")

	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(want)) != string(body) {
		t.Fatal("Comms wire is stale")
	}
}

func TestCommsBindingGoldens(t *testing.T) {
	in := commstest.Real(t)
	in.Names = map[string]string{} // The original root fixture had no loaded franchise directory.
	host := &fakeHost{in: in}
	a := commsapp.New(host)
	for _, tc := range []struct {
		name  string
		scope commsapp.RawCommsScope
	}{
		{"all", commsapp.RawCommsScope{}}, {"trade", commsapp.RawCommsScope{Kind: "trade", Team: "0001"}},
		{"invalid", commsapp.RawCommsScope{Kind: "post"}}, {"unknown", commsapp.RawCommsScope{Team: "9999"}},
	} {
		board, err := a.TargetComms(tc.scope)
		checkCommsGolden(t, tc.name, struct {
			Board commsapp.CommsBoard
			Error string
		}{board, commsError(err)})
	}
	opened := ""
	host.open = func(url string) { opened = url }
	for _, destination := range []string{"league-board", "polls", "invalid"} {
		opened = ""
		err := a.TargetOpenComms(destination)
		checkCommsGolden(t, "open-"+destination, struct{ URL, Error string }{opened, commsError(err)})
	}
}
func commsError(err error) string {
	if err != nil {
		return err.Error()
	}
	return ""
}

func checkCommsGolden[T any](t *testing.T, name string, value T) {
	t.Helper()
	got, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	got = append(got, '\n')
	dir := commstest.RepoPath("testdata", "wire", "comms")
	path := filepath.Join(dir, name+".json")
	if os.Getenv("TWR_WRITE_WIRE") == "1" {
		if err := os.MkdirAll(dir, 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, got, 0o600); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("%s: Comms binding contract changed", path)
	}
}

func TestCommandCatalog(t *testing.T) {
	commands := commsapp.Commands()
	ids := map[string]bool{}
	for _, command := range commands {
		if command.ID == "" || ids[command.ID] || command.Method == "" || command.Version == "" || command.Role == "" || command.Gravity == "" {
			t.Fatal(command)
		}
		ids[command.ID] = true
		if command.Availability != "both" && command.Availability != "server" && command.Availability != "desktop" {
			t.Fatal(command)
		}
		if command.Method == "TargetOpenComms" && command.Availability != "desktop" {
			t.Fatal("browser handoff is desktop-only")
		}
	}
	if len(commands) != 3 || !ids["comms.read"] || !ids["comms.refresh"] || !ids["comms.open"] {
		t.Fatal(commands)
	}
	commands[0].ID = "changed"
	if commsapp.Commands()[0].ID != "comms.read" {
		t.Fatal("catalog storage leaked")
	}
}

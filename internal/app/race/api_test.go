package race

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/report"
	"github.com/secureprospective/TheWarRoom/internal/app/season"
)

type fakeHost struct {
	input                season.Snapshot
	readyErr, refreshErr error
	calls                []string
	useHeld              bool
}

func (h *fakeHost) Ready() error { h.calls = append(h.calls, "ready"); return h.readyErr }
func (h *fakeHost) RaceInput(snapshot func() season.Held) season.Snapshot {
	h.calls = append(h.calls, "input")
	in := h.input
	if h.useHeld {
		in.Held = snapshot()
	}
	return in
}
func (h *fakeHost) RefreshRace(_ bool, _ func() season.Held) error {
	h.calls = append(h.calls, "refresh")
	return h.refreshErr
}
func repoPath(parts ...string) string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(append([]string{filepath.Dir(file), "..", "..", ".."}, parts...)...)
}
func TestRaceWireMatchesGo(t *testing.T) {
	t.Parallel()
	in := realRace(t, "2026")
	in.Held.Power = map[string]season.PowerRankingsResult{season.ViewSeason: testPower()}
	board, err := New(&fakeHost{input: in}, &Held{}).TargetRace(in.Week, in.Team, in.Mine)
	if err != nil {
		t.Fatal(err)
	}
	board.Standings.Rows = board.Standings.Rows[:5]
	for i := range board.Board.Reports {
		r := &board.Board.Reports[i]
		r.Rows = r.Rows[:min(len(r.Rows), 3)]
	}
	body, err := json.MarshalIndent(struct {
		Race RaceBoard `json:"race"`
	}{board}, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	file := repoPath("frontend", "src", "app", "race", "wire.testdata.json")
	body = append(body, '\n')
	if os.Getenv("TWR_WRITE_WIRE") == "1" {
		if err := os.WriteFile(file, body, 0o644); err != nil { //nolint:gosec // a committed test fixture
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	if string(want) != string(body) {
		t.Fatal("race wire stale; TWR_WRITE_WIRE=1 go test -run TestRaceWireMatchesGo .")
	}
}

func TestRaceWireTwins(t *testing.T) {
	b, err := Build(realRace(t, "2026"))
	if err != nil {
		t.Fatal(err)
	}
	for _, board := range []Board{{}, {Weeks: []report.Choice{}, Teams: []report.Choice{}}, b} {
		want, err := json.Marshal(board)
		if err != nil {
			t.Fatal(err)
		}
		got, err := json.Marshal(wireRace(board))
		if err != nil {
			t.Fatal(err)
		}
		if string(want) != string(got) {
			t.Fatal("race wire twin changed")
		}
	}
}

func TestRaceAPIStartupDenialAndOrder(t *testing.T) {
	h := &fakeHost{readyErr: errors.New("fixture startup failed")}
	_, err := New(h, &Held{}).TargetRace(40, "unknown", "unknown")
	if err == nil || err.Error() != "race: startup: fixture startup failed" || !reflect.DeepEqual(h.calls, []string{"ready"}) {
		t.Fatal(err, h.calls)
	}
	h = &fakeHost{input: season.Snapshot{Zone: time.UTC}}
	api := New(h, &Held{})
	if _, err = api.TargetRace(40, "", ""); err == nil || err.Error() != "race: week 40 has no scored games" || !reflect.DeepEqual(h.calls, []string{"ready", "input"}) {
		t.Fatal(err, h.calls)
	}
	h.refreshErr = errors.New("race refresh: fixture failure")
	if err = api.TargetRaceRefresh(true); err != h.refreshErr {
		t.Fatal(err)
	}
}

func TestRaceAPIUsesItsHolder(t *testing.T) {
	in := realRace(t, "2026")
	held := &Held{}
	publish(held, in.Held)
	in.Held = season.Held{}
	h := &fakeHost{input: in, useHeld: true}
	b, err := New(h, held).TargetRace(0, "0001", "0001")
	if err != nil || len(b.Standings.Rows) == 0 || !reflect.DeepEqual(h.calls, []string{"ready", "input"}) {
		t.Fatal(err, h.calls)
	}
}

func TestRaceWireGoldens(t *testing.T) {
	for _, name := range []string{"unfetched", "held"} {
		in := season.Snapshot{Zone: time.UTC}
		if name == "held" {
			in = realRace(t, "2026")
			in.Held.Power = map[string]season.PowerRankingsResult{season.ViewSeason: testPower()}
		}
		b, err := New(&fakeHost{input: in}, &Held{}).TargetRace(in.Week, in.Team, in.Mine)
		if err != nil {
			t.Fatal(err)
		}
		if name == "held" {
			b.Standings.Rows = b.Standings.Rows[:5]
			for i := range b.Board.Reports {
				r := &b.Board.Reports[i]
				r.Rows = r.Rows[:min(len(r.Rows), 3)]
			}
		}
		got, err := json.MarshalIndent(b, "", " ")
		if err != nil {
			t.Fatal(err)
		}
		want, err := os.ReadFile(repoPath("testdata", "wire", "race", name+".json"))
		if err != nil {
			t.Fatal(err)
		}
		if string(want) != string(append(got, '\n')) {
			t.Fatal(name, "wire golden changed")
		}
	}
}

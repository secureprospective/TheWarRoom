package draft

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/draft/drafttest"
	"github.com/secureprospective/TheWarRoom/internal/app/report"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/draftreports"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/players"
	"github.com/secureprospective/TheWarRoom/internal/lab"
	"github.com/secureprospective/TheWarRoom/internal/normalize"
)

type fakeHost struct {
	startup error
	load    func() (*lab.Data, error)
	calls   []string
	in      Input
	players report.Held[players.RawPlayer]
	refresh func(bool) error
	opened  string
	target  string
	lookup  normalize.Lookup
}

func (h *fakeHost) Ready() error { h.calls = append(h.calls, "ready"); return h.startup }
func (h *fakeHost) DraftReady() (context.Context, int, *time.Location, error) {
	return context.Background(), 2026, time.UTC, h.Ready()
}
func (h *fakeHost) DraftFacts() (*lab.Data, string, error) {
	d, err := h.load()
	return d, "test build", err
}

func (h *fakeHost) DraftInputs() (report.Held[draftreports.Pick], report.Held[draftreports.FuturePick], normalize.Lookup) {
	h.calls = append(h.calls, "feeds", "lookup")
	return report.Held[draftreports.Pick]{Value: h.in.Results.Value, Feed: h.in.Results.Feed},
		report.Held[draftreports.FuturePick]{Value: h.in.Future.Value, Feed: h.in.Future.Feed}, h.lookup
}
func (h *fakeHost) DraftLeague() (int, map[string]string, *time.Location) {
	h.calls = append(h.calls, "league")
	return 2026, h.in.Franchises, time.UTC
}
func (h *fakeHost) DraftValues(context.Context) (map[string]float64, string) {
	h.calls = append(h.calls, "values")
	return h.in.Values, h.in.ValueNote
}
func (h *fakeHost) DraftNext() string { h.calls = append(h.calls, "next"); return h.in.Next }
func (h *fakeHost) DraftDestination() string {
	h.calls = append(h.calls, "destination")
	return h.target
}
func (h *fakeHost) DraftPlayers(context.Context) report.Held[players.RawPlayer] { return h.players }
func (h *fakeHost) RefreshDraftReports(force bool) error                        { return h.refresh(force) }

func newTestAPI(t *testing.T) *API {
	t.Helper()
	return New(&fakeHost{load: sync.OnceValues(lab.Load)})
}

func TestAPIReadyBeforeReads(t *testing.T) {
	h := &fakeHost{startup: errors.New("not ready")}
	a := New(h)
	if _, err := a.TargetDraftReports(); err == nil || err.Error() != "draft reports: startup: not ready" {
		t.Fatal(err)
	}
	if _, err := a.TargetNFLDraft(); err == nil || err.Error() != "NFL draft: startup: not ready" {
		t.Fatal(err)
	}
	if err := a.TargetOpenMFLDraftRoom(); err == nil || err.Error() != "open draft room: not ready" {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(h.calls, []string{"ready", "ready", "ready"}) {
		t.Fatal(h.calls)
	}
	h.load = func() (*lab.Data, error) { t.Fatal("invalid scope reached facts"); return nil, nil }
	if New(h).TargetDraftLab(RawDraftScope{}).Status != report.ReadingFailed {
		t.Fatal("invalid input accepted")
	}
}

func TestAPIReportsReadOrderAndForce(t *testing.T) {
	h := &fakeHost{in: Input{Next: "Next draft: no date set by the commissioner."}}
	a := New(h)
	if _, err := a.TargetDraftReports(); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(h.calls, []string{"ready", "feeds", "lookup", "values", "next", "league", "destination"}) {
		t.Fatal(h.calls)
	}
	for _, force := range []bool{false, true} {
		h.refresh = func(got bool) error {
			if got != force {
				t.Fatal(got)
			}
			return errors.New("original refresh refusal")
		}
		if err := a.TargetDraftReportsRefresh(force); err == nil || err.Error() != "original refresh refusal" {
			t.Fatal(err)
		}
	}
	if err := a.TargetOpenMFLDraftRoom(); err == nil || !strings.Contains(err.Error(), "MFL host not held") {
		t.Fatal(err)
	}
	h.target = "https://discovered.example/2026/options?L=fixture&O=17"
	if err := a.TargetOpenMFLDraftRoom(); err != nil {
		t.Fatal(err)
	}
	board, err := a.TargetDraftReports()
	if err != nil || h.opened != board.MFLURL || !strings.HasSuffix(h.opened, "&O=17") {
		t.Fatal(err, h.opened)
	}

}

func TestAPIReportInputsRemainUnchanged(t *testing.T) {
	d := drafttest.Held(t)
	in := Input{Season: 2026, Results: Held[draftreports.Pick]{Value: d.Picks, Feed: d.Read},
		Future: Held[draftreports.FuturePick]{Value: d.Future, Feed: d.FutureRead}, Names: d.Names,
		Franchises: d.Franchises, Values: d.Values, ValueNote: d.ValueNote, Next: d.Next, Zone: time.UTC}
	h := &fakeHost{in: in}
	before := testBoard(t)
	a := New(h)
	first, err := a.TargetDraftReports()
	if err != nil {
		t.Fatal(err)
	}
	first.Reports[0].Rows[0].Cells[0] = "mutated"
	again, err := a.TargetDraftReports()
	if err != nil || again.Reports[0].Rows[0].Cells[0] != before.Reports[0].Rows[0].Cells[0] || in.Results.Value[0].Round != d.Picks[0].Round {
		t.Fatal("report retained output storage", err)
	}
}

func (h *fakeHost) OpenDraftRoom() error {
	if err := h.Ready(); err != nil {
		return fmt.Errorf("open draft room: %w", err)
	}
	target := h.DraftDestination()
	if target == "" {
		return fmt.Errorf("open draft room: MFL host not held; refresh first")
	}
	h.opened = target
	return nil
}

func TestAPIHeldWireMatchesCanonicalGolden(t *testing.T) {
	in := input(drafttest.Held(t), time.UTC)
	rows := []players.RawPlayer{}
	for id, name := range in.Names {
		rows = append(rows, players.RawPlayer{ID: id, Name: name, Position: "WR"})
	}
	lookup, err := normalize.NewLookup(rows)
	if err != nil {
		t.Fatal(err)
	}
	got, err := New(&fakeHost{in: in, lookup: lookup}).TargetDraftReports()
	if err != nil {
		t.Fatal(err)
	}
	body, err := json.MarshalIndent(got, "", " ")
	if err != nil {
		t.Fatal(err)
	}
	want, err := os.ReadFile(drafttest.RepoPath("testdata", "wire", "draft", "reports-held.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(append(body, '\n'), want) {
		t.Fatal("held API wire changed")
	}
}

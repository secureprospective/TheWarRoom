package archive

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"reflect"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/app/archive/archivetest"
	"github.com/secureprospective/TheWarRoom/internal/app/report"
)

type fakeHost struct {
	snapshot          Snapshot
	stale             bool
	readyErr, readErr error
	calls             []string
	opponent          string
}

func (h *fakeHost) Ready() error                    { h.calls = append(h.calls, "ready"); return h.readyErr }
func (h *fakeHost) ArchiveContext() context.Context { return context.Background() }
func (h *fakeHost) ArchiveOpponent(string) string {
	h.calls = append(h.calls, "opponent")
	return h.opponent
}
func (h *fakeHost) ArchiveFacts(ctx context.Context) (report.ArchiveSnapshot, bool, error) {
	h.calls = append(h.calls, "facts")
	if _, ok := ctx.Deadline(); !ok {
		panic("archive read has no deadline")
	}
	return h.snapshot, h.stale, h.readErr
}

func TestArchiveWireMatchesGolden(t *testing.T) {
	cache := reduced(t)
	for _, c := range []struct {
		name, team, other string
		season            int
		stale             bool
		snapshot          Snapshot
	}{
		{name: "unimported"},
		{name: "league", snapshot: cache},
		{name: "season-stale", season: 2015, stale: true, snapshot: cache},
		{name: "franchise", team: "0001", other: "0002", snapshot: cache},
	} {
		h := &fakeHost{snapshot: c.snapshot, stale: c.stale}
		b, err := New(h).TargetArchive(c.season, c.team, c.other)
		if err != nil {
			t.Fatal(c.name, err)
		}
		got, err := json.MarshalIndent(b, "", " ")
		if err != nil {
			t.Fatal(err)
		}
		want, err := os.ReadFile(archivetest.RepoPath("testdata", "wire", "archive", c.name+".json"))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(want, append(got, '\n')) {
			t.Errorf("%s differs from the Archive binding's output", c.name)
		}
	}
}

func TestArchiveStartupDenial(t *testing.T) {
	h := &fakeHost{readyErr: errors.New("fixture startup failure")}
	_, err := New(h).TargetArchive(-1, "invalid", "invalid")
	if err == nil || err.Error() != "archive: startup: fixture startup failure" || !reflect.DeepEqual(h.calls, []string{"ready"}) {
		t.Fatal(err, h.calls)
	}
}

func TestArchiveReadOrderAndDenials(t *testing.T) {
	cache := reduced(t)
	for _, tc := range []struct {
		name, team, other, want string
		season                  int
	}{
		{name: "default", team: "0001"},
		{name: "explicit", team: "0001", other: "0002"},
		{name: "bad season", season: 1900, want: "archive: season is not in the snapshot"},
		{name: "bad team", team: "invalid", want: "archive: franchise is not in the snapshot"},
		{name: "bad other", other: "invalid", want: "archive: franchise is not in the snapshot"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := &fakeHost{snapshot: cache, opponent: "0002"}
			b, err := New(h).TargetArchive(tc.season, tc.team, tc.other)
			if tc.want != "" {
				if err == nil || err.Error() != tc.want {
					t.Fatal(err)
				}
			} else if err != nil || b.Other != "0002" {
				t.Fatal(b.Other, err)
			}
			calls := []string{"ready", "facts"}
			if tc.other == "" {
				calls = []string{"ready", "opponent", "facts"}
			}
			if !reflect.DeepEqual(h.calls, calls) {
				t.Fatal(h.calls)
			}
		})
	}
	h := &fakeHost{readErr: errors.New("archive: fixture read failure")}
	_, err := New(h).TargetArchive(1900, "invalid", "")
	if err != h.readErr {
		t.Fatal(err)
	}
}

func TestArchiveWireTwinsMatch(t *testing.T) {
	for _, b := range []Board{
		{},
		{Board: report.Board{Reports: []report.Report{{}}}, Seasons: []report.Choice{}, Teams: []report.Choice{}},
		populatedBoard(t),
	} {
		want, err := json.Marshal(b)
		if err != nil {
			t.Fatal(err)
		}
		got, err := json.Marshal(wireArchive(b))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(want, got) {
			t.Fatalf("archive wire twin differs: %s != %s", want, got)
		}
	}
}

func populatedBoard(t *testing.T) Board {
	t.Helper()
	b, err := Build(reduced(t), false, 2015, "0001", "0002")
	if err != nil {
		t.Fatal(err)
	}
	return b
}

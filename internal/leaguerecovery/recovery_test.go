package leaguerecovery

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/measures"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

func TestOfflineStandUpExclusivityAndValidation(t *testing.T) {
	t.Parallel()
	for _, mode := range []string{"success", "occupied", "invalid", "newer", "missing"} {
		t.Run(mode, func(t *testing.T) {
			t.Parallel()
			source := filepath.Join(t.TempDir(), "source.db")
			p, err := db.Open(t.Context(), source, history.SchemaVersion())
			if err != nil {
				t.Fatal(err)
			}
			s := history.New(p, &measures.Registry{})
			if err := s.Initialize(t.Context()); err != nil {
				t.Fatal(err)
			}
			body, err := os.ReadFile("../commissioner/testdata/native-assets.wire.json")
			if err != nil {
				t.Fatal(err)
			}
			var held commissioner.NativeAssets
			if err := json.Unmarshal(body, &held); err != nil {
				t.Fatal(err)
			}
			if held.Snapshot == nil {
				t.Fatal("missing snapshot")
			}
			point := commissioner.RawRosterRestorePoint{Version: 1, League: "native.recovery", Revision: 42, CapturedAt: time.Now().UTC(), Snapshot: *held.Snapshot}
			point.Snapshot.Picks = nil
			point.Snapshot.Pool = nil
			point.Snapshot.Champions = nil
			point.Snapshot.Champion = ""
			point.Snapshot.StandingsOrder = nil
			if err := point.Validate(); err != nil {
				t.Fatal(err)
			}
			if mode == "invalid" {
				point.Snapshot.Players[0].Salary = -1
			}
			payload, err := json.Marshal(point)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := p.Write().ExecContext(t.Context(), `INSERT INTO league_roster_restore VALUES(?,?,?)`, point.League, point.Revision, string(payload)); err != nil {
				t.Fatal(err)
			}
			if mode == "newer" {
				if _, err := p.Write().ExecContext(t.Context(), `PRAGMA user_version=999`); err != nil {
					t.Fatal(err)
				}
			}
			if err := p.Close(); err != nil {
				t.Fatal(err)
			}
			original, err := os.ReadFile(source)
			if err != nil {
				t.Fatal(err)
			}
			target := filepath.Join(t.TempDir(), "new-league")
			if mode == "occupied" {
				if err := os.Mkdir(target, 0700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(target, "sentinel"), []byte("keep"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			if mode == "missing" {
				source = filepath.Join(t.TempDir(), "missing.db")
			}
			_, err = StandUp(t.Context(), RawRequest{Source: source, League: point.League, Destination: target})
			if mode != "success" {
				if err == nil {
					t.Fatal("unsafe recovery succeeded")
				}
				if mode == "occupied" {
					body, err := os.ReadFile(filepath.Join(target, "sentinel"))
					if err != nil || string(body) != "keep" {
						t.Fatal("existing destination changed")
					}
				} else if _, err := os.Lstat(target); !os.IsNotExist(err) {
					t.Fatal("failed recovery left a destination")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			info, err := os.Stat(target)
			if err != nil || info.Mode().Perm() != 0700 {
				t.Fatal("recovery directory is not private")
			}
			after, err := os.ReadFile(source)
			if err != nil || !reflect.DeepEqual(original, after) {
				t.Fatal("source database changed")
			}
			restored, err := db.Open(t.Context(), filepath.Join(target, "history.db"), history.SchemaVersion())
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = restored.Close() }()
			recovered := history.New(restored, &measures.Registry{})
			got, err := recovered.RosterRestorePoint(t.Context(), point.League)
			if err != nil || !reflect.DeepEqual(got.Snapshot, point.Snapshot) {
				t.Fatalf("recovered league differs: %v", err)
			}
		})
	}
}

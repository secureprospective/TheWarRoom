package history

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/domain"
)

func TestRosterRecoveryRoundTrip(t *testing.T) {
	t.Parallel()
	for _, backup := range []bool{false, true} {
		for _, status := range []domain.RosterStatus{domain.RosterActive, domain.RosterIR, domain.RosterTaxi} {
			t.Run(string(status)+map[bool]string{false: "/point", true: "/R2-snapshot"}[backup], func(t *testing.T) {
				t.Parallel()
				source, _, _ := newStore(t)
				if _, err := source.ExecuteCommissioner(t.Context(), createWith(commissioner.PresetLegacy, 0), "creator", nil); err != nil {
					t.Fatal(err)
				}
				seed := heldSeedCommand(t)
				seed.Args.Native.Players[0].Status = status
				seed.Args.Native.Players[0].Salary = 123456789
				seed.Args.Native.Players[0].ExpirationYear = 2031
				if _, err := source.ExecuteCommissioner(t.Context(), seed, "creator", nil); err != nil {
					t.Fatal(err)
				}
				if backup {
					path := filepath.Join(t.TempDir(), "r2-payload.db")
					if err := source.pools.CaptureHosted(t.Context(), path, func() error { return nil }); err != nil {
						t.Fatal(err)
					}
					p, err := db.Open(t.Context(), path, SchemaVersion())
					if err != nil {
						t.Fatal(err)
					}
					defer func() { _ = p.Close() }()
					source = New(p, source.reg)
					if err := source.Initialize(t.Context()); err != nil {
						t.Fatal(err)
					}
				}
				point, err := source.RosterRestorePoint(t.Context(), seed.League)
				if err != nil {
					t.Fatal(err)
				}
				target, _, _ := newStore(t)
				message, err := target.StandUpRosters(t.Context(), point)
				if err != nil {
					t.Fatal(err)
				}
				assets, err := target.NativeAssets(t.Context(), point.League)
				if err != nil || assets.Snapshot == nil {
					t.Fatalf("recovered assets: %v", err)
				}
				recovered, err := target.RosterRestorePoint(t.Context(), point.League)
				if err != nil || !reflect.DeepEqual(recovered.Snapshot, point.Snapshot) {
					t.Fatalf("round trip changed cents, years, statuses or directory: %v", err)
				}
				if !reflect.DeepEqual(assets.Snapshot.Players, point.Snapshot.Players) {
					t.Fatal("replay differs from point")
				}
				if !strings.Contains(message, "read-only") {
					t.Fatal(message)
				}
				location, err := target.TransferStatus(t.Context(), point.League)
				if err != nil || location.Writable {
					t.Fatalf("recovery granted authority: %+v %v", location, err)
				}
				before := historyJSON(t, target.pools.Read(), `SELECT json_group_array(payload) FROM league_event`)
				if _, err := target.StandUpRosters(t.Context(), point); err == nil {
					t.Fatal("overwrote recovered league")
				}
				if after := historyJSON(t, target.pools.Read(), `SELECT json_group_array(payload) FROM league_event`); after != before {
					t.Fatal("failed retry changed ledger")
				}
			})
		}
	}
}

func TestRosterRecoveryPreservesSourceAndPreset(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, preset, source string
		teams                int
		old                  bool
	}{
		{"native 12", commissioner.PresetEmpty, commissioner.SourceNative, 12, false},
		{"native Legacy rules 12", commissioner.PresetLegacy, commissioner.SourceNative, 12, false},
		{"native Default32", commissioner.PresetDefault32, commissioner.SourceNative, 32, false},
		{"Legacy 32", commissioner.PresetLegacy, commissioner.SourceMFLImport, 32, false},
		{"pre-source Legacy 32", commissioner.PresetLegacy, commissioner.SourceMFLImport, 32, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			point := playedRosterPoint(t, tc.preset, tc.source, tc.teams)
			if point.Source != tc.source || point.Preset != tc.preset {
				t.Fatalf("captured source/preset=%q/%q", point.Source, point.Preset)
			}
			if tc.old {
				point.Source, point.Preset = "", ""
			}
			target, _, _ := newStore(t)
			if _, err := target.StandUpRosters(t.Context(), point); err != nil {
				t.Fatal(err)
			}
			recovered, err := target.RosterRestorePoint(t.Context(), point.League)
			if err != nil || !reflect.DeepEqual(recovered.Snapshot, point.Snapshot) ||
				recovered.Source != tc.source || recovered.Preset != tc.preset {
				t.Fatalf("source, preset or contracts changed: %+v %v", recovered, err)
			}
			records, err := target.nativeRecords(t.Context(), point.League)
			if err != nil || records[0].Command.Args.Source != tc.source || records[0].Command.Args.Preset != tc.preset ||
				len(records[0].Command.Args.Franchises) != tc.teams {
				t.Fatalf("recovered creation: %+v %v", records, err)
			}
			location, err := target.TransferStatus(t.Context(), point.League)
			if err != nil || location.Writable {
				t.Fatalf("recovery granted authority: %+v %v", location, err)
			}
		})
	}
}

func playedRosterPoint(t *testing.T, preset, source string, teams int) commissioner.RawRosterRestorePoint {
	t.Helper()
	create := createWith(preset, teams)
	create.Args.Source = source
	create, err := create.Prepared()
	if err != nil {
		t.Fatal(err)
	}
	seed := heldSeedCommand(t)
	seed.Args.Native.Franchises = seed.Args.Native.Franchises[:teams]
	seed.Args.Native.Picks = nil
	seed.Args.Native.Players[0].Salary = 123456789
	seed.Args.Native.Players[0].ExpirationYear = 2031
	at := seed.Args.Native.RosterAt
	records := []commissioner.Record{
		{Command: create, Principal: "creator", Receipt: commissioner.Receipt{Entry: "create", Revision: 1, RecordedAt: at}},
		{Command: seed, Principal: "creator", Receipt: commissioner.Receipt{Entry: "played", Revision: 2, RecordedAt: at}},
	}
	point, seeded, err := commissioner.RosterRestorePoint(seed.League, records)
	if err != nil || !seeded {
		t.Fatalf("capture played rosters: seeded=%v error=%v", seeded, err)
	}
	return point
}

package commissioner

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
)

func TestAppCreationStartsBySource(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, preset, source string
		teams, lots          int
		started              bool
	}{
		{"default32", commissioner.PresetDefault32, commissioner.SourceNative, 32, 32, true},
		{"empty", commissioner.PresetEmpty, commissioner.SourceNative, 12, 0, true},
		{"native Legacy rules", commissioner.PresetLegacy, commissioner.SourceNative, 12, 0, true},
		{"import Default32 rules", commissioner.PresetDefault32, commissioner.SourceMFLImport, 32, 0, false},
		{"Legacy compatibility", commissioner.PresetLegacy, "", 32, 0, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			app := commissionerTestApp(t)
			opsClock(app, 9)
			raw := opsCommand(commissioner.Create, 0, "create", commissioner.RawArgs{Name: "Our league",
				Preset: tc.preset, Source: tc.source, Season: 2026, FranchiseCount: tc.teams})
			raw.Reason = "Start our league"
			first := executeCommissionerTest(t, app, raw)
			assets, err := app.TargetNativeAssets(raw.League)
			if err != nil || assets.Seeded != tc.started {
				t.Fatalf("assets: %+v %v", assets, err)
			}
			records, err := app.CommissionerRecords(t.Context(), raw.League)
			if err != nil {
				t.Fatal(err)
			}
			wantRecords := 1
			if tc.started {
				wantRecords = 2
				seed := records[len(records)-1].Command
				if seed.Command != commissioner.NativeSeed || seed.Reason != raw.Reason || len(seed.Args.Native.Pool) != tc.lots ||
					len(seed.Args.Native.Players) != 0 || len(seed.Args.Native.Picks) != 0 {
					t.Fatalf("start: %+v", seed)
				}
				if tc.preset == commissioner.PresetDefault32 {
					old, err := commissioner.NativeStart(raw.League, records[:1], records[0].Receipt.RecordedAt)
					if err != nil {
						t.Fatal(err)
					}
					before, err := json.Marshal(old)
					if err != nil {
						t.Fatal(err)
					}
					after, err := json.Marshal(seed.Args.Native)
					if err != nil || !bytes.Equal(before, after) {
						t.Fatalf("starting bytes changed: %s / %s: %v", before, after, err)
					}
				}
			}
			if len(records) != wantRecords {
				t.Fatalf("records: %d, want %d", len(records), wantRecords)
			}
			jobs, err := app.TargetCommissionerJobs(raw.League)
			if err != nil {
				t.Fatal(err)
			}
			hasStart := false
			for _, job := range jobs.Jobs {
				hasStart = hasStart || job.ID == "seed"
				if tc.name == "default32" && job.ID == "bids" && job.State != commissioner.JobLive {
					t.Fatalf("bids closed: %+v", job)
				}
			}
			if hasStart == tc.started {
				t.Fatalf("start job visibility: %+v", jobs)
			}
			opsClock(app, 10)
			if retry := executeCommissionerTest(t, app, raw); retry != first {
				t.Fatalf("retry changed receipt: %+v %+v", first, retry)
			}
			records, err = app.CommissionerRecords(t.Context(), raw.League)
			if err != nil || len(records) != wantRecords {
				t.Fatalf("retry appended: %d %v", len(records), err)
			}
			raw.Reason = "Different"
			if _, err := app.TargetCommissionerExecute(previewCommandBody(t, raw)); err == nil {
				t.Fatal("changed retry accepted")
			}
		})
	}
}

package commissioner

import (
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

// A 32-Team Default league records its start through the normal seed request without reading held data
// (the fixture's held-data port is nil), and its assets read back one O-Line card per franchise.
func TestDefault32StartsThroughTheSeedRequest(t *testing.T) {
	t.Parallel()
	app := commissionerTestApp(t)
	opsClock(app, 9)
	create := opsCommand(commissioner.Create, 0, "create", commissioner.RawArgs{Name: "Ops league",
		Preset: commissioner.PresetDefault32, Season: 2026})
	create.Reason = "Start our league"
	if _, err := app.history.ExecuteCommissioner(t.Context(), create, app.session, nil); err != nil {
		t.Fatal(err)
	}
	jobs, err := app.TargetCommissionerJobs(create.League)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, job := range jobs.Jobs {
		if job.ID == "seed" {
			found = job.Name == "Start the league"
		}
	}
	if !found {
		t.Fatalf("older native league cannot start: %+v", jobs)
	}
	input := RawNativeSeedRequest{League: "native.ops", Expected: 1, Request: "start", Reason: "Hand out the lines"}
	if _, err := app.TargetNativeSeed(input); err != nil {
		t.Fatal(err)
	}
	opsClock(app, 10)
	if _, err := app.TargetNativeSeed(input); err != nil {
		t.Fatalf("a retried start returns its receipt: %v", err)
	}
	changed := input
	changed.Reason = "Different request body"
	if _, err := app.TargetNativeSeed(changed); err == nil {
		t.Fatal("a default start retry accepted a changed reason")
	}
	assets, err := app.TargetNativeAssets("native.ops")
	if err != nil || !assets.Seeded || len(assets.Snapshot.Players) != 0 || len(assets.Snapshot.Pool) != 32 {
		t.Fatalf("assets: %v %+v", err, assets)
	}
	for _, p := range assets.Snapshot.Pool {
		if facts, ok := nativecheck.LineCardFacts(p.Player); !ok || p.Name != facts.Name {
			t.Fatalf("%+v is not a named O-Line card", p)
		}
	}
}

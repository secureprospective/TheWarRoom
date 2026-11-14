package main

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

// The upgrade that gives each league its own file prints the proof of every table against what
// server.db held; the proof reads the same until a league file changes, then the command fails.
func TestUpgradeProvesEveryLeagueTable(t *testing.T) {
	s := newTwoLeagueServer(t)
	var output bytes.Buffer
	if err := runUpgrade(t.Context(), []string{"-db", s.path, "-save", filepath.Join(t.TempDir(), "saved")}, &output); err != nil {
		t.Fatalf("upgrade: %v\n%s", err, output.String())
	}
	if !strings.Contains(output.String(), "Upgraded the league file") || !strings.Contains(output.String(), "gave its 2 league(s)") || strings.Contains(output.String(), "DIFFERS") {
		t.Fatalf("upgrade report:\n%s", output.String())
	}
	output.Reset()
	if err := runConversionProof(t.Context(), []string{"-db", s.path}, &output); err != nil {
		t.Fatalf("conversion-proof: %v\n%s", err, output.String())
	}
	report := output.String()
	for _, want := range []string{"gave its 2 league(s)", "League " + fixtureLeague + ":", "League " + secondLeague + ":", "league_event", "league_recovery_log"} {
		if !strings.Contains(report, want) {
			t.Fatalf("report lacks %q:\n%s", want, report)
		}
	}
	if strings.Contains(report, "DIFFERS") || strings.Count(report, " same\n") != strings.Count(report, " rows  digest ") || strings.Count(report, " same\n") < 2*8 {
		t.Fatalf("a table differs right after conversion:\n%s", report)
	}
	pools, err := db.Open(t.Context(), mustLeaguePath(t, leaguesRoot(s.path), secondLeague), history.LeagueSchemaVersion())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pools.Write().ExecContext(t.Context(), `INSERT INTO league_recovery_log(sequence,league,revision,recorded_at,message)
		VALUES(1000,'`+secondLeague+`',2,'2026-11-14T00:00:01Z','a later write')`); err != nil {
		t.Fatal(err)
	}
	if err := pools.Close(); err != nil {
		t.Fatal(err)
	}
	output.Reset()
	err = runConversionProof(t.Context(), []string{"-db", s.path}, &output)
	if err == nil || !strings.Contains(err.Error(), "1 table(s) differ") || !strings.Contains(output.String(), "DIFFERS (server.db held 1 rows") {
		t.Fatalf("a changed league file was not reported: %v\n%s", err, output.String())
	}
}

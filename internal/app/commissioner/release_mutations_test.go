package commissioner

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	catalog "github.com/secureprospective/TheWarRoom/internal/commandcatalog"
	domain "github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/governance"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/leagueweek"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

type releaseMutation struct {
	Args          domain.RawArgs `json:"args"`
	Visibility    string         `json:"visibility"`
	Effective     string         `json:"effective"`
	AdmittedRoles []string       `json:"admittedRoles"`
}

func releaseMutationRows(t *testing.T) map[string]releaseMutation {
	t.Helper()
	body, err := os.ReadFile("testdata/release-mutations.json")
	if err != nil {
		t.Fatal(err)
	}
	var rows map[string]releaseMutation
	if err := json.Unmarshal(body, &rows); err != nil {
		t.Fatal(err)
	}
	ids := []string{}
	for _, command := range NativeCommands() {
		if slices.Contains(command.Modes, catalog.Execute) {
			ids = append(ids, command.ID)
			row, ok := rows[command.ID]
			if !ok {
				t.Fatalf("catalog mutation %s has no release matrix fixture", command.ID)
			}
			switch command.Method {
			case "TargetCommissionerExecute", "TargetNativeTradeAction", "TargetNativeWaiverAction", "TargetNativeBidAction", "TargetNativeLineupSet", "TargetNativeDraftPick", "TargetNativeSeed":
			default:
				t.Fatalf("catalog adapter %s has no release matrix", command.Method)
			}
			for _, role := range row.AdmittedRoles {
				if !slices.Contains([]string{"commissioner", "deputy", "second-deputy", "DOT", "owner", "outsider"}, role) {
					t.Fatalf("%s: untested role %s", command.ID, role)
				}
			}
		}
	}
	if len(ids) != len(rows) {
		t.Fatal("release matrix contains obsolete mutation fixtures")
	}
	return rows
}

func (r releaseMutation) raw(t *testing.T, command, league, principal string, revision int64) domain.RawCommand {
	t.Helper()
	body, err := json.Marshal(r.Args)
	if err != nil {
		t.Fatal(err)
	}
	var args domain.RawArgs
	if err := json.Unmarshal([]byte(strings.ReplaceAll(string(body), "$actor", principal)), &args); err != nil {
		t.Fatal(err)
	}
	raw := domain.RawCommand{League: league, Command: command, Args: args, Expected: revision,
		Request: "matrix." + command, Reason: "Disposable release matrix change", Effective: r.Effective, Visibility: r.Visibility}
	if err := raw.Validate(); err != nil {
		t.Fatalf("%s fixture does not pass Raw validation: %v", command, err)
	}
	return raw
}

func TestReleaseMatrixMutationFixtures(t *testing.T) {
	t.Parallel()
	for command, row := range releaseMutationRows(t) {
		t.Run(command, func(t *testing.T) { row.raw(t, command, "native.matrix", "creator", 1) })
	}
}

func releaseWorkflowLeague(t *testing.T, store *history.Store, principal governance.Principal) string {
	t.Helper()
	revision := int64(0)
	record := func(command string, args domain.RawArgs) {
		t.Helper()
		raw := domain.RawCommand{League: "native.hosted", Command: command, Args: args, Expected: revision, Request: fmt.Sprintf("setup.%d", revision), Reason: "Set up disposable matrix league", Effective: "now", Visibility: "league"}
		receipt, err := store.ExecuteCommissioner(t.Context(), raw, string(principal), nil)
		if err != nil {
			t.Fatal(err)
		}
		revision = receipt.Revision
	}
	record(domain.Create, domain.RawArgs{Name: "Matrix league", Preset: domain.PresetLegacy, Season: 2026, FranchiseCount: 32})
	seed := releaseMutationRows(t)[domain.NativeSeed].Args.Native
	record(domain.NativeSeed, domain.RawArgs{Native: seed})
	for _, rule := range []struct{ field, value string }{
		{"calendar.timezone", `"UTC"`},
		{"calendar.boundary", `"start_of_day"`},
		{"calendar.windows", `{"matrix":{"start":{"month":1,"day":1},"opens":["lineup","draft_pick","ir","taxi","cut","waiver_claim","ufa_bid","trade"]}}`},
	} {
		record(domain.RulesSet, domain.RawArgs{Field: rule.field, Value: json.RawMessage(rule.value)})
	}
	record("commish.seat.replace", domain.RawArgs{People: &domain.RawPeople{Franchise: "franchise.01", Principal: string(principal)}})
	record(domain.CaretakerSeat, domain.RawArgs{People: &domain.RawPeople{Franchise: "franchise.03", Principal: string(principal)}})
	at := time.Now().UTC()
	evidence := history.RawFootballEvidence{Version: 1, Season: 2026, Source: "Disposable release matrix evidence", AsOf: at,
		Players: map[string]history.RawFootballPlayer{"0099": {Identity: "matrix.0099", Facts: nativecheck.PlayerFacts{Name: "Matrix quarterback", Position: leaguerules.QB, Team: "DAL", DesignationHeld: true, DraftYear: 2026, DraftYearHeld: true}},
			"5001": {Identity: "matrix.5001", Facts: nativecheck.PlayerFacts{Name: "Matrix rookie", Position: leaguerules.QB, Team: "DAL", DesignationHeld: true, DraftYear: 2027, DraftYearHeld: true}}}}
	for week := 1; week <= 22; week++ {
		evidence.Weeks = append(evidence.Weeks, leagueweek.Week{Number: week, Games: []leagueweek.Game{{Teams: [2]string{"DAL", "NYG"}, Kickoff: at.Add(24 * time.Hour)}}})
	}
	body, err := json.Marshal(evidence)
	if err != nil {
		t.Fatal(err)
	}
	hash := sha256.Sum256(body)
	digest := hex.EncodeToString(hash[:])
	if err := store.ImportFootball(t.Context(), body, digest); err != nil {
		t.Fatal(err)
	}
	return digest
}

package hostedfixture

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/governance"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/leagueweek"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

// CreateWorkflowLeague records a complete disposable roster, reserve and draft playground.
func CreateWorkflowLeague(ctx context.Context, store *history.Store, principal governance.Principal) (string, error) {
	at := time.Now().UTC()
	revision := int64(0)
	record := func(command string, args commissioner.RawArgs) error {
		raw := commissioner.RawCommand{League: "native.hosted", Command: command, Args: args, Expected: revision, Reason: "Set up disposable practice league", Request: fmt.Sprintf("practice.%d", revision), Effective: "now", Visibility: "league"}
		receipt, err := store.ExecuteCommissioner(ctx, raw, string(principal), nil)
		if err != nil {
			return fmt.Errorf("practice %s: %w", command, err)
		}
		revision = receipt.Revision
		return nil
	}
	if err := record(commissioner.Create, commissioner.RawArgs{Name: "Practice league", Preset: commissioner.PresetLegacy, Season: 2026, FranchiseCount: 32}); err != nil {
		return "", err
	}
	seed := &commissioner.RawNativeSnapshot{Season: 2026, RosterAt: at, DraftAt: at, FutureAt: at}
	order := []string{}
	for n := 1; n <= 32; n++ {
		id := fmt.Sprintf("franchise.%02d", n)
		order = append(order, id)
		seed.Franchises = append(seed.Franchises, commissioner.NativeFranchise{ID: id, HeldID: fmt.Sprintf("%04d", n), Name: fmt.Sprintf("Practice team %d", n), Abbreviation: fmt.Sprintf("P%d", n)})
		seed.Picks = append(seed.Picks, commissioner.NativePick{Season: 2027, Round: 1, Original: id, Owner: id})
	}
	for _, id := range []string{"0099", "0100", "0101"} {
		seed.Players = append(seed.Players, commissioner.NativePlayer{ID: id, Name: map[string]string{"0099": "Practice quarterback", "0100": "Injured prospect", "0101": "Young receiver"}[id], Franchise: "franchise.01", Status: domain.RosterActive, Salary: 100000000, ExpirationYear: 2028, ContractStatus: domain.CStatusUFA})
	}
	seed.Pool = []commissioner.NativePoolEntry{{Player: "6001", Name: "Practice free agent", Season: 2026}}
	seed.Players = append(seed.Players, commissioner.NativePlayer{ID: "0102", Name: "Other team receiver", Franchise: "franchise.02", Status: domain.RosterActive, Salary: 100000000, ExpirationYear: 2028, ContractStatus: domain.CStatusUFA})
	if err := record(commissioner.NativeSeed, commissioner.RawArgs{Native: seed}); err != nil {
		return "", err
	}
	if err := setupWorkflowRules(record, at, order); err != nil {
		return "", err
	}
	if err := record("commish.seat.replace", commissioner.RawArgs{People: &commissioner.RawPeople{Franchise: "franchise.01", Principal: string(principal)}}); err != nil {
		return "", err
	}
	if err := record(commissioner.WaiverOrderSet, commissioner.RawArgs{Waiver: &commissioner.RawWaiverArgs{Order: order}}); err != nil {
		return "", err
	}
	if err := record(commissioner.DraftOrderSet, commissioner.RawArgs{Draft: &commissioner.RawDraftArgs{Season: 2027, Order: order}}); err != nil {
		return "", err
	}
	if err := record(commissioner.DraftStart, commissioner.RawArgs{Draft: &commissioner.RawDraftArgs{Season: 2027, At: time.Now().UTC().Add(2 * time.Second)}}); err != nil {
		return "", err
	}
	return importPracticeFootball(ctx, store, at)
}

func setupWorkflowRules(record func(string, commissioner.RawArgs) error, at time.Time, order []string) error {
	for _, rule := range []struct{ field, value string }{
		{"roster.limits", `{"min":0,"max":100}`}, {"cap.amount", `100000000000`},
		{"calendar.timezone", `"America/Chicago"`}, {"calendar.boundary", `"start_of_day"`},
		{"calendar.windows", `{"rookie_draft":{"start":{"month":1,"day":1},"opens":["draft_pick","lineup","ir","taxi","cut","waiver_claim","ufa_bid","trade"]}}`},
		{"lineup.positions", `{"QB":{"min":1,"max":1},"WR":{"min":0,"max":2}}`},
		{"lineup.groups", `{"total":{"min":1,"max":3}}`},
		{"bids.standFor", `120000000000`}, {"waivers.duration", `120000000000`}, {"bids.ufa", `"blind"`}, {"trades.review", `"commissioner"`}, {"draft.rounds", `1`}, {"draft.clock", `86400000000000`},
	} {
		if err := record(commissioner.RulesSet, commissioner.RawArgs{Field: rule.field, Value: json.RawMessage(rule.value)}); err != nil {
			return err
		}
	}
	alignment := map[string]string{}
	for i, id := range order {
		alignment[id] = fmt.Sprintf("Practice division %d", i/4+1)
	}
	divisions, err := json.Marshal(alignment)
	if err != nil {
		return fmt.Errorf("practice divisions: %w", err)
	}
	if err := record(commissioner.RulesSet, commissioner.RawArgs{Field: "league.alignment", Value: divisions}); err != nil {
		return err
	}
	deadline, err := json.Marshal(at.AddDate(0, 0, 30))
	if err != nil {
		return fmt.Errorf("practice deadline: %w", err)
	}
	if err := record(commissioner.RulesSet, commissioner.RawArgs{Field: "calendar.tradeDeadline", Value: deadline}); err != nil {
		return err
	}
	if err := record(commissioner.ScheduleGenerate, commissioner.RawArgs{Schedule: &commissioner.RawScheduleArgs{Weeks: 1, Pattern: leaguerules.DivisionWeighted}}); err != nil {
		return err
	}
	if err := record(commissioner.SchedulePublish, commissioner.RawArgs{Schedule: &commissioner.RawScheduleArgs{}}); err != nil {
		return err
	}
	return nil
}

func importPracticeFootball(ctx context.Context, store *history.Store, at time.Time) (string, error) {
	raw := history.RawFootballEvidence{Version: 1, Season: 2026, Source: "Disposable practice players, not real NFL advice", AsOf: at, Players: map[string]history.RawFootballPlayer{}}
	for _, p := range []struct {
		id, name string
		position leaguerules.Position
	}{{"0099", "Practice quarterback", leaguerules.QB}, {"0100", "Injured prospect", leaguerules.WR}, {"0101", "Young receiver", leaguerules.WR}, {"5001", "Practice rookie", leaguerules.RB}, {"6001", "Practice free agent", leaguerules.WR}, {"0102", "Other team receiver", leaguerules.WR}} {
		facts := nativecheck.PlayerFacts{Name: p.name, Position: p.position, Team: "DAL", DesignationHeld: true, DraftYear: 2026, DraftYearHeld: true}
		if p.id == "5001" {
			facts.DraftYear = 2027
		}
		if p.id == "0100" {
			facts.Designation = leaguerules.InjuredReserve
		}
		raw.Players[p.id] = history.RawFootballPlayer{Identity: "practice." + p.id, Facts: facts}
	}
	for week := 1; week <= 22; week++ {
		raw.Weeks = append(raw.Weeks, leagueweek.Week{Number: week, Games: []leagueweek.Game{{Teams: [2]string{"DAL", "NYG"}, Kickoff: at.Add(24 * time.Hour)}}})
	}
	body, err := json.Marshal(raw)
	if err != nil {
		return "", fmt.Errorf("encode practice facts: %w", err)
	}
	hash := sha256.Sum256(body)
	digest := hex.EncodeToString(hash[:])
	if err := store.ImportFootball(ctx, body, digest); err != nil {
		return "", fmt.Errorf("import practice facts: %w", err)
	}
	return digest, nil
}

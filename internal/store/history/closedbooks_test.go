//go:build leaguebench

package history

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/fantasyscore/source"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

type booksProbe func([]commissioner.Record) ([]byte, error)

func readingProbe[T any](read func([]commissioner.Record) (T, error)) booksProbe {
	return func(records []commissioner.Record) ([]byte, error) {
		value, err := read(records)
		body, encodeErr := json.Marshal(value)
		if encodeErr != nil {
			return nil, fmt.Errorf("encode probe: %w", encodeErr)
		}
		return body, err
	}
}

func sameBooksProbe(t *testing.T, probe booksProbe, folded, full []commissioner.Record) {
	t.Helper()
	got, gotErr := probe(folded)
	want, wantErr := probe(full)
	var gotRefusal, wantRefusal *commissioner.Refusal
	if !bytes.Equal(got, want) || fmt.Sprint(gotErr) != fmt.Sprint(wantErr) || errors.As(gotErr, &gotRefusal) != errors.As(wantErr, &wantRefusal) || errors.Is(gotErr, ErrRevisionConflict) != errors.Is(wantErr, ErrRevisionConflict) {
		t.Fatalf("summary/full differ\nsummary: %s, %v\nfull: %s, %v", got, gotErr, want, wantErr)
	}
}

func TestClosedBooksDifferential(t *testing.T) {
	s, _ := openBigLeague(t, filepath.Join(t.TempDir(), "books.db"))
	g := seedClosedSeasons(t, s, 80, 2020, 2023)
	s.SetBidEnvironment(bigLeagueEnvironment{})
	g.moves(leagueVolume{year: 2023, drops: 2, adds: 2})
	g.trades(leagueVolume{year: 2023, trades: 1, assets: 1})
	g.now = time.Date(2024, 4, 1, 12, 0, 0, 0, time.UTC)
	g.command("creator", commissioner.SeasonRollover, commissioner.RawArgs{Rollover: &commissioner.RawRolloverArgs{To: 2024}}, false)
	g.moves(leagueVolume{year: 2024, drops: 2, adds: 2})
	g.trades(leagueVolume{year: 2024, trades: 1, assets: 1, proposals: 1})
	full, folded := closedBooksRecords(t, s)
	probes := closedBooksReadings(t, s, g)
	for name, probe := range probes {
		t.Run(name, func(t *testing.T) { sameBooksProbe(t, probe, folded, full) })
	}
	testBooksDecisions(t, s, g, folded, full)
	// Authoritative history is untouched and every completed rollover has one row.
	var summaries int
	if err := s.pools.Read().QueryRow(`SELECT count(*) FROM league_season_summary`).Scan(&summaries); err != nil {
		t.Fatal(err)
	}
	if summaries != 4 {
		t.Fatalf("closed summaries=%d want 4", summaries)
	}
	var body []byte
	if err := s.pools.Read().QueryRow(`SELECT payload FROM league_season_summary ORDER BY revision DESC LIMIT 1`).Scan(&body); err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"missing", "latest-missing", "checksum", "source"} {
		t.Run("fallback/"+mode, func(t *testing.T) { testStoredBooksFallback(t, s, bigLeagueID, body, mode, full) })
	}
}

func closedBooksRecords(t *testing.T, s *Store) ([]commissioner.Record, []commissioner.Record) {
	t.Helper()
	events, err := s.Events(t.Context(), bigLeagueID, commissioner.Stream, 0)
	if err != nil {
		t.Fatal(err)
	}
	full, err := commissionerRecords(events)
	if err != nil {
		t.Fatal(err)
	}
	var body []byte
	if err := s.pools.Read().QueryRow(`SELECT payload FROM league_season_summary ORDER BY revision DESC LIMIT 1`).Scan(&body); err != nil {
		t.Fatal(err)
	}
	var raw commissioner.RawClosedBooks
	if err := json.Unmarshal(body, &raw); err != nil {
		t.Fatal(err)
	}
	if err := raw.Validate(); err != nil {
		t.Fatal(err)
	}
	folded, ok := commissioner.WithClosedBooks(full, raw)
	if !ok {
		t.Fatal("valid latest summary was not accepted")
	}
	// A cold Store must attach the persisted checkpoint, not require a new rollover.
	cold := New(s.pools, s.reg)
	cold.SetClock(s.now)
	stored, err := cold.nativeRecords(t.Context(), bigLeagueID)
	if err != nil {
		t.Fatal(err)
	}
	if reflect.DeepEqual(stored, full) {
		t.Fatal("cold store did not attach the persisted summary")
	}
	if !bytes.Contains(mustBooksJSON(t, stored), []byte("generated.")) {
		t.Fatal("lost authoritative requests")
	}
	return full, folded
}

func mustBooksJSON(t *testing.T, records []commissioner.Record) []byte {
	t.Helper()
	body, err := json.Marshal(records)
	if err != nil {
		t.Fatal(err)
	}
	return body
}

func closedBooksReadings(t *testing.T, s *Store, g *bigLeagueBuilder) map[string]booksProbe {
	t.Helper()
	at := g.now
	team := bigTeam(0)
	owner := bigOwner(0)
	env := commissioner.LineupFacts{Facts: g.facts, Zone: time.UTC}
	probes := map[string]booksProbe{
		"projection": readingProbe(func(r []commissioner.Record) (commissioner.Projection, error) { return commissioner.Replay(r), nil }),
		"assets": readingProbe(func(r []commissioner.Record) (commissioner.NativeAssets, error) {
			return commissioner.NativeAssetsReading(bigLeagueID, r, at)
		}),
		"directory": readingProbe(func(r []commissioner.Record) (commissioner.NativeDirectory, error) {
			return commissioner.NativeDirectoryReading(bigLeagueID, r, at)
		}),
		"rules": readingProbe(func(r []commissioner.Record) (commissioner.Rules, error) {
			return commissioner.RulesReading(bigLeagueID, r, at)
		}),
		"jobs": readingProbe(func(r []commissioner.Record) (commissioner.Jobs, error) {
			return commissioner.JobsReading(bigLeagueID, r, at)
		}),
		"notes": readingProbe(func(r []commissioner.Record) (commissioner.LeagueNotes, error) {
			return commissioner.NotesReading(bigLeagueID, r)
		}),
		"corrections": readingProbe(func(r []commissioner.Record) (commissioner.Corrections, error) {
			return commissioner.CorrectionsReading(bigLeagueID, r, at)
		}),
		"history": readingProbe(func(r []commissioner.Record) (commissioner.FranchiseHistory, error) {
			return commissioner.FranchiseHistoryReading(bigLeagueID, r, team)
		}),
		"cap": readingProbe(func(r []commissioner.Record) (commissioner.NativeCapCheck, error) {
			return commissioner.NativeCapCheckReading(commissioner.RawNativeCapScope{League: bigLeagueID, ViewerZone: "UTC"}, r, at)
		}),
		"roster": readingProbe(func(r []commissioner.Record) (commissioner.NativeRosterCheck, error) {
			return commissioner.NativeRosterCheckReading(bigLeagueID, r, at, g.facts)
		}),
		"reserve": readingProbe(func(r []commissioner.Record) (commissioner.NativeReserveCheck, error) {
			return commissioner.NativeReserveCheckReading(bigLeagueID, r, at, g.facts)
		}),
		"house": readingProbe(func(r []commissioner.Record) (commissioner.NativeHouseRules, error) {
			return commissioner.NativeHouseRulesReading(bigLeagueID, r, at)
		}),
		"waivers": readingProbe(func(r []commissioner.Record) (commissioner.NativeWaivers, error) {
			return commissioner.NativeWaiversReading(bigLeagueID, team, owner, r, at, time.UTC, g.facts)
		}),
		"bids": readingProbe(func(r []commissioner.Record) (commissioner.NativeBids, error) {
			return commissioner.NativeBidsReading(bigLeagueID, team, owner, r, at, time.UTC, g.facts)
		}),
		"trades": readingProbe(func(r []commissioner.Record) (commissioner.NativeTrades, error) {
			return commissioner.NativeTradesReading(bigLeagueID, team, owner, r, at, time.UTC)
		}),
		"calendar": readingProbe(func(r []commissioner.Record) (commissioner.Calendar, error) {
			return commissioner.CalendarReading(bigLeagueID, r, at, at.Year(), leaguerules.Action("trade"), team, time.UTC)
		}),
		"lineup": readingProbe(func(r []commissioner.Record) (commissioner.NativeLineup, error) {
			return commissioner.NativeLineupReading(bigLeagueID, team, owner, 1, r, at, env)
		}),
		"player": readingProbe(func(r []commissioner.Record) (commissioner.NativePlayerScore, error) {
			return commissioner.NativePlayerScoreReading(bigLeagueID, source.RawNativeScope{Player: g.players[0].ID, Season: at.Year(), Week: 1}, r, source.NativeEvidence{}, at)
		}),
	}
	for _, principal := range []string{"creator", owner, "outsider"} {
		probes["people/"+principal] = readingProbe(func(r []commissioner.Record) (commissioner.People, error) {
			return commissioner.PeopleReading(bigLeagueID, principal, r)
		})
		probes["ledger/"+principal] = readingProbe(func(r []commissioner.Record) (commissioner.Ledger, error) {
			state, err := commissioner.GovernanceState(r)
			if err != nil {
				return commissioner.Ledger{}, err
			}
			return commissioner.Reading(bigLeagueID, principal, r, commissioner.GovernanceAuthorizer{State: state})
		})
		probes["digests/"+principal] = readingProbe(func(r []commissioner.Record) (commissioner.Digests, error) {
			state, err := commissioner.GovernanceState(r)
			if err != nil {
				return commissioner.Digests{}, err
			}
			return commissioner.DigestsReading(bigLeagueID, principal, r, commissioner.GovernanceAuthorizer{State: state})
		})
	}
	for year := 2020; year <= 2025; year++ {
		probes[fmt.Sprintf("season/%d", year)] = readingProbe(func(r []commissioner.Record) (leaguerules.RuleSet, error) {
			return commissioner.RulesInForce(r, year, at)
		})
		probes[fmt.Sprintf("schedule/%d", year)] = readingProbe(func(r []commissioner.Record) (commissioner.Schedule, error) {
			return commissioner.ScheduleSeasonReading(bigLeagueID, r, year, at)
		})
		probes[fmt.Sprintf("results/%d", year)] = readingProbe(func(r []commissioner.Record) (commissioner.NativeResults, error) {
			return commissioner.NativeResultsReading(t.Context(), bigLeagueID, r, s.nativeSources(bigLeagueID, year), year, 1, at)
		})
		probes[fmt.Sprintf("standings/%d", year)] = readingProbe(func(r []commissioner.Record) (commissioner.NativeStandings, error) {
			return commissioner.NativeStandingsReading(t.Context(), bigLeagueID, r, s.nativeSources(bigLeagueID, year), year, at)
		})
		probes[fmt.Sprintf("bracket/%d", year)] = readingProbe(func(r []commissioner.Record) (commissioner.NativeBracket, error) {
			return commissioner.NativeBracketReading(t.Context(), bigLeagueID, r, s.nativeSources(bigLeagueID, year), year, at, env)
		})
		probes[fmt.Sprintf("draft/%d", year)] = readingProbe(func(r []commissioner.Record) (commissioner.NativeDraft, error) {
			return commissioner.NativeDraftReading(commissioner.RawNativeDraftScope{League: bigLeagueID, Season: year, Franchise: team, At: at}, r, time.UTC, nil, g.facts)
		})
	}
	addBooksReadings(t, s, g, probes)
	return probes
}

func testBooksDecisions(t *testing.T, s *Store, g *bigLeagueBuilder, folded, full []commissioner.Record) {
	t.Helper()
	candidates := []commissioner.RawCommand{
		{Command: commissioner.RulesSet, Args: commissioner.RawArgs{Field: "cap.amount", Value: json.RawMessage(`100000000000002`)}},
		{Command: commissioner.RulesSet, Args: commissioner.RawArgs{Field: "cap.amount", Value: latestBooksRule(full, "cap.amount")}},
		{Command: commissioner.Revert, Args: commissioner.RawArgs{Entry: full[len(full)-1].Receipt.Entry}},
		{Command: commissioner.Revert, Args: commissioner.RawArgs{Entry: full[15].Receipt.Entry}},
		{Command: "commish.role.grant", Args: commissioner.RawArgs{People: &commissioner.RawPeople{Principal: bigOwner(0), Role: "DOT"}}},
		{Command: commissioner.SeasonRollover, Args: commissioner.RawArgs{Rollover: &commissioner.RawRolloverArgs{To: 2025}}},
	}
	for i, raw := range candidates {
		raw.League, raw.Expected, raw.Request = bigLeagueID, g.revision, fmt.Sprintf("probe.%d", i)
		raw.Effective, raw.Visibility, raw.Reason = "now", "league", "Differential decision"
		if raw.Command == "commish.role.grant" {
			raw.Visibility = "private"
		}
		prepared, canonical, err := preparedCommand(raw, "creator", true)
		if err != nil {
			t.Fatal(err)
		}
		for _, principal := range []string{"creator", bigOwner(0), "outsider"} {
			t.Run(fmt.Sprintf("decision/%d/%s", i, principal), func(t *testing.T) {
				type result struct {
					Role  string
					Retry *commissioner.Receipt
				}
				probe := readingProbe(func(r []commissioner.Record) (result, error) {
					role, retry, err := s.checkCommissionerCommand(prepared, principal, nil, r, canonical, appendFacts{players: g.facts}, s.now())
					return result{role, retry}, err
				})
				sameBooksProbe(t, probe, folded, full)
				type trialResult struct {
					Before *commissioner.RolloverBefore
					Plan   *commissioner.RolloverPlan
				}
				sameBooksProbe(t, readingProbe(func(r []commissioner.Record) (trialResult, error) {
					raw := deepClone(prepared)
					before, plan, err := s.prepareCommissionerTrial(t.Context(), &raw, principal, r, g.now, g.facts, time.UTC, appendFacts{players: g.facts})
					return trialResult{before, plan}, err
				}), folded, full)
			})
		}
	}
	// Request identity and receipts from closed seasons must remain available.
	retry := full[2]
	sameBooksProbe(t, readingProbe(func(r []commissioner.Record) (commissioner.Receipt, error) {
		receipt, _, err := retriedReceipt(retry.Command, retry.Canonical, r)
		return receipt, err
	}), folded, full)
}

func latestBooksRule(records []commissioner.Record, field string) json.RawMessage {
	for i := len(records) - 1; i >= 0; i-- {
		if records[i].Command.Command == commissioner.RulesSet && records[i].Command.Args.Field == field {
			return records[i].Command.Args.Value
		}
	}
	return nil
}

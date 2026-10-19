package commissioner

import (
	"fmt"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

func TestTimezoneOnlyNeedsConfirm(t *testing.T) {
	for _, timing := range []string{"now", "next_rollover"} {
		raw := ruleRaw("calendar.timezone", `"Europe/London"`, timing, "")
		definition, err := raw.Definition()
		if err != nil || definition.Gravity != "G1" || raw.Validate() != nil {
			t.Fatalf("timezone %s: %+v %v, validation %v", timing, definition, err, raw.Validate())
		}
	}
	if got := safeguardsFor("calendar.timezone"); got.Now != "confirm" || got.NextRollover != "confirm" {
		t.Fatalf("timezone safeguards %+v", got)
	}
	if got := safeguardsFor("calendar.boundary"); got.Now != "reason" || got.NextRollover != "hold" {
		t.Fatalf("boundary safeguards changed: %+v", got)
	}
}

func TestLegacyCreationForAnySize(t *testing.T) {
	for count, divisions := range map[int]int{2: 2, 6: 2, 12: 2, 16: 4, 24: 6, 32: 8} {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			raw, err := (RawCommand{League: "native.test", Command: Create,
				Args:   RawArgs{Name: "Ours", Preset: PresetLegacy, Season: 2026, FranchiseCount: count},
				Reason: "Start", Request: "create", Effective: "now", Visibility: "league"}).Prepared()
			if err != nil || raw.Validate() != nil || len(raw.Args.Franchises) != count {
				t.Fatalf("create %d: %+v %v", count, raw, err)
			}
			again, err := raw.Prepared()
			if err != nil || !sameSeed(raw.Args.Seed, again.Args.Seed) {
				t.Fatal("retry changed seed", err)
			}
			records := []Record{{Command: raw, Receipt: Receipt{Entry: "create", RecordedAt: time.Unix(1, 0)}}}
			rules, err := resolveRecords(records)
			if err != nil {
				t.Fatal(err)
			}
			if rules.League.Teams.Value != count || rules.League.Conferences.Value != 2 ||
				rules.League.Divisions.Value != divisions {
				t.Fatalf("%d teams: want %d divisions of about four: %+v", count, divisions, rules.League)
			}
			field := rules.Playoffs
			if count == 32 && (field.PerConference.Value != 7 || field.DivisionWinners.Value != 4 ||
				field.WildCards.Value != 3) {
				t.Fatalf("Legacy playoff field lost: %+v", field)
			}
			if count != 32 && (field.PerConference.State != leaguerules.NeedsRuling ||
				field.DivisionWinners.State != leaguerules.NeedsRuling || field.WildCards.State != leaguerules.NeedsRuling) {
				t.Fatalf("%d teams: a smaller playoff field was invented: %+v", count, field)
			}
			if count == 32 {
				if rules.Playoffs.ByeSeed.Value != 1 || len(rules.Playoffs.Graph.Value) != 7 {
					t.Fatal("Legacy bracket lost")
				}
			} else if rules.Playoffs.Graph.State != leaguerules.NeedsRuling ||
				rules.Playoffs.ByeSeed.State != leaguerules.NeedsRuling ||
				rules.Schedule.Pattern.State != leaguerules.NeedsRuling {
				t.Fatal("small bracket or schedule guessed")
			}
			if count == 2 && (rules.Trades.MaximumParties.Value != 2 ||
				rules.Trades.Review.State != leaguerules.NeedsRuling) {
				t.Fatal("small trade rules unchanged")
			}
			if rules.Trades.DOTSeats.State != leaguerules.NeedsRuling {
				t.Fatal("DOT seats invented")
			}
		})
	}
}

func waitingRecord(entry, value string, at time.Time) Record {
	return Record{Command: ruleRaw("roster.limits", value, "next_rollover", ""),
		Receipt: Receipt{Entry: entry, RecordedAt: at}, Principal: "creator"}
}

func TestNewestWaitingRuleWinsAndUndoRestores(t *testing.T) {
	at := time.Date(2026, 10, 14, 4, 17, 3, 0, time.UTC)
	records := newLedger(t, PresetLegacy).records
	records = append(records, waitingRecord("first", `{"min":20,"max":70}`, at),
		waitingRecord("second", `{"min":22,"max":75}`, at.Add(time.Minute)))
	assertWaiting := func(winner string, replaced map[string]string) {
		t.Helper()
		_, pending, err := ruleHistory(records, at.Add(time.Hour))
		if err != nil {
			t.Fatal(err)
		}
		changes := pending["roster.limits"]
		if len(changes) != len(replaced)+1 {
			t.Fatalf("history lost: %+v", changes)
		}
		ledger, err := Reading("native.test", "creator", records, CreatorAuthorizer{Creator: "creator"})
		if err != nil {
			t.Fatal(err)
		}
		for _, change := range changes {
			if change.Entry == winner {
				if change.When != waitingWords || change.ReplacedAt != "" {
					t.Fatalf("winner not waiting: %+v", change)
				}
			} else if change.When != "Replaced by a later change" || change.ReplacedAt != replaced[change.Entry] {
				t.Fatalf("replacement missing: %+v", change)
			}
			for _, entry := range ledger.Entries {
				if entry.ID == change.Entry && entry.ReplacedAt != change.ReplacedAt {
					t.Fatalf("ledger disagrees: %+v", entry)
				}
			}
		}
	}
	second := at.Add(time.Minute).Format(time.RFC3339Nano)
	assertWaiting("second", map[string]string{"first": second})
	records = append(records, Record{Command: RawCommand{Command: Revert, Args: RawArgs{Entry: "second"}},
		Receipt: Receipt{Entry: "undo-second", RecordedAt: at.Add(2 * time.Minute)}})
	assertWaiting("first", map[string]string{})
	records = append(records, waitingRecord("third", `{"min":24,"max":78}`, at.Add(3*time.Minute)))
	third := at.Add(3 * time.Minute).Format(time.RFC3339Nano)
	assertWaiting("third", map[string]string{"first": third})
	records = append(records, Record{Command: RawCommand{Command: Revert, Args: RawArgs{Entry: "undo-second"}},
		Receipt: Receipt{Entry: "restore-second", RecordedAt: at.Add(4 * time.Minute)}})
	assertWaiting("third", map[string]string{"first": third, "second": third})
	records = append(records, Record{Command: RawCommand{Command: Revert, Args: RawArgs{Entry: "third"}},
		Receipt: Receipt{Entry: "undo-third", RecordedAt: at.Add(5 * time.Minute)}})
	assertWaiting("second", map[string]string{"first": second})
}

type CreatorAuthorizer struct{ Creator string }

func (a CreatorAuthorizer) Allow(_ string, principal, _ string) (string, error) {
	if a.Creator != "" && a.Creator == principal {
		return "commissioner", nil
	}
	return "", fmt.Errorf("only this league's commissioner may make this change")
}

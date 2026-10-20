package history

import (
	"crypto/sha256"
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

// rolloverDB is a seeded Legacy league on a database file that can be physically closed and reopened.
type rolloverDB struct {
	path  string
	pools *db.Pools
	s     *Store
	c     *clock
}

func (r *rolloverDB) open(t *testing.T) {
	t.Helper()
	pools, err := db.Open(t.Context(), r.path, SchemaVersion())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = pools.Close() })
	r.s, r.c, r.pools = reopen(t, pools, registry(t, ""))
}

// reopenFile closes the database for real and opens the same file again.
func (r *rolloverDB) reopenFile(t *testing.T) {
	t.Helper()
	at := r.c.now
	if err := r.pools.Close(); err != nil {
		t.Fatal(err)
	}
	r.open(t)
	r.c.now = at
}

// version is the league version a change must be made against: the number of entries in its ledger.
func (r *rolloverDB) version(t *testing.T) int64 {
	t.Helper()
	events, err := r.s.Events(t.Context(), "native.test", commissioner.Stream, 0)
	if err != nil {
		t.Fatal(err)
	}
	return int64(len(events))
}

func (r *rolloverDB) do(t *testing.T, raw commissioner.RawCommand) commissioner.Receipt {
	t.Helper()
	receipt, err := r.s.ExecuteCommissioner(t.Context(), raw, "creator", nil)
	if err != nil {
		t.Fatalf("%s %s: %v", raw.Command, raw.Request, err)
	}
	return receipt
}

func (r *rolloverDB) rule(t *testing.T, field, value, effective string) commissioner.Receipt {
	t.Helper()
	rev := r.version(t)
	reason := "Our league's ruling"
	return r.do(t, ruleCommand(field, value, effective, reason, rev))
}

func newRolloverDB(t *testing.T) *rolloverDB {
	t.Helper()
	r := &rolloverDB{path: filepath.Join(t.TempDir(), "history.db")}
	r.open(t)
	r.do(t, createWith(commissioner.PresetLegacy, 0))
	r.do(t, heldSeedCommand(t))
	r.rule(t, "calendar.boundary", `"start_of_day"`, "now")
	r.rule(t, "calendar.tradeDeadline", `"2026-11-09T05:00:00Z"`, "now")
	r.rule(t, "roster.limits", `{"min":20,"max":70}`, "next_rollover")
	r.c.now = r.c.now.Add(time.Hour)
	r.rule(t, "roster.limits", `{"min":22,"max":75}`, "next_rollover")
	r.c.now = time.Date(2027, 3, 1, 5, 0, 0, 0, time.UTC)
	return r
}

func (r *rolloverDB) rollover(t *testing.T, to int, dry bool, request string) commissioner.RawCommand {
	t.Helper()
	return commissioner.RawCommand{League: "native.test", Command: commissioner.SeasonRollover,
		Args:   commissioner.RawArgs{Rollover: &commissioner.RawRolloverArgs{To: to, DryRun: dry}},
		Reason: "Turn the league year", Expected: r.version(t), Request: request, Effective: "now",
		Visibility: "league"}
}

// fingerprint hashes every row of every table, so a write of any kind changes it.
func fingerprint(t *testing.T, s *Store) [sha256.Size]byte {
	t.Helper()
	hash := sha256.New()
	for _, name := range tableNames(t, s) {
		fmt.Fprintf(hash, "table %s\n", name)
		rows, err := s.pools.Read().QueryContext(t.Context(), `SELECT * FROM `+name)
		if err != nil {
			t.Fatal(err)
		}
		hashRows(t, hash, rows)
	}
	var sum [sha256.Size]byte
	copy(sum[:], hash.Sum(nil))
	return sum
}

func tableNames(t *testing.T, s *Store) []string {
	t.Helper()
	tables, err := s.pools.Read().QueryContext(t.Context(),
		`SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' ORDER BY name`)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tables.Close() }()
	var names []string
	for tables.Next() {
		var name string
		if err := tables.Scan(&name); err != nil {
			t.Fatal(err)
		}
		names = append(names, name)
	}
	if err := tables.Err(); err != nil {
		t.Fatal(err)
	}
	return names
}

func hashRows(t *testing.T, hash interface{ Write([]byte) (int, error) }, rows *sql.Rows) {
	t.Helper()
	defer func() { _ = rows.Close() }()
	columns, err := rows.Columns()
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		values := make([]sql.RawBytes, len(columns))
		targets := make([]any, len(columns))
		for i := range values {
			targets[i] = &values[i]
		}
		if err := rows.Scan(targets...); err != nil {
			t.Fatal(err)
		}
		for _, value := range values {
			_, _ = hash.Write(append(value, 0))
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
}

func (r *rolloverDB) playerStatuses(t *testing.T) (int, []domain.RosterStatus) {
	t.Helper()
	assets, err := r.s.NativeAssets(t.Context(), "native.test")
	if err != nil || !assets.Seeded {
		t.Fatalf("assets: %v", err)
	}
	var statuses []domain.RosterStatus
	for _, p := range assets.Snapshot.Players {
		statuses = append(statuses, p.Status)
	}
	return assets.Snapshot.Season, statuses
}

func TestSeasonRolloverDryRunWritesNothing(t *testing.T) {
	t.Parallel()
	r := newRolloverDB(t)
	before := fingerprint(t, r.s)
	receipt := r.do(t, r.rollover(t, 2027, true, "dry"))
	if receipt.Plan == nil || receipt.Plan.To != 2027 || receipt.Plan.Blocked || receipt.Entry != "" ||
		!strings.Contains(receipt.Message, "nothing was recorded") {
		t.Fatalf("a dry run must return the plan and no entry: %+v", receipt)
	}
	if fingerprint(t, r.s) != before {
		t.Fatal("a dry run changed the database")
	}
	again := r.do(t, r.rollover(t, 2027, true, "dry"))
	if !reflect.DeepEqual(again.Plan, receipt.Plan) || fingerprint(t, r.s) != before {
		t.Fatal("a repeated dry run differs or wrote")
	}
}

func TestSeasonRolloverRefusals(t *testing.T) {
	t.Parallel()
	r := newRolloverDB(t)
	before := fingerprint(t, r.s)
	for _, to := range []int{2026, 2028, 2025} {
		for _, dry := range []bool{true, false} {
			_, err := r.s.ExecuteCommissioner(t.Context(), r.rollover(t, to, dry, "wrong"), "creator", nil)
			var refused *commissioner.Refusal
			if !errors.As(err, &refused) || !strings.Contains(refused.Reason(), "Move only from season 2026 to season 2027") {
				t.Fatalf("to %d dry %v: %v", to, dry, err)
			}
		}
	}
	for _, principal := range []string{"stranger", "owner-one"} {
		if _, err := r.s.ExecuteCommissioner(t.Context(), r.rollover(t, 2027, false, "intruder"), principal, nil); err == nil {
			t.Fatalf("%s rolled the league over", principal)
		}
	}
	stale := r.rollover(t, 2027, false, "stale")
	stale.Expected--
	if _, err := r.s.ExecuteCommissioner(t.Context(), stale, "creator", nil); !errors.Is(err, ErrRevisionConflict) {
		t.Fatalf("a stale league version was accepted: %v", err)
	}
	noReason := r.rollover(t, 2027, false, "quiet")
	noReason.Reason = " "
	if _, err := r.s.ExecuteCommissioner(t.Context(), noReason, "creator", nil); err == nil {
		t.Fatal("a rollover with no reason was accepted")
	}
	if fingerprint(t, r.s) != before {
		t.Fatal("a refused rollover changed the database")
	}
}

func TestSeasonRolloverWritesOneEntryAndRetriesToTheSameReceipt(t *testing.T) {
	t.Parallel()
	r := newRolloverDB(t)
	events, err := r.s.Events(t.Context(), "native.test", commissioner.Stream, 0)
	if err != nil {
		t.Fatal(err)
	}
	command := r.rollover(t, 2027, false, "go")
	receipt := r.do(t, command)
	after, err := r.s.Events(t.Context(), "native.test", commissioner.Stream, 0)
	if err != nil || len(after) != len(events)+1 || receipt.Plan == nil || receipt.Entry == "" {
		t.Fatalf("a rollover must write exactly one entry: %d -> %d %+v %v", len(events), len(after), receipt, err)
	}
	record := after[len(after)-1].Payload.Commissioner
	if record.Before == nil || record.Receipt.Plan == nil || record.Receipt.Plan.BeforeDigest == "" ||
		record.Receipt.Plan.BeforeDigest != receipt.Plan.BeforeDigest {
		t.Fatalf("the entry holds no before-state digest and plan: %+v", record)
	}
	fingerprintAfter := fingerprint(t, r.s)
	retry := r.do(t, command)
	if retry.Entry != receipt.Entry || !reflect.DeepEqual(retry.Plan, receipt.Plan) ||
		fingerprint(t, r.s) != fingerprintAfter {
		t.Fatal("a retry wrote again or returned a different receipt")
	}
	changed := command
	changed.Args = commissioner.RawArgs{Rollover: &commissioner.RawRolloverArgs{To: 2027, DryRun: true}}
	if _, err := r.s.ExecuteCommissioner(t.Context(), changed, "creator", nil); err == nil {
		t.Fatal("a request key was reused for a different rollover")
	}
}

func (r *rolloverDB) readings(t *testing.T) (season, rosterSeason, rulesSeason int, rosterMax string) {
	t.Helper()
	check, err := r.s.NativeRosterCheck(t.Context(), "native.test", nativecheck.Facts{})
	if err != nil {
		t.Fatal(err)
	}
	directory, err := r.s.NativeDirectory(t.Context(), "native.test")
	if err != nil {
		t.Fatal(err)
	}
	assetSeason, _ := r.playerStatuses(t)
	return assetSeason, directory.RosterSeason, check.RulesSeason, string(ruleRow(t, r.s, "roster.limits").Current)
}

func TestSeasonRolloverReplaysAndRevertsAfterReopening(t *testing.T) {
	t.Parallel()
	r := newRolloverDB(t)
	beforeSeason, beforeRoster, beforeRules, beforeLimits := r.readings(t)
	if beforeSeason != 2026 || beforeRoster != 2026 || beforeRules != 2026 || !strings.Contains(beforeLimits, "80") {
		t.Fatalf("before: %d %d %d %s", beforeSeason, beforeRoster, beforeRules, beforeLimits)
	}
	receipt := r.do(t, r.rollover(t, 2027, false, "go"))
	for pass := range 2 {
		if pass == 1 {
			r.reopenFile(t)
		}
		season, roster, rules, limits := r.readings(t)
		if season != 2027 || roster != 2027 || rules != 2027 || !strings.Contains(limits, "75") {
			t.Fatalf("pass %d after the rollover: %d %d %d %s", pass, season, roster, rules, limits)
		}
		if _, statuses := r.playerStatuses(t); slicesContain(statuses, domain.RosterIR) ||
			slicesContain(statuses, domain.RosterTaxi) {
			t.Fatalf("pass %d: a reserve player was not returned to active: %v", pass, statuses)
		}
	}
	revert := reverseCommand(receipt.Entry, r.version(t), "undo-rollover")
	r.do(t, revert)
	for pass := range 2 {
		if pass == 1 {
			r.reopenFile(t)
		}
		season, roster, rules, limits := r.readings(t)
		if season != beforeSeason || roster != beforeRoster || rules != beforeRules || limits != beforeLimits {
			t.Fatalf("pass %d after the revert: %d %d %d %s", pass, season, roster, rules, limits)
		}
		if _, statuses := r.playerStatuses(t); !slicesContain(statuses, domain.RosterIR) ||
			!slicesContain(statuses, domain.RosterTaxi) {
			t.Fatalf("pass %d: the revert did not put reserve players back: %v", pass, statuses)
		}
	}
}

func slicesContain(statuses []domain.RosterStatus, want domain.RosterStatus) bool {
	for _, status := range statuses {
		if status == want {
			return true
		}
	}
	return false
}

func TestSeasonRolloverCannotBeReversedAroundLaterChanges(t *testing.T) {
	t.Parallel()
	r := newRolloverDB(t)
	receipt := r.do(t, r.rollover(t, 2027, false, "go"))
	r.c.now = r.c.now.Add(time.Hour)
	later := r.rule(t, "roster.limits", `{"min":10,"max":90}`, "now")
	_, err := r.s.ExecuteCommissioner(t.Context(), reverseCommand(receipt.Entry, r.version(t), "too-early"), "creator", nil)
	var refused *commissioner.Refusal
	if !errors.As(err, &refused) || !strings.Contains(refused.Reason(), "Reverse the later changes first") {
		t.Fatalf("a rollover with later changes was reversed: %v", err)
	}
	r.do(t, reverseCommand(later.Entry, r.version(t), "undo-later"))
	r.do(t, reverseCommand(receipt.Entry, r.version(t), "undo-rollover"))
}

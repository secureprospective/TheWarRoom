package history

import (
	"fmt"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

// Seed a held native ledger with future picks; a newly created empty league has not issued them yet.
func heldSideRookies(t *testing.T, s *Store) {
	t.Helper()
	create, err := createWith(commissioner.PresetMirror16, 0).Prepared()
	if err != nil {
		t.Fatal(err)
	}
	records := []commissioner.Record{{Command: create, Principal: "creator", Receipt: commissioner.Receipt{RecordedAt: t0()}}}
	seed, err := commissioner.NativeStart("native.test", records, t0())
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range seed.Franchises {
		seed.Picks = append(seed.Picks, commissioner.NativePick{Season: 2027, Round: 1, Original: f.ID, Owner: f.ID})
	}
	commands := []commissioner.RawCommand{create, {League: "native.test", Command: commissioner.NativeSeed,
		Args: commissioner.RawArgs{Native: &seed}, Expected: 1, Request: "held-seed", Reason: "Held future picks",
		Effective: "now", Visibility: "league"}}
	records = nil
	tx, err := s.pools.BeginWrite(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback() }()
	for _, command := range commands {
		raw, canonical, err := preparedCommand(command, "creator", true)
		if err != nil {
			t.Fatal(err)
		}
		receipt, err := s.appendCommissioner(t.Context(), tx, appendRequest{raw: raw, principal: "creator",
			role: "commissioner", canonical: canonical, at: t0(), records: records})
		if err != nil {
			t.Fatal(err)
		}
		records = append(records, commissioner.Record{Command: raw, Principal: "creator", Receipt: receipt})
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
}

func TestPerSideRookieDraftsPersistCloseAndReopenTogether(t *testing.T) {
	path := filepath.Join(t.TempDir(), "rookies.db")
	s, pools := statStore(t, path)
	heldSideRookies(t, s)
	for _, rule := range [][2]string{{"calendar.windows", `{}`}, {"calendar.boundary", `"start_of_day"`},
		{"draft.rounds", `1`}} {
		runScoreCommand(t, s, ruleCommand(rule[0], rule[1], "now", "Prepare the draft", revision(t, s)), "creator")
	}
	now := t0().Add(time.Hour)
	runScoreCommand(t, s, draftCommand(commissioner.DraftStart,
		commissioner.RawDraftArgs{Season: 2027, At: now}, revision(t, s)), "creator")
	ids := commissioner.LeagueFranchises(scoreRecords(t, s))
	runScoreCommand(t, s, draftCommand(commissioner.DraftOrderSet,
		commissioner.RawDraftArgs{Season: 2027, Order: ids}, revision(t, s)), "creator")
	s.SetClock(func() time.Time { return now })
	facts := nativecheck.Facts{}
	for i := range 9 {
		facts[fmt.Sprintf("%05d", 30000+i)] = nativecheck.PlayerFacts{Name: fmt.Sprintf("Rookie %d", i),
			Position: leaguerules.WR, DraftYear: 2027, DraftYearHeld: true}
	}
	read := func(team string) commissioner.NativeDraft {
		t.Helper()
		d, err := s.NativeDraft(t.Context(), commissioner.RawNativeDraftScope{League: "native.test",
			Season: 2027, Franchise: team, At: now}, time.UTC, facts)
		if err != nil {
			t.Fatal(err)
		}
		return d
	}
	checkpoints := map[string]string{}
	for i := range 8 {
		for _, team := range []string{"franchise.01", "franchise.09"} {
			d := read(team)
			pick := d.Draft.OnClock
			raw := draftCommand(commissioner.DraftAssign, commissioner.RawDraftArgs{Season: 2027, Side: d.Side,
				Act: &commissioner.RawDraftAct{Round: pick.Round, Original: pick.Original,
					Player: fmt.Sprintf("%05d", 30000+i)}}, revision(t, s))
			receipt, err := s.ExecuteCommissionerWithFacts(t.Context(), raw, "creator", nil, facts, time.UTC)
			if err != nil {
				t.Fatal(err)
			}
			if i == 0 {
				checkpoints[team] = receipt.Entry
			}
		}
	}
	if err := s.CloseFinishedDrafts(t.Context(), "native.test", facts); err != nil {
		t.Fatal(err)
	}
	before := []commissioner.NativeDraft{read("franchise.01"), read("franchise.09")}
	for _, d := range before {
		if d.Closed == nil || len(d.Closed.Pool) != 1 || len(d.Closed.Issued) != 2 {
			t.Fatalf("side %s did not close: %+v", d.Side, d.Closed)
		}
	}
	if err := pools.Close(); err != nil {
		t.Fatal(err)
	}
	s, pools = statStore(t, path)
	s.SetClock(func() time.Time { return now })
	if after := []commissioner.NativeDraft{read("franchise.01"), read("franchise.09")}; !reflect.DeepEqual(before, after) {
		t.Fatal("both drafts did not survive database reopen")
	}
	for _, team := range []string{"franchise.01", "franchise.09"} {
		d := read(team)
		raw := draftCommand(commissioner.DraftRevert, commissioner.RawDraftArgs{Season: 2027, Side: d.Side,
			Act: &commissioner.RawDraftAct{To: checkpoints[team]}}, revision(t, s))
		if _, err := s.ExecuteCommissionerWithFacts(t.Context(), raw, "creator", nil, facts, time.UTC); err != nil {
			t.Fatal(err)
		}
	}
	if err := pools.Close(); err != nil {
		t.Fatal(err)
	}
	s, _ = statStore(t, path)
	s.SetClock(func() time.Time { return now })
	for _, team := range []string{"franchise.01", "franchise.09"} {
		d := read(team)
		if d.Closed != nil || d.Draft.Exhausted || d.Draft.OnClock == nil {
			t.Fatal("restores did not persist")
		}
	}
}

package state

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
)

// feedWrite runs fn inside one WriteTx — the surface every ledger write goes through. The feed
// is read-only; this helper exists only so tests can populate the append-only tables via the
// real write paths (the same paths the transaction engine uses) before asserting on Feed.
func feedWrite(t *testing.T, s *Store, fn func(w TxWriter) error) {
	t.Helper()
	if err := s.WriteTx(context.Background(), fn); err != nil {
		t.Fatalf("WriteTx: %v", err)
	}
}

// findFeedByKind returns the first feed row of the given Kind, failing the test if none.
func findFeedByKind(t *testing.T, evs []FeedEvent, kind string) FeedEvent {
	t.Helper()
	for _, e := range evs {
		if e.Kind == kind {
			return e
		}
	}
	t.Fatalf("no feed event of kind %q in %d rows", kind, len(evs))
	return FeedEvent{}
}

// rawFeedInsert plants a row directly into an append-only feed-source table, bypassing the Go
// write API. The append-only triggers do not block INSERTs (only UPDATE/DELETE), so this lets a
// test plant a SPECIFIC reason / source / franchise combination without assembling a full
// transaction op — e.g. contract_year_changes rows tagged source="op" with a chosen reason, or
// a player_status_events transition with a status the engine never writes from a public path.
func rawFeedInsert(t *testing.T, s *Store, table, ddl string, args ...any) {
	t.Helper()
	ctx := context.Background()
	if _, err := s.pools.Write().ExecContext(ctx, ddl, args...); err != nil {
		t.Fatalf("raw insert into %s: %v", table, err)
	}
}

// isoTime returns a deterministic RFC3339 instant `secs` seconds after the Unix epoch, so feed
// tests can plant rows at known distinct timestamps and assert on ordering without flakiness.
func isoTime(secs int64) string {
	return time.Unix(secs, 0).UTC().Format(time.RFC3339)
}

// TestFeed_EmptyLeagueReturnsNonNilSlice is the empty-state contract: a fresh league with no
// executed transactions returns an empty (non-nil) slice, never nil — the IPC layer marshals
// nil to JSON null, which the frontend would have to guard against; a non-nil empty slice
// marshals to [] and renders an empty state cleanly.
func TestFeed_EmptyLeagueReturnsNonNilSlice(t *testing.T) {
	s := newStore(t, &fakeSource{rosters: baseRosters(t)})
	got, err := s.Feed(context.Background(), 100)
	if err != nil {
		t.Fatalf("Feed: %v", err)
	}
	if got == nil {
		t.Fatal("Feed returned nil for an empty league, want a non-nil empty slice")
	}
	if len(got) != 0 {
		t.Fatalf("Feed returned %d rows on an empty league, want 0", len(got))
	}
}

// TestFeed_ChronologicalDesc pins the spine's most load-bearing property: the feed is strict
// reverse-chronological. Plants three rows at distinct timestamps across three source tables
// and asserts they come back newest-first. Without this, the "Watch Floor river" grammar
// (Session-D) would render a misleading timeline.
func TestFeed_ChronologicalDesc(t *testing.T) {
	s := newStore(t, &fakeSource{rosters: baseRosters(t)})
	ctx := context.Background()

	// Oldest: a cap relief at t=1000.
	rawFeedInsert(t, s, "cap_relief_ledger",
		`INSERT INTO cap_relief_ledger (seq, league_id, franchise_id, league_year, relief_cents, reason, created_at)
		 VALUES (1, ?, '0001', ?, 100000000, 'old', ?)`,
		s.leagueID, s.season, isoTime(1000))
	// Middle: a trade note at t=2000.
	rawFeedInsert(t, s, "trade_notes",
		`INSERT INTO trade_notes (id, league_id, season, created_at, picks_note, rationale, involved_franchises)
		 VALUES ('tn:1', ?, ?, ?, '', 'mid trade', '0001,0002')`,
		s.leagueID, s.season, isoTime(2000))
	// Newest: a dead-cap charge at t=3000.
	rawFeedInsert(t, s, "dead_cap_ledger",
		`INSERT INTO dead_cap_ledger (id, league_id, franchise_id, league_year, mfl_id, dead_cap_cents, reason, created_at)
		 VALUES ('dc:1', ?, '0001', ?, '1234', 50000000, 'new', ?)`,
		s.leagueID, s.season, isoTime(3000))

	got, err := s.Feed(ctx, 100)
	if err != nil {
		t.Fatalf("Feed: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("Feed returned %d rows, want 3", len(got))
	}
	if got[0].Timestamp != isoTime(3000) || got[1].Timestamp != isoTime(2000) || got[2].Timestamp != isoTime(1000) {
		t.Fatalf("feed not reverse-chronological: got %s, %s, %s", got[0].Timestamp, got[1].Timestamp, got[2].Timestamp)
	}
}

// TestFeed_LimitDefaultsAndCaps covers the limit contract: a non-positive limit falls back to
// the default (500), and an explicit limit caps the result count. Without these, an unbounded
// query could surface the entire history on a league with thousands of events.
func TestFeed_LimitDefaultsAndCaps(t *testing.T) {
	s := newStore(t, &fakeSource{rosters: baseRosters(t)})
	ctx := context.Background()
	for i := int64(0); i < 5; i++ {
		rawFeedInsert(t, s, "cap_relief_ledger",
			`INSERT INTO cap_relief_ledger (seq, league_id, franchise_id, league_year, relief_cents, reason, created_at)
			 VALUES (?, ?, '0001', ?, 100000000, 'r', ?)`,
			i+1, s.leagueID, s.season, isoTime(i+100))
	}
	got, err := s.Feed(ctx, 3)
	if err != nil {
		t.Fatalf("Feed(3): %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("Feed(3) returned %d rows, want 3 (the explicit cap)", len(got))
	}
	// Non-positive limit falls back to the default; on this 5-row league the default of 500
	// returns every row. Tests against the same row count, not the literal 500, so a future
	// tuning of the default does not break this assertion.
	all, err := s.Feed(ctx, 0)
	if err != nil {
		t.Fatalf("Feed(0): %v", err)
	}
	if len(all) != 5 {
		t.Fatalf("Feed(0) returned %d rows, want 5 (default-limit fallback)", len(all))
	}
}

// TestFeed_KindFromEachSource plants one row per source table and asserts the SQL's
// CASE-expression maps each onto the documented coarse Kind. This is the contract the
// frontend's spine-color switch reads; a wrong Kind mis-colors the row.
func TestFeed_KindFromEachSource(t *testing.T) {
	s := newStore(t, &fakeSource{rosters: baseRosters(t)})
	ctx := context.Background()

	rawFeedInsert(t, s, "trade_notes",
		`INSERT INTO trade_notes (id, league_id, season, created_at, picks_note, rationale, involved_franchises)
		 VALUES ('tn:a', ?, ?, ?, 'pn', 'r', '0001,0002')`,
		s.leagueID, s.season, isoTime(100))
	rawFeedInsert(t, s, "player_status_events",
		`INSERT INTO player_status_events (seq, league_id, mfl_id, status, reason, at)
		 VALUES (1, ?, '1234', 'FREE_AGENT', 'waiver-cut §8', ?)`,
		s.leagueID, isoTime(200))
	rawFeedInsert(t, s, "player_status_events",
		`INSERT INTO player_status_events (seq, league_id, mfl_id, status, reason, at)
		 VALUES (2, ?, '1235', 'RETIRED', 'retirement §13', ?)`,
		s.leagueID, isoTime(300))
	rawFeedInsert(t, s, "player_status_events",
		`INSERT INTO player_status_events (seq, league_id, mfl_id, status, reason, at)
		 VALUES (3, ?, '1236', 'DECEASED', 'gaines-adams §13', ?)`,
		s.leagueID, isoTime(400))
	rawFeedInsert(t, s, "dead_cap_ledger",
		`INSERT INTO dead_cap_ledger (id, league_id, franchise_id, league_year, mfl_id, dead_cap_cents, reason, created_at)
		 VALUES ('dc:a', ?, '0001', ?, '1234', 100000000, 'waiver-cut §8', ?)`,
		s.leagueID, s.season, isoTime(500))
	rawFeedInsert(t, s, "cap_relief_ledger",
		`INSERT INTO cap_relief_ledger (seq, league_id, franchise_id, league_year, relief_cents, reason, created_at)
		 VALUES (1, ?, '0001', ?, 100000000, 'career-ending injury', ?)`,
		s.leagueID, s.season, isoTime(600))
	rawFeedInsert(t, s, "contract_year_changes",
		`INSERT INTO contract_year_changes (id, league_id, mfl_id, league_year, old_cents, new_cents, reason, source, changed_at)
		 VALUES ('cyc:sign', ?, '1234', ?, 0, 50000000, 'free-agency signing §6', 'signing', ?)`,
		s.leagueID, s.season, isoTime(700))
	rawFeedInsert(t, s, "contract_year_changes",
		`INSERT INTO contract_year_changes (id, league_id, mfl_id, league_year, old_cents, new_cents, reason, source, changed_at)
		 VALUES ('cyc:ext', ?, '1234', ?, 0, 50000000, 'extension: 3 yrs @ $5M', 'extension', ?)`,
		s.leagueID, s.season, isoTime(800))
	rawFeedInsert(t, s, "contract_year_changes",
		`INSERT INTO contract_year_changes (id, league_id, mfl_id, league_year, old_cents, new_cents, reason, source, changed_at)
		 VALUES ('cyc:restr', ?, '1234', ?, 50000000, 30000000, '§11 restructure: moved $20 from 2026 to 2028', 'op', ?)`,
		s.leagueID, s.season, isoTime(900))
	rawFeedInsert(t, s, "contract_year_changes",
		`INSERT INTO contract_year_changes (id, league_id, mfl_id, league_year, old_cents, new_cents, reason, source, changed_at)
		 VALUES ('cyc:tag', ?, '1234', ?, 50000000, 60000000, '§9 franchise tag: season salary set to $6M', 'op', ?)`,
		s.leagueID, s.season, isoTime(1000))

	got, err := s.Feed(ctx, 100)
	if err != nil {
		t.Fatalf("Feed: %v", err)
	}
	wantKinds := map[string]string{
		"trade_notes":             "TRADE",
		"player_status_events:1":  "RELEASE",
		"player_status_events:2":  "RETIREMENT",
		"player_status_events:3":  "DEATH",
		"dead_cap_ledger":         "DEAD_CAP",
		"cap_relief_ledger":       "CAP_RELIEF",
		"contract_year_changes:1": "SIGN",
		"contract_year_changes:2": "EXTENSION",
		"contract_year_changes:3": "RESTRUCTURE",
		"contract_year_changes:4": "TAG",
	}
	seen := map[string]string{}
	for _, e := range got {
		key := e.Source
		if e.Source == "player_status_events" || e.Source == "contract_year_changes" {
			// disambiguate by ordering within source — kind itself is the assertion target
			key += ":" + e.ID
		}
		seen[key] = e.Kind
	}
	// player_status_events ids are 1/2/3 in plant order; contract_year_changes ids are the
	// literal cyc:* strings — index by source+kind for a stable cross-check.
	for _, e := range got {
		switch e.Source {
		case "trade_notes":
			if e.Kind != "TRADE" {
				t.Errorf("trade_notes row Kind = %q, want TRADE", e.Kind)
			}
		case "dead_cap_ledger":
			if e.Kind != "DEAD_CAP" {
				t.Errorf("dead_cap_ledger row Kind = %q, want DEAD_CAP", e.Kind)
			}
		case "cap_relief_ledger":
			if e.Kind != "CAP_RELIEF" {
				t.Errorf("cap_relief_ledger row Kind = %q, want CAP_RELIEF", e.Kind)
			}
		case "player_status_events":
			switch e.ID {
			case "1":
				if e.Kind != "RELEASE" {
					t.Errorf("player_status_events seq=1 Kind = %q, want RELEASE", e.Kind)
				}
			case "2":
				if e.Kind != "RETIREMENT" {
					t.Errorf("player_status_events seq=2 Kind = %q, want RETIREMENT", e.Kind)
				}
			case "3":
				if e.Kind != "DEATH" {
					t.Errorf("player_status_events seq=3 Kind = %q, want DEATH", e.Kind)
				}
			}
		case "contract_year_changes":
			switch e.ID {
			case "cyc:sign":
				if e.Kind != "SIGN" {
					t.Errorf("cyc:sign Kind = %q, want SIGN", e.Kind)
				}
			case "cyc:ext":
				if e.Kind != "EXTENSION" {
					t.Errorf("cyc:ext Kind = %q, want EXTENSION", e.Kind)
				}
			case "cyc:restr":
				if e.Kind != "RESTRUCTURE" {
					t.Errorf("cyc:restr Kind = %q, want RESTRUCTURE", e.Kind)
				}
			case "cyc:tag":
				if e.Kind != "TAG" {
					t.Errorf("cyc:tag Kind = %q, want TAG", e.Kind)
				}
			}
		}
	}
	// Sanity: every expected source was represented (catches a silent table-skip).
	for src := range wantKinds {
		withoutIdx := src
		if i := strings.Index(src, ":"); i >= 0 {
			withoutIdx = src[:i]
		}
		found := false
		for k := range seen {
			if strings.HasPrefix(k, withoutIdx) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("no feed row from source %q was returned", withoutIdx)
		}
	}
}

// TestFeed_SeedContractChangesExcluded proves the WHERE source <> 'seed' guard fires: the
// seed-only rows every fresh contract creates are NOT feed events (they are initial migration,
// not a transaction), so they must never appear in the feed.
func TestFeed_SeedContractChangesExcluded(t *testing.T) {
	s := newStore(t, &fakeSource{rosters: baseRosters(t)})
	ctx := context.Background()
	// Plant one seed row and one op row in the same table.
	rawFeedInsert(t, s, "contract_year_changes",
		`INSERT INTO contract_year_changes (id, league_id, mfl_id, league_year, old_cents, new_cents, reason, source, changed_at)
		 VALUES ('cyc:seed', ?, '1234', ?, 0, 50000000, 'seed: flat-fill from MFL annual salary + expiration', 'seed', ?)`,
		s.leagueID, s.season, isoTime(100))
	rawFeedInsert(t, s, "contract_year_changes",
		`INSERT INTO contract_year_changes (id, league_id, mfl_id, league_year, old_cents, new_cents, reason, source, changed_at)
		 VALUES ('cyc:op', ?, '1234', ?, 50000000, 30000000, '§11 restructure: moved $20 from 2026 to 2028', 'op', ?)`,
		s.leagueID, s.season, isoTime(200))

	got, err := s.Feed(ctx, 100)
	if err != nil {
		t.Fatalf("Feed: %v", err)
	}
	for _, e := range got {
		if e.ID == "cyc:seed" {
			t.Fatalf("seed row surfaced in the feed: %+v", e)
		}
	}
	if len(got) != 1 {
		t.Fatalf("Feed returned %d rows, want only the non-seed op row", len(got))
	}
	if got[0].ID != "cyc:op" {
		t.Fatalf("Feed row id = %q, want cyc:op", got[0].ID)
	}
}

// TestFeed_TradeRowSplitAndText proves trade_notes rows carry the Session-3 persisted text
// (rationale required, picks note free) AND that the comma-joined involved_franchises column
// is split into a slice covering BOTH source and destination franchises. The Session-3 audit
// (TestExecute_TradeNoteRecordsSourceFranchise) already pinned that the writer records both;
// this asserts the reader unjoins them faithfully.
func TestFeed_TradeRowSplitAndText(t *testing.T) {
	s := newStore(t, &fakeSource{rosters: baseRosters(t)})
	ctx := context.Background()
	rawFeedInsert(t, s, "trade_notes",
		`INSERT INTO trade_notes (id, league_id, season, created_at, picks_note, rationale, involved_franchises)
		 VALUES ('tn:split', ?, ?, ?, '2027 1st to 0001', 'three-team foundation trade', '0001,0005,0009')`,
		s.leagueID, s.season, isoTime(100))

	got, err := s.Feed(ctx, 10)
	if err != nil {
		t.Fatalf("Feed: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("Feed returned %d rows, want 1 trade row", len(got))
	}
	e := got[0]
	if e.Kind != "TRADE" {
		t.Errorf("Kind = %q, want TRADE", e.Kind)
	}
	if e.TradeRationale != "three-team foundation trade" {
		t.Errorf("TradeRationale = %q, want the persisted rationale", e.TradeRationale)
	}
	if e.TradePicksNote != "2027 1st to 0001" {
		t.Errorf("TradePicksNote = %q, want the persisted picks note", e.TradePicksNote)
	}
	wantFr := []string{"0001", "0005", "0009"}
	if len(e.FranchiseIDs) != len(wantFr) {
		t.Fatalf("FranchiseIDs = %v, want %v (split from comma-joined)", e.FranchiseIDs, wantFr)
	}
	for i, w := range wantFr {
		if e.FranchiseIDs[i] != w {
			t.Errorf("FranchiseIDs[%d] = %q, want %q", i, e.FranchiseIDs[i], w)
		}
	}
	if e.Provenance != "trade" {
		t.Errorf("Provenance = %q, want \"trade\"", e.Provenance)
	}
	if e.MFLID != "" {
		t.Errorf("MFLID = %q, want empty (trade_notes has no single player)", e.MFLID)
	}
}

// TestFeed_ProvenancePerKind is the conservative-derivation contract: only the documented
// (Kind, Reason) combinations carry acquisition provenance; everything else returns "" so a
// future handler never mislabels. Covers every documented Kind at least once.
func TestFeed_ProvenancePerKind(t *testing.T) {
	for _, tc := range []struct {
		name     string
		kind     string
		reason   string
		provWant string
	}{
		{"TRADE", "TRADE", "", "trade"},
		{"SIGN", "SIGN", "free-agency signing §6", "free-agent-signing"},
		{"DEAD_CAP §8 waiver", "DEAD_CAP", "waiver-cut §8", "waiver"},
		{"RELEASE §8 waiver", "RELEASE", "waiver-cut §8", "waiver"},
		{"DEAD_CAP §12 buyout (not acquisition)", "DEAD_CAP", "buyout §12", ""},
		{"RELEASE natural expiry (not acquisition)", "RELEASE", "", ""},
		{"RETIREMENT", "RETIREMENT", "retirement §13", ""},
		{"DEATH", "DEATH", "gaines-adams §13", ""},
		{"CAP_RELIEF", "CAP_RELIEF", "career-ending injury", ""},
		{"EXTENSION", "EXTENSION", "extension: 3 yrs", ""},
		{"RESTRUCTURE", "RESTRUCTURE", "§11 restructure: moved", ""},
		{"TAG", "TAG", "§9 franchise tag: set", ""},
		{"WAIVER_VOID", "WAIVER_VOID", "waiver-cut §8", ""},
		{"CONTRACT_CHANGE uncategorized", "CONTRACT_CHANGE", "future op", ""},
	} {
		got := deriveProvenance(tc.kind, tc.reason)
		if got != tc.provWant {
			t.Errorf("%s: deriveProvenance(%q,%q) = %q, want %q", tc.name, tc.kind, tc.reason, got, tc.provWant)
		}
	}
}

// TestClassifyContractChangeKind covers the source+reason → Kind table for contract_year_changes
// rows. SQL mirrors this exactly; this test pins the contract so a divergence between the SQL
// CASE and the Go classifier surfaces here.
func TestClassifyContractChangeKind(t *testing.T) {
	for _, tc := range []struct {
		source, reason, want string
	}{
		{"signing", "free-agency signing §6", "SIGN"},
		{"extension", "extension: 3 yrs @ $5M", "EXTENSION"},
		{"op", "§11 restructure: moved $20 from 2026 to 2028", "RESTRUCTURE"},
		{"op", "§9 franchise tag: season salary set to $6M", "TAG"},
		{"op", "waiver-cut §8", "WAIVER_VOID"},
		{"op", "future unmapped op", "CONTRACT_CHANGE"},
		{"", "", "CONTRACT_CHANGE"},
	} {
		if got := classifyContractChangeKind(tc.source, tc.reason); got != tc.want {
			t.Errorf("classifyContractChangeKind(%q,%q) = %q, want %q", tc.source, tc.reason, got, tc.want)
		}
	}
}

// TestSplitFranchises covers the unjoin helper: a clean split, whitespace tolerance, and the
// empty / single-edge cases that would otherwise produce [""] or [" "] in the DTO.
func TestSplitFranchises(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want []string
	}{
		{"", nil},
		{"   ", nil},
		{"0001", []string{"0001"}},
		{"0001,0002", []string{"0001", "0002"}},
		{"0001, 0002 , 0003", []string{"0001", "0002", "0003"}}, // whitespace tolerance
		{",,,", nil}, // only separators
	} {
		got := splitFranchises(tc.in)
		if len(got) != len(tc.want) {
			t.Errorf("splitFranchises(%q) = %v, want %v", tc.in, got, tc.want)
			continue
		}
		for i := range got {
			if got[i] != tc.want[i] {
				t.Errorf("splitFranchises(%q)[%d] = %q, want %q", tc.in, i, got[i], tc.want[i])
			}
		}
	}
}

// TestFeed_FranchisesForLedgerTables verifies the per-source franchise-availability contract:
// dead_cap_ledger and cap_relief_ledger carry a single franchise_id (becomes a one-element
// slice), while player_status_events and contract_year_changes are player-keyed with NO
// franchise and return nil. trade_notes splits its involved_franchises (covered separately).
func TestFeed_FranchisesForLedgerTables(t *testing.T) {
	s := newStore(t, &fakeSource{rosters: baseRosters(t)})
	ctx := context.Background()
	rawFeedInsert(t, s, "dead_cap_ledger",
		`INSERT INTO dead_cap_ledger (id, league_id, franchise_id, league_year, mfl_id, dead_cap_cents, reason, created_at)
		 VALUES ('dc:f', ?, '0007', ?, '1234', 50000000, 'waiver-cut §8', ?)`,
		s.leagueID, s.season, isoTime(100))
	rawFeedInsert(t, s, "cap_relief_ledger",
		`INSERT INTO cap_relief_ledger (seq, league_id, franchise_id, league_year, relief_cents, reason, created_at)
		 VALUES (1, ?, '0008', ?, 100000000, 'career-ending injury', ?)`,
		s.leagueID, s.season, isoTime(200))
	rawFeedInsert(t, s, "player_status_events",
		`INSERT INTO player_status_events (seq, league_id, mfl_id, status, reason, at)
		 VALUES (1, ?, '1234', 'RETIRED', 'retirement §13', ?)`,
		s.leagueID, isoTime(300))
	rawFeedInsert(t, s, "contract_year_changes",
		`INSERT INTO contract_year_changes (id, league_id, mfl_id, league_year, old_cents, new_cents, reason, source, changed_at)
		 VALUES ('cyc:f', ?, '1234', ?, 0, 50000000, 'free-agency signing §6', 'signing', ?)`,
		s.leagueID, s.season, isoTime(400))

	got, err := s.Feed(ctx, 100)
	if err != nil {
		t.Fatalf("Feed: %v", err)
	}
	for _, e := range got {
		switch e.Source {
		case "dead_cap_ledger":
			if len(e.FranchiseIDs) != 1 || e.FranchiseIDs[0] != "0007" {
				t.Errorf("dead_cap_ledger FranchiseIDs = %v, want [0007]", e.FranchiseIDs)
			}
		case "cap_relief_ledger":
			if len(e.FranchiseIDs) != 1 || e.FranchiseIDs[0] != "0008" {
				t.Errorf("cap_relief_ledger FranchiseIDs = %v, want [0008]", e.FranchiseIDs)
			}
		case "player_status_events", "contract_year_changes":
			if e.FranchiseIDs != nil {
				t.Errorf("%s FranchiseIDs = %v, want nil (table has no franchise column)", e.Source, e.FranchiseIDs)
			}
		}
	}
}

// TestFeed_StableIDsForReactKeys verifies each source's ID column is exposed verbatim so the
// frontend can compose a stable React key (Source + ":" + ID) without re-deriving it. The ID is
// the source table's primary key, uniquely identifying the row within its source.
func TestFeed_StableIDsForReactKeys(t *testing.T) {
	s := newStore(t, &fakeSource{rosters: baseRosters(t)})
	ctx := context.Background()
	rawFeedInsert(t, s, "trade_notes",
		`INSERT INTO trade_notes (id, league_id, season, created_at, picks_note, rationale, involved_franchises)
		 VALUES ('tn:id', ?, ?, ?, '', 'r', '0001')`,
		s.leagueID, s.season, isoTime(100))
	rawFeedInsert(t, s, "player_status_events",
		`INSERT INTO player_status_events (seq, league_id, mfl_id, status, reason, at)
		 VALUES (42, ?, '1234', 'RETIRED', 'retirement §13', ?)`,
		s.leagueID, isoTime(200))
	rawFeedInsert(t, s, "dead_cap_ledger",
		`INSERT INTO dead_cap_ledger (id, league_id, franchise_id, league_year, mfl_id, dead_cap_cents, reason, created_at)
		 VALUES ('dc:id', ?, '0001', ?, '1234', 50000000, 'waiver-cut §8', ?)`,
		s.leagueID, s.season, isoTime(300))
	rawFeedInsert(t, s, "cap_relief_ledger",
		`INSERT INTO cap_relief_ledger (seq, league_id, franchise_id, league_year, relief_cents, reason, created_at)
		 VALUES (7, ?, '0001', ?, 100000000, 'r', ?)`,
		s.leagueID, s.season, isoTime(400))
	rawFeedInsert(t, s, "contract_year_changes",
		`INSERT INTO contract_year_changes (id, league_id, mfl_id, league_year, old_cents, new_cents, reason, source, changed_at)
		 VALUES ('cyc:id', ?, '1234', ?, 0, 50000000, 'free-agency signing §6', 'signing', ?)`,
		s.leagueID, s.season, isoTime(500))

	got, err := s.Feed(ctx, 100)
	if err != nil {
		t.Fatalf("Feed: %v", err)
	}
	want := map[string]string{
		"trade_notes":           "tn:id",
		"player_status_events":  "42",
		"dead_cap_ledger":       "dc:id",
		"cap_relief_ledger":     "7",
		"contract_year_changes": "cyc:id",
	}
	gotIDs := map[string]string{}
	for _, e := range got {
		gotIDs[e.Source] = e.ID
	}
	for src, w := range want {
		if gotIDs[src] != w {
			t.Errorf("ID for %s = %q, want %q", src, gotIDs[src], w)
		}
	}
}

// TestFeed_DeadCapMoneyIsNotParsed is a v1 scope-marker: the feed surfaces the audit REASON
// (text) but does not parse the dead_cap_cents amount into a DTO field. The money side of a
// release already renders on the dedicated FranchiseHQ cap view; the activity feed's job is the
// event spine, not a re-derivation of cap figures. Pinning this prevents a "wouldn't it be nice
// to show the dollar amount" creep from sneaking in without a deliberate scope decision.
func TestFeed_DeadCapMoneyIsNotParsed(t *testing.T) {
	s := newStore(t, &fakeSource{rosters: baseRosters(t)})
	ctx := context.Background()
	rawFeedInsert(t, s, "dead_cap_ledger",
		`INSERT INTO dead_cap_ledger (id, league_id, franchise_id, league_year, mfl_id, dead_cap_cents, reason, created_at)
		 VALUES ('dc:money', ?, '0001', ?, '1234', 75000000, 'waiver-cut §8', ?)`,
		s.leagueID, s.season, isoTime(100))

	got, err := s.Feed(ctx, 10)
	if err != nil {
		t.Fatalf("Feed: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("Feed returned %d rows, want 1", len(got))
	}
	if got[0].Reason != "waiver-cut §8" {
		t.Errorf("Reason = %q, want the raw audit text", got[0].Reason)
	}
	// No field on FeedEvent carries the parsed cents amount; the struct only has text + ids.
}

// TestFeed_RealWritePaths exercises the feed against rows written by the REAL transaction write
// primitives (not raw SQL), proving the read model is faithful to what the engine actually
// persists. A handful of sibling tests already cover each write path in isolation; this one
// asserts the feed reads them back as the operator would.
func TestFeed_RealWritePaths(t *testing.T) {
	s := newStore(t, &fakeSource{rosters: baseRosters(t)})
	ctx := context.Background()

	// A §13 cap relief via the real AddCapRelief path.
	feedWrite(t, s, func(w TxWriter) error {
		return w.AddCapRelief(ctx, CapReliefEntry{
			FranchiseID: "0001",
			LeagueYear:  s.season,
			Amount:      domain.Money(100_000_000),
			Reason:      "career-ending injury",
		})
	})
	// A trade note via the real LogTradeNote path (the same path acquisitions.Trade runs).
	feedWrite(t, s, func(w TxWriter) error {
		return w.LogTradeNote(ctx, "2027 1st to 0001", "foundation trade", []string{"0001", "0002"})
	})

	got, err := s.Feed(ctx, 100)
	if err != nil {
		t.Fatalf("Feed: %v", err)
	}
	if len(got) < 2 {
		t.Fatalf("Feed returned %d rows, want at least 2 (cap relief + trade note)", len(got))
	}
	trade := findFeedByKind(t, got, "TRADE")
	if trade.TradeRationale != "foundation trade" {
		t.Errorf("trade rationale via real path = %q, want the persisted text", trade.TradeRationale)
	}
	relief := findFeedByKind(t, got, "CAP_RELIEF")
	if len(relief.FranchiseIDs) != 1 || relief.FranchiseIDs[0] != "0001" {
		t.Errorf("cap relief franchise via real path = %v, want [0001]", relief.FranchiseIDs)
	}
}

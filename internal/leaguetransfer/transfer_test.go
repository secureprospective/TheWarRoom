package leaguetransfer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/governance"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/leagueweek"
	"github.com/secureprospective/TheWarRoom/internal/measures"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

func transferFixture(t *testing.T) (*history.Store, *db.Pools, RawExport) {
	t.Helper()
	pools, err := db.Open(t.Context(), filepath.Join(t.TempDir(), "source.db"), history.SchemaVersion())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = pools.Close() })
	registry, err := measures.Embedded()
	if err != nil {
		t.Fatal(err)
	}
	store := history.New(pools, registry)
	if err := store.Initialize(t.Context()); err != nil {
		t.Fatal(err)
	}
	at := time.Date(2026, 10, 24, 12, 0, 0, 0, time.UTC)
	store.SetClock(func() time.Time { return at })
	raw := commissioner.RawCommand{League: "native.test", Command: commissioner.Create, Args: commissioner.RawArgs{Name: "Moving league", Preset: commissioner.PresetLegacy, Season: 2026}, Request: "create", Reason: "Create the league", Effective: "now", Visibility: "league"}
	runTransferCommand(t, store, raw, "creator")
	seed, err := os.ReadFile("../commissioner/testdata/native-assets.wire.json")
	if err != nil {
		t.Fatal(err)
	}
	var assets commissioner.NativeAssets
	if err := json.Unmarshal(seed, &assets); err != nil {
		t.Fatal(err)
	}
	assets.Snapshot.Players = assets.Snapshot.Players[:2]
	for i := range assets.Snapshot.Players {
		p := &assets.Snapshot.Players[i]
		p.Franchise = []string{"franchise.01", "franchise.32"}[i]
		p.Status = "ROSTER"
		p.Salary = 100000
	}
	raw.Command = commissioner.NativeSeed
	raw.Args = commissioner.RawArgs{Native: assets.Snapshot}
	runTransferCommand(t, store, raw, "creator")
	for field, value := range map[string]string{"schedule.regularWeeks": "3", "lineup.invalidPenalty": "\"none\""} {
		raw.Command = commissioner.RulesSet
		raw.Args = commissioner.RawArgs{Field: field, Value: json.RawMessage(value)}
		runTransferCommand(t, store, raw, "creator")
	}
	raw.Command = commissioner.ScheduleGenerate
	raw.Args = commissioner.RawArgs{Schedule: &commissioner.RawScheduleArgs{Weeks: 3}}
	runTransferCommand(t, store, raw, "creator")
	raw.Command = commissioner.SchedulePublish
	raw.Args = commissioner.RawArgs{Schedule: &commissioner.RawScheduleArgs{}}
	runTransferCommand(t, store, raw, "creator")
	body := transferEvidence(t, at)
	football := digest(body)
	if err := store.ImportFootball(t.Context(), body, football); err != nil {
		t.Fatal(err)
	}
	pinned, err := store.LoadFootball(t.Context(), football)
	if err != nil {
		t.Fatal(err)
	}
	store.SetLineupEnvironment(pinned)
	addTransferLineups(t, store, raw)
	addTransferBid(t, store, raw, pinned.PlayerFacts())
	aliases := []governance.VerifiedAlias{{League: "native.test", Alias: "creator", Account: "acct.12345678-1234-4234-8234-123456789abc"}}
	out, err := Capture(t.Context(), store, "native.test", "aaaaaaaa-bbbb-4ccc-8ddd-eeeeeeeeeeee", football, at, aliases)
	if err != nil {
		t.Fatal(err)
	}
	return store, pools, out
}

func runTransferCommand(t *testing.T, store *history.Store, raw commissioner.RawCommand, principal string) {
	t.Helper()
	head, err := store.CommittedHead(t.Context(), raw.League, commissioner.Stream)
	if err != nil {
		t.Fatal(err)
	}
	raw.Expected = head
	raw.Request = fmt.Sprintf("change.%d", head)
	if _, err := store.ExecuteCommissioner(t.Context(), raw, principal, nil); err != nil {
		t.Fatalf("%s: %v", raw.Command, err)
	}
}

func addTransferLineups(t *testing.T, store *history.Store, raw commissioner.RawCommand) {
	t.Helper()
	for i, franchise := range []string{"franchise.01", "franchise.32"} {
		owner := fmt.Sprintf("owner.%d", i)
		invite := "invite." + owner
		raw.Command = "commish.seat.invite"
		raw.Visibility = "private"
		raw.Args = commissioner.RawArgs{People: &commissioner.RawPeople{ID: invite, Franchise: franchise, Principal: owner}}
		runTransferCommand(t, store, raw, "creator")
		raw.Command = "commish.seat.invite.redeem"
		raw.Args.People = &commissioner.RawPeople{ID: invite, Principal: owner}
		runTransferCommand(t, store, raw, owner)
		raw.Command = commissioner.LineupCorrect
		raw.Visibility = "league"
		raw.Args = commissioner.RawArgs{Lineup: &commissioner.RawLineupChange{Season: 2026, Week: 1, Franchise: franchise, Starters: []string{[]string{"0099", "0100"}[i]}}}
		runTransferCommand(t, store, raw, "creator")
	}
}

type transferBidSeason struct{}

func (transferBidSeason) BidEnvironment(context.Context, string, time.Time) (commissioner.BidEnvironment, error) {
	return commissioner.BidEnvironment{Season: nativecheck.BidSeason{Week: 5, WeekKnown: true}}, nil
}

func addTransferBid(t *testing.T, store *history.Store, raw commissioner.RawCommand, facts nativecheck.Facts) {
	t.Helper()
	for _, rule := range []struct{ field, value string }{
		{"roster.limits", `{"min":0,"max":100}`}, {"cap.amount", `100000000000`}, {"calendar.timezone", `"UTC"`}, {"calendar.boundary", `"start_of_day"`},
		{"calendar.windows", `{"native":{"start":{"month":1,"day":1},"opens":["cut","waiver_claim","ufa_bid"]}}`},
	} {
		raw.Command = commissioner.RulesSet
		raw.Args = commissioner.RawArgs{Field: rule.field, Value: json.RawMessage(rule.value)}
		runTransferCommand(t, store, raw, "creator")
	}
	head, err := store.CommittedHead(t.Context(), raw.League, commissioner.Stream)
	if err != nil {
		t.Fatal(err)
	}
	store.SetBidEnvironment(transferBidSeason{})
	raw.Expected = head
	raw.Request = "hidden.bid"
	raw.Command = commissioner.BidPlace
	raw.Visibility = "private"
	raw.Args = commissioner.RawArgs{Bid: &commissioner.RawBidArgs{Player: "5001", Franchise: "franchise.01", Salary: 500000000, Years: 1}}
	if _, err := store.ExecuteCommissionerWithFacts(t.Context(), raw, "owner.0", nil, facts, time.UTC); err != nil {
		t.Fatal(err)
	}
}

func transferEvidence(t *testing.T, at time.Time) []byte {
	t.Helper()
	weekly, err := os.ReadFile("../fantasyscore/source/testdata/weeks.csv")
	if err != nil {
		t.Fatal(err)
	}
	plays, err := os.ReadFile("../fantasyscore/source/testdata/plays.csv")
	if err != nil {
		t.Fatal(err)
	}
	raw := history.RawFootballEvidence{Version: 1, Season: 2026, Source: "commissioner reference import", AsOf: at,
		Players: map[string]history.RawFootballPlayer{
			"5001": {Identity: "00-0039999", Facts: nativecheck.PlayerFacts{Name: "Free agent", Position: leaguerules.QB, Team: "DAL", DesignationHeld: true, DraftYear: 2020, DraftYearHeld: true}},
			"0099": {Identity: "00-0023459", Facts: nativecheck.PlayerFacts{Name: "Quarterback", Position: leaguerules.QB, Team: "DAL", DesignationHeld: true}},
			"0100": {Identity: "00-0023853", Facts: nativecheck.PlayerFacts{Name: "Kicker", Position: leaguerules.PK, Team: "NYG", DesignationHeld: true}}},
		Weeks: []leagueweek.Week{{Number: 1, Games: []leagueweek.Game{{Teams: [2]string{"DAL", "NYG"}, Kickoff: at.Add(time.Hour)}}}},
		Stats: []history.RawFootballStats{{AsOf: at.Add(-time.Hour), Weekly: strings.ReplaceAll(string(weekly), ",2025,", ",2026,"), Plays: strings.ReplaceAll(string(plays), ",2025,", ",2026,")}}}
	body, err := json.Marshal(raw)
	if err != nil {
		t.Fatal(err)
	}
	return body
}

func TestEncryptedExportRoundtripFrozenReplayParity(t *testing.T) {
	store, _, raw := transferFixture(t)
	key := [32]byte{1, 2, 3}
	var encrypted bytes.Buffer
	if err := Encrypt(t.Context(), &encrypted, raw, key); err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(encrypted.Bytes(), []byte("Moving league")) {
		t.Fatal("plaintext escaped")
	}
	got, err := Decrypt(t.Context(), &encrypted, key)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(raw, got) {
		t.Fatal("durable content changed in transit")
	}
	records, err := history.TransferRecords(got.Events)
	if err != nil {
		t.Fatal(err)
	}
	evidence, err := history.TransferFootball(got.Blobs[got.Football])
	if err != nil {
		t.Fatal(err)
	}
	isolated, err := history.TransferReadings(t.Context(), got.League, records, evidence, got.FrozenAt)
	if err != nil {
		t.Fatal(err)
	}
	assertDirectTransferParity(t, store, isolated)
	if len(got.Aliases) != 1 || evidence.Identities()["0099"] != "00-0023459" {
		t.Fatal("lost linked or player identities")
	}
}

func assertDirectTransferParity(t *testing.T, store *history.Store, got history.TransferProjection) {
	t.Helper()
	assets, err := store.NativeAssets(t.Context(), "native.test")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(assets, got.Assets) {
		t.Fatal("roster differs")
	}
	records, err := store.Events(t.Context(), "native.test", commissioner.Stream, 0)
	if err != nil {
		t.Fatal(err)
	}
	ledger := make([]commissioner.Record, 0, len(records))
	for _, e := range records {
		ledger = append(ledger, *e.Payload.Commissioner)
	}
	capReading, err := commissioner.NativeCapCheckReading(commissioner.RawNativeCapScope{League: "native.test", ViewerZone: "UTC"}, ledger, time.Date(2026, 10, 24, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(capReading, got.Cap) {
		t.Fatal("cap differs")
	}
	for _, lineup := range got.Lineups {
		direct, err := store.NativeLineup(t.Context(), "native.test", lineup.Franchise, "creator", 1, time.Date(2026, 10, 24, 12, 0, 0, 0, time.UTC))
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(direct, lineup) {
			t.Fatal("lineup differs")
		}
	}
	pinnedBody, err := history.TransferFootball(transferEvidence(t, time.Date(2026, 10, 24, 12, 0, 0, 0, time.UTC)))
	if err != nil {
		t.Fatal(err)
	}
	for i, bids := range got.Bids {
		direct, err := store.NativeBids(t.Context(), "native.test", bids.Franchise, fmt.Sprintf("owner.%d", i), time.Date(2026, 10, 24, 12, 0, 0, 0, time.UTC), time.UTC, pinnedBody.PlayerFacts())
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(direct, bids) {
			t.Fatal("bid differs")
		}
	}
	results, err := store.NativeResults(t.Context(), "native.test", 2026, 1)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(results, got.Results[0]) || results.Games[0].Status != commissioner.GameFinal {
		t.Fatalf("results differ or are incomplete: %+v", results.Games[0])
	}
	standings, err := store.NativeStandings(t.Context(), "native.test", 2026)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(standings, got.Standings) {
		t.Fatal("standings differ")
	}
	if len(got.Bids) != 2 || got.Bids[0].ActiveBids != 1 || len(got.Lineups) != 32 || !got.Lineups[0].Set {
		t.Fatal("lineups or owner bids missing")
	}
}

func TestExportRejectsMissingBlobBadDigestAndReplayMismatch(t *testing.T) {
	_, _, original := transferFixture(t)
	cases := map[string]func(*RawExport){
		"missing blob":    func(r *RawExport) { delete(r.Blobs, r.Football) },
		"bad blob digest": func(r *RawExport) { r.Blobs[r.Football][0] ^= 1 },
		"missing event":   func(r *RawExport) { r.Events = r.Events[1:] },
		"duplicate event": func(r *RawExport) { r.Events = append(r.Events, r.Events[0]) },
		"other league":    func(r *RawExport) { r.Events[0].League = "native.other" },
		"new format":      func(r *RawExport) { r.Version = 2 },
		"new command": func(r *RawExport) {
			r.Events[0].Payload = strings.Replace(r.Events[0].Payload, `"commandVersion":1`, `"commandVersion":99`, 1)
		},
		"changed receipt":    func(r *RawExport) { r.Events[0].Actor = "someone else" },
		"cross-league alias": func(r *RawExport) { r.Aliases[0].League = "native.other" },
		"roster parity":      func(r *RawExport) { r.Projection.Roster = "wrong" },
		"cap parity":         func(r *RawExport) { r.Projection.Cap = "wrong" },
		"lineup parity":      func(r *RawExport) { r.Projection.Lineup = "wrong" },
		"bid parity":         func(r *RawExport) { r.Projection.Bid = "wrong" },
		"result parity":      func(r *RawExport) { r.Projection.Result = "wrong" },
		"standings parity":   func(r *RawExport) { r.Projection.Standings = "wrong" },
	}
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatal(err)
	}
	for name, change := range cases {
		t.Run(name, func(t *testing.T) {
			var bad RawExport
			if err := json.Unmarshal(data, &bad); err != nil {
				t.Fatal(err)
			}
			change(&bad)
			// Even a correctly authenticated, internally rehashed document must obey the domain contract.
			bad.Content, err = bad.contentDigest()
			if err != nil {
				t.Fatal(err)
			}
			if err := bad.Validate(t.Context()); err == nil {
				t.Fatal("damaged league export accepted")
			}
		})
	}
}

func TestEncryptedExportRejectsWrongKeyTamperingTruncationAndVersion(t *testing.T) {
	_, _, raw := transferFixture(t)
	key := [32]byte{9}
	var encrypted bytes.Buffer
	if err := Encrypt(t.Context(), &encrypted, raw, key); err != nil {
		t.Fatal(err)
	}
	original := encrypted.Bytes()
	if _, err := Decrypt(t.Context(), bytes.NewReader(original), [32]byte{8}); err == nil {
		t.Fatal("wrong key accepted")
	}
	for _, position := range []int{0, len(envelope), len(original) - 1} {
		changed := bytes.Clone(original)
		changed[position] ^= 1
		if _, err := Decrypt(t.Context(), bytes.NewReader(changed), key); err == nil {
			t.Fatal("tampering accepted")
		}
	}
	for _, length := range []int{0, 1, len(envelope), len(original) - 1} {
		if _, err := Decrypt(t.Context(), bytes.NewReader(original[:length]), key); err == nil {
			t.Fatal("truncated file accepted")
		}
	}
	if _, err := Decrypt(t.Context(), bytes.NewReader(append(bytes.Clone(original), 0)), key); err == nil {
		t.Fatal("trailing data accepted")
	}
}

func TestExportCarriesNoSchemaOrWritableAuthority(t *testing.T) {
	store, pools, raw := transferFixture(t)
	at := raw.FrozenAt
	if err := store.SetAuthorityPolicy(t.Context(), history.AuthorityPolicy{League: raw.League, Domain: "roster", CutoverAt: &at}); err != nil {
		t.Fatal(err)
	}
	var schemaBefore int
	if err := pools.Read().QueryRowContext(t.Context(), "PRAGMA user_version").Scan(&schemaBefore); err != nil {
		t.Fatal(err)
	}
	if err := raw.Validate(t.Context()); err != nil {
		t.Fatal(err)
	}
	authority, err := store.Authority(t.Context(), raw.League, "roster", at)
	if err != nil || authority != history.AuthorityLocal {
		t.Fatal("export changed source authority", err)
	}
	var schemaAfter int
	if err := pools.Read().QueryRowContext(t.Context(), "PRAGMA user_version").Scan(&schemaAfter); err != nil || schemaAfter != schemaBefore {
		t.Fatal("export changed schema", err)
	}
	body, err := json.Marshal(raw)
	if err != nil {
		t.Fatal(err)
	}
	for _, marker := range []string{"user_version", "authority_policy", "schemaVersion", "password_hash", "account_sessions"} {
		if bytes.Contains(body, []byte(marker)) {
			t.Fatalf("copied %s", marker)
		}
	}
	body = append(bytes.TrimSuffix(body, []byte("}")), []byte(`,"schemaVersion":999,"active":true}`)...)
	if _, err := decode(t.Context(), body); err == nil {
		t.Fatal("copied schema/activation marker accepted")
	}
}

func TestCaptureRejectsMissingSelectedEvidenceAndDamagedAudit(t *testing.T) {
	store, pools, raw := transferFixture(t)
	if _, err := Capture(t.Context(), store, raw.League, raw.SourceEpoch, strings.Repeat("0", 64), raw.FrozenAt, nil); err == nil {
		t.Fatal("missing selected evidence accepted")
	}
	if _, err := pools.Write().ExecContext(t.Context(), "DROP TRIGGER league_event_no_update; UPDATE league_event SET actor='rewritten' WHERE revision=1"); err != nil {
		t.Fatal(err)
	}
	if _, err := Capture(t.Context(), store, raw.League, raw.SourceEpoch, raw.Football, raw.FrozenAt, nil); err == nil {
		t.Fatal("damaged source audit accepted")
	}
}

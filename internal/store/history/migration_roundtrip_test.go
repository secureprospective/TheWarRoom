package history

import (
	"context"
	"errors"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/leaguemigration"
	"github.com/secureprospective/TheWarRoom/internal/leaguemigration/mflsource"
	"github.com/secureprospective/TheWarRoom/internal/leaguemigration/migrationtest"
)

// migrationLedger is the store as the move uses it.
type migrationLedger struct{ s *Store }

func (l migrationLedger) Records(ctx context.Context, league string) ([]commissioner.Record, error) {
	return l.s.nativeRecords(ctx, league)
}

func (l migrationLedger) Batch(ctx context.Context, raws []commissioner.RawCommand, principal string) ([]commissioner.Receipt, error) {
	return l.s.ExecuteCommissionerBatch(ctx, raws, principal, nil)
}

func (l migrationLedger) Execute(ctx context.Context, raw commissioner.RawCommand, principal string) (commissioner.Receipt, error) {
	return l.s.ExecuteCommissioner(ctx, raw, principal, nil)
}

func migrationService(t *testing.T, export mflsource.Export) (*leaguemigration.Service, *Store, *migrationtest.Source) {
	t.Helper()
	s, c, _ := newStore(t)
	c.now = migrationtest.ReadAt.Add(time.Hour)
	src := &migrationtest.Source{Export: export}
	svc := &leaguemigration.Service{Source: func(string) (leaguemigration.Source, error) { return src, nil },
		Ledger: migrationLedger{s}, Now: func() time.Time { return c.now }}
	return svc, s, src
}

func migrationLeagues(t *testing.T) map[string]mflsource.Export {
	t.Helper()
	legacy, err := migrationtest.Legacy()
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]mflsource.Export{"Legacy NFL 14432": legacy}
	for name, spec := range migrationtest.Specs() {
		out[name] = migrationtest.Synthetic(name, spec)
	}
	return out
}

// TestMigrationRoundTrip imports each fixture league, reviews it, cuts it over, then reads the native
// league back and compares it to the MFL source field by field. Zero differences are allowed.
func TestMigrationRoundTrip(t *testing.T) {
	t.Parallel()
	for name, export := range migrationLeagues(t) {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			svc, s, _ := migrationService(t, export)
			ctx := t.Context()
			receipts, err := svc.Import(ctx, export.LeagueID, "Move the league over from MFL", "creator")
			if err != nil || len(receipts) != 2 {
				t.Fatalf("import: %v %+v", err, receipts)
			}
			req := leaguemigration.Request{MFLLeague: export.LeagueID, League: leaguemigration.AppLeagueID(export.LeagueID, export.Season)}
			records, err := s.nativeRecords(ctx, req.League)
			if err != nil || commissioner.LeagueSource(records) != commissioner.SourceMFLImport {
				t.Fatalf("before the cutover the league must follow MFL: %v", err)
			}
			review, err := svc.Review(ctx, req)
			if err != nil {
				t.Fatal(err)
			}
			if review.Differences != 0 {
				t.Fatalf("review found differences right after import: %+v %+v", review.LeagueWide, review.Franchises)
			}
			if _, err := svc.Cutover(ctx, req, review.Digest, "", "creator"); err == nil {
				t.Fatal("a cutover without a typed reason was recorded")
			}
			if _, err := svc.Cutover(ctx, req, "stale", "The league is ready", "creator"); err == nil {
				t.Fatal("a cutover from a review the commissioner did not read was recorded")
			}
			if _, err := svc.Cutover(ctx, req, review.Digest, "The league is ready", "creator"); err != nil {
				t.Fatal(err)
			}
			assertNativeEqualsMFL(t, s, svc, export, req)
		})
	}
}

func assertNativeEqualsMFL(t *testing.T, s *Store, svc *leaguemigration.Service, export mflsource.Export,
	req leaguemigration.Request) {
	t.Helper()
	ctx := t.Context()
	records, err := s.nativeRecords(ctx, req.League)
	if err != nil {
		t.Fatal(err)
	}
	if commissioner.LeagueSource(records) != commissioner.SourceNative || !commissioner.CutOver(records) {
		t.Fatal("after the cutover the league must be native")
	}
	export.ReadAt = svc.Now() // the instant the service read MFL at
	plan, err := mflsource.Prepare(export)
	if err != nil {
		t.Fatal(err)
	}
	app, err := leaguemigration.ReadApp(req.League, records, svc.Now())
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range leaguemigration.Compare(plan, app) {
		t.Errorf("lossy field %s %s: MFL %s, app %s", d.Franchise, d.Field, d.MFL, d.App)
	}
	got, want := app.Snapshot, plan.Seed
	// Reading fills these empty collections; nil and empty hold the same nothing.
	got.Sides = nil
	for _, s := range []*commissioner.RawNativeSnapshot{&got, &want} {
		if len(s.StandingsOrder) == 0 {
			s.StandingsOrder = nil
		}
		if len(s.Conferences) == 0 {
			s.Conferences = nil
		}
	}
	gv, wv := reflect.ValueOf(got), reflect.ValueOf(want)
	for i := range gv.NumField() {
		if !reflect.DeepEqual(gv.Field(i).Interface(), wv.Field(i).Interface()) {
			t.Errorf("lossy snapshot field %s:\napp %+v\nmfl %+v", gv.Type().Field(i).Name,
				gv.Field(i).Interface(), wv.Field(i).Interface())
		}
	}
	assertAppHoldsSource(t, export, app)
	if app.Name != export.Name || len(app.Directory.Franchises) != len(export.Config.Franchises) {
		t.Errorf("league name or franchises lost: %q %d", app.Name, len(app.Directory.Franchises))
	}
}

// TestMigrationLeavesNothingOnAFailedStep fails the import at every step: the read of each MFL part, and
// after each recorded change before the commit. Nothing is left behind, and the import simply runs again.
func TestMigrationLeavesNothingOnAFailedStep(t *testing.T) {
	t.Parallel()
	export := migrationtest.Synthetic("Fault league", migrationtest.Specs()["8 teams"])
	league := leaguemigration.AppLeagueID(export.LeagueID, export.Season)
	steps := []string{"read:league", "read:rosters", "read:adjustments", "read:draft", "read:players",
		"record:0", "record:1"}
	for _, step := range steps {
		t.Run(step, func(t *testing.T) {
			t.Parallel()
			svc, s, src := migrationService(t, export)
			kind, at, _ := strings.Cut(step, ":")
			injected := errors.New("the computer stopped")
			if kind == "read" {
				src.FailAt = at
			} else {
				s.batchFault = func(i int) error {
					if at == strconv.Itoa(i) {
						return injected
					}
					return nil
				}
			}
			if _, err := svc.Import(t.Context(), export.LeagueID, "Move over", "creator"); err == nil {
				t.Fatal("the failed import reported success")
			}
			if records, err := s.nativeRecords(t.Context(), league); err != nil || len(records) != 0 {
				t.Fatalf("a failed import left %d entries behind: %v", len(records), err)
			}
			if n := count(t, s, "league_event"); n != 0 {
				t.Fatalf("a failed import left %d ledger rows", n)
			}
			src.FailAt, s.batchFault = "", nil
			if _, err := svc.Import(t.Context(), export.LeagueID, "Move over", "creator"); err != nil {
				t.Fatalf("starting the import again failed: %v", err)
			}
		})
	}
}

func TestMigrationReviewWordsADifferencePerFranchise(t *testing.T) {
	t.Parallel()
	export := migrationtest.Synthetic("Review league", migrationtest.Specs()["8 teams"])
	svc, _, src := migrationService(t, export)
	ctx := t.Context()
	if _, err := svc.Import(ctx, export.LeagueID, "Move over", "creator"); err != nil {
		t.Fatal(err)
	}
	// MFL moves on after the import: two contracts change and a pick is traded.
	changed := src.Export
	changed.Rosters = append([]mflsource.Rostered{}, changed.Rosters...)
	changed.Rosters[0].Salary, changed.Rosters[1].ContractYear = "9.99", "2031"
	changed.Future = append(changed.Future[:0:0], changed.Future...)
	changed.Future[1].Owner = "0005"
	src.Export = changed
	req := leaguemigration.Request{MFLLeague: export.LeagueID, League: leaguemigration.AppLeagueID(export.LeagueID, 2026)}
	review, err := svc.Review(ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	if review.Differences != 3 {
		t.Fatalf("differences=%d %+v", review.Differences, review.Franchises)
	}
	first := strings.Join(review.Franchises[0].Lines, " | ")
	if !strings.Contains(first, "2 contracts differ: Player 0100 salary") || !strings.Contains(first, "Roster matches") {
		t.Errorf("franchise 1 lines: %s", first)
	}
	// The traded pick is listed under the franchise MFL says now owns it.
	fifth := strings.Join(review.Franchises[4].Lines, " | ")
	if !strings.Contains(fifth, "1 pick differs: the 2027 round 1 pick of franchise.02 owner") {
		t.Errorf("franchise 5 lines: %s", fifth)
	}
	again, err := svc.Review(ctx, req)
	if err != nil || again.Digest != review.Digest {
		t.Fatalf("a review is not re-runnable to the same words: %v", err)
	}
}

func TestMigrationInvitesAfterCutover(t *testing.T) {
	t.Parallel()
	export := migrationtest.Synthetic("Invite league", migrationtest.Specs()["empty franchise no email"])
	svc, s, _ := migrationService(t, export)
	ctx := t.Context()
	req := leaguemigration.Request{MFLLeague: export.LeagueID, League: leaguemigration.AppLeagueID(export.LeagueID, 2026)}
	if _, err := svc.Import(ctx, export.LeagueID, "Move over", "creator"); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Invites(ctx, req); err == nil {
		t.Fatal("invitations were prepared before the cutover")
	}
	review, err := svc.Review(ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Cutover(ctx, req, review.Digest, "Ready", "creator"); err != nil {
		t.Fatal(err)
	}
	list, err := svc.Invites(ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	if len(list.List) != 8 || list.Ready != 7 || list.List[0].Ready || !strings.Contains(list.List[0].Words, "no email") {
		t.Fatalf("one invitation per franchise, the emailless owner listed: %+v", list)
	}
	if _, err := svc.SendInvites(ctx, req, "stale", "", "creator"); err == nil {
		t.Fatal("an unreviewed invitation list was sent")
	}
	receipts, err := svc.SendInvites(ctx, req, list.Digest, "", "creator")
	if err != nil || len(receipts) != 7 {
		t.Fatalf("send: %v %d", err, len(receipts))
	}
	after, err := svc.Invites(ctx, req)
	if err != nil || after.Ready != 0 || !strings.Contains(after.List[1].Words, "Invitation sent") {
		t.Fatalf("sent invitations must show as sent: %v %+v", err, after.List[1])
	}
	redeem := commissioner.RawPeople{ID: after.List[1].ID, Principal: "owner2@example.com"}
	records, _ := s.nativeRecords(ctx, req.League)
	if _, err := s.ExecuteCommissioner(ctx, commissioner.RawCommand{League: req.League,
		Command: "commish.seat.invite.redeem", Args: commissioner.RawArgs{People: &redeem},
		Expected: commissioner.ViewRevision(ctx, records, int64(len(records))), Request: "redeem", Reason: "Join",
		Effective: "now", Visibility: "private"}, "owner2@example.com", nil); err != nil {
		t.Fatalf("an owner could not accept: %v", err)
	}
	claimed, err := svc.Invites(ctx, req)
	if err != nil || !claimed.List[1].Claimed || !strings.Contains(claimed.List[1].Words, "Claimed by owner2@example.com") {
		t.Fatalf("a claimed seat must show: %v %+v", err, claimed.List[1])
	}
}

func TestCutoverIsOnceOnlyAndPermanent(t *testing.T) {
	t.Parallel()
	export := migrationtest.Synthetic("Once league", migrationtest.Specs()["8 teams"])
	svc, s, _ := migrationService(t, export)
	ctx := t.Context()
	league := leaguemigration.AppLeagueID(export.LeagueID, 2026)
	cutover := func(request string) (commissioner.Receipt, error) {
		records, _ := s.nativeRecords(ctx, league)
		return s.ExecuteCommissioner(ctx, commissioner.RawCommand{League: league, Command: commissioner.Cutover,
			Expected: int64(len(records)), Request: request, Reason: "Ready", Effective: "now", Visibility: "league"},
			"creator", nil)
	}
	if _, err := svc.Import(ctx, export.LeagueID, "Move over", "creator"); err != nil {
		t.Fatal(err)
	}
	first, err := cutover("first")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := cutover("second"); err == nil || !strings.Contains(err.Error(), "already been moved over") {
		t.Fatalf("a second cutover: %v", err)
	}
	records, _ := s.nativeRecords(ctx, league)
	undo := reverseCommand(first.Entry, int64(len(records)), "undo-cutover")
	if _, err := s.ExecuteCommissioner(ctx, undo, "creator", nil); err == nil {
		t.Fatal("the cutover was reversed")
	}
	if err := commissioner.CheckCutoverAppend(undo, records); err == nil || !strings.Contains(err.Error(), "permanent") {
		t.Fatalf("the cutover's own check allows a reversal: %v", err)
	}
	if _, err := s.ExecuteCommissioner(ctx, createWith(commissioner.PresetDefault32, 0), "creator", nil); err != nil {
		t.Fatal(err)
	}
	if _, err := s.ExecuteCommissioner(ctx, commissioner.RawCommand{League: "native.test", Command: commissioner.Cutover,
		Expected: 1, Request: "native", Reason: "Ready", Effective: "now", Visibility: "league"}, "creator", nil); err == nil ||
		!strings.Contains(err.Error(), "only a league imported from MFL") {
		t.Fatalf("an app league was cut over: %v", err)
	}
}

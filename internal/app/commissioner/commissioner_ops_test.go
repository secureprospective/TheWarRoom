package commissioner

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/commissioner/commissionertest"
	"github.com/secureprospective/TheWarRoom/internal/commissioner"
)

const opsLeague = "native.ops"

func opsClock(app *ledgerFixture, hour int) {
	app.history.SetClock(func() time.Time { return time.Date(2026, 10, 14, hour, 0, 0, 0, time.UTC) })
}

func opsCommand(command string, expected int64, request string,
	args commissioner.RawArgs) commissioner.RawCommand {
	return commissioner.RawCommand{League: opsLeague, Command: command, Args: args, Expected: expected,
		Request: request, Effective: "now", Visibility: "league"}
}

// opsApp starts a Legacy league as "creator" with every owner reading as "owner".
func opsApp(t *testing.T) *ledgerFixture {
	t.Helper()
	app := commissionerTestApp(t)
	opsClock(app, 9)
	create := opsCommand(commissioner.Create, 0, "create", commissioner.RawArgs{Name: "Ops league",
		Preset: commissioner.PresetLegacy, Source: commissioner.SourceMFLImport, Season: 2026})
	create.Reason = "Start our league"
	executeCommissionerTest(t, app, create)
	return app
}

func readAs(app *ledgerFixture, principal string) func() {
	was := app.session
	app.session = principal
	return func() { app.session = was }
}

func TestAPreviewPostsNothingAndOwnersCannotPost(t *testing.T) {
	t.Parallel()
	app := opsApp(t)
	raw := commissionertest.NoticeCommand("native.ops", commissioner.NoticePublish, 1, "p1",
		"Draft moved to Sunday", true)
	preview := mustPreview(t, app, raw)
	if len(preview.Blockers) != 0 || preview.Safeguard != "confirm" || len(preview.Fields) != 1 ||
		preview.Fields[0].After != "Draft moved to Sunday" || preview.Fields[0].Before != "Not posted" ||
		!strings.Contains(preview.Fields[0].AfterSource, "At the top") || preview.Target.Version != 1 {
		t.Fatalf("notice preview: %+v", preview)
	}
	reading, err := app.TargetCommissionerNotices(opsLeague)
	if err != nil || len(reading.Notices) != 0 || len(reading.Mail.Recorded) != 0 || reading.Revision != 1 {
		t.Fatalf("a dry run posted a notice owners can see: %+v %v", reading, err)
	}
	blank := commissionertest.NoticeCommand("native.ops", commissioner.NoticePublish, 1, "p2", "", false)
	if got := mustPreview(t, app,
		blank); len(got.Blockers) != 1 || got.Blockers[0] != "Write the notice before posting it" {
		t.Fatalf("an empty notice must be refused in Go's words: %+v", got)
	}
	seat := opsCommand("commish.seat.replace", 1, "seat", commissioner.RawArgs{People: &commissioner.RawPeople{
		Franchise: "franchise.01", Principal: "owner"}})
	seat.Reason = "Seat the owner"
	executeCommissionerTest(t, app, seat)
	raw = commissionertest.NoticeCommand("native.ops", commissioner.NoticePublish, 2, "p3",
		"Draft moved to Sunday", true)
	defer readAs(app, "owner")()
	preview = mustPreview(t, app, raw)
	if len(preview.Blockers) == 0 {
		t.Fatalf("an owner who holds a franchise may not post a notice: %+v", preview)
	}
	if _, err := app.TargetCommissionerExecute(previewCommandBody(t, raw)); err == nil {
		t.Fatal("an owner posted an official notice")
	}
	if reading, err = app.TargetCommissionerNotices(opsLeague); err != nil || len(reading.Notices) != 0 {
		t.Fatalf("a refused notice was recorded: %+v %v", reading, err)
	}
}

// The ledger is filtered after Go decides what a person may read, so no filter finds a private row.
func TestNoLedgerFilterShowsAnOwnerARoleGrant(t *testing.T) {
	t.Parallel()
	app, league := shellLedgerApp(t)
	restore := readAs(app, "owner")
	defer restore()
	for _, kind := range []string{"", "setup", "rules", "people", "notices", "corrections"} {
		for _, person := range []string{"", "creator", "secret-deputy", "Local operator"} {
			body, err := json.Marshal(commissioner.RawLedgerFilter{Kind: kind, Person: person})
			if err != nil {
				t.Fatal(err)
			}
			reading, err := app.TargetCommissionerLedgerFiltered(league, string(body))
			if err != nil {
				t.Fatal(err)
			}
			for _, entry := range reading.Entries {
				if strings.Contains(entry.What, "Co-commissioner") || strings.Contains(entry.What, "secret-deputy") {
					t.Errorf("kind %q person %q showed an owner %+v", kind, person, entry)
				}
			}
		}
	}
	restore()
	reading, err := app.TargetCommissionerLedgerFiltered(league, `{"kind":"people"}`)
	if err != nil || len(reading.Entries) != 1 || !strings.Contains(reading.Entries[0].What, "Co-commissioner") {
		t.Fatalf("the commissioner must still find the grant: %+v %v", reading, err)
	}
}

func TestPostedNoticesAreFindableInTheLedgerByKind(t *testing.T) {
	t.Parallel()
	app := opsApp(t)
	opsClock(app, 10)
	executeCommissionerTest(t, app, commissionertest.NoticeCommand("native.ops", commissioner.NoticePublish,
		1, "n1", "Waivers run Tuesday", true))
	reading, err := app.TargetCommissionerLedgerFiltered(opsLeague, `{"kind":"notices"}`)
	if err != nil || len(reading.Entries) != 1 ||
		reading.Entries[0].What != "Posted an official notice, pinned: “Waivers run Tuesday”" {
		t.Fatalf("a posted notice belongs to its own kind of change: %+v %v", reading, err)
	}
	other, err := app.TargetCommissionerLedgerFiltered(opsLeague, `{"kind":"people"}`)
	if err != nil || len(other.Entries) != 0 {
		t.Fatalf("a notice is not a people change: %+v %v", other, err)
	}
}

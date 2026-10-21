package main

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/governance"
)

const opsLeague = "native.ops"

func opsClock(app *App, hour int) {
	app.core.History.SetClock(func() time.Time { return time.Date(2026, 10, 14, hour, 0, 0, 0, time.UTC) })
}

func opsCommand(command string, expected int64, request string, args commissioner.RawArgs) commissioner.RawCommand {
	return commissioner.RawCommand{League: opsLeague, Command: command, Args: args, Expected: expected,
		Request: request, Effective: "now", Visibility: "league"}
}

func noticeCommand(command string, expected int64, request, text string, pin bool) commissioner.RawCommand {
	return opsCommand(command, expected, request, commissioner.RawArgs{Note: &commissioner.RawNote{Text: text, Pin: pin}})
}

func ruleCommand(expected int64, request, field, value, effective, reason string) commissioner.RawCommand {
	raw := opsCommand(commissioner.RulesSet, expected, request, commissioner.RawArgs{Field: field,
		Value: json.RawMessage(value)})
	raw.Effective, raw.Reason = effective, reason
	return raw
}

// opsApp starts a Legacy league as "creator" with every owner reading as "owner".
func opsApp(t *testing.T) *App {
	t.Helper()
	app := commissionerTestApp(t)
	opsClock(app, 9)
	create := opsCommand(commissioner.Create, 0, "create", commissioner.RawArgs{Name: "Ops league",
		Preset: commissioner.PresetLegacy, Season: 2026})
	create.Reason = "Start our league"
	executeCommissionerTest(t, app, create)
	return app
}

func readAs(app *App, principal string) func() {
	was := app.core.CommissionerSession
	app.core.CommissionerSession = testCommissionerSession{principal}
	return func() { app.core.CommissionerSession = was }
}

func TestCommissionerNoticesWire(t *testing.T) {
	t.Parallel()
	app := opsApp(t)
	opsClock(app, 10)
	plain := executeCommissionerTest(t, app, noticeCommand(commissioner.NoticePublish, 1, "n1", "Waivers run Tuesday", false))
	opsClock(app, 11)
	pinned := executeCommissionerTest(t, app, noticeCommand(commissioner.NoticePublish, 2, "n2", "Rookie draft is Saturday", true))
	opsClock(app, 12)
	executeCommissionerTest(t, app, noticeCommand(commissioner.NotePost, 3, "n3", "House rule: no trades in week 18", true))
	views := map[string]commissioner.LeagueNotes{}
	read := func(name, principal string) commissioner.LeagueNotes {
		defer readAs(app, principal)()
		reading, err := app.TargetCommissionerNotices(opsLeague)
		if err != nil {
			t.Fatal(err)
		}
		views[name] = reading
		return reading
	}
	commish, owner := read("commissioner", "creator"), read("owner", "owner")
	if len(commish.Notices) != 2 || commish.Notices[0].ID != pinned.Entry || !commish.Notices[0].Pinned ||
		commish.Notices[1].ID != plain.Entry || commish.Notices[1].Pinned {
		t.Fatalf("a pinned notice must lead: %+v", commish.Notices)
	}
	if len(commish.Notes) != 1 || commish.Notes[0].Kind != "note" || len(commish.Mail.Recorded) != 2 {
		t.Fatalf("league notes are not official notices and the mail log lists only notices: %+v", commish)
	}
	if owner.Revision != 4 || len(owner.Notices) != 2 || owner.Notices[0].Text != "Rookie draft is Saturday" {
		t.Fatalf("every owner reads the same notices: %+v", owner)
	}
	opsClock(app, 13)
	withdraw := opsCommand(commissioner.Revert, 4, "w1", commissioner.RawArgs{Entry: pinned.Entry})
	withdraw.Reason = "Posted by mistake"
	executeCommissionerTest(t, app, withdraw)
	after := read("withdrawn", "owner")
	if len(after.Notices) != 1 || after.Notices[0].ID != plain.Entry || len(after.Mail.Recorded) != 2 ||
		!after.Mail.Recorded[0].Withdrawn || after.Mail.Recorded[1].Withdrawn {
		t.Fatalf("a withdrawn notice must leave the owners' view and stay in the mail log: %+v", after)
	}
	body, err := json.MarshalIndent(views, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	checkWire(t, "frontend/src/app/commissioner/notices.wire.testdata.json", body, "notices")
	if _, err := app.TargetCommissionerNotices("mfl.123"); err == nil {
		t.Fatal("an imported league has no app notices")
	}
}

func TestAPreviewPostsNothingAndOwnersCannotPost(t *testing.T) {
	t.Parallel()
	app := opsApp(t)
	raw := noticeCommand(commissioner.NoticePublish, 1, "p1", "Draft moved to Sunday", true)
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
	blank := noticeCommand(commissioner.NoticePublish, 1, "p2", "", false)
	if got := mustPreview(t, app, blank); len(got.Blockers) != 1 || got.Blockers[0] != "Write the notice before posting it" {
		t.Fatalf("an empty notice must be refused in Go's words: %+v", got)
	}
	seat := opsCommand("commish.seat.replace", 1, "seat", commissioner.RawArgs{People: &commissioner.RawPeople{
		Franchise: "franchise.01", Principal: "owner"}})
	seat.Reason = "Seat the owner"
	executeCommissionerTest(t, app, seat)
	raw = noticeCommand(commissioner.NoticePublish, 2, "p3", "Draft moved to Sunday", true)
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

func TestCommissionerDigestsWire(t *testing.T) {
	t.Parallel()
	app := opsApp(t)
	opsClock(app, 10)
	executeCommissionerTest(t, app, ruleCommand(1, "r1", "roster.limits", `{"min":20,"max":70}`, "next_rollover", ""))
	opsClock(app, 11)
	executeCommissionerTest(t, app, ruleCommand(2, "r2", "bids.ufa", `"blind"`, "now", "Agreed at the owners meeting"))
	opsClock(app, 12)
	executeCommissionerTest(t, app, ruleCommand(3, "r3", "notify.reminders", `false`, "now", ""))
	executeCommissionerTest(t, app, noticeCommand(commissioner.NoticePublish, 4, "n1", "Welcome", false))
	opsClock(app, 13)
	grant := opsCommand("commish.role.grant", 5, "g1", commissioner.RawArgs{People: &commissioner.RawPeople{
		Principal: "secret-deputy", Role: governance.CoCommissioner}})
	grant.Reason, grant.Visibility = "Needs a second pair of hands", "private"
	executeCommissionerTest(t, app, grant)
	views := map[string]commissioner.Digests{}
	for _, who := range []string{"creator", "owner"} {
		restore := readAs(app, who)
		reading, err := app.TargetCommissionerDigests(opsLeague)
		restore()
		if err != nil {
			t.Fatal(err)
		}
		views[who] = reading
	}
	got := func(d commissioner.Digests) []string {
		var out []string
		for _, e := range d.Digests {
			out = append(out, e.Sentence)
		}
		return out
	}
	owner := strings.Join(got(views["owner"]), "\n")
	for _, want := range []string{"Active roster size: 18 to 80 → 20 to 70, from the next rollover",
		"Unrestricted free agent bidding: Open → Blind", "The reason given: Agreed at the owners meeting.", "Created league “Ops league”"} {
		if !strings.Contains(owner, want) && !strings.Contains(strings.Join(got(views["creator"]), "\n"), want) {
			t.Errorf("no digest says %q:\n%s", want, owner)
		}
	}
	if len(views["owner"].Digests) != 3 || len(views["creator"].Digests) != 4 {
		t.Fatalf("one digest for each hold or typed-reason change, none for a confirm: %v / %v",
			got(views["owner"]), got(views["creator"]))
	}
	for _, sentence := range got(views["owner"]) {
		if strings.Contains(sentence, "Co-commissioner") || strings.Contains(sentence, "secret-deputy") ||
			strings.Contains(sentence, "Welcome") || strings.Contains(sentence, "deadlines") {
			t.Errorf("an owner's digest says too much: %s", sentence)
		}
	}
	body, err := json.MarshalIndent(views, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	checkWire(t, "frontend/src/app/commissioner/digests.wire.testdata.json", body, "digests")
}

func TestNotificationSettingsAreRulesThatNeedOnlyAConfirm(t *testing.T) {
	t.Parallel()
	app := opsApp(t)
	rules, err := app.TargetCommissionerRules(opsLeague)
	if err != nil {
		t.Fatal(err)
	}
	var found []string
	for _, section := range rules.Sections {
		for _, row := range section.Rows {
			if strings.HasPrefix(row.Field, "notify.") {
				found = append(found, row.Field+"="+row.Value)
				if row.Competitive || row.Safeguards.Now != "confirm" || row.Safeguards.NextRollover != "confirm" {
					t.Errorf("%s should need one confirm: %+v", row.Field, row.Safeguards)
				}
			}
		}
	}
	if strings.Join(found, ",") != "notify.transactions=On,notify.reminders=On" {
		t.Fatalf("the league starts with both on: %v", found)
	}
}

func TestACommissionerSetRuleCarriesWhoWhenAndWhichChange(t *testing.T) {
	t.Parallel()
	app := opsApp(t)
	opsClock(app, 15)
	receipt := executeCommissionerTest(t, app,
		ruleCommand(1, "r1", "roster.limits", `{"min":20,"max":70}`, "now", "Agreed"))
	rules, err := app.TargetCommissionerRules(opsLeague)
	if err != nil {
		t.Fatal(err)
	}
	var governed, preset int
	for _, section := range rules.Sections {
		for _, row := range section.Rows {
			if row.Governed == nil {
				preset++
				continue
			}
			governed++
			if row.Field != "roster.limits" || row.Governed.Who != "creator" || row.Governed.Entry != receipt.Entry ||
				!row.Governed.When.Equal(time.Date(2026, 10, 14, 15, 0, 0, 0, time.UTC)) ||
				row.Governed.Change != "Active roster size: 18 to 80 → 20 to 70, in force now" {
				t.Errorf("%s: %+v", row.Field, row.Governed)
			}
		}
	}
	if governed != 1 || preset < 50 {
		t.Fatalf("only the value the commissioner changed is governed: %d governed, %d preset", governed, preset)
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
	executeCommissionerTest(t, app, noticeCommand(commissioner.NoticePublish, 1, "n1", "Waivers run Tuesday", true))
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

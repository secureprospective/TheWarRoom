package commissioner

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func noteRaw(command, text string, pin bool) RawCommand {
	return RawCommand{League: "native.test", Command: command, Args: RawArgs{Note: &RawNote{Text: text, Pin: pin}},
		Expected: 1, Request: "note", Effective: "now", Visibility: "league"}
}

func TestANoticeIsOnlyEverPlainTextShownToEveryOwnerNow(t *testing.T) {
	for _, command := range []string{NoticePublish, NotePost} {
		if err := noteRaw(command, "Rookie draft is Saturday", true).Validate(); err != nil {
			t.Fatalf("%s: %v", command, err)
		}
		if d, err := noteRaw(command, "x", false).Definition(); err != nil || d.Gravity != "G1" {
			t.Fatalf("%s should need one confirm: %+v %v", command, d, err)
		}
	}
	long := strings.Repeat("a", noteLimit+1)
	for name, raw := range map[string]RawCommand{
		"empty":      noteRaw(NoticePublish, "", false),
		"blank":      noteRaw(NoticePublish, "  \n ", false),
		"spaces":     noteRaw(NoticePublish, " hello ", false),
		"too long":   noteRaw(NoticePublish, long, false),
		"control":    noteRaw(NoticePublish, "bell\a", false),
		"private":    func() RawCommand { r := noteRaw(NoticePublish, "hi", false); r.Visibility = "private"; return r }(),
		"later":      func() RawCommand { r := noteRaw(NoticePublish, "hi", false); r.Effective = "next_rollover"; return r }(),
		"no note":    {League: "native.test", Command: NoticePublish, Expected: 1, Request: "n", Effective: "now", Visibility: "league"},
		"extra name": func() RawCommand { r := noteRaw(NotePost, "hi", false); r.Args.Name = "Mine"; return r }(),
		"on a rule": func() RawCommand {
			r := ruleRaw("house.texts", `{"a":"b"}`, "now", "")
			r.Args.Note = &RawNote{Text: "hi"}
			return r
		}(),
	} {
		if err := raw.Validate(); err == nil {
			t.Errorf("%s: a notice that cannot be shown as written was accepted", name)
		}
	}
	if err := noteRaw(NoticePublish, strings.Repeat("é", noteLimit), false).Validate(); err != nil {
		t.Fatalf("the limit counts characters, not bytes: %v", err)
	}
}

func note(entry string, command string, text string, pin bool, minute int, visibility string) Record {
	raw := noteRaw(command, text, pin)
	raw.Visibility = visibility
	return Record{Command: raw, Principal: "local.abc", Role: "commissioner", Gravity: "G1",
		Receipt: Receipt{Entry: entry, League: raw.League, Revision: int64(minute),
			RecordedAt: time.Date(2026, 10, 14, 9, minute, 0, 0, time.UTC)}}
}

func withdrawal(entry, target string, minute int) Record {
	return Record{Command: RawCommand{League: "native.test", Command: Revert, Args: RawArgs{Entry: target},
		Visibility: "league", Effective: "now"}, Principal: "local.abc", Gravity: "G3",
		Receipt: Receipt{Entry: entry, League: "native.test", Revision: int64(minute),
			RecordedAt: time.Date(2026, 10, 14, 9, minute, 0, 0, time.UTC)}}
}

func TestOwnersReadOnlyNoticesThatAreRecordedAndPinnedOnesLead(t *testing.T) {
	records := []Record{
		{Command: RawCommand{Command: Create}, Receipt: Receipt{Entry: "act.0", Revision: 1}},
		note("act.a", NoticePublish, "First", false, 1, "league"),
		note("act.b", NoticePublish, "Pinned old", true, 2, "league"),
		note("act.c", NoticePublish, "Third", false, 3, "league"),
		note("act.d", NotePost, "Pinned note", true, 4, "league"),
		note("act.e", NoticePublish, "Pinned new", true, 5, "league"),
	}
	reading, err := NotesReading("native.test", records)
	if err != nil {
		t.Fatal(err)
	}
	var texts []string
	for _, n := range reading.Notices {
		texts = append(texts, n.Text)
	}
	if strings.Join(texts, "|") != "Pinned new|Pinned old|Third|First" {
		t.Fatalf("pinned notices lead, each group newest first: %v", texts)
	}
	if len(reading.Notes) != 1 || reading.Notes[0].Text != "Pinned note" || !reading.Notes[0].Pinned ||
		reading.Notes[0].Kind != "note" {
		t.Fatalf("a league note is not an official notice: %+v", reading.Notes)
	}
	// Reading the ledger as it stood before the post shows nothing of it.
	before, err := NotesReading("native.test", records[:3])
	if err != nil || len(before.Notices) != 2 || before.Notices[0].Text != "Pinned old" {
		t.Fatalf("a notice is visible before it is posted: %+v %v", before.Notices, err)
	}
	reading, err = NotesReading("native.test", append(records, withdrawal("act.f", "act.b", 6)))
	if err != nil || len(reading.Notices) != 3 || reading.Notices[0].Text != "Pinned new" {
		t.Fatalf("a withdrawn notice is still shown to owners: %+v %v", reading.Notices, err)
	}
	if len(reading.Mail.Recorded) != 4 || !reading.Mail.Recorded[2].Withdrawn || reading.Mail.Recorded[2].Text != "Pinned old" {
		t.Fatalf("the mail log must list every notice recorded, withdrawn ones marked: %+v", reading.Mail.Recorded)
	}
	if !strings.Contains(reading.Mail.Words, "sends no email") {
		t.Fatalf("the mail log must say the app sends no mail: %q", reading.Mail.Words)
	}
}

func TestAPrivateEntryNeverReachesTheNotices(t *testing.T) {
	records := []Record{{Command: RawCommand{Command: Create}, Receipt: Receipt{Entry: "act.0", Revision: 1}},
		note("act.a", NoticePublish, "Secret", true, 1, "private")}
	reading, err := NotesReading("native.test", records)
	if err != nil || len(reading.Notices) != 0 || len(reading.Mail.Recorded) != 0 {
		t.Fatalf("private entry leaked: %+v %v", reading, err)
	}
}

func TestReleaseNotesAppearWithoutAnyoneAskingForThem(t *testing.T) {
	reading, err := NotesReading("native.test", []Record{{Command: RawCommand{Command: Create},
		Receipt: Receipt{Entry: "act.0", Revision: 1}}})
	if err != nil || len(reading.Releases) == 0 || len(reading.Notices) != 0 || len(reading.Notes) != 0 {
		t.Fatalf("release notes need no post: %+v %v", reading, err)
	}
	for i, release := range reading.Releases {
		if release.Date == "" || release.Title == "" || release.Words == "" ||
			(i > 0 && release.Date > reading.Releases[i-1].Date) {
			t.Fatalf("release %d is incomplete or out of order: %+v", i, release)
		}
	}
}

// The app cannot send a message because the code that runs a notice has no way to reach one.
func TestTheCommissionerPackageHasNoWayToSendAMessage(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil || len(files) == 0 {
		t.Fatal("no source files", err)
	}
	for _, file := range files {
		if strings.HasSuffix(file, "_test.go") {
			continue
		}
		parsed, err := parser.ParseFile(token.NewFileSet(), file, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatal(err)
		}
		for _, spec := range parsed.Imports {
			path, _ := strconv.Unquote(spec.Path.Value)
			if path == "net" || strings.HasPrefix(path, "net/") || path == "os/exec" || path == "syscall" ||
				strings.Contains(path, "smtp") || strings.Contains(path, "mail") {
				t.Errorf("%s imports %s, which could carry a message out of the app", file, path)
			}
		}
		source, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		for _, word := range []string{"mailto:", "sendmail", "smtp."} {
			if strings.Contains(string(source), word) {
				t.Errorf("%s mentions %s", file, word)
			}
		}
	}
}

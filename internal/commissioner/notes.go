package commissioner

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	NoticePublish = "commish.notice.publish"
	NotePost      = "commish.whatsnew.post"
	// noteLimit keeps a notice to what a person reads in one go.
	noteLimit = 1000
)

// RawNote is the text of an official notice or a league note, and whether it stays pinned at the top.
type RawNote struct {
	Text string `json:"text"`
	Pin  bool   `json:"pin,omitempty"`
}

type noteDefinition struct{ command, control, gravity, label, title, kind string }

func noteDefinitions() []noteDefinition {
	return []noteDefinition{
		{NoticePublish, "CM-052", "G1", "Posted an official notice", "Official notice", "notice"},
		{NotePost, "CM-053", "G1", "Posted a league note", "League note", "note"},
	}
}

func noteDefinitionFor(command string) (noteDefinition, bool) {
	for _, d := range noteDefinitions() {
		if d.command == command {
			return d, true
		}
	}
	return noteDefinition{}, false
}

// validateNote refuses anything but the note itself: it is shown to owners as written.
func (a RawArgs) validateNote() error {
	if a.Note == nil || a.Name != "" || a.Entry != "" || a.People != nil {
		return fmt.Errorf("send the text of the notice")
	}
	text := a.Note.Text
	switch {
	case strings.TrimSpace(text) == "":
		return Refused(fmt.Errorf("write the notice before posting it"))
	case text != strings.TrimSpace(text):
		return fmt.Errorf("remove surrounding spaces from the notice")
	case utf8.RuneCountInString(text) > noteLimit:
		return Refused(fmt.Errorf("keep the notice to %d characters", noteLimit))
	case strings.IndexFunc(text, func(r rune) bool { return unicode.IsControl(r) && r != '\n' }) >= 0:
		return fmt.Errorf("the notice can hold text only")
	}
	return nil
}

// clip shortens text for one line of the ledger without cutting a word's letters in half.
func clip(text string, limit int) string {
	runes := []rune(strings.Join(strings.Fields(text), " "))
	if len(runes) <= limit {
		return string(runes)
	}
	return strings.TrimSpace(string(runes[:limit])) + "…"
}

// noteLabel says, for the ledger, what a posted notice or league note was.
func noteLabel(r Record) string {
	d, _ := noteDefinitionFor(r.Command.Command)
	pinned := ""
	if r.Command.Args.Note.Pin {
		pinned = ", pinned"
	}
	return fmt.Sprintf("%s%s: “%s”", d.label, pinned, clip(r.Command.Args.Note.Text, 140))
}

// revertNoteLabel describes a reversal whose original change was a notice or league note; ok is false
// otherwise.
func revertNoteLabel(records []Record, i int) (string, bool) {
	target := records[i]
	for target.Command.Command == Revert {
		found := false
		for _, r := range records[:i] {
			if r.Receipt.Entry == target.Command.Args.Entry {
				target, found = r, true
				break
			}
		}
		if !found {
			return "", false
		}
	}
	d, ok := noteDefinitionFor(target.Command.Command)
	if !ok {
		return "", false
	}
	verb, noun := "Withdrew", strings.ToLower(strings.TrimPrefix(d.label, "Posted "))
	if Replay(records[:i+1]).Active[target.Receipt.Entry] {
		verb = "Restored"
	}
	return fmt.Sprintf("%s %s: “%s”", verb, noun, clip(target.Command.Args.Note.Text, 140)), true
}

package commissioner

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// Note is a posted notice or league note as owners read it.
type Note struct {
	ID        string    `json:"id"`
	Kind      string    `json:"kind"`
	Text      string    `json:"text"`
	Pinned    bool      `json:"pinned"`
	Who       string    `json:"who"`
	When      time.Time `json:"when"`
	Withdrawn bool      `json:"withdrawn"`
	// Change is the ledger's own words for the entry that posted it.
	Change string `json:"change"`
}

// Release is one line of what changed in the app. The app writes these; nobody posts them.
type Release struct {
	Date  string `json:"date"`
	Title string `json:"title"`
	Words string `json:"words"`
}

// MailLog says what the app sent, in plain words, and lists the notices it recorded instead.
type MailLog struct {
	Words    string `json:"words"`
	Recorded []Note `json:"recorded"`
}

// LeagueNotes is what owners read: pinned and newest official notices, league notes, the app's release
// notes, and the mail log. Withdrawn items appear only in the mail log, marked as withdrawn.
type LeagueNotes struct {
	League   string    `json:"league"`
	Revision int64     `json:"revision"`
	Notices  []Note    `json:"notices"`
	Notes    []Note    `json:"notes"`
	Releases []Release `json:"releases"`
	Mail     MailLog   `json:"mail"`
}

const noMailWords = "This app sends no email and no messages to anyone. A notice is recorded in the league " +
	"ledger and shown to owners inside the app. These are the notices it recorded."

// releaseNotes are what changed in the app, newest first.
func releaseNotes() []Release {
	return []Release{
		{"2026-10-14", "Commissioner tools", "Start a league of your own in the app, set its rules, give and take " +
			"roles, and see every change in the league ledger."},
		{"2026-10-13", "Trade desk and reports", "The trade desk keeps exit plans, countdowns tick live, reports open in a " +
			"drawer and news times say when they are estimates."},
		{"2026-10-12", "Franchise HQ and the home page", "Try a what-if on your roster, use the row menus, and read " +
			"league dates straight from the rulebook."},
		{"2026-10-11", "Contracts, the Lab and the bid pool", "Contract moves, the Lab screens and the free-agent " +
			"bid pool arrive, with money kept to the exact cent."},
		{"2026-10-10", "The first target screens", "The trade desk, the home seasonal card and alerts, League " +
			"Pulse Now, and IR and taxi moves arrive."},
	}
}

// NotesReading reads the posted notices from the ledger alone. A notice is shown only once its entry is
// recorded, and stops being shown when a later entry withdraws it.
func NotesReading(league string, records []Record) (LeagueNotes, error) {
	if len(records) == 0 {
		return LeagueNotes{}, fmt.Errorf("this app league has not been created")
	}
	active := Replay(records).Active
	out := LeagueNotes{League: league, Revision: int64(len(records)), Notices: []Note{}, Notes: []Note{},
		Releases: releaseNotes(), Mail: MailLog{Words: noMailWords, Recorded: []Note{}}}
	for i := len(records) - 1; i >= 0; i-- {
		r := records[i]
		d, ok := noteDefinitionFor(r.Command.Command)
		if !ok || r.Command.Visibility != "league" || r.Command.Args.Note == nil {
			continue
		}
		note := Note{ID: r.Receipt.Entry, Kind: d.kind, Text: r.Command.Args.Note.Text,
			Pinned: r.Command.Args.Note.Pin, Who: personWords(r.Principal), When: r.Receipt.RecordedAt,
			Withdrawn: !active[r.Receipt.Entry], Change: noteLabel(r)}
		if d.kind == "notice" {
			out.Mail.Recorded = append(out.Mail.Recorded, note)
		}
		if note.Withdrawn {
			continue
		}
		if d.kind == "notice" {
			out.Notices = append(out.Notices, note)
		} else {
			out.Notes = append(out.Notes, note)
		}
	}
	// Records run oldest first and were walked newest first; pinned items lead, each group newest first.
	sort.SliceStable(out.Notices, func(a, b int) bool { return out.Notices[a].Pinned && !out.Notices[b].Pinned })
	sort.SliceStable(out.Notes, func(a, b int) bool { return out.Notes[a].Pinned && !out.Notes[b].Pinned })
	return out, nil
}

// Digest is one plain sentence about a change that asked for a hold or a typed reason.
type Digest struct {
	ID       string    `json:"id"`
	Sentence string    `json:"sentence"`
	When     time.Time `json:"when"`
}

// Digests is the league's newest digests first. Go writes each sentence from the ledger.
type Digests struct {
	League  string   `json:"league"`
	Digests []Digest `json:"digests"`
}

// digestLimit keeps the digest to the latest changes; the ledger holds all of them.
const digestLimit = 50

// DigestsReading writes a sentence for every change a person had to hold or give a reason for. It reads
// the ledger as this principal may, so a private change never reaches a digest an owner can see.
func DigestsReading(league, principal string, records []Record, auth Authorizer) (Digests, error) {
	ledger, err := Reading(league, principal, records, auth)
	if err != nil {
		return Digests{}, err
	}
	gravity := make(map[string]string, len(records))
	for _, r := range records {
		gravity[r.Receipt.Entry] = r.Gravity
	}
	out := Digests{League: league, Digests: []Digest{}}
	for _, e := range ledger.Entries {
		if g := gravity[e.ID]; g != "G2" && g != "G3" {
			continue
		}
		sentence := fmt.Sprintf("%s recorded this change: %s.", e.Who, strings.TrimRight(e.What, "."))
		if e.Why != "" {
			sentence += fmt.Sprintf(" The reason given: %s", strings.TrimRight(e.Why, ".")+".")
		}
		out.Digests = append(out.Digests, Digest{ID: e.ID, Sentence: sentence, When: e.When})
		if len(out.Digests) == digestLimit {
			break
		}
	}
	return out, nil
}

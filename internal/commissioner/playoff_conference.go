package commissioner

import (
	"fmt"
	"slices"

	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

// singleConference names the one conference of a league that has only one.
const singleConference = "League"

// conferenceLayout says which conference every franchise plays in. Words is empty when the layout is
// known, and otherwise says what the commissioner still has to settle.
type conferenceLayout struct {
	Of          map[string]string
	Conferences []string
	Words       string
}

// conferencesFor reads the league's conferences from its division alignment and the conference of each
// division recorded in the starting snapshot. A league with one conference needs no record of it.
func conferencesFor(records []Record, rules leaguerules.RuleSet, franchises []string) conferenceLayout {
	alignment := rules.League.Alignment
	if alignment.State != leaguerules.IsKnown {
		return conferenceLayout{Words: "Needs a ruling: which division each team is in"}
	}
	held := heldConferences(records)
	out := conferenceLayout{Of: map[string]string{}}
	for _, id := range franchises {
		division := alignment.Value[id]
		if division == "" {
			return conferenceLayout{Words: fmt.Sprintf("Needs a ruling: which division the team %q is in", id)}
		}
		conference := held[division]
		if rules.League.Conferences.State == leaguerules.IsKnown && rules.League.Conferences.Value == 1 {
			conference = singleConference
		}
		if conference == "" {
			return conferenceLayout{Words: fmt.Sprintf("Needs a ruling: which conference the division %q is in", division)}
		}
		out.Of[id] = conference
		if !slices.Contains(out.Conferences, conference) {
			out.Conferences = append(out.Conferences, conference)
		}
	}
	slices.Sort(out.Conferences)
	known := rules.League.Conferences
	if known.State == leaguerules.IsKnown && known.Value != len(out.Conferences) {
		return conferenceLayout{Words: fmt.Sprintf("The league is set up with %d conferences but its divisions sit in %d",
			known.Value, len(out.Conferences))}
	}
	return out
}

// heldConferences is the division-to-conference record the league was started with.
func heldConferences(records []Record) map[string]string {
	for _, r := range records {
		if r.Command.Command == NativeSeed && r.Command.Args.Native != nil {
			return r.Command.Args.Native.Conferences
		}
	}
	return nil
}

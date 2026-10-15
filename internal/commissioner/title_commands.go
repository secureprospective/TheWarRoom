package commissioner

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"unicode"
)

const (
	TitleSet = "title.set"

	// yearToken in a title label stands for the season's own year.
	yearToken      = "{year}"
	maxTitleLength = 80
)

// RawTitleChange is the title.set argument (Args.Title). An empty label puts back the default.
type RawTitleChange struct {
	Season int    `json:"season"`
	Label  string `json:"label,omitempty"`
}

// DecodeTitleChange reads a title command's argument. A label that names a year other than the season's
// own is refused, so a title can never carry an old year.
func DecodeTitleChange(a RawArgs) (RawTitleChange, error) {
	other := a
	other.Title = nil
	if a.Title == nil || !reflect.DeepEqual(other, RawArgs{}) {
		return RawTitleChange{}, fmt.Errorf("send only the details of this title change")
	}
	raw := *a.Title
	if raw.Season < 1920 || raw.Season > 9999 {
		return raw, fmt.Errorf("choose the season this title is for")
	}
	if strings.TrimSpace(raw.Label) != raw.Label || len(raw.Label) > maxTitleLength {
		return raw, fmt.Errorf("write the title in at most %d characters, without spaces at either end", maxTitleLength)
	}
	for _, year := range yearsIn(raw.Label) {
		if year != raw.Season {
			return raw, fmt.Errorf("the label names %d but this title is for %d; write %s to stamp the season automatically",
				year, raw.Season, yearToken)
		}
	}
	return raw, nil
}

// yearsIn lists the four-digit years written in a label.
func yearsIn(label string) []int {
	var years []int
	run := ""
	flush := func() {
		if year, err := strconv.Atoi(run); err == nil && len(run) == 4 {
			years = append(years, year)
		}
		run = ""
	}
	for _, r := range label {
		if unicode.IsDigit(r) {
			run += string(r)
			continue
		}
		flush()
	}
	flush()
	return years
}

// DefaultTitle is the label a season's champion carries until the commissioner sets another.
func DefaultTitle(season int) string { return fmt.Sprintf("%d League Champion", season) }

// BracketTitle is the label the season's champion carries.
type BracketTitle struct {
	Season  int    `json:"season"`
	Label   string `json:"label"`
	Default bool   `json:"default"`
	Entry   string `json:"entry,omitempty"`
	Reason  string `json:"reason,omitempty"`
}

// TitleReading is the label standing for a season: the last title.set that is still in force, else the default.
func TitleReading(records []Record, season int) BracketTitle {
	out := BracketTitle{Season: season, Label: DefaultTitle(season), Default: true}
	active := Replay(records).Active
	for _, r := range records {
		if r.Command.Command != TitleSet || !active[r.Receipt.Entry] || r.Command.Args.Title == nil ||
			r.Command.Args.Title.Season != season {
			continue
		}
		if label := r.Command.Args.Title.Label; label != "" {
			out = BracketTitle{Season: season, Label: strings.ReplaceAll(label, yearToken, strconv.Itoa(season)),
				Entry: r.Receipt.Entry, Reason: r.Command.Reason}
		} else {
			out = BracketTitle{Season: season, Label: DefaultTitle(season), Default: true, Entry: r.Receipt.Entry,
				Reason: r.Command.Reason}
		}
	}
	return out
}

func titleLabel(c RawTitleChange) string {
	if c.Label == "" {
		return fmt.Sprintf("Put back the default champion title for season %d", c.Season)
	}
	return fmt.Sprintf("Set the season %d champion title to %q", c.Season, c.Label)
}

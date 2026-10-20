package bids

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/engine/bid"
	"github.com/secureprospective/TheWarRoom/internal/envelope"
)

type Action struct {
	ID       string `json:"id"`
	Label    string `json:"label"`
	Disabled bool   `json:"disabled"`
	Reason   string `json:"reason"`
}
type Form struct {
	Salary string `json:"salary"`
	Beat   string `json:"beat"`
}

func BidStateWord(s envelope.State) string {
	words := map[envelope.State]string{
		envelope.Draft: "Draft", envelope.Blocked: "Blocked", envelope.Ready: "Ready",
		envelope.HandedOff: "Handed off", envelope.NotYetDone: "Not yet done", envelope.BidPending: "Bid pending",
		envelope.NotVerified: "Not verified", envelope.Landed: "Landed", envelope.AwardedOffApp: "Awarded off-app",
		envelope.Withdrawn: "Withdrawn", envelope.Stale: "Stale", envelope.Failed: "Failed",
	}
	if w, ok := words[s]; ok {
		return w
	}
	return string(s)
}

func BidOpen(s envelope.State) bool {
	return !slices.Contains([]envelope.State{envelope.Landed, envelope.AwardedOffApp, envelope.Withdrawn,
		envelope.Stale, envelope.Failed}, s)
}

func FormOf(f envelope.BidFacts) Form {
	whole, frac := int64(f.Salary)/1e8, int64(f.Salary)%1e8
	form := Form{Salary: strconv.FormatInt(whole, 10)}
	if frac != 0 {
		form.Salary += "." + strings.TrimRight(fmt.Sprintf("%08d", frac), "0")
	}
	if f.Beat != nil {
		form.Beat = f.Beat.String()
	}
	return form
}

func SentenceEnd(s string) string {
	s = strings.TrimSpace(s)
	if s == "" || strings.ContainsAny(s[len(s)-1:], ".!?") {
		return s
	}
	return s + "."
}

func Noted(r envelope.Receipt) envelope.Receipt {
	r.Audit = slices.Clone(r.Audit)
	for i := range r.Audit {
		r.Audit[i].Note = SentenceEnd(r.Audit[i].Note)
	}
	return r
}

func BeatText(f envelope.BidFacts) string {
	text := "No bid to beat entered"
	if f.Beat != nil {
		text = "Bid to beat " + f.Beat.String()
	}
	if f.Snipe {
		text += " · Snipe"
	}
	return text
}

func BidActions(s envelope.State, status string, rules []bid.Result, problem string) []Action {
	copyPost := Action{ID: "hand-off", Label: "Copy ProBoards post"}
	again := Action{ID: "copy", Label: "Copy post again"}
	switch {
	case slices.Contains([]envelope.State{envelope.Blocked, envelope.Ready, envelope.Draft}, s):
		reason := BidBlockReason(rules, problem)
		if reason == "" && s == envelope.Blocked {
			reason = status + "; change the figures to check it again"
		}
		if reason == "" && s == envelope.Draft {
			reason = "This plan is not checked yet"
		}
		if reason != "" {
			copyPost.Disabled, copyPost.Reason = true, SentenceEnd(reason)
		}
		out := []Action{copyPost, {ID: "revise", Label: "Change figures"}}
		out = append(out, Action{ID: "withdraw", Label: "Remove plan"})
		return out
	case slices.Contains([]envelope.State{envelope.HandedOff, envelope.NotYetDone, envelope.NotVerified}, s):
		out := []Action{again, {ID: "posted", Label: "I posted it"}}
		out = append(out, Action{ID: "unposted", Label: "I didn't post it"})
		return out
	case s == envelope.BidPending:
		return []Action{again, {ID: "withdraw", Label: "Bid was not processed"}}
	case s == envelope.Stale:
		return []Action{{ID: "withdraw", Label: "Remove plan"}}
	}
	return []Action{}
}

func BidBlockReason(rules []bid.Result, problem string) string {
	if problem != "" {
		return problem
	}
	if bid.Blocked(rules) {
		return bid.Note(rules)
	}
	return ""
}

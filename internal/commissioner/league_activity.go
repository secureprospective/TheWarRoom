package commissioner

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"slices"
	"sort"
	"strings"
	"time"
)

const (
	ActivityNotice = "notice"
	ActivityTrade  = "trade"
	ActivityRoster = "roster"

	defaultActivityLimit = 50
	maximumActivityLimit = 100
)

// RawLeagueActivityScope is the member-facing scope for public league events only.
type RawLeagueActivityScope struct {
	League string                   `json:"league"`
	Filter *RawLeagueActivityFilter `json:"filter,omitempty"`
}

// RawLeagueActivityFilter narrows a public event page; before is exclusive.
type RawLeagueActivityFilter struct {
	Kind      string `json:"kind,omitempty"`
	Franchise string `json:"franchise,omitempty"`
	Before    string `json:"before,omitempty"`
	Limit     int    `json:"limit,omitempty"`
}

func (r RawLeagueActivityScope) Validate() error {
	if !strings.HasPrefix(r.League, "native.") || r.League == "native." || len(r.League) > 128 ||
		strings.TrimSpace(r.League) != r.League {
		return fmt.Errorf("choose an app league")
	}
	if r.Filter != nil {
		return r.Filter.Validate()
	}
	return nil
}

func (r RawLeagueActivityFilter) Validate() error {
	if r.Kind != "" && r.Kind != ActivityNotice && r.Kind != ActivityTrade && r.Kind != ActivityRoster {
		return fmt.Errorf("choose a public activity kind")
	}
	if r.Franchise != "" && (strings.TrimSpace(r.Franchise) != r.Franchise || len(r.Franchise) > 128) {
		return fmt.Errorf("choose a franchise")
	}
	if r.Limit < 0 || r.Limit > maximumActivityLimit {
		return fmt.Errorf("choose up to %d activity events", maximumActivityLimit)
	}
	if r.Before != "" {
		if _, err := time.Parse(time.RFC3339Nano, r.Before); err != nil {
			return fmt.Errorf("choose an activity cursor time: %w", err)
		}
	}
	return nil
}

// LeagueActivityEvent is a deliberately small projection. It has no ledger payload, actor reason,
// bid, note, or pending-offer fields.
type LeagueActivityEvent struct {
	ID         string    `json:"id"`
	Kind       string    `json:"kind"`
	When       time.Time `json:"when"`
	Summary    string    `json:"summary"`
	Franchises []string  `json:"franchises,omitempty"`
}

type LeagueActivity struct {
	League   string                `json:"league"`
	Revision int64                 `json:"revision"`
	Events   []LeagueActivityEvent `json:"events"`
}

// LeagueActivityReading projects only explicit public event kinds. Unknown ledger commands are hidden.
func LeagueActivityReading(scope RawLeagueActivityScope, records []Record) (LeagueActivity, error) {
	if err := scope.Validate(); err != nil {
		return LeagueActivity{}, err
	}
	active := Replay(records).Active
	tradeStates := make(map[string]string)
	all := make([]LeagueActivityEvent, 0)
	for _, record := range records {
		if !active[record.Receipt.Entry] || record.Command.League != scope.League {
			continue
		}
		if event, ok := publicActivityEvent(record, tradeStates); ok {
			all = append(all, event)
		}
	}
	out := LeagueActivity{League: scope.League, Revision: activityRevision(all), Events: make([]LeagueActivityEvent, 0)}
	filter := RawLeagueActivityFilter{}
	if scope.Filter != nil {
		filter = *scope.Filter
	}
	before := time.Time{}
	if filter.Before != "" {
		var err error
		before, err = time.Parse(time.RFC3339Nano, filter.Before)
		if err != nil {
			return LeagueActivity{}, fmt.Errorf("choose an activity cursor time: %w", err)
		}
	}
	sort.SliceStable(all, func(i, j int) bool { return all[i].When.Before(all[j].When) })
	limit := filter.Limit
	if limit == 0 {
		limit = defaultActivityLimit
	}
	for i := len(all) - 1; i >= 0; i-- {
		event := all[i]
		if !activityMatches(event, filter, before) {
			continue
		}
		out.Events = append(out.Events, event)
		if len(out.Events) == limit {
			break
		}
	}
	return out, nil
}

func publicActivityEvent(record Record, tradeStates map[string]string) (LeagueActivityEvent, bool) {
	if record.Command.Visibility != "league" || record.Receipt.Entry == "" {
		return LeagueActivityEvent{}, false
	}
	base := LeagueActivityEvent{ID: record.Receipt.Entry, When: record.Receipt.RecordedAt}
	switch record.Command.Command {
	case NoticePublish:
		return publicNoticeEvent(record, base)
	case RosterCut, WaiverProcess, WaiverOverride, BidsProcess, BidsOverride:
		return publicRosterActivityEvent(record, base)
	default:
		if publicTradeActivityCommand(record.Command.Command) {
			return completedTradeEvent(record, base, tradeStates)
		}
		return LeagueActivityEvent{}, false
	}
}

func publicNoticeEvent(record Record, event LeagueActivityEvent) (LeagueActivityEvent, bool) {
	if record.Command.Args.Note == nil {
		return LeagueActivityEvent{}, false
	}
	event.Kind, event.Summary = ActivityNotice, record.Command.Args.Note.Text
	return event, true
}

func publicRosterActivityEvent(record Record, event LeagueActivityEvent) (LeagueActivityEvent, bool) {
	switch record.Command.Command {
	case RosterCut:
		args, outcome := record.Command.Args.Waiver, record.Command.Args.WaiverOutcome
		if args == nil || outcome == nil || outcome.Player == nil {
			return LeagueActivityEvent{}, false
		}
		event.Kind, event.Summary, event.Franchises = ActivityRoster, "Player released from a roster", []string{args.Franchise}
	case WaiverProcess, WaiverOverride:
		outcome := record.Command.Args.WaiverOutcome
		if outcome == nil || outcome.Winner == "" {
			return LeagueActivityEvent{}, false
		}
		event.Kind, event.Summary, event.Franchises = ActivityRoster, "Waiver claim awarded", []string{outcome.Winner}
	case BidsProcess, BidsOverride:
		outcome := record.Command.Args.BidOutcome
		if outcome == nil || outcome.Winner == "" {
			return LeagueActivityEvent{}, false
		}
		event.Kind, event.Summary, event.Franchises = ActivityRoster, "Player added to a roster", []string{outcome.Winner}
	default:
		return LeagueActivityEvent{}, false
	}
	return event, true
}

func publicTradeActivityCommand(command string) bool {
	switch command {
	case TradeOffer, TradeAccept, TradeDecline, TradeWithdraw, TradeReview, TradeRecuse,
		TradeAppeal, TradeAppealRule, TradeOverride:
		return true
	default:
		return false
	}
}

func completedTradeEvent(record Record, event LeagueActivityEvent, states map[string]string) (LeagueActivityEvent, bool) {
	outcome := record.Command.Args.TradeOutcome
	if outcome == nil {
		return LeagueActivityEvent{}, false
	}
	id := outcome.ID
	if id == "" {
		id = record.Receipt.Entry
	}
	previous := states[id]
	states[id] = outcome.Status
	if outcome.Status != "approved" || previous == "approved" {
		return LeagueActivityEvent{}, false
	}
	event.Kind, event.Summary = ActivityTrade, "Trade completed"
	event.Franchises = slices.Clone(outcome.Parties)
	return event, true
}

func activityMatches(event LeagueActivityEvent, filter RawLeagueActivityFilter, before time.Time) bool {
	if filter.Kind != "" && filter.Kind != event.Kind || (!before.IsZero() && !event.When.Before(before)) {
		return false
	}
	if filter.Franchise == "" || event.Kind == ActivityNotice {
		return true
	}
	return slices.Contains(event.Franchises, filter.Franchise)
}

func activityRevision(events []LeagueActivityEvent) int64 {
	digest := sha256.New()
	for _, event := range events {
		_, _ = digest.Write([]byte(event.ID))
		_, _ = digest.Write([]byte{0})
	}
	return int64(binary.BigEndian.Uint64(digest.Sum(nil)[:8]) & ((1 << 53) - 1))
}

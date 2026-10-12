package main

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/engine/bid"
	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
	"github.com/secureprospective/TheWarRoom/internal/normalize"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

type RawCommsScope struct {
	Team string `json:"team"`
	Kind string `json:"kind"`
}

func (r RawCommsScope) Validate() error {
	if r.Team != "" {
		if err := ingestion.FeedFranchise(r.Team); err != nil {
			return fmt.Errorf("league feed: %w", err)
		}
	}
	if r.Kind != "" && !slices.Contains([]string{"trade", "add-drop", "ir", "taxi", "other"}, r.Kind) {
		return fmt.Errorf("league feed: invalid kind %q", r.Kind)
	}
	return nil
}

type CommsEvent struct {
	Sentence string `json:"sentence"`
	Date     string `json:"date"`
}

type CommsBoard struct {
	Feed         Report       `json:"feed"`
	Transactions Report       `json:"transactions"`
	Events       []CommsEvent `json:"events"`
	BoardNote    string       `json:"boardNote"`
	LeagueTalk   string       `json:"leagueTalk"`
	BoardPosts   []Report     `json:"boardPosts"`
	Team         string       `json:"team"`
	Kind         string       `json:"kind"`
	Teams        []RaceChoice `json:"teams"`
	Kinds        []RaceChoice `json:"kinds"`
}

type commsInput struct {
	transactions heldSeasonFeed[[]leaguefeed.Transaction]
	boards       [2]heldSeasonFeed[[]ingestion.RawLeaguePost]
	names        map[string]string
	lookup       normalize.Lookup
	mirror       state.MirrorSnapshot
	zone         *time.Location
}

func buildComms(in commsInput, scope RawCommsScope) (CommsBoard, error) {
	if err := scope.Validate(); err != nil {
		return CommsBoard{}, err
	}
	if scope.Team != "" && in.names[scope.Team] == "" {
		return CommsBoard{}, fmt.Errorf("league feed: franchise is not in the league")
	}
	out := CommsBoard{Team: scope.Team, Kind: scope.Kind, LeagueTalk: "The league talks on ProBoards.",
		Events: []CommsEvent{}, Teams: []RaceChoice{{Value: "", Label: "All franchises"}},
		Kinds: []RaceChoice{{Value: "", Label: "All moves"}, {Value: "trade", Label: "Trades"},
			{Value: "add-drop", Label: "Adds and drops"}, {Value: "ir", Label: "IR moves"},
			{Value: "taxi", Label: "Taxi moves"}, {Value: "other", Label: "Other transactions"}}}
	ids := make([]string, 0, len(in.names))
	for id := range in.names {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	for _, id := range ids {
		out.Teams = append(out.Teams, RaceChoice{Value: id, Label: in.names[id]})
	}
	out.Feed = reportShell("C-05", "League feed", "Newest first, from held MFL transactions. A salary and contract end are the player's current contract on MFL, not the terms when the move was made; missing terms are labelled. No messages are sent.", in.transactions, in.zone, "Local time", "Move")
	out.Transactions = out.Feed
	out.Transactions.ID, out.Transactions.Title = "M-032", "Transactions"
	out.Transactions.SortAscending, out.Transactions.SortDescending = "Sort oldest first", "Sort newest first"
	out.Transactions.Link = &ReportLink{Label: "Read the league feed", Endpoint: "C-05"}
	rows := slices.Clone(in.transactions.value)
	slices.SortStableFunc(rows, func(a, b leaguefeed.Transaction) int { return b.Time.Compare(a.Time) })
	for _, tx := range rows {
		if !commsMatches(tx, scope) {
			continue
		}
		sentence := in.transactionSentence(tx)
		date := reportTime(tx.Time, in.zone)
		out.Events = append(out.Events, CommsEvent{Sentence: sentence, Date: date})
		out.Transactions.Rows = append(out.Transactions.Rows, ReportRow{Cells: []string{date, sentence}, Sort: []float64{float64(tx.Time.Unix()), float64(tx.Time.Unix())}})
	}
	if len(out.Events) == 0 && out.Feed.Status == "ok" {
		out.Feed.Status, out.Feed.Note = "empty", "No league moves match these filters."
		out.Transactions.Status, out.Transactions.Note = out.Feed.Status, out.Feed.Note
	}
	out.BoardPosts, out.BoardNote = commsPosts(in)
	return out, nil
}

func commsKind(tx leaguefeed.Transaction) string {
	switch tx.Kind {
	case "TRADE":
		return "trade"
	case "FREE_AGENT", "LOAD_ROSTERS":
		return "add-drop"
	case "IR":
		return "ir"
	case "TAXI":
		return "taxi"
	default:
		return "other"
	}
}

func commsMatches(tx leaguefeed.Transaction, scope RawCommsScope) bool {
	team := scope.Team == "" || tx.Franchise == scope.Team || (tx.Trade != nil && tx.Trade.Counterparty == scope.Team)
	return team && (scope.Kind == "" || commsKind(tx) == scope.Kind)
}

func (in commsInput) team(id string) string {
	if name := in.names[id]; name != "" {
		return name
	}
	return "A franchise whose name is unavailable"
}

func (in commsInput) asset(a leaguefeed.Asset, contract bool) string {
	switch {
	case a.Player != nil:
		name, ok := in.lookup.Name(a.Player.String())
		if !ok {
			return "a player whose details are unavailable; refresh from MFL"
		}
		name = proseName(name)
		if facts, ok := in.lookup.Facts(a.Player.String()); ok {
			name += fmt.Sprintf(" (%s, %s)", facts.Position, facts.Team)
		}
		if contract {
			name += in.contract(a.Player.String())
		}
		return name
	case a.FuturePick != nil:
		p := a.FuturePick
		owner := in.team(p.Franchise)
		possessive := "'s"
		if strings.HasSuffix(owner, "s") {
			possessive = "'"
		}
		return fmt.Sprintf("%s%s %d round %d pick", owner, possessive, p.Year, p.Round)
	case a.CurrentPick != nil:
		return fmt.Sprintf("round %d pick %d", a.CurrentPick.Round, a.CurrentPick.Pick)
	case a.BlindBidCents != nil:
		return bid.MoneyText(domain.Money(*a.BlindBidCents)) + " in bid money"
	default:
		return "an asset whose details are unavailable"
	}
}

func (in commsInput) contract(id string) string {
	for _, p := range in.mirror.Players {
		if p.MFLID != id {
			continue
		}
		if p.ContractYear < in.mirror.Season {
			return " (" + bid.MoneyText(p.Salary) + "; contract end unavailable)"
		}
		return fmt.Sprintf(" (%s, contract through %d)", bid.MoneyText(p.Salary), p.ContractYear)
	}
	return " (salary and contract unavailable)"
}

// proseName turns MFL's "Last, First" into the "First Last" a sentence reads.
func proseName(name string) string {
	last, first, ok := strings.Cut(name, ", ")
	if !ok || first == "" || strings.Contains(first, ", ") {
		return name
	}
	return first + " " + last
}

func (in commsInput) assets(rows []leaguefeed.Asset) string {
	names := make([]string, 0, len(rows))
	for _, a := range rows {
		names = append(names, in.asset(a, false))
	}
	if len(names) == 0 {
		return "no assets"
	}
	return strings.Join(names, ", ")
}

func (in commsInput) transactionSentence(tx leaguefeed.Transaction) string {
	name := in.team(tx.Franchise)
	if tx.Trade != nil {
		return fmt.Sprintf("%s traded %s to %s for %s.", name, in.assets(tx.Trade.Gave), in.team(tx.Trade.Counterparty), in.assets(tx.Trade.CounterpartyGave))
	}
	changes := []struct {
		change             *leaguefeed.RosterChange
		incoming, outgoing string
		contract           bool
	}{
		{tx.AddsDrops, "added %s", "dropped %s", true}, {tx.IR, "activated %s from IR", "placed %s on IR", false},
		{tx.Taxi, "promoted %s from taxi", "moved %s to taxi", false},
	}
	for _, c := range changes {
		if c.change == nil {
			continue
		}
		clauses := []string{}
		for _, side := range []struct {
			rows []leaguefeed.Asset
			verb string
		}{{c.change.In, c.incoming}, {c.change.Out, c.outgoing}} {
			for _, a := range side.rows {
				clauses = append(clauses, fmt.Sprintf(side.verb, in.asset(a, c.contract)))
			}
		}
		if len(clauses) > 0 {
			return name + " " + strings.Join(clauses, "; ") + "."
		}
	}
	label := strings.ToLower(strings.ReplaceAll(tx.Kind, "_", " "))
	return name + " recorded " + label + " on MFL; event details are unavailable in this export."
}

func commsPosts(in commsInput) ([]Report, string) {
	reports := []Report{}
	unavailable, count := false, 0
	for i, feed := range in.boards {
		id, title := "M-120", "MFL message board"
		if i == 1 {
			id, title = "M-123", "MFL polls"
		}
		r := reportShell(id, title, "Held MFL exports; discussion and voting open on MFL. The app never posts or votes.", feed, in.zone, "Title", "Author franchise", "Local time")
		unavailable = unavailable || feed.refreshError != ""
		if feed.fetchedAt.IsZero() {
			unavailable = true
			r.Note = title + " not available; refresh from MFL. An authentication error is not an empty board."
		}
		rows := slices.Clone(feed.value)
		slices.SortStableFunc(rows, func(a, b ingestion.RawLeaguePost) int { return strings.Compare(b.Timestamp, a.Timestamp) })
		for _, row := range rows {
			title := row.Title
			if title == "" {
				title = row.Subject
			}
			at, err := ingestion.FeedTime(row.Timestamp)
			if err != nil {
				unavailable = true
				continue
			}
			r.Rows = append(r.Rows, ReportRow{Cells: []string{title, in.team(row.Franchise), reportTime(at, in.zone)}})
		}
		count += len(r.Rows)
		reports = append(reports, r)
	}
	if unavailable {
		return reports, "MFL's message board or polls are not available. Refresh from MFL; league talk is on ProBoards."
	}
	if count == 0 {
		return reports, "No posts on MFL's message board or polls this season."
	}
	return reports, "Discussion and voting stay on MFL; the app only opens the page."
}

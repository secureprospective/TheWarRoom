// Package comms builds league communications from immutable caller-owned snapshots.
package comms

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/report"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/engine/bid"
	"github.com/secureprospective/TheWarRoom/internal/facts"
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

type Event struct {
	Sentence string `json:"sentence"`
	Date     string `json:"date"`
}

type Board struct {
	Feed         report.Report   `json:"feed"`
	Transactions report.Report   `json:"transactions"`
	Events       []Event         `json:"events"`
	BoardNote    string          `json:"boardNote"`
	LeagueTalk   string          `json:"leagueTalk"`
	BoardPosts   []report.Report `json:"boardPosts"`
	Team         string          `json:"team"`
	Kind         string          `json:"kind"`
	Teams        []report.Choice `json:"teams"`
	Kinds        []report.Choice `json:"kinds"`
}

type Input struct {
	Transactions report.Held[leaguefeed.Transaction]
	Boards       [2]report.Held[ingestion.RawLeaguePost]
	Names        map[string]string
	Lookup       normalize.Lookup
	Mirror       state.MirrorSnapshot
	Zone         *time.Location
	Rosters      facts.RosterWeekColumns
	RosterTimes  map[[2]int64]int64
	RosterNote   string
}

func Build(in Input, scope RawCommsScope) (Board, error) {
	if err := scope.Validate(); err != nil {
		return Board{}, err
	}
	if scope.Team != "" && in.Names[scope.Team] == "" {
		return Board{}, fmt.Errorf("league feed: franchise is not in the league")
	}
	out := Board{Team: scope.Team, Kind: scope.Kind, LeagueTalk: "The league talks on ProBoards.",
		Events: []Event{}, Teams: []report.Choice{{Value: "", Label: "All franchises"}},
		Kinds: []report.Choice{{Value: "", Label: "All moves"}, {Value: "trade", Label: "Trades"},
			{Value: "add-drop", Label: "Adds and drops"}, {Value: "ir", Label: "IR moves"},
			{Value: "taxi", Label: "Taxi moves"}, {Value: "other", Label: "Other transactions"}}}
	ids := make([]string, 0, len(in.Names))
	for id := range in.Names {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	for _, id := range ids {
		out.Teams = append(out.Teams, report.Choice{Value: id, Label: in.Names[id]})
	}
	out.Feed = report.Shell("C-05", "League feed", "Newest first, from held MFL transactions. Adds show the current MFL contract. Drops show the last held weekly contract before the move; missing salary is labelled. No messages are sent.", in.Transactions.Feed, len(in.Transactions.Value), in.Zone, "Local time", "Move")
	out.Feed.Note += in.RosterNote
	out.Transactions = out.Feed
	out.Transactions.ID, out.Transactions.Title = "M-032", "Transactions"
	out.Transactions.SortAscending, out.Transactions.SortDescending = "Sort oldest first", "Sort newest first"
	out.Transactions.Link = &report.Link{Label: "Read the league feed", Endpoint: "C-05"}
	rows := slices.Clone(in.Transactions.Value)
	slices.SortStableFunc(rows, func(a, b leaguefeed.Transaction) int { return b.Time.Compare(a.Time) })
	for _, tx := range rows {
		if !commsMatches(tx, scope) {
			continue
		}
		sentence := in.transactionSentence(tx)
		date := report.Time(tx.Time, in.Zone)
		out.Events = append(out.Events, Event{Sentence: sentence, Date: date})
		out.Transactions.Rows = append(out.Transactions.Rows, report.Row{Cells: []string{date, sentence}, Sort: []float64{float64(tx.Time.Unix()), float64(tx.Time.Unix())}})
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

func (in Input) team(id string) string {
	if name := in.Names[id]; name != "" {
		return name
	}
	return "A franchise whose name is unavailable"
}

func (in Input) asset(a leaguefeed.Asset, contract bool) string {
	switch {
	case a.Player != nil:
		name, ok := in.Lookup.Name(a.Player.String())
		if !ok {
			return "a player whose details are unavailable; refresh from MFL"
		}
		name = report.ProseName(name)
		if facts, ok := in.Lookup.Facts(a.Player.String()); ok {
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

func (in Input) contract(id string) string {
	for _, p := range in.Mirror.Players {
		if p.MFLID != id {
			continue
		}
		if p.ContractYear < in.Mirror.Season {
			return " (" + bid.MoneyText(p.Salary) + "; contract end unavailable)"
		}
		return fmt.Sprintf(" (%s, contract through %d)", bid.MoneyText(p.Salary), p.ContractYear)
	}
	return " (salary not held)"
}

func (in Input) assets(rows []leaguefeed.Asset) string {
	names := make([]string, 0, len(rows))
	for _, a := range rows {
		names = append(names, in.asset(a, false))
	}
	if len(names) == 0 {
		return "no assets"
	}
	return strings.Join(names, ", ")
}

func (in Input) transactionSentence(tx leaguefeed.Transaction) string {
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
				asset := in.asset(a, c.contract)
				if c.contract && side.verb == c.outgoing && a.Player != nil {
					asset = in.asset(a, false) + in.droppedContract(a.Player.String(), tx.Franchise, tx.Time)
				}
				clauses = append(clauses, fmt.Sprintf(side.verb, asset))
			}
		}
		if len(clauses) > 0 {
			return name + " " + strings.Join(clauses, "; ") + "."
		}
	}
	label := strings.ToLower(strings.ReplaceAll(tx.Kind, "_", " "))
	return name + " recorded " + label + " on MFL; event details are unavailable in this export."
}

func commsPosts(in Input) ([]report.Report, string) {
	reports := []report.Report{}
	unavailable, count := false, 0
	for i, feed := range in.Boards {
		id, title := "M-120", "MFL message board"
		if i == 1 {
			id, title = "M-123", "MFL polls"
		}
		r := report.Shell(id, title, "Held MFL exports; discussion and voting open on MFL. The app never posts or votes.", feed.Feed, len(feed.Value), in.Zone, "Title", "Author franchise", "Local time")
		unavailable = unavailable || feed.RefreshError != ""
		if feed.FetchedAt.IsZero() {
			unavailable = true
			r.Note = title + " not available; refresh from MFL. An authentication error is not an empty board."
			if feed.RefreshError != "" {
				r.Note = title + " not available: " + feed.RefreshError + "."
			}
		}
		rows := slices.Clone(feed.Value)
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
			r.Rows = append(r.Rows, report.Row{Cells: []string{title, in.team(row.Franchise), report.Time(at, in.Zone)}})
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

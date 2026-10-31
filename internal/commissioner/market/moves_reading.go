package market

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

// NativeMove uses the HQ receipt shape without MFL targets or synthetic MFL transitions.
type NativeMove struct {
	CorrelationID string            `json:"correlationId"`
	Spec          NativeMoveSpec    `json:"spec"`
	State         string            `json:"state"`
	Audit         []NativeMoveAudit `json:"audit"`
}

type NativeMoveSpec struct {
	Intent      string            `json:"intent"`
	LeagueID    string            `json:"leagueId"`
	FranchiseID string            `json:"franchiseId"`
	Subject     NativeMoveSubject `json:"subject"`
	Expected    struct{}          `json:"expected"`
	Gravity     string            `json:"gravity"`
	Undo        string            `json:"undo"`
	Target      NativeMoveTarget  `json:"target"`
	Deadline    *time.Time        `json:"deadline,omitempty"`
}

type NativeMoveSubject struct {
	Players []string `json:"players"`
	Picks   []string `json:"picks"`
}

type NativeMoveTarget struct {
	Kind string `json:"kind"`
}

type NativeMoveAudit struct {
	At    time.Time `json:"at"`
	From  string    `json:"from"`
	Event string    `json:"event"`
	To    string    `json:"to"`
	Note  string    `json:"note"`
}

type NativeMoves struct {
	League    string       `json:"league"`
	Franchise string       `json:"franchise"`
	Planned   []NativeMove `json:"planned"`
	Reason    string       `json:"reason"`
	Moves     []NativeMove `json:"moves"`
}

func nativeMove(league, franchise, id, intent, state, note string, at time.Time) NativeMove {
	return NativeMove{CorrelationID: id, State: state,
		Spec: NativeMoveSpec{Intent: intent, LeagueID: league, FranchiseID: franchise,
			Subject: NativeMoveSubject{Players: []string{}, Picks: []string{}},
			Gravity: "G0", Undo: "irreversible", Target: NativeMoveTarget{Kind: "native"}},
		Audit: []NativeMoveAudit{{At: at, From: "draft", Event: "recorded", To: state, Note: note}}}
}

func NativeMovesReading(league, franchise, principal string, records []commissioner.Record, at time.Time,
	zone *time.Location, facts nativecheck.Facts) (NativeMoves, error) {
	if !strings.HasPrefix(league, "native.") || len(league) <= len("native.") || strings.TrimSpace(franchise) == "" || at.IsZero() {
		return NativeMoves{}, fmt.Errorf("choose an app league, franchise and reading time")
	}
	for _, r := range records {
		if r.Command.League != league {
			return NativeMoves{}, fmt.Errorf("move records belong to another league")
		}
	}
	records = commissioner.RecordsThrough(records, at)
	if zone == nil {
		zone = time.UTC
	}
	trades, err := NativeTradesReading(league, franchise, principal, records, at, zone)
	if err != nil {
		return NativeMoves{}, fmt.Errorf("read move trades: %w", err)
	}
	bids, err := NativeBidsReading(league, franchise, principal, records, at, zone, facts)
	if err != nil {
		return NativeMoves{}, fmt.Errorf("read move bids: %w", err)
	}
	waivers, err := NativeWaiversReading(league, franchise, principal, records, at, zone, facts)
	if err != nil {
		return NativeMoves{}, fmt.Errorf("read move waivers: %w", err)
	}
	out := NativeMoves{League: league, Franchise: franchise, Planned: []NativeMove{},
		Reason: "native plans not yet stored", Moves: []NativeMove{}}
	out.appendPendingBids(bids)
	out.appendPendingWaivers(waivers, records)
	out.appendWaitingTrades(trades, records)
	f, err := commissioner.NewFrame(league, records, at, facts, zone)
	if err != nil {
		return NativeMoves{}, fmt.Errorf("read committed moves: %w", err)
	}
	out.appendLanded(records, f.Book)
	return out, nil
}

func (out *NativeMoves) appendPendingBids(bids NativeBids) {
	for _, p := range bids.Players {
		if p.YourBid == nil {
			continue
		}
		m := nativeMove(out.League, out.Franchise, p.YourBid.ID, commissioner.BidPlace, "handed_off",
			p.Name+": "+p.YourBid.Contract+" · "+p.Closes, p.YourBid.PlacedAt)
		m.Spec.Subject.Players = append(m.Spec.Subject.Players, p.Player)
		m.Spec.Deadline = p.ClosesAt
		out.Moves = append(out.Moves, m)
	}
}

func (out *NativeMoves) appendPendingWaivers(waivers NativeWaivers, records []commissioner.Record) {
	for _, c := range waivers.Claims {
		i := slices.IndexFunc(waivers.Players, func(p NativeWaiverPlayer) bool { return p.Window == c.Window })
		if i < 0 {
			continue
		}
		p := waivers.Players[i]
		if p.Status != "On waivers" && p.Status != "Closed; awaiting processing" {
			continue
		}
		for _, r := range records {
			if r.Receipt.Entry == c.ID {
				m := nativeMove(out.League, out.Franchise, c.ID, commissioner.WaiverClaim, "handed_off",
					p.Name+": claim waiting for processing · "+p.Closes, r.Receipt.RecordedAt)
				m.Spec.Subject.Players = append(m.Spec.Subject.Players, p.Player)
				m.Spec.Deadline = &p.ClosesAt
				out.Moves = append(out.Moves, m)
			}
		}
	}
}

func (out *NativeMoves) appendWaitingTrades(trades NativeTrades, records []commissioner.Record) {
	for _, t := range trades.Offers {
		// The desk read is broader than this private queue. No non-party trade enters it.
		if !slices.Contains(t.Parties, out.Franchise) {
			continue
		}
		state := ""
		switch t.Status {
		case "offered":
			state = "handed_off"
			if !slices.Contains(t.Accepted, out.Franchise) {
				state = "needs_you"
			}
		case "review", "appealed":
			state = "handed_off"
		case "approved":
			state = "landed"
		}
		if state == "" {
			continue
		}
		m := nativeMove(out.League, out.Franchise, t.ID, commissioner.TradeOffer, state, tradeMoveWords(t), tradeMoveTime(t, records))
		if t.Status == "offered" {
			m.Spec.Deadline = &t.ExpiresAt
		}
		for _, asset := range t.Assets {
			if asset.Player != "" {
				m.Spec.Subject.Players = append(m.Spec.Subject.Players, asset.Player)
			} else {
				m.Spec.Subject.Picks = append(m.Spec.Subject.Picks, fmt.Sprintf("%d round %d · %s", asset.Season, asset.Round, asset.Original))
			}
		}
		out.Moves = append(out.Moves, m)
	}
}

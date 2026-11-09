package market

import (
	"fmt"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

func tradeMoveWords(t NativeTradeOffer) string {
	words := []string{t.Words}
	for _, d := range t.Details {
		words = append(words, strings.TrimSpace(d.Label+" "+d.Contract))
	}
	return strings.Join(words, " · ")
}

func tradeMoveEvidence(t NativeTradeOffer, records []commissioner.Record) (time.Time, string) {
	at, entry := t.OfferedAt, t.ID
	active := commissioner.Replay(records).Active
	for _, r := range records {
		if v := r.Command.Args.TradeOutcome; active[r.Receipt.Entry] && v != nil &&
			(v.ID == t.ID || r.Receipt.Entry == t.ID) {
			at, entry = r.Receipt.RecordedAt, r.Receipt.Entry
		}
	}
	return at, entry
}

func (out *NativeMoves) appendLanded(records []commissioner.Record, book commissioner.MarketBook) {
	active := commissioner.Replay(records).Active
	for _, r := range records {
		if !active[r.Receipt.Entry] {
			continue
		}
		switch r.Command.Command {
		case commissioner.RosterCut:
			w := r.Command.Args.WaiverOutcome
			if w != nil && w.Player != nil && w.Player.Franchise == out.Franchise {
				out.landedPlayer(r, r.Receipt.Entry, commissioner.RosterCut, w.Player.ID,
					fmt.Sprintf("Released %s · dead cap %s", w.Player.Name, w.Dead))
			}
		case commissioner.BidsProcess, commissioner.BidsOverride:
			out.landBid(r, &book.Bids)
		case commissioner.WaiverProcess, commissioner.WaiverOverride:
			out.landClaim(r, &book)
		}
	}
}

func (out *NativeMoves) landedPlayer(r commissioner.Record, id, intent, player, words string) {
	m := nativeMove(out.League, out.Franchise, id, intent, "landed", words+" · Committed in league ledger: "+r.Receipt.Entry, r.Receipt.RecordedAt)
	m.Spec.Subject.Players = append(m.Spec.Subject.Players, player)
	for i := range out.Moves {
		if out.Moves[i].CorrelationID == id {
			out.Moves[i] = m
			return
		}
	}
	out.Moves = append(out.Moves, m)
}

func (out *NativeMoves) landBid(r commissioner.Record, book *commissioner.BidBook) {
	b := r.Command.Args.BidOutcome
	if b == nil || b.Winner != out.Franchise || b.WinningBid == "" {
		return
	}
	a, i, err := book.Find(b.WinningBid)
	if err != nil || a.WinningBid != b.WinningBid || a.Winner != out.Franchise || !a.DecidedAt.Equal(r.Receipt.RecordedAt) {
		return
	}
	bid := a.Bids[i]
	out.landedPlayer(r, b.WinningBid, commissioner.BidPlace, a.Player,
		"Signed player "+a.Player+" · "+nativecheck.ContractWords(bid.Salary, bid.Years))
}

func (out *NativeMoves) landClaim(r commissioner.Record, book *commissioner.MarketBook) {
	args, result := r.Command.Args.Waiver, r.Command.Args.WaiverOutcome
	if args == nil || result == nil || result.Winner != out.Franchise {
		return
	}
	w, err := book.Window(args.Window)
	if r.Command.Command == commissioner.WaiverOverride {
		w, _, err = book.Claim(args.Claim)
	}
	if err != nil || w.Winner != out.Franchise || !w.ClosedAt.Equal(result.CloseAt) {
		return
	}
	for _, c := range w.Claims {
		if c.Franchise == out.Franchise {
			out.landedPlayer(r, c.ID, commissioner.WaiverClaim, w.Player.ID, "Waiver claim awarded to your franchise")
			return
		}
	}
}

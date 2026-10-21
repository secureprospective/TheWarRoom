package contracts

import (
	"fmt"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/transactions"
)

// MoveDTO is one leg of a trade.
type MoveDTO struct {
	MFLID         string `json:"mflID"`
	ToFranchiseID string `json:"toFranchiseID"`
}

// TransactionRequest is the typed IPC payload for every transaction. Kind selects which fields
// are read. Money arrives as millions strings and is parsed to exact cents server-side, never
// as a JS number; every other figure is resolved from authoritative state.
type TransactionRequest struct {
	Kind  string    `json:"kind"`
	Moves []MoveDTO `json:"moves"`
	// TRADE: PicksNote is free text (there is no pick-ownership ledger yet); Rationale is required.
	PicksNote    string `json:"picksNote"`
	Rationale    string `json:"rationale"`
	MFLID        string `json:"mflID"`
	Status       string `json:"status"`
	MoveMillions string `json:"moveMillions"`
	AddedYears   int    `json:"addedYears"` // EXTENSION: years to add (1-3)
	ToPhase      string `json:"toPhase"`    // ADVANCE_PHASE: target phase
	Note         string `json:"note"`
	// Special situations (rulebook §13): RETIREMENT and DEATH read MFLID; CAP_RELIEF reads
	// FranchiseID, AmountMillions and Reason.
	FranchiseID    string `json:"franchiseID"`
	AmountMillions string `json:"amountMillions"`
	Reason         string `json:"reason"`
	// SIGN (§6): MFLID, FranchiseID, SalaryMillions (flat per year) and Years (1-4).
	SalaryMillions string `json:"salaryMillions"`
	Years          int    `json:"years"`
	// SET_SIGNING_WINDOW (§6): open or close the commissioner signing window.
	WindowOpen bool `json:"windowOpen"`
	// SET_TRADE_DEADLINE (§14): an ISO-8601 instant; empty clears the deadline.
	TradeDeadline string `json:"tradeDeadline"`
	// Calendar ops: EventID is minted by the frontend on schedule and resent on reschedule or
	// cancel. Payload is the eventual op's fields, stored as-is and run only when the event fires.
	EventID     string `json:"eventID"`
	EventKind   string `json:"eventKind"`
	ScheduledAt string `json:"scheduledAt"`
	Payload     string `json:"payload"`
}

// buildRequest maps the DTO onto a sealed transactions.Request; an unknown Kind is rejected
// here.
func buildRequest(req TransactionRequest) (transactions.Request, error) {
	switch req.Kind {
	case string(transactions.KindTrade):
		moves := make([]transactions.PlayerMove, len(req.Moves))
		for i, m := range req.Moves {
			moves[i] = transactions.PlayerMove{MFLID: m.MFLID, ToFranchiseID: m.ToFranchiseID}
		}
		return transactions.Trade{Moves: moves, PicksNote: req.PicksNote, Rationale: req.Rationale}, nil
	case string(transactions.KindRosterStatus):
		return transactions.RosterStatusChange{
			MFLID:  req.MFLID,
			Status: domain.RosterStatus(req.Status),
		}, nil
	case string(transactions.KindWaiver):
		return transactions.Waiver{MFLID: req.MFLID}, nil
	case string(transactions.KindBuyout):
		// §12: offseason only, two per team per season.
		return transactions.Buyout{MFLID: req.MFLID}, nil
	case string(transactions.KindAdvancePhase):
		return transactions.AdvancePhase{To: domain.Phase(req.ToPhase), Note: req.Note}, nil
	case string(transactions.KindRolloverSeason):
		// §14: the season boundary, PLAYOFFS(N) to OFFSEASON(N+1).
		return transactions.RolloverSeason{Note: req.Note}, nil
	case string(transactions.KindSetSigningWindow):
		return transactions.SetSigningWindow{Open: req.WindowOpen, Note: req.Note}, nil
	case string(transactions.KindRetirement):
		// §13: 30% of the remaining contract becomes dead cap.
		return transactions.Retirement{MFLID: req.MFLID}, nil
	case string(transactions.KindDeath):
		// §13 Gaines Adams Rule: removed with zero dead cap.
		return transactions.Death{MFLID: req.MFLID}, nil
	case string(transactions.KindScheduleEvent):
		return transactions.ScheduleEvent{Event: calendarEvent(req)}, nil
	case string(transactions.KindRescheduleEvent):
		return transactions.RescheduleEvent{Event: calendarEvent(req)}, nil
	case string(transactions.KindCancelEvent):
		return transactions.CancelEvent{Event: calendarEvent(req)}, nil
	default:
		return buildContractRequest(req)
	}
}

// buildContractRequest maps the contract kinds and those whose parsing can fail: tag, extension,
// restructure and signing, cap relief (money parsing) and the trade deadline (time parsing).
func buildContractRequest(req TransactionRequest) (transactions.Request, error) {
	switch req.Kind {
	case string(transactions.KindTag):
		return transactions.Tag{MFLID: req.MFLID}, nil
	case string(transactions.KindExtension):
		return transactions.Extension{MFLID: req.MFLID, AddedYears: req.AddedYears}, nil
	case string(transactions.KindSetTradeDeadline):
		// An empty deadline is the zero time, which clears it.
		var deadline time.Time
		if req.TradeDeadline != "" {
			d, err := time.Parse(time.RFC3339, req.TradeDeadline)
			if err != nil {
				return nil, fmt.Errorf("transactions: trade deadline %q is not RFC3339: %w", req.TradeDeadline, err)
			}
			deadline = d
		}
		return transactions.SetTradeDeadline{Deadline: deadline, Note: req.Note}, nil
	case string(transactions.KindCapRelief):
		// §13 cap relief: a discretionary commissioner credit.
		amount, err := domain.ParseMoneyMillions(req.AmountMillions)
		if err != nil {
			return nil, fmt.Errorf("cap relief amount: %w", err)
		}
		return transactions.CapRelief{FranchiseID: req.FranchiseID, Amount: amount, Reason: req.Reason}, nil
	case string(transactions.KindRestructure):
		move, err := domain.ParseMoneyMillions(req.MoveMillions)
		if err != nil {
			return nil, fmt.Errorf("restructure move: %w", err)
		}
		return transactions.Restructure{MFLID: req.MFLID, Move: move}, nil
	case string(transactions.KindSign):
		// §6: the salary is the agreed figure. Eligibility, lockout and the floor resolve in the
		// transaction.
		salary, err := domain.ParseMoneyMillions(req.SalaryMillions)
		if err != nil {
			return nil, fmt.Errorf("sign salary: %w", err)
		}
		return transactions.Sign{MFLID: req.MFLID, FranchiseID: req.FranchiseID, Salary: salary, Years: req.Years}, nil
	default:
		return nil, fmt.Errorf("unknown transaction kind %q", req.Kind)
	}
}

// calendarEvent maps the DTO's calendar fields. The payload stays opaque here; the operation's
// own builder parses it when the event fires.
func calendarEvent(req TransactionRequest) transactions.CalendarEvent {
	return transactions.CalendarEvent{
		EventID:     req.EventID,
		EventKind:   req.EventKind,
		ScheduledAt: req.ScheduledAt,
		Payload:     req.Payload,
	}
}

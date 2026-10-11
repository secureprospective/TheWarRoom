package envelope

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/engine/contractacts"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
	"github.com/secureprospective/TheWarRoom/internal/transactions"
)

type ContractFacts struct {
	Player      playerid.PlayerID     `json:"player" ts_type:"string"`
	Name        string                `json:"name"`
	Position    string                `json:"position"`
	Season      int                   `json:"season"`
	Intent      string                `json:"intent"`
	Level       int                   `json:"level"`
	Years       int                   `json:"years"`
	Source      int                   `json:"source"`
	Destination int                   `json:"destination"`
	Move        domain.Money          `json:"move"`
	Baseline    []contractacts.Cell   `json:"baseline"`
	Status      domain.ContractStatus `json:"status"`
	Info        string                `json:"info"`
	Terms       contractacts.Terms    `json:"terms"`
	TeamFigures []ContractFigure      `json:"teamFigures"`
	Reasons     []string              `json:"reasons"`
}

type ContractFigure struct {
	Label  string `json:"label"`
	Before string `json:"before"`
	After  string `json:"after"`
	Change string `json:"change"`
}

func IsContract(intent string) bool {
	return slices.Contains([]string{"rfa.tender", "fa.resign", "contract.tag", "contract.extend", "contract.restructure"}, intent)
}

func ContractWindow(intent string, phase domain.Phase) string {
	if intent == "rfa.tender" || intent == "fa.resign" {
		if phase != domain.PhaseOffseason {
			return "Offseason only: can't be done until the offseason."
		}
		return ""
	}
	kinds := map[string]transactions.Kind{"contract.tag": transactions.KindTag, "contract.extend": transactions.KindExtension, "contract.restructure": transactions.KindRestructure}
	kind, ok := kinds[intent]
	if !ok {
		return "Unsupported contract Act"
	}
	if !slices.Contains(transactions.LegalOps(phase), kind) {
		return "This Act is not allowed in the current season phase."
	}
	return ""
}

type ContractCheck struct{ Reasons []string }

func (c ContractCheck) Evaluate(s Spec, _ snapshot.Snapshot) CheckResult {
	if !IsContract(s.Intent) {
		return CheckResult{Blocked: true, Note: "Contract check: unsupported intent"}
	}
	for _, reason := range c.Reasons {
		if reason != "" && !strings.HasPrefix(reason, "Not checked:") && !strings.HasPrefix(reason, "Using ") {
			return CheckResult{Blocked: true, Note: strings.TrimRight(reason, ".; ") + "."}
		}
	}
	return CheckResult{Note: reasonNote(c.Reasons)}
}

func DraftContract(at time.Time, spec Spec, facts ContractFacts, generate func() string) (Envelope, error) {
	e, err := New(at, spec, generate)
	if err != nil {
		return Envelope{}, fmt.Errorf("draft contract: %w", err)
	}
	next, err := e.Check(at, snapshot.Snapshot{}, ContractCheck{facts.Reasons})
	if err != nil {
		return Envelope{}, fmt.Errorf("check contract: %w", err)
	}
	return next, nil
}

func (e Envelope) ContractPosted(at time.Time) (Envelope, error) {
	if !IsContract(e.spec.Intent) || e.state != HandedOff {
		return Envelope{}, ErrIllegalTransition{From: e.state, Event: AwaitBid}
	}
	return e.move(at, AwaitBid, "Pending: you said you posted this to the commissioner on ProBoards; not yet verified")
}
func (e Envelope) WithdrawContract(at time.Time) (Envelope, error) {
	if !IsContract(e.spec.Intent) {
		return Envelope{}, ErrIllegalTransition{From: e.state, Event: Withdraw}
	}
	note := RemovePlanNote
	for _, entry := range e.audit {
		if entry.To == HandedOff {
			note = "Withdrawn: you said you did not post this contract request"
		}
		if entry.To == BidPending {
			note = "Withdrawn: you said the commissioner did not process this request."
		}
	}
	if e.state == BidPending || e.state == NotVerified {
		note = "Withdrawn: you said the commissioner did not process this request."
	}
	return e.move(at, Withdraw, note)
}

type ContractEvidence struct {
	Player      playerid.PlayerID
	FranchiseID string
	Cells       []contractacts.Cell
	Observed    bool
	Status      domain.ContractStatus
	Info        string
}

type ContractPredicate struct{ Facts ContractFacts }

func (ContractPredicate) Sources() []Source { return []Source{Rosters, Contracts} }
func (p ContractPredicate) Evaluate(s Spec, o Observation) Verdict {
	_, present := rosterStatus(s, o.Rosters.Value)
	if !present {
		return Verdict{Event: Partial, Note: "Not verified: player is no longer on my roster"}
	}
	for _, held := range o.Contracts.Value {
		if held.Player != p.Facts.Player || held.FranchiseID != s.FranchiseID {
			continue
		}
		if !held.Observed {
			return Verdict{Event: Partial, Note: "Not verified: MFL annual salary and expiration do not prove each contract cell; inferred future cells are not evidence"}
		}
		if slices.Equal(held.Cells, p.Facts.Baseline) && held.Status == p.Facts.Status && held.Info == p.Facts.Info {
			return Verdict{Event: NoChange, Note: "No matching contract change observed"}
		}
		if slices.Equal(held.Cells, p.Facts.Terms.Cells) {
			return Verdict{Event: Match, Note: "Landed: the contracts export shows every planned year and salary cell on my player"}
		}
		return Verdict{Event: Partial, Note: "Not verified: planned " + contractCellsText(p.Facts.Terms.Cells) + "; MFL now shows " + contractCellsText(held.Cells)}
	}
	return Verdict{Event: Partial, Note: "Not verified: player's contracts export missing"}
}

func contractCellsText(cells []contractacts.Cell) string {
	parts := make([]string, len(cells))
	for i, c := range cells {
		parts[i] = fmt.Sprintf("%d at %s", c.Season, c.Salary)
	}
	return strings.Join(parts, ", ")
}

func ContractInvalidation(f ContractFacts, held ContractEvidence, windowReason string) string {
	if held.FranchiseID == "" {
		return "Roster: player left my roster"
	}
	if !slices.Equal(f.Baseline, held.Cells) || f.Status != held.Status || f.Info != held.Info {
		return "Contract: player's contract changed since this plan was drafted"
	}
	if windowReason != "" {
		return windowReason
	}
	return ""
}

package contracts

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	mflwhatif "github.com/secureprospective/TheWarRoom/internal/mflref/whatif"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/envelope"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

// Release is the twin of the root RawReleaseRequest: the same fields in the same order.
type Release struct {
	Mine      string `json:"mine"`
	Action    string `json:"action"`
	Move      string `json:"move"`
	ID        string `json:"id"`
	Confirmed bool   `json:"confirmed"`
}

func (r Release) Validate() error {
	if err := (snapshot.RawContractScope{Mine: r.Mine, Sort: "name"}).Validate(); err != nil {
		return fmt.Errorf("release scope: %w", err)
	}
	if r.Mine == "" || !slices.Contains([]string{"read", "draft", "handoff", "rebase", "withdraw"}, r.Action) {
		return fmt.Errorf("release: franchise and known action required")
	}
	if r.Action == "draft" && r.Move == "" {
		return fmt.Errorf("release: what-if move required")
	}
	if (r.Action == "handoff" || r.Action == "rebase" || r.Action == "withdraw") && r.ID == "" {
		return fmt.Errorf("release: plan id required")
	}
	return nil
}

// ReleasePlan is a cut or buyout plan as the owner reads it; it is main.ReleasePlanView.
type ReleasePlan struct {
	ID            string           `json:"id"`
	Player        string           `json:"player"`
	Label         string           `json:"label"`
	Charge        string           `json:"charge"`
	Confirmation  string           `json:"confirmation"`
	Post          string           `json:"post"`
	Channel       string           `json:"channel"`
	Receipt       envelope.Receipt `json:"receipt"`
	CanHandOff    bool             `json:"canHandOff"`
	CanRebase     bool             `json:"canRebase"`
	RebaseLabel   string           `json:"rebaseLabel"`
	WithdrawLabel string           `json:"withdrawLabel"`
	Cut           bool             `json:"cut"`
}

// Plan words a release plan from its receipt and the facts it was drafted on.
func Plan(r envelope.Receipt, f envelope.ReleaseFacts) ReleasePlan {
	cut := r.Spec.Intent == envelope.CutIntent
	verb, channel := "Buy out ", "Post this to the commissioner on ProBoards."
	if cut {
		verb, channel = "Cut ", "Release on MFL; post player, contract and cap hit on ProBoards (§8)."
	}
	cost := snapshot.ContractAmountOf(f.Charge).Text
	return ReleasePlan{ID: r.CorrelationID, Player: f.Name, Label: verb + f.Name, Charge: cost,
		Confirmation: fmt.Sprintf("This releases %s; %s dead cap this season. It cannot be undone.", f.Name, cost),
		Post:         fmt.Sprintf("%s\n%s: annual salary %s; contract expires %d; %d paid years after %d.\nPlanned dead cap this season: %s. Please process this %s request.\nCannot %s this player for the rest of %s.", verb+f.Name, f.Name, snapshot.ContractAmountOf(f.Salary).Text, f.Expiration, max(0, f.Expiration-f.Season), f.Season, cost, strings.TrimSpace(verb), lockVerb(cut), lockSeason(cut)),
		Channel:      channel, Receipt: r, CanHandOff: r.State == envelope.Ready, CanRebase: r.State == envelope.Stale, RebaseLabel: "Reprice from today's MFL rosters", WithdrawLabel: withdrawLabel(r.State, cut), Cut: cut}
}

func withdrawLabel(s envelope.State, cut bool) string {
	if s == envelope.Draft || s == envelope.Ready || s == envelope.Blocked || s == envelope.Stale {
		return "Remove plan"
	}
	if s == envelope.HandedOff || s == envelope.NotYetDone || s == envelope.NotVerified {
		if cut {
			return "I didn't drop him"
		}
		return "I didn't post it"
	}
	return ""
}

func lockVerb(cut bool) string {
	if cut {
		return "re-sign"
	}
	return "bid on"
}

func lockSeason(cut bool) string {
	if cut {
		return "this season"
	}
	return "this season; eligible again the following offseason"
}

// WithdrawNote is the audit note of a withdrawn release: the owner's own words once it was handed off.
func WithdrawNote(r envelope.Receipt) string {
	note := envelope.RemovePlanNote
	for _, entry := range r.Audit {
		if entry.To != envelope.HandedOff {
			continue
		}
		note = "You said you did not post it, so the plan is withdrawn."
		if r.Spec.Intent == envelope.CutIntent {
			note = "You said you did not drop him, so the plan is withdrawn."
		}
	}
	return note
}

// ReleaseFacts is what a release plan is drafted on: the what-if move, the player's held contract in the
// what-if's baseline, its stamps and the franchise's adjustments.
func ReleaseFacts(s mflwhatif.WhatIfSession, m mflwhatif.WhatIfMove) (envelope.ReleaseFacts, error) {
	pid, err := playerid.New(m.Player)
	if err != nil {
		return envelope.ReleaseFacts{}, fmt.Errorf("release player: %w", err)
	}
	p, found := Player(s.Snapshot, s.Mine, m.Player)
	if !found {
		return envelope.ReleaseFacts{}, fmt.Errorf("release: player not in what-if baseline")
	}
	adjustments, err := Adjustments(s.Snapshot, s.Mine)
	if err != nil {
		return envelope.ReleaseFacts{}, err
	}
	return envelope.ReleaseFacts{Player: pid, Name: m.Name, Season: s.Snapshot.Season, Charge: m.Charge, Salary: p.Salary, Expiration: p.ContractYear, ContractInfo: p.ContractInfo,
		RostersStamp: s.Stamps.Rosters, ContractStamp: s.Stamps.Contract, DeadCapStamp: s.Stamps.DeadCap, Adjustments: adjustments}, nil
}

// Player is the franchise's held contract for a player, if the franchise holds him.
func Player(m state.MirrorSnapshot, mine, player string) (state.MirrorPlayer, bool) {
	for _, p := range m.Players {
		if p.MFLID == player && p.FranchiseID == mine {
			return p, true
		}
	}
	return state.MirrorPlayer{}, false
}

// Adjustments is the franchise's held salary adjustments as JSON, to tell later whether they changed.
func Adjustments(m state.MirrorSnapshot, mine string) (string, error) {
	rows := []state.Adjustment{}
	for _, r := range m.Adjustments {
		if r.FranchiseID == mine {
			rows = append(rows, r)
		}
	}
	body, err := json.Marshal(rows)
	if err != nil {
		return "", fmt.Errorf("release adjustments: %w", err)
	}
	return string(body), nil
}

// StateLabel names an open plan's state for the what-if.
func StateLabel(s envelope.State) string {
	labels := map[envelope.State]string{envelope.Ready: "Ready", envelope.Blocked: "Blocked", envelope.Stale: "Stale", envelope.HandedOff: "Handed off", envelope.NotYetDone: "Not yet done", envelope.NotVerified: "Not verified", envelope.Draft: "Draft"}
	return labels[s]
}

// ObservedCharge is the one MFL adjustment that names the player and dead cap, if exactly one does.
func ObservedCharge(f envelope.ReleaseFacts, mine string, m state.MirrorSnapshot) (domain.Money, bool) {
	var amount domain.Money
	matches := 0
	for _, adj := range m.Adjustments {
		if adj.FranchiseID != mine {
			continue
		}
		words := strings.FieldsFunc(strings.ToLower(adj.Description), func(r rune) bool { return r < '0' || r > '9' })
		identified := false
		for _, word := range words {
			identified = identified || word == f.Player.String()
		}
		dead := strings.Contains(strings.ToLower(adj.Description), "dead cap")
		if identified && dead {
			amount = adj.Amount
			matches++
		}
	}
	return amount, matches == 1
}

// ChargeNote compares a landed release's planned dead cap with the charge MFL shows for it.
func ChargeNote(f envelope.ReleaseFacts, mine string, m state.MirrorSnapshot) string {
	amount, seen := ObservedCharge(f, mine, m)
	note := "Landed: cannot see a player-linked dead-cap charge in MFL salary adjustments."
	if seen {
		note = snapshot.ContractAmountOf(f.Charge).Text + " planned vs " + snapshot.ContractAmountOf(amount).Text + " now (MFL player-labelled dead cap)"
		if amount == f.Charge {
			note = "Landed: planned dead cap matches " + snapshot.ContractAmountOf(amount).Text + " now"
		}
	}
	return note
}

// NeedsValidation says whether a plan in this state is checked against the mirror again.
func NeedsValidation(s envelope.State) bool {
	return s == envelope.Ready || s == envelope.HandedOff || s == envelope.NotYetDone || s == envelope.NotVerified
}

// ContractChanged says whether the held contract differs from the one the plan was drafted on.
func ContractChanged(p state.MirrorPlayer, f envelope.ReleaseFacts) bool {
	return p.Salary != f.Salary || p.ContractYear != f.Expiration || p.ContractInfo != f.ContractInfo
}

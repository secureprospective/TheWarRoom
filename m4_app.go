package main

import (
	"context"
	"sort"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/normalize"
	"github.com/secureprospective/TheWarRoom/internal/transactions"
)

// m4Timeout bounds the M4 reads: the first call after a cold start may fetch the players
// directory.
const m4Timeout = 120 * time.Second

// M4Player is one roster player with identity resolved; money is float millions for display
// only.
type M4Player struct {
	MFLID        string  `json:"mflID"`
	Name         string  `json:"name"`
	Position     string  `json:"position"`
	RosterStatus string  `json:"rosterStatus"`
	Salary       float64 `json:"salary"`
	CapSalary    float64 `json:"capSalary"`
}

// RosterResult is one franchise's named roster and cap. Warning reports a names outage, so the
// UI shows ids with a reason instead of failing.
type RosterResult struct {
	OK          bool       `json:"ok"`
	FranchiseID string     `json:"franchiseID"`
	CapUsed     float64    `json:"capUsed"`
	Players     []M4Player `json:"players"`
	Warning     string     `json:"warning"`
	Detail      string     `json:"detail"`
}

// GetRoster reads one franchise's roster and joins player names and positions.
func (a *App) GetRoster(franchiseID string) RosterResult {
	if a.startupErr != nil {
		return RosterResult{Detail: a.startupErr.Error()}
	}
	if a.state == nil {
		return RosterResult{Detail: "state store not initialized"}
	}
	if err := a.state.Err(); err != nil {
		return RosterResult{FranchiseID: franchiseID, Detail: "state is stale after a failed reload: " + err.Error()}
	}
	fs, ok := a.state.Reader().FranchiseState(franchiseID)
	if !ok {
		return RosterResult{FranchiseID: franchiseID, Detail: "no such franchise (or it holds no players)"}
	}
	ctx, cancel := context.WithTimeout(a.ctx, m4Timeout)
	defer cancel()
	lk, warning := a.resolveDirectory(ctx)

	players := make([]M4Player, len(fs.Players))
	for i, p := range fs.Players {
		m := M4Player{
			MFLID:        p.MFLID,
			RosterStatus: string(p.RosterStatus),
			Salary:       p.Salary.Millions(),
			CapSalary:    p.CapSalary.Millions(),
		}
		if f, ok := lk.Facts(p.MFLID); ok {
			m.Name, m.Position = f.Name, string(f.Position)
		} else {
			m.Name = "(unknown id " + p.MFLID + ")"
		}
		players[i] = m
	}
	return RosterResult{OK: true, FranchiseID: franchiseID, CapUsed: fs.CapUsed.Millions(), Players: players, Warning: warning}
}

// FreeAgentPoolResult is the signable pool (latest status FREE_AGENT, on no roster). Salary
// and status come from the signing terms.
type FreeAgentPoolResult struct {
	OK      bool       `json:"ok"`
	Players []M4Player `json:"players"`
	Warning string     `json:"warning"`
	Detail  string     `json:"detail"`
}

// GetFreeAgentPool reads the free-agent pool with player identity joined.
func (a *App) GetFreeAgentPool() FreeAgentPoolResult {
	if a.startupErr != nil {
		return FreeAgentPoolResult{Detail: a.startupErr.Error()}
	}
	if a.state == nil {
		return FreeAgentPoolResult{Detail: "state store not initialized"}
	}
	if err := a.state.Err(); err != nil {
		return FreeAgentPoolResult{Detail: "state is stale after a failed reload: " + err.Error()}
	}
	ctx, cancel := context.WithTimeout(a.ctx, m4Timeout)
	defer cancel()
	ids, err := a.state.FreeAgents(ctx)
	if err != nil {
		return FreeAgentPoolResult{Detail: err.Error()}
	}
	lk, warning := a.resolveDirectory(ctx)

	players := make([]M4Player, 0, len(ids))
	for _, id := range ids {
		m := M4Player{MFLID: id, RosterStatus: "FREE_AGENT"}
		if f, ok := lk.Facts(id); ok {
			m.Name, m.Position = f.Name, string(f.Position)
		} else {
			m.Name = "(unknown id " + id + ")"
		}
		players = append(players, m)
	}
	return FreeAgentPoolResult{OK: true, Players: players, Warning: warning}
}

// M4Franchise is one entry in the franchise rail. Name is empty when MFL has none on file;
// the UI then shows the id.
type M4Franchise struct {
	FranchiseID string `json:"franchiseID"`
	Name        string `json:"name"`
	PlayerCount int    `json:"playerCount"`
}

// FranchisesResult is the franchise directory behind the left rail.
type FranchisesResult struct {
	OK         bool          `json:"ok"`
	Franchises []M4Franchise `json:"franchises"`
	Detail     string        `json:"detail"`
}

// GetFranchises lists the franchises with display names and roster sizes.
func (a *App) GetFranchises() FranchisesResult {
	if a.startupErr != nil {
		return FranchisesResult{Detail: a.startupErr.Error()}
	}
	if a.state == nil {
		return FranchisesResult{Detail: "state store not initialized"}
	}
	if err := a.state.Err(); err != nil {
		return FranchisesResult{Detail: "state is stale after a failed reload: " + err.Error()}
	}
	names := map[string]string{}
	if a.rulebook != nil {
		names = a.rulebook.FranchiseNames()
	}
	r := a.state.Reader()
	ids := r.Franchises()
	sort.Strings(ids) // ids are zero-padded, so lexical order is numeric
	out := make([]M4Franchise, 0, len(ids))
	for _, id := range ids {
		count := 0
		if fs, ok := r.FranchiseState(id); ok {
			count = len(fs.Players)
		}
		out = append(out, M4Franchise{FranchiseID: id, Name: names[id], PlayerCount: count})
	}
	return FranchisesResult{OK: true, Franchises: out}
}

// LegalOpsResult is the op kinds that are legal in the current season phase.
type LegalOpsResult struct {
	OK     bool     `json:"ok"`
	Phase  string   `json:"phase"`
	Kinds  []string `json:"kinds"`
	Detail string   `json:"detail"`
}

// GetLegalOps returns the phase-legal op kinds from transactions.LegalOps, so the UI never
// re-encodes the policy. It is a coarse filter: preview and commit still run the authoritative
// check.
func (a *App) GetLegalOps() LegalOpsResult {
	if a.startupErr != nil {
		return LegalOpsResult{Detail: a.startupErr.Error()}
	}
	if a.state == nil {
		return LegalOpsResult{Detail: "state store not initialized"}
	}
	ctx, cancel := context.WithTimeout(a.ctx, 10*time.Second)
	defer cancel()
	ph, err := a.state.CurrentPhase(ctx)
	if err != nil {
		return LegalOpsResult{Detail: err.Error()}
	}
	kinds := transactions.LegalOps(ph)
	out := make([]string, len(kinds))
	for i, k := range kinds {
		out[i] = string(k)
	}
	return LegalOpsResult{OK: true, Phase: string(ph), Kinds: out}
}

// PreviewTransaction dry-runs a request exactly as ExecuteTransaction would, then rolls back:
// the confirm screen learns whether it would commit, or the reason it would not. Nothing is
// stored, and the preview result is never fed back as input.
func (a *App) PreviewTransaction(req TransactionRequest) TransactionResult {
	if a.startupErr != nil {
		return TransactionResult{Detail: a.startupErr.Error()}
	}
	if a.coordinator == nil {
		return TransactionResult{Detail: "transaction coordinator not initialized"}
	}
	ctx, cancel := context.WithTimeout(a.ctx, 30*time.Second)
	defer cancel()

	// TAG and EXTENSION resolve their price or floor server-side from the players directory, so
	// they preview through their own verbs. Only ids and counts cross the wire, never money.
	switch req.Kind {
	case string(transactions.KindTag):
		dir, derr := a.directory(ctx)
		if derr != nil {
			return TransactionResult{Kind: req.Kind, Detail: "resolve players DB for the §9 tag price: " + derr.Error()}
		}
		rec, terr := a.coordinator.PreviewTag(ctx, req.MFLID, dir)
		return a.receiptResult(req.Kind, rec, terr)
	case string(transactions.KindExtension):
		dir, derr := a.directory(ctx)
		if derr != nil {
			return TransactionResult{Kind: req.Kind, Detail: "resolve players DB for the §10 position floor: " + derr.Error()}
		}
		rec, terr := a.coordinator.PreviewExtension(ctx, req.MFLID, req.AddedYears, dir)
		return a.receiptResult(req.Kind, rec, terr)
	}

	txn, err := buildRequest(req)
	if err != nil {
		return TransactionResult{Detail: err.Error()}
	}
	if sign, ok := txn.(transactions.Sign); ok {
		dir, derr := a.directory(ctx)
		if derr != nil {
			return TransactionResult{Kind: req.Kind, Detail: "resolve players DB for the §6 min-salary floor: " + derr.Error()}
		}
		rec, terr := a.coordinator.PreviewSign(ctx, sign, dir)
		return a.receiptResult(req.Kind, rec, terr)
	}
	rec, terr := a.coordinator.Preview(ctx, txn)
	return a.receiptResult(req.Kind, rec, terr)
}

// receiptResult maps a coordinator outcome onto TransactionResult: a rejection carries the
// reason; a success carries the receipt and the previewed cap impact.
func (a *App) receiptResult(kind string, rec transactions.Receipt, err error) TransactionResult {
	if err != nil {
		return TransactionResult{Kind: kind, Detail: err.Error()}
	}
	return TransactionResult{
		OK:              true,
		Kind:            string(rec.Kind),
		PlayersAffected: rec.PlayersAffected,
		At:              rec.At.Format(time.RFC3339),
		CapDeltas:       a.capDeltaDTOs(rec.CapDeltas),
	}
}

// capDeltaDTOs formats the signed cap deltas with franchise names. Never nil.
func (a *App) capDeltaDTOs(deltas []transactions.CapDelta) []CapDeltaDTO {
	names := map[string]string{}
	if a.rulebook != nil {
		names = a.rulebook.FranchiseNames()
	}
	out := make([]CapDeltaDTO, 0, len(deltas))
	for _, d := range deltas {
		amount := d.Cents.String()
		if d.Cents > 0 {
			amount = "+" + amount // Money.String already prefixes − on a credit
		}
		out = append(out, CapDeltaDTO{
			FranchiseID:   d.FranchiseID,
			FranchiseName: names[d.FranchiseID],
			Amount:        amount,
			Cents:         d.Cents.Cents(),
			Reason:        d.Reason,
		})
	}
	return out
}

// resolveDirectory returns the players directory, or an empty one plus a warning on failure, so
// a names outage shows ids instead of blanking a good read.
func (a *App) resolveDirectory(ctx context.Context) (normalize.Lookup, string) {
	lk, err := a.directory(ctx)
	if err != nil {
		return normalize.Lookup{}, "player names unavailable (players-DB fetch failed: " + err.Error() + ") — roster/cap are complete"
	}
	return lk, ""
}

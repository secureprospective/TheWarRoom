package main

import (
	"context"
	"sort"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/normalize"
	"github.com/secureprospective/TheWarRoom/internal/transactions"
)

// m4_app.go holds the M4 Transaction-UI read surface: name-enriched franchise rosters,
// a name-enriched free-agent pool, and the franchise directory. These back the real
// operator workspace (a franchise rail → named roster → per-player action panel),
// replacing the raw-mflID dev panel over time (D8 phased cutover). They are READ-ONLY
// (never the writer) and resolve player identity server-side so no mflID is ever shown
// to a human (design decision D2). Player names/positions come from the players-DB
// Lookup; a Lookup failure DEGRADES (ids + a warning) rather than hiding persisted
// roster state — the GetRankings idiom.
//
// FRANCHISE NAMES are deferred: the league export parsed today (internal/ingestion/league)
// does not carry franchise names, so the rail shows franchise IDs until a follow-up extends
// that fetcher. Player identity — the dominant win — is fully resolved here.

// m4Timeout bounds the M4 read IPC calls. On a warm DB the Lookup is cached and these
// return in microseconds; on the first call after a cold start the lazy players-DB fetch
// can run, so the budget matches the M1 reads rather than the 3 s ping.
const m4Timeout = 120 * time.Second

// M4Player is one roster player as the operator workspace renders it: identity resolved,
// money at the display edge (float millions — no money math happens frontend-side).
type M4Player struct {
	MFLID        string  `json:"mflID"`
	Name         string  `json:"name"`
	Position     string  `json:"position"`
	RosterStatus string  `json:"rosterStatus"`
	Salary       float64 `json:"salary"`
	CapSalary    float64 `json:"capSalary"`
}

// RosterResult is one franchise's named roster + derived cap. Warning carries a non-fatal
// degradation (names unavailable) so the UI can show ids with an explanation rather than fail.
type RosterResult struct {
	OK          bool       `json:"ok"`
	FranchiseID string     `json:"franchiseID"`
	CapUsed     float64    `json:"capUsed"`
	Players     []M4Player `json:"players"`
	Warning     string     `json:"warning"`
	Detail      string     `json:"detail"`
}

// GetRoster reads one franchise's current roster through the read-only surface and joins
// player identity (name/position) from the players-DB Lookup. It powers the center roster
// table of the transaction workspace.
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

// FreeAgentPoolResult is the name-enriched signable pool (latest status FREE_AGENT, off every
// roster). Free agents carry no salary/status — those are set by the signing terms.
type FreeAgentPoolResult struct {
	OK      bool       `json:"ok"`
	Players []M4Player `json:"players"`
	Warning string     `json:"warning"`
	Detail  string     `json:"detail"`
}

// GetFreeAgentPool reads the free-agency pool and joins player identity. It powers the
// "Free Agents" view the operator signs from.
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

// M4Franchise is one entry in the franchise rail: its id, a display name (empty until the
// franchise-name follow-up lands — the UI falls back to the id), and its roster size.
type M4Franchise struct {
	FranchiseID string `json:"franchiseID"`
	Name        string `json:"name"`
	PlayerCount int    `json:"playerCount"`
}

// FranchisesResult is the franchise directory backing the left rail.
type FranchisesResult struct {
	OK         bool          `json:"ok"`
	Franchises []M4Franchise `json:"franchises"`
	Detail     string        `json:"detail"`
}

// GetFranchises lists the league's franchises (id + roster size) for the rail. Franchise
// display names are deferred (see the file header); the id is the stable label until then.
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
	r := a.state.Reader()
	ids := r.Franchises()
	sort.Strings(ids) // stable rail order (franchise ids are zero-padded, so lexical == numeric)
	out := make([]M4Franchise, 0, len(ids))
	for _, id := range ids {
		count := 0
		if fs, ok := r.FranchiseState(id); ok {
			count = len(fs.Players)
		}
		out = append(out, M4Franchise{FranchiseID: id, PlayerCount: count})
	}
	return FranchisesResult{OK: true, Franchises: out}
}

// PreviewTransaction dry-runs a request through the Coordinator (design D5): it validates,
// phase-gates, and applies EXACTLY as ExecuteTransaction would, then rolls back — so the
// staged-confirm UI learns whether the move would commit (OK=true) or the authoritative reason
// it would be rejected (Detail), with nothing persisted. The confirm step then calls
// ExecuteTransaction, which re-sends the SAME intent and commits authoritatively (D4) — the
// preview result is never fed back in as input. SIGN resolves its §6 min-salary-floor draft-year
// input via the players-DB directory, mirroring executeSign. (TAG/§10 previews, which need their
// own directory verbs, are a later-slice follow-up; they are not slice-1 ops.)
func (a *App) PreviewTransaction(req TransactionRequest) TransactionResult {
	if a.startupErr != nil {
		return TransactionResult{Detail: a.startupErr.Error()}
	}
	if a.coordinator == nil {
		return TransactionResult{Detail: "transaction coordinator not initialized"}
	}
	ctx, cancel := context.WithTimeout(a.ctx, 30*time.Second)
	defer cancel()

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
		if terr != nil {
			return TransactionResult{Kind: req.Kind, Detail: terr.Error()}
		}
		return TransactionResult{OK: true, Kind: string(rec.Kind), PlayersAffected: rec.PlayersAffected, At: rec.At.Format(time.RFC3339)}
	}
	rec, err := a.coordinator.Preview(ctx, txn)
	if err != nil {
		return TransactionResult{Kind: req.Kind, Detail: err.Error()}
	}
	return TransactionResult{OK: true, Kind: string(rec.Kind), PlayersAffected: rec.PlayersAffected, At: rec.At.Format(time.RFC3339)}
}

// resolveDirectory returns the cached players-DB Lookup, DEGRADING on failure: a nil-facts
// Lookup plus a human warning, so a names-unavailable state (e.g. an MFL outage on a warm DB)
// surfaces ids with an explanation instead of blanking a perfectly good roster read. Mirrors
// GetRankings' display-only-join degradation.
func (a *App) resolveDirectory(ctx context.Context) (normalize.Lookup, string) {
	lk, err := a.directory(ctx)
	if err != nil {
		return normalize.Lookup{}, "player names unavailable (players-DB fetch failed: " + err.Error() + ") — roster/cap are complete"
	}
	return lk, ""
}

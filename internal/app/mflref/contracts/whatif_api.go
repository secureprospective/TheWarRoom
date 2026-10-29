package contracts

import (
	"context"
	"fmt"
	"slices"
	"time"

	mflwhatif "github.com/secureprospective/TheWarRoom/internal/mflref/whatif"

	"github.com/secureprospective/TheWarRoom/internal/app/report"
	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/envelope"
	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
	"github.com/secureprospective/TheWarRoom/internal/transactions"
)

type RawWhatIfRequest struct {
	Mine   string `json:"mine"`
	Action string `json:"action"`
	Player string `json:"player"`
	Move   string `json:"move"`
}

func (r RawWhatIfRequest) Validate() error {
	return WhatIf(r).Validate()
}

// TargetWhatIf reads or changes only the persisted simulation; never a plan or MFL request.
func (a *API) TargetWhatIf(raw RawWhatIfRequest) WhatIfReading {
	if err := raw.Validate(); err != nil {
		return a.whatIfReading(WhatIfView{}, mflwhatif.WhatIfSession{}, err)
	}
	if err := a.host.Ready(); err != nil {
		return a.whatIfReading(WhatIfView{}, mflwhatif.WhatIfSession{}, err)
	}
	if err := a.host.RequireOwnFranchise(raw.Mine); err != nil {
		return a.whatIfReading(WhatIfView{}, mflwhatif.WhatIfSession{}, err)
	}
	var out WhatIfReading
	a.host.SerializeWhatIf(func() {
		ctx, cancel := context.WithTimeout(a.host.ContractsContext(), timeout)
		defer cancel()
		out = a.whatIf(ctx, raw)
	})
	return out
}

// whatIf runs under the what-if and refresh serializers.
func (a *API) whatIf(ctx context.Context, raw RawWhatIfRequest) WhatIfReading {
	st := a.stores()
	session, store, err := OpenWhatIf(ctx, st, raw)
	if err != nil {
		return a.whatIfReading(WhatIfView{}, session, err)
	}
	stamps, err := st.League.Stamps(ctx)
	if err != nil {
		return a.whatIfReading(WhatIfView{}, session, fmt.Errorf("what-if current stamps: %w", err))
	}
	stale := stamps.NewerThan(session.Stamps)
	if stale && len(session.Moves) == 0 {
		session, store, err = OpenWhatIf(ctx, st, RawWhatIfRequest{Mine: raw.Mine, Action: "reset"})
		if err != nil {
			return a.whatIfReading(WhatIfView{}, session, err)
		}
		stale = false
	}
	var notice string
	if raw.Action != "read" && raw.Action != "reset" {
		if stale {
			notice = "MFL changed since this what-if started. Nothing changed; Reset first."
		} else {
			next, nextStore, changeErr := a.changeWhatIf(ctx, session, raw)
			if changeErr != nil {
				notice = changeErr.Error()
			} else {
				session, store = next, nextStore
			}
		}
	}
	view, err := a.whatIfViewThrough(ctx, session, store, stale, session.Snapshot.Season)
	view.Notice = notice
	return a.whatIfReading(view, session, err)
}

func (a *API) whatIfReading(view WhatIfView, s mflwhatif.WhatIfSession, err error) WhatIfReading {
	r := contractReading(a, view, s.Stamps.Rosters, "Your what-if, started from MFL rosters as of "+stampText(s.Stamps.Rosters), "Cut = 35% of each paid cell after this season, or 50% for a known restructured contract; claimed and final-year cuts cost $0. Sum exact cents, round half-up once, then snap once to $10,000. Buyout = mean of remaining paid cells times 60% / 75% / 90% for 2 / 3 / 4 years, then one snap; at most two per season. Future salaries carry today’s MFL annual salary through contract expiration; MFL does not report those years separately. The exports do not establish restructure history: seeded contracts assume 35%, not a proven unrestructured verdict. This season's used begins at the mirror total; the engine's ledger delta is applied without changing the original MFL adjustments. MFL adjustments remain literal adjustments, not proven dead cap. The whole modelled charge lands in the cut year; future seasons have no modelled dead cap. Buyouts here model offseason terms without asserting that today's window is open; offseason start is not held. No submission or plan is created.", len(s.Moves), err)
	r.Baseline = s.Stamps.Rosters
	r.HowCounted += fmt.Sprintf(" Separate contract-cell update time: %s. Separate dead-cap update time: %s.", stampText(s.Stamps.Contract), stampText(s.Stamps.DeadCap))
	r.Method = "Coordinator handlers on a generation-scoped whatif.db ledger; integer cents"
	r.Provenance.Freshness.Note = "Baseline held until Reset; survives restart. The mirror is never written."
	return WhatIfReading(r)
}
func stampText(s string) string {
	if s == "" {
		return "not held by this mirror"
	}
	at, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		return "date unavailable"
	}
	return report.Time(at, time.Local)
}

// OpenWhatIf loads a franchise's persisted what-if, or starts it from the mirror on first use and on
// reset. The caller holds the what-if and refresh serializers.
func OpenWhatIf(ctx context.Context, st Stores, raw RawWhatIfRequest) (mflwhatif.WhatIfSession, *state.Store, error) {
	sessions := mflwhatif.NewWhatIfSessions(st.WhatifPools)
	session, found, err := sessions.Load(ctx, raw.Mine)
	if err != nil {
		return session, nil, fmt.Errorf("what-if session: %w", err)
	}
	if !found || raw.Action == "reset" {
		session, err = MirrorWhatIf(ctx, st, raw.Mine)
		if err != nil {
			return session, nil, err
		}
		return replayWhatIf(ctx, st.WhatifPools, session)
	}
	store, err := LoadWhatIf(ctx, st.WhatifPools, session)
	return session, store, err
}

// LoadWhatIf seeds a generation-scoped what-if ledger from the session's pinned baseline.
func LoadWhatIf(ctx context.Context, pools *db.Pools, s mflwhatif.WhatIfSession) (*state.Store, error) {
	store := state.New(pools, s.Generation, s.Snapshot.Season, s)
	if err := store.Initialize(ctx, state.SnapshotSource{Snapshot: s.Snapshot}); err != nil {
		return nil, fmt.Errorf("what-if ledger: %w", err)
	}
	if err := store.Err(); err != nil {
		return nil, fmt.Errorf("what-if ledger: %w", err)
	}
	return store, nil
}

func (a *API) changeWhatIf(ctx context.Context, s mflwhatif.WhatIfSession, raw RawWhatIfRequest) (mflwhatif.WhatIfSession, *state.Store, error) {
	previous := s
	s.Moves = slices.Clone(s.Moves)
	if raw.Action == "undo" {
		index := slices.IndexFunc(s.Moves, func(m mflwhatif.WhatIfMove) bool { return m.ID == raw.Move })
		if index < 0 {
			return previous, nil, fmt.Errorf("what-if: move no longer exists")
		}
		s.Moves = slices.Delete(s.Moves, index, index+1)
	} else {
		if slices.ContainsFunc(s.Moves, func(m mflwhatif.WhatIfMove) bool { return m.Player == raw.Player }) {
			return previous, nil, fmt.Errorf("what-if: player already removed; Undo first")
		}
		kind := string(transactions.KindWaiver)
		if raw.Action == "buyout" {
			kind = string(transactions.KindBuyout)
		}
		name, found := "", false
		for _, p := range s.Snapshot.Players {
			if p.MFLID == raw.Player && p.FranchiseID == s.Mine {
				found = true
				name = "Player " + p.MFLID
			}
		}
		if !found {
			return previous, nil, fmt.Errorf("what-if: only your baseline roster can be modelled")
		}
		directory, _ := a.host.ContractsDirectory()
		for _, p := range directory.Players.Value {
			if p.ID.String() == raw.Player && p.Name != "" {
				name = p.Name
			}
		}
		s.Moves = append(s.Moves, mflwhatif.WhatIfMove{ID: fmt.Sprint(time.Now().UnixNano()), Kind: kind, Player: raw.Player, Name: name})
	}
	next, store, err := replayWhatIf(ctx, a.stores().WhatifPools, s)
	if err != nil {
		return previous, nil, err
	}
	return next, store, nil
}

// ScratchWhatIf rebuilds a franchise's what-if in a throwaway scratch pool for a trade projection.
// It reads the saved session (or starts one from the mirror) but writes nothing to whatif.db: the
// ledger and its moves are replayed into scratch, and a new session is not published.
func ScratchWhatIf(ctx context.Context, st Stores, scratch *db.Pools, mine string) (mflwhatif.WhatIfSession, *state.Store, error) {
	session, found, err := mflwhatif.NewWhatIfSessions(st.WhatifPools).Load(ctx, mine)
	if err != nil {
		return session, nil, fmt.Errorf("what-if session: %w", err)
	}
	if !found {
		if session, err = MirrorWhatIf(ctx, st, mine); err != nil {
			return session, nil, err
		}
		return replayLedger(ctx, scratch, session)
	}
	// The saved session already holds each move's charge; replay only rebuilds its ledger.
	replay := session
	replay.Moves = slices.Clone(session.Moves)
	_, store, err := replayLedger(ctx, scratch, replay)
	return session, store, err
}

func replayWhatIf(ctx context.Context, pools *db.Pools, s mflwhatif.WhatIfSession) (mflwhatif.WhatIfSession, *state.Store, error) {
	s, store, err := replayLedger(ctx, pools, s)
	if err != nil {
		return s, nil, err
	}
	if err := mflwhatif.NewWhatIfSessions(pools).Save(ctx, s); err != nil {
		return s, nil, fmt.Errorf("what-if publish: %w", err)
	}
	return s, store, nil
}

// replayLedger seeds a new generation from the session's baseline and executes its moves in order.
func replayLedger(ctx context.Context, pools *db.Pools, s mflwhatif.WhatIfSession) (mflwhatif.WhatIfSession, *state.Store, error) {
	s.Generation = "whatif:" + s.Mine + ":" + fmt.Sprint(time.Now().UnixNano())
	store, err := LoadWhatIf(ctx, pools, s)
	if err != nil {
		return s, nil, err
	}
	coord, err := transactions.New(store.Writer(), nil, nil)
	if err != nil {
		return s, nil, fmt.Errorf("what-if coordinator: %w", err)
	}
	s.SeedUsed, _ = store.CapUsed(s.Mine)
	for i, m := range s.Moves {
		player, ok := store.Player(m.Player)
		if !ok || player.FranchiseID != s.Mine {
			return s, nil, fmt.Errorf("what-if replay: player not on your roster")
		}
		req, err := buildRequest(TransactionRequest{Kind: m.Kind, MFLID: m.Player})
		if err != nil {
			return s, nil, fmt.Errorf("what-if request: %w", err)
		}
		receipt, err := coord.Execute(ctx, req)
		if err != nil {
			return s, nil, fmt.Errorf("what-if replay: %w", err)
		}
		if len(receipt.CapDeltas) > 1 {
			return s, nil, fmt.Errorf("what-if replay: missing charge")
		}
		s.Moves[i].Charge = 0
		if len(receipt.CapDeltas) == 1 {
			s.Moves[i].Charge = receipt.CapDeltas[0].Cents
		}
		s.Moves[i].Explanation = Rule(m.Kind, player, s.Snapshot.Season)
	}
	return s, store, nil
}

func (a *API) whatIfViewThrough(ctx context.Context, s mflwhatif.WhatIfSession, store *state.Store, stale bool, horizon int) (WhatIfView, error) {
	var ledger Ledger
	if store != nil {
		ledger = store
	}
	v, err := DescribeWhatIf(ctx, s, ledger, stale, horizon, func(moves []WhatIfMove) error {
		return whatIfPlanLabels(ctx, a.stores(), s.Mine, moves)
	})
	return wireWhatIf(v), err
}

// wireWhatIf keeps every list of the what-if a list, never null, as the root converter did.
func wireWhatIf(v WhatIfView) WhatIfView {
	out := WhatIfView{Stale: v.Stale, Banner: v.Banner, Empty: v.Empty, Moves: make([]WhatIfMove, len(v.Moves)),
		Figures: wireWhatIfFigures(v.Figures), Choices: make([]WhatIfChoice, len(v.Choices)), Notice: v.Notice}
	copy(out.Moves, v.Moves)
	copy(out.Choices, v.Choices)
	return out
}

// wireWhatIfFigures keeps the figures a list, never null.
func wireWhatIfFigures(figures []WhatIfFigure) []WhatIfFigure {
	out := make([]WhatIfFigure, len(figures))
	copy(out, figures)
	return out
}

func whatIfPlanLabels(ctx context.Context, st Stores, mine string, moves []WhatIfMove) error {
	if st.Moves != nil {
		receipts, err := st.Moves.List(ctx, ingestion.LeagueID, mine)
		if err != nil {
			return fmt.Errorf("what-if plans: %w", err)
		}
		for i, move := range moves {
			for _, receipt := range receipts {
				if envelope.IsRelease(receipt.Spec.Intent) && receipt.Spec.Expected.Player.String() == move.Player && envelope.ReleaseOpen(receipt.State) {
					moves[i].Plan = "Plan: " + StateLabel(receipt.State)
					break
				}
			}
		}
	}
	return nil
}

// MirrorWhatIf builds an unpersisted baseline; counterparty projections must not read private plans.
func MirrorWhatIf(ctx context.Context, st Stores, mine string) (mflwhatif.WhatIfSession, error) {
	var session mflwhatif.WhatIfSession
	m, _, used := st.League.ContractsView()
	amount, ok := used[mine]
	if !ok {
		return session, fmt.Errorf("what-if: franchise %s is not in the mirror", mine)
	}
	stamps, err := st.League.Stamps(ctx)
	if err != nil {
		return session, fmt.Errorf("what-if baseline: %w", err)
	}
	session = mflwhatif.WhatIfSession{Mine: mine, Snapshot: m, Stamps: stamps, Used: amount, Moves: []mflwhatif.WhatIfMove{}, TaxiPercent: st.Rulebook.TaxiCapPercent(), IRPercent: st.Rulebook.IRCapPercent()}
	if capText := st.Rulebook.GetSalaryCap(); capText != "" {
		capAmount, err := domain.ParseMoneyMillions(capText)
		if err != nil {
			return session, fmt.Errorf("what-if cap: %w", err)
		}
		session.Cap = &capAmount
	}
	return session, nil
}

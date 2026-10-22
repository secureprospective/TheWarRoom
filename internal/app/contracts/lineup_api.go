package contracts

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/url"
	"slices"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/report"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/envelope"
	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
	"github.com/secureprospective/TheWarRoom/internal/leagueweek"
	"github.com/secureprospective/TheWarRoom/internal/lineup"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
	"github.com/secureprospective/TheWarRoom/internal/store/moves"
)

// LineupReading is one franchise's saved MFL lineup for the held week, the rest of its active
// roster as the bench, and whether the league's own starter rules accept it. StarterCount is a
// full lineup's size under those rules (0 when they are unreadable).
type LineupReading struct {
	Franchise    string              `json:"franchise"`
	Week         int                 `json:"week"`
	StarterCount int                 `json:"starterCount"`
	Starters     []LineupPlayer      `json:"starters"`
	Bench        []LineupPlayer      `json:"bench"`
	Check        lineup.Result       `json:"check"`
	Provenance   snapshot.Provenance `json:"provenance"`
	RulesSource  snapshot.Provenance `json:"rulesSource"`
}

// TargetLineup reads held state only (no network, never refreshMu). A missing lineup or
// unreadable rules is not an error: the reading's provenance and Check say why.
func (a *API) TargetLineup(franchiseID string) (LineupReading, error) {
	lineups, err := a.host.ContractsSeason()
	if err != nil {
		return LineupReading{}, fmt.Errorf("target lineup: %w", err)
	}
	snap, loaded := a.host.ContractsDirectory()
	if !loaded {
		return LineupReading{}, fmt.Errorf("target lineup: no snapshot loaded")
	}
	known := slices.ContainsFunc(snap.Franchises.Value, func(f snapshot.Franchise) bool {
		return f.ID == franchiseID
	})
	if !known {
		return LineupReading{}, fmt.Errorf("target lineup: unknown franchise %q", franchiseID)
	}
	cfg := a.stores().Rulebook.ActiveConfig()
	rules, rulesErr := lineup.ParseRules(cfg.Starters)
	r := LineupReading{
		Franchise: franchiseID, Week: lineups.Value.Week, StarterCount: rules.StarterCount,
		Starters: []LineupPlayer{}, Bench: []LineupPlayer{}, Provenance: lineups.Provenance,
		RulesSource: snapshot.Provenance{Kind: "live", Source: cfg.Source, Freshness: a.host.RulesFreshness()},
	}
	r.populate(snap, lineups.Value, rules)
	r.judge(rules, rulesErr)
	return r, nil
}

// judge fills Check; unreadable rules are reported in the reading, not as the binding's error.
func (r *LineupReading) judge(rules lineup.Rules, rulesErr error) {
	if rulesErr != nil {
		note := "The league's lineup rules could not be read: " + rulesErr.Error()
		r.Check = UnreadableRules(rulesErr)
		r.RulesSource.Freshness = domain.Freshness{State: domain.FreshFail, Note: note}
		return
	}
	positions := make([]domain.Position, 0, len(r.Starters))
	for _, p := range r.Starters {
		positions = append(positions, p.Position)
	}
	r.Check = lineup.Check(rules, positions)
}

func (r *LineupReading) populate(snap snapshot.Snapshot, saved leaguefeed.Lineups, rules lineup.Rules) {
	starters, bench, found := Layout(snap, saved, r.Franchise, rules)
	r.Starters = append(r.Starters, starters...)
	if !found && r.Provenance.Freshness.State != domain.FreshFail {
		fresh := &r.Provenance.Freshness
		fresh.State = domain.FreshFail
		fresh.Note = report.JoinNote(fresh.Note, "no saved lineup for franchise "+r.Franchise)
	}
	r.Bench = append(r.Bench, bench...)
}

// LineupSelection checks a lineup being edited: the snapshot is loaded, the franchise is held and
// the starters are distinct player IDs.
func LineupSelection(h Selector, franchiseID string, starters []string) (snapshot.Snapshot, []playerid.PlayerID, error) {
	if err := h.Ready(); err != nil {
		return snapshot.Snapshot{}, nil, fmt.Errorf("lineup selection: startup: %w", err)
	}
	snap, loaded := h.ContractsDirectory()
	if !loaded {
		return snapshot.Snapshot{}, nil, fmt.Errorf("lineup selection: no snapshot loaded")
	}
	if !slices.ContainsFunc(snap.Franchises.Value, func(f snapshot.Franchise) bool {
		return f.ID == franchiseID
	}) {
		return snapshot.Snapshot{}, nil, fmt.Errorf("lineup selection: unknown franchise %q", franchiseID)
	}
	ids, err := (Selection{Starters: starters}).Validate()
	if err != nil {
		return snapshot.Snapshot{}, nil, err
	}
	return snap, ids, nil
}

// TargetCheckLineup judges a lineup being edited in HQ: local only, no network, never refreshMu,
// never saved. Unknown positions are problems in the Result, not errors.
func (a *API) TargetCheckLineup(franchiseID string, starters []string) (lineup.Result, error) {
	snap, ids, err := LineupSelection(a.host, franchiseID, starters)
	if err != nil {
		return lineup.Result{}, fmt.Errorf("target check lineup: %w", err)
	}
	rules, err := lineup.ParseRules(a.stores().Rulebook.ActiveConfig().Starters)
	if err != nil {
		return UnreadableRules(err), nil
	}
	return lineup.Check(rules, envelope.LineupPositions(ids, snap)), nil
}

// LineupPlan is the request and check a lineup draft builds for franchiseID now. It is a package
// function, never bound; root cross-feature scenarios use it to stage a competing plan.
func LineupPlan(a *API, franchiseID string, ids []playerid.PlayerID, now time.Time) (envelope.LineupRequest, envelope.LineupCheck) {
	return a.lineupRequest(franchiseID, ids, now)
}

func (a *API) lineupRequest(
	franchiseID string, ids []playerid.PlayerID, now time.Time,
) (envelope.LineupRequest, envelope.LineupCheck) {
	held, feed := a.host.ContractsLineups()
	week := a.host.ContractsWeek()
	req := envelope.LineupRequest{
		LeagueID: ingestion.LeagueID, FranchiseID: franchiseID, Week: held.Week,
		Starters: ids, Target: envelope.Target{Kind: envelope.Unmapped},
	}
	c := envelope.LineupCheck{BaselineKnown: !feed.FetchedAt.IsZero() && feed.RefreshError == ""}
	if c.BaselineKnown {
		for _, saved := range held.Franchises {
			if saved.Franchise == franchiseID {
				req.Baseline = saved.Starters
			}
		}
	}
	rules, err := lineup.ParseRules(a.stores().Rulebook.ActiveConfig().Starters)
	if err != nil {
		c.Blocks = append(c.Blocks, UnreadableRules(err).Problems[0].Message)
	}
	c.Rules = rules
	if c.BaselineKnown && req.Week != week.Number {
		c.Blocks = append(c.Blocks, fmt.Sprintf(
			"MFL's saved lineups are for week %d, the league is in week %d", req.Week, week.Number))
	}
	// With no held feed, scope the blocked plan to the known league week, not an invented week.
	if req.Week == 0 {
		req.Week = week.Number
	}
	// The plan can land until the week's last kickoff; earlier games lock only their own players.
	switch {
	case len(week.Games) == 0:
		c.Blocks = append(c.Blocks, "lineup lock unknown")
	case !leagueweek.LastLock(week).After(now):
		c.Blocks = append(c.Blocks, fmt.Sprintf("week %d lineups are locked", week.Number))
	default:
		last := leagueweek.LastLock(week)
		req.Deadline = &last
	}
	host, season := a.host.ContractsLeagueHost()
	if host == "" {
		c.Blocks = append(c.Blocks, "MFL host not yet known")
	} else {
		req.Target = envelope.Target{Kind: envelope.Mapped, URL: fmt.Sprintf(
			"https://%s/%d/lineup?L=%s&WEEK=%d&F=%s", host, season,
			url.QueryEscape(req.LeagueID), req.Week, url.QueryEscape(franchiseID))}
	}
	return req, c
}

// TargetDraftLineup saves a checked lineup.set plan for the held week, after moving earlier live
// plans for the same franchise and week to Stale.
func (a *API) TargetDraftLineup(franchiseID string, starters []string) (envelope.Receipt, error) {
	if err := a.host.RequireOwnFranchise(franchiseID); err != nil {
		return envelope.Receipt{}, err //nolint:wrapcheck // Preserve the shared authorization refusal.
	}
	snap, ids, err := LineupSelection(a.host, franchiseID, starters)
	if err != nil {
		return envelope.Receipt{}, fmt.Errorf("target draft lineup: %w", err)
	}
	var random [16]byte
	if _, err := rand.Read(random[:]); err != nil {
		return envelope.Receipt{}, fmt.Errorf("target draft lineup: correlation ID: %w", err)
	}
	var receipt envelope.Receipt
	a.host.SerializeMoves(func() { receipt, err = a.draftLineup(franchiseID, ids, snap, "lineup-"+hex.EncodeToString(random[:])) })
	return receipt, err
}

// draftLineup runs under the plan-ledger serializer.
func (a *API) draftLineup(franchiseID string, ids []playerid.PlayerID, snap snapshot.Snapshot, id string) (envelope.Receipt, error) {
	now := a.host.ContractsNow()
	req, check := a.lineupRequest(franchiseID, ids, now)
	e, err := envelope.DraftLineup(now, req, snap, check, func() string { return id })
	if err != nil {
		return envelope.Receipt{}, fmt.Errorf("target draft lineup: %w", err)
	}
	ctx, cancel := context.WithTimeout(a.host.ContractsContext(), timeout)
	defer cancel()
	store := a.stores().Moves
	if err := SupersedeMoves(ctx, store, e, now); err != nil {
		return envelope.Receipt{}, err
	}
	if err := store.Save(ctx, e); err != nil {
		return envelope.Receipt{}, fmt.Errorf("target draft lineup: save: %w", err)
	}
	return e.Receipt(), nil
}

// SupersedeMoves moves the franchise's live plans that compete with next to Stale. The caller holds
// the plan-ledger serializer.
func SupersedeMoves(ctx context.Context, store *moves.Store, next envelope.Envelope, now time.Time) error {
	spec := next.Receipt().Spec
	receipts, err := store.List(ctx, spec.LeagueID, spec.FranchiseID)
	if err != nil {
		return fmt.Errorf("supersede move: list: %w", err)
	}
	// Only a plan that can still be handed off or land competes; drafts and blocked plans are inert.
	live := []envelope.State{envelope.Ready, envelope.HandedOff, envelope.NotYetDone, envelope.NotVerified,
		envelope.DOTReview}
	for _, r := range receipts {
		if !Competing(r.Spec, spec) || !slices.Contains(live, r.State) {
			continue
		}
		old, err := store.Get(ctx, r.CorrelationID)
		if err != nil {
			return fmt.Errorf("supersede move: load: %w", err)
		}
		stale, err := old.Invalidate(now, "superseded by "+next.ID())
		if err != nil {
			return fmt.Errorf("supersede move: invalidate: %w", err)
		}
		if err := store.Save(ctx, stale); err != nil {
			return fmt.Errorf("supersede move: save: %w", err)
		}
	}
	return nil
}

func (a *API) TargetRosterReports(mine string) ([]report.Report, error) {
	if err := a.host.Ready(); err != nil {
		return nil, fmt.Errorf("roster reports: startup: %w", err)
	}
	snap, err := a.host.RosterSnapshot(mine)
	if err != nil {
		return nil, err //nolint:wrapcheck // The host returns the personal feature's existing refusal words.
	}
	ctx, cancel := context.WithTimeout(a.host.ContractsContext(), timeout)
	defer cancel()
	lastWeek := a.host.ContractsWeek().Number - 1
	season, week, byes := a.host.RosterFeeds(ctx, lastWeek)
	rules, err := lineup.ParseRules(a.stores().Rulebook.ActiveConfig().Starters)
	stats, bye := RosterReports(snap, mine, season, week, byes, rules, time.Local)
	if err != nil {
		bye.Note += " Starter bounds unavailable: " + err.Error()
	}
	return []report.Report{stats, bye}, nil
}

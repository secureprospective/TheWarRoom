package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	contractsapp "github.com/secureprospective/TheWarRoom/internal/app/contracts"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/envelope"
	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/nflschedule"
	"github.com/secureprospective/TheWarRoom/internal/leagueweek"
	"github.com/secureprospective/TheWarRoom/internal/lineup"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
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

// LineupPlayer is a player in the lineup; Position is empty when the directory lacks him.
type LineupPlayer struct {
	ID       string          `json:"id"`
	Position domain.Position `json:"position,omitempty" ts_type:"string"`
}

// TargetLineup reads held state only (no network, never refreshMu). A missing lineup or
// unreadable rules is not an error: the reading's provenance and Check say why.
func (a *App) TargetLineup(franchiseID string) (LineupReading, error) {
	season, err := a.TargetSeason()
	if err != nil {
		return LineupReading{}, fmt.Errorf("target lineup: %w", err)
	}
	a.targetSnapshotMu.Lock()
	snap, loaded := a.targetSnapshot, a.hasTargetSnapshot
	a.targetSnapshotMu.Unlock()
	if !loaded {
		return LineupReading{}, fmt.Errorf("target lineup: no snapshot loaded")
	}
	known := slices.ContainsFunc(snap.Franchises.Value, func(f snapshot.Franchise) bool {
		return f.ID == franchiseID
	})
	if !known {
		return LineupReading{}, fmt.Errorf("target lineup: unknown franchise %q", franchiseID)
	}
	cfg := a.rulebook.ActiveConfig()
	rules, rulesErr := lineup.ParseRules(cfg.Starters)
	r := LineupReading{
		Franchise: franchiseID, Week: season.Lineups.Value.Week, StarterCount: rules.StarterCount,
		Starters: []LineupPlayer{}, Bench: []LineupPlayer{}, Provenance: season.Lineups.Provenance,
		RulesSource: snapshot.Provenance{Kind: "live", Source: cfg.Source, Freshness: a.rulesFreshness()},
	}
	r.populate(snap, season, rules)
	r.judge(rules, rulesErr)
	return r, nil
}

// judge fills Check; unreadable rules are reported in the reading, not as the binding's error.
func (r *LineupReading) judge(rules lineup.Rules, rulesErr error) {
	if rulesErr != nil {
		note := "The league's lineup rules could not be read: " + rulesErr.Error()
		r.Check = unreadableLineupRules(rulesErr)
		r.RulesSource.Freshness = Freshness{State: FreshFail, Note: note}
		return
	}
	positions := make([]domain.Position, 0, len(r.Starters))
	for _, p := range r.Starters {
		positions = append(positions, p.Position)
	}
	r.Check = lineup.Check(rules, positions)
}

func (r *LineupReading) populate(snap snapshot.Snapshot, season SeasonReading, rules lineup.Rules) {
	starters, bench, found := contractsapp.Layout(snap, season.Lineups.Value, r.Franchise, rules)
	for _, p := range starters {
		r.Starters = append(r.Starters, LineupPlayer(p))
	}
	if !found && r.Provenance.Freshness.State != FreshFail {
		fresh := &r.Provenance.Freshness
		fresh.State = FreshFail
		fresh.Note = joinNote(fresh.Note, "no saved lineup for franchise "+r.Franchise)
	}
	for _, p := range bench {
		r.Bench = append(r.Bench, LineupPlayer(p))
	}
}

func joinNote(notes ...string) string {
	kept := slices.DeleteFunc(notes, func(n string) bool { return n == "" })
	for i, note := range kept {
		kept[i] = strings.TrimRight(note, ".; ")
	}
	return strings.Join(kept, "; ")
}

// rulesFreshness is live from the last successful rules sync; before one, the stored rulebook
// is the last good copy and says so.
func (a *App) rulesFreshness() Freshness {
	checked := a.rulesCheckedAt.Load()
	if checked == 0 {
		return Freshness{State: FreshStale, Note: "stored league settings; not yet checked with MFL this run"}
	}
	return liveFreshness(time.Unix(checked, 0))
}

func unreadableLineupRules(err error) lineup.Result { return contractsapp.UnreadableRules(err) }

func (a *App) lineupSelection(
	franchiseID string, starters []string,
) (snapshot.Snapshot, []playerid.PlayerID, error) {
	if err := a.ready(); err != nil {
		return snapshot.Snapshot{}, nil, fmt.Errorf("lineup selection: startup: %w", err)
	}
	a.targetSnapshotMu.Lock()
	snap, loaded := a.targetSnapshot, a.hasTargetSnapshot
	a.targetSnapshotMu.Unlock()
	if !loaded {
		return snapshot.Snapshot{}, nil, fmt.Errorf("lineup selection: no snapshot loaded")
	}
	if !slices.ContainsFunc(snap.Franchises.Value, func(f snapshot.Franchise) bool {
		return f.ID == franchiseID
	}) {
		return snapshot.Snapshot{}, nil, fmt.Errorf("lineup selection: unknown franchise %q", franchiseID)
	}
	ids, err := (contractsapp.Selection{Starters: starters}).Validate()
	if err != nil {
		return snapshot.Snapshot{}, nil, err //nolint:wrapcheck // the service's errors already start "lineup selection"
	}
	return snap, ids, nil
}

// TargetCheckLineup judges a lineup being edited in HQ: local only, no network, never refreshMu,
// never saved. Unknown positions are problems in the Result, not errors.
func (a *App) TargetCheckLineup(franchiseID string, starters []string) (lineup.Result, error) {
	snap, ids, err := a.lineupSelection(franchiseID, starters)
	if err != nil {
		return lineup.Result{}, fmt.Errorf("target check lineup: %w", err)
	}
	rules, err := lineup.ParseRules(a.rulebook.ActiveConfig().Starters)
	if err != nil {
		return unreadableLineupRules(err), nil
	}
	return lineup.Check(rules, envelope.LineupPositions(ids, snap)), nil
}

func (a *App) lineupRequest(
	franchiseID string, ids []playerid.PlayerID, now time.Time,
) (envelope.LineupRequest, envelope.LineupCheck) {
	a.seasonMu.Lock()
	held := a.seasonLineups
	a.seasonMu.Unlock()
	a.weekMu.Lock()
	week := a.week
	a.weekMu.Unlock()
	req := envelope.LineupRequest{
		LeagueID: ingestion.LeagueID, FranchiseID: franchiseID, Week: held.value.Week,
		Starters: ids, Target: envelope.Target{Kind: envelope.Unmapped},
	}
	c := envelope.LineupCheck{BaselineKnown: !held.fetchedAt.IsZero() && held.refreshError == ""}
	if c.BaselineKnown {
		for _, saved := range held.value.Franchises {
			if saved.Franchise == franchiseID {
				req.Baseline = saved.Starters
			}
		}
	}
	rules, err := lineup.ParseRules(a.rulebook.ActiveConfig().Starters)
	if err != nil {
		c.Blocks = append(c.Blocks, unreadableLineupRules(err).Problems[0].Message)
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
	host := ""
	if a.mflClient != nil {
		host = a.mflClient.LeagueHost()
	}
	if host == "" {
		c.Blocks = append(c.Blocks, "MFL host not yet known")
	} else {
		req.Target = envelope.Target{Kind: envelope.Mapped, URL: fmt.Sprintf(
			"https://%s/%d/lineup?L=%s&WEEK=%d&F=%s", host, a.season,
			url.QueryEscape(req.LeagueID), req.Week, url.QueryEscape(franchiseID))}
	}
	return req, c
}

// TargetDraftLineup saves a checked lineup.set plan for the held week, after moving earlier live
// plans for the same franchise and week to Stale.
func (a *App) TargetDraftLineup(franchiseID string, starters []string) (envelope.Receipt, error) {
	snap, ids, err := a.lineupSelection(franchiseID, starters)
	if err != nil {
		return envelope.Receipt{}, fmt.Errorf("target draft lineup: %w", err)
	}
	var random [16]byte
	if _, err := rand.Read(random[:]); err != nil {
		return envelope.Receipt{}, fmt.Errorf("target draft lineup: correlation ID: %w", err)
	}
	a.movesMu.Lock()
	defer a.movesMu.Unlock()
	now := a.movesNow()
	req, check := a.lineupRequest(franchiseID, ids, now)
	id := "lineup-" + hex.EncodeToString(random[:])
	e, err := envelope.DraftLineup(now, req, snap, check, func() string { return id })
	if err != nil {
		return envelope.Receipt{}, fmt.Errorf("target draft lineup: %w", err)
	}
	ctx, cancel := context.WithTimeout(a.ctx, m2Timeout)
	defer cancel()
	if err := a.supersedeLineups(ctx, e, now); err != nil {
		return envelope.Receipt{}, err
	}
	if err := a.moves.Save(ctx, e); err != nil {
		return envelope.Receipt{}, fmt.Errorf("target draft lineup: save: %w", err)
	}
	return e.Receipt(), nil
}

func (a *App) supersedeLineups(ctx context.Context, next envelope.Envelope, now time.Time) error {
	return a.supersedeMoves(ctx, next, now)
}

func (a *App) supersedeMoves(ctx context.Context, next envelope.Envelope, now time.Time) error {
	spec := next.Receipt().Spec
	receipts, err := a.moves.List(ctx, spec.LeagueID, spec.FranchiseID)
	if err != nil {
		return fmt.Errorf("supersede move: list: %w", err)
	}
	// Only a plan that can still be handed off or land competes; drafts and blocked plans are inert.
	live := []envelope.State{envelope.Ready, envelope.HandedOff, envelope.NotYetDone, envelope.NotVerified,
		envelope.DOTReview}
	for _, r := range receipts {
		if !contractsapp.Competing(r.Spec, spec) || !slices.Contains(live, r.State) {
			continue
		}
		old, err := a.moves.Get(ctx, r.CorrelationID)
		if err != nil {
			return fmt.Errorf("supersede move: load: %w", err)
		}
		stale, err := old.Invalidate(now, "superseded by "+next.ID())
		if err != nil {
			return fmt.Errorf("supersede move: invalidate: %w", err)
		}
		if err := a.moves.Save(ctx, stale); err != nil {
			return fmt.Errorf("supersede move: save: %w", err)
		}
	}
	return nil
}

func (a *App) TargetRosterReports(mine string) ([]Report, error) {
	if err := a.ready(); err != nil {
		return nil, fmt.Errorf("roster reports: startup: %w", err)
	}
	snap, err := a.personalSnapshot(mine)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(a.ctx, m4Timeout)
	defer cancel()
	a.weekMu.Lock()
	lastWeek := a.week.Number - 1
	a.weekMu.Unlock()
	season := scorePeriod(ctx, a, "YTD")
	week := scorePeriod(ctx, a, strconv.Itoa(lastWeek))
	byes := archivedOnly(ctx, a, nflschedule.ByeExport, true, nflschedule.ParseByes)
	rules, err := lineup.ParseRules(a.rulebook.ActiveConfig().Starters)
	stats, bye := contractsapp.RosterReports(snap, mine, contractsapp.Scores{Feed: season.state(), Rows: season.value},
		contractsapp.Scores{Feed: week.state(), Rows: week.value}, contractsapp.Byes{Feed: byes.state(), Rows: byes.value}, rules, time.Local)
	if err != nil {
		bye.Note += " Starter bounds unavailable: " + err.Error()
	}
	return []Report{wireReport(stats), wireReport(bye)}, nil
}

func scorePeriod(ctx context.Context, a *App, period string) heldSeasonFeed[[]ingestion.RawPlayerScore] {
	return archivedOnly(ctx, a, "playerScores", false, func(body []byte) ([]ingestion.RawPlayerScore, error) {
		rows, err := ingestion.ParsePlayerScores(body)
		if err != nil {
			return nil, fmt.Errorf("player scores: parse: %w", err)
		}
		out := []ingestion.RawPlayerScore{}
		for _, r := range rows {
			if r.Week == period {
				out = append(out, r)
			}
		}
		if len(out) == 0 {
			return nil, fmt.Errorf("player scores: period %s not held", period)
		}
		return out, nil
	})
}

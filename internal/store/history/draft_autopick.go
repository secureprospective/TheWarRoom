package history

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"log"
	"slices"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

// DraftRanking orders an available draft player pool from highest to lowest priority.
type DraftRanking interface {
	Rank(ctx context.Context, league string, season int, pool []string) ([]string, error)
}

// DynastyDraftRanking orders players by the newest stored model run's dynasty_ppg descending (now_ppg
// when Season is set), then unscored players by this week's NativeProjections median descending,
// breaking ties by ID.
type DynastyDraftRanking struct {
	Store  *Store
	Season bool
}

// Rank returns pool sorted by stored dynasty_ppg (newest model run), then weekly projection median.
func (r DynastyDraftRanking) Rank(ctx context.Context, league string, season int,
	pool []string) ([]string, error) {
	if len(pool) == 0 {
		return nil, nil
	}
	scored, err := r.latestDynastyScores(ctx)
	if err != nil {
		return nil, err
	}
	var withScore, unscored []string
	for _, id := range pool {
		if _, ok := scored[id]; ok {
			withScore = append(withScore, id)
		} else {
			unscored = append(unscored, id)
		}
	}
	slices.SortFunc(withScore, func(a, b string) int {
		return cmp.Or(cmp.Compare(scored[b], scored[a]), cmp.Compare(a, b))
	})
	if len(unscored) > 0 {
		if err := r.sortByProjection(ctx, league, season, unscored); err != nil {
			return nil, err
		}
	}
	return append(withScore, unscored...), nil
}

func (r DynastyDraftRanking) latestDynastyScores(ctx context.Context) (map[string]float64, error) {
	run, ok, err := r.Store.latestModelRun(ctx)
	if err != nil || !ok {
		return nil, err
	}
	rows, err := r.Store.ModelScores(ctx, run.ID)
	if err != nil {
		return nil, err
	}
	out := make(map[string]float64, len(rows))
	for _, sc := range rows {
		if !sc.Unavailable {
			out[sc.MFLID] = sc.DynastyPPG
			if r.Season {
				out[sc.MFLID] = sc.NowPPG
			}
		}
	}
	return out, nil
}

func (r DynastyDraftRanking) sortByProjection(ctx context.Context, league string, season int,
	ids []string) error {
	week := r.Store.projectionWeek(ctx, league, r.Store.now())
	proj, err := r.Store.NativeProjections(ctx, league, season, week, ids)
	if err != nil {
		return err
	}
	byID := make(map[string]NativeProjection, len(proj.Players))
	for _, p := range proj.Players {
		byID[p.Player] = p
	}
	slices.SortFunc(ids, func(a, b string) int {
		pa, pb := byID[a], byID[b]
		if pa.Available != pb.Available {
			if pa.Available {
				return -1
			}
			return 1
		}
		if pa.Available {
			if c := cmp.Compare(pb.Median, pa.Median); c != 0 {
				return c
			}
		}
		return cmp.Compare(a, b)
	})
	return nil
}

func (s *Store) projectionWeek(ctx context.Context, league string, at time.Time) int {
	if s.bidEnv == nil {
		return 1
	}
	env, err := s.bidEnv.BidEnvironment(ctx, league, at)
	if err == nil && env.Season.WeekKnown && env.Season.Week >= 1 && env.Season.Week <= 22 {
		return env.Season.Week
	}
	return 1
}

// autoPickDraftAuthorizer is deliberately narrower than any human role: the app itself may only
// pick for a vacant franchise or an expired human autopick turn.
type autoPickDraftAuthorizer struct {
	records   []commissioner.Record
	franchise string
	key       commissioner.DraftKey
	at        time.Time
	standings commissioner.DraftStandings
}

func (a autoPickDraftAuthorizer) Allow(_ string, principal, command string) (string, error) {
	if principal != commissioner.DraftAutoPickPrincipal || command != commissioner.DraftPick {
		return "", errors.New("automatic draft pick can only record draft picks")
	}
	auto, err := commissioner.AutomaticDraftTurn(a.records, a.key, a.franchise, a.at, a.standings)
	if err != nil {
		return "", fmt.Errorf("check franchise seat: %w", err)
	}
	if !auto {
		return "", errors.New("this turn is not eligible for an automatic pick")
	}
	return "automatic", nil
}

func authorizeAutoPickDraft(raw commissioner.RawCommand, principal string,
	auth commissioner.Authorizer, records []commissioner.Record) (string, error) {
	role, err := authorizeCommandWithAuthorizer(raw, principal, auth)
	if err != nil {
		return "", err
	}
	if raw.Args.Draft == nil || raw.Args.Draft.Pick == nil {
		return "", errors.New("automatic draft pick requires a draft selection")
	}
	a, ok := auth.(autoPickDraftAuthorizer)
	if !ok {
		return "", errors.New("automatic draft authorizer required")
	}
	auto, err := commissioner.AutomaticDraftTurn(records, a.key,
		raw.Args.Draft.Pick.Franchise, a.at, a.standings)
	if err != nil {
		return "", fmt.Errorf("check franchise seat: %w", err)
	}
	if !auto {
		return "", errors.New("this turn is not eligible for an automatic pick")
	}
	return role, nil
}

// ProcessDueAutoPicks advances every open, unpaused draft of the league while a vacant franchise
// is on the clock, stopping when a seated franchise is on the clock or the draft is exhausted.
func (s *Store) ProcessDueAutoPicks(ctx context.Context, league string, at time.Time,
	facts nativecheck.Facts, ranking DraftRanking) error {
	_, err := s.ProcessDueAutoPicksUpTo(ctx, league, at, facts, ranking, 0)
	return err
}

// ProcessDueAutoPicksUpTo is ProcessDueAutoPicks that stops after limit picks (0: no limit) and says
// whether more were due, so one league's long run of empty seats cannot hold up every other league's
// due work: the worker takes a batch per league per pass and comes straight back.
func (s *Store) ProcessDueAutoPicksUpTo(ctx context.Context, league string, at time.Time,
	facts nativecheck.Facts, ranking DraftRanking, limit int) (bool, error) {
	if at.IsZero() {
		at = s.now()
	}
	standings := s.draftStandings(ctx, league)
	stalled := map[commissioner.DraftKey]bool{}
	cache := map[commissioner.DraftKey][]string{}
	for made := 0; ; {
		// Each committed pick is recorded at the store clock; read the draft through that instant so the
		// next iteration sees it. A frozen clock keeps the caller's time.
		if now := s.now(); now.After(at) {
			at = now
		}
		records, err := s.nativeRecords(ctx, league)
		if err != nil {
			return false, err
		}
		turn, ok, err := commissioner.DueAutoPick(records, at, facts, standings, stalled)
		if err != nil {
			return false, fmt.Errorf("read due auto-picks: %w", err)
		}
		if !ok {
			return false, nil
		}
		if limit > 0 && made >= limit {
			return true, nil
		}
		turnRanking := ranking
		if turnRanking == nil { // the league's draft.autopickRanking rule
			turnRanking = DynastyDraftRanking{Store: s, Season: turn.Ranking == leaguerules.SeasonRanking}
		}
		ranked, err := cachedRankPool(ctx, turnRanking, league, turn, cache)
		if err != nil {
			return false, fmt.Errorf("rank draft pool: %w", err)
		}
		picked, err := s.pickFirstAccepted(ctx, league, at, records, turn, ranked, facts, standings)
		if err != nil {
			return false, err
		}
		if !picked {
			stalled[turn.Key] = true
		} else {
			made++
		}
	}
}

func cachedRankPool(ctx context.Context, ranking DraftRanking, league string,
	turn commissioner.AutoPickTurn, cache map[commissioner.DraftKey][]string) ([]string, error) {
	avail := make(map[string]bool, len(turn.Pool))
	for _, id := range turn.Pool {
		avail[id] = true
	}
	if prev, ok := cache[turn.Key]; ok {
		out := make([]string, 0, len(turn.Pool))
		for _, id := range prev {
			if avail[id] {
				out = append(out, id)
			}
		}
		if len(out) == len(turn.Pool) {
			return out, nil
		}
	}
	ranked, err := ranking.Rank(ctx, league, turn.Key.Season, turn.Pool)
	if err != nil {
		return nil, fmt.Errorf("rank draft pool: %w", err)
	}
	cache[turn.Key] = ranked
	return ranked, nil
}

func (s *Store) pickFirstAccepted(ctx context.Context, league string, at time.Time,
	records []commissioner.Record, turn commissioner.AutoPickTurn, ranked []string,
	facts nativecheck.Facts, standings commissioner.DraftStandings) (bool, error) {
	reqKey := autoPickRequestKey(league, turn.Key, turn.Round, turn.Slot)
	skipped := 0
	for _, needOnly := range []bool{true, false} {
		picked, err := s.pickMatching(ctx, league, at, records, turn, ranked, facts, standings,
			reqKey, needOnly, &skipped)
		if err != nil || picked {
			return picked, err
		}
	}
	log.Printf("draft autopick: league=%s round=%d slot=%d skipped=%d (no legal pick)",
		league, turn.Round, turn.Slot, skipped)
	return false, nil
}

func (s *Store) pickMatching(ctx context.Context, league string, at time.Time,
	records []commissioner.Record, turn commissioner.AutoPickTurn, ranked []string,
	facts nativecheck.Facts, standings commissioner.DraftStandings, reqKey string,
	needOnly bool, skipped *int) (bool, error) {
	for _, playerID := range ranked {
		pf, ok := facts.Of(playerID)
		needed := ok && nativecheck.LineupNeedsPosition(turn.Lineup, turn.Held, pf.Position)
		if needed != needOnly {
			continue
		}
		in := commissioner.DraftPickInput{League: league, Season: turn.Key.Season,
			Franchise: turn.Franchise, Player: playerID, Request: reqKey,
			Expected: int64(len(records)), Startup: turn.Key.Startup}
		raw, err := commissioner.PlanDraftPickOnClock(in, records, facts, at, time.UTC, standings)
		if err != nil {
			var refusal *commissioner.Refusal
			if errors.As(err, &refusal) {
				*skipped++
				continue
			}
			return false, fmt.Errorf("plan automatic draft pick: %w", err)
		}
		auth := autoPickDraftAuthorizer{
			records: records, franchise: turn.Franchise, key: turn.Key,
			at: at, standings: standings,
		}
		if _, err := s.ExecuteCommissionerWithFacts(ctx, raw, commissioner.DraftAutoPickPrincipal,
			auth, facts, time.UTC); err != nil {
			return false, fmt.Errorf("record automatic draft pick: %w", err)
		}
		if *skipped > 0 {
			log.Printf("draft autopick: league=%s round=%d slot=%d skipped=%d",
				league, turn.Round, turn.Slot, *skipped)
		}
		return true, nil
	}
	return false, nil
}

func autoPickRequestKey(league string, key commissioner.DraftKey, round, slot int) string {
	kind := "rookie"
	if key.Startup {
		kind = "startup"
	}
	if key.Side != "" {
		return fmt.Sprintf("draft-autopick.%s.%s.%d.%s.%d.%d",
			league, key.Side, key.Season, kind, round, slot)
	}
	return fmt.Sprintf("draft-autopick.%s.%d.%s.%d.%d", league, key.Season, kind, round, slot)
}

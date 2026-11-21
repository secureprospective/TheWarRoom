package leagueload

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/archive"
	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/nflschedule"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/players"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/transactions"
	"github.com/secureprospective/TheWarRoom/internal/leaguemigration"
	"github.com/secureprospective/TheWarRoom/internal/leaguemigration/mflsource"
	"github.com/secureprospective/TheWarRoom/internal/mfl"
	"github.com/secureprospective/TheWarRoom/internal/mflkey"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

const defaultMirrorFetchRPS = 2

// MirrorFetchRequest configures one pull of MFL exports into a mirror history store using the
// desktop Prepare fetch path. MFL is only ever read.
type MirrorFetchRequest struct {
	MFLLeague string
	Year      int // optional; 0 discovers the league's current season
	Store     *history.Store
	Transport http.RoundTripper // nil uses http.DefaultTransport
	KeySource func(ctx context.Context, league string, season int) (mflkey.Key, error)
	Now       func() time.Time
	RPS       float64 // optional; defaults to 2 (matching desktop Prepare)
}

// MirrorFetchResult summarizes what was fetched and archived into the mirror database.
type MirrorFetchResult struct {
	League       string    `json:"league"`
	LeagueName   string    `json:"leagueName"`
	MFLLeague    string    `json:"mflLeague"`
	Season       int       `json:"season"`
	Franchises   int       `json:"franchises"`
	Players      int       `json:"players"`
	Contracts    int       `json:"contracts"`
	Transactions int       `json:"transactions"`
	ScoreWeeks   int       `json:"scoreWeeks"`
	NFLWeeks     int       `json:"nflWeeks"`
	FetchedAt    time.Time `json:"fetchedAt"`
}

// FetchMirror pulls fresh MFL league exports, the player directory, and NFL schedule weeks into
// req.Store through archive.Transport, reusing (*mflsource.Source).Read and mflsource.Prepare
// (the desktop Prepare path). It verifies the resulting mirror can build football evidence offline.
func FetchMirror(ctx context.Context, req MirrorFetchRequest) (MirrorFetchResult, error) {
	if err := checkMirrorFetchRequest(req); err != nil {
		return MirrorFetchResult{}, err
	}
	now, rps, mflLeague := mirrorFetchDefaults(req)
	var seasonHint atomic.Int64
	if req.Year != 0 {
		seasonHint.Store(int64(req.Year))
	} else {
		seasonHint.Store(int64(now().Year()))
	}
	client, err := newMirrorMFLClient(req, mflLeague, rps, &seasonHint)
	if err != nil {
		return MirrorFetchResult{}, err
	}
	candidate, playerCount, err := readPreparedMirror(ctx, client, mflLeague, req.Year, now, &seasonHint)
	if err != nil {
		return MirrorFetchResult{}, err
	}
	seasonHint.Store(int64(candidate.Season))
	nflWeeks, err := fetchMirrorNFLWeeks(ctx, client, candidate.Season, candidate.Plan)
	if err != nil {
		return MirrorFetchResult{}, err
	}
	mirror := Mirror{Store: req.Store}
	fetchedAt := now().UTC()
	if _, _, err := mirror.Football(ctx, candidate.Season, fetchedAt); err != nil {
		return MirrorFetchResult{}, fmt.Errorf("verify mirror football evidence: %w", err)
	}
	txns, err := countArchivedTransactions(ctx, mirror, candidate.Season, mflLeague)
	if err != nil {
		return MirrorFetchResult{}, err
	}
	return buildMirrorFetchResult(mflLeague, candidate, playerCount, txns, nflWeeks, fetchedAt), nil
}

func mirrorFetchDefaults(req MirrorFetchRequest) (func() time.Time, float64, string) {
	now := req.Now
	if now == nil {
		now = time.Now
	}
	rps := req.RPS
	if rps == 0 {
		rps = defaultMirrorFetchRPS
	}
	return now, rps, strings.TrimSpace(req.MFLLeague)
}

func readPreparedMirror(
	ctx context.Context,
	client *mfl.Client,
	mflLeague string,
	year int,
	now func() time.Time,
	seasonHint *atomic.Int64,
) (leaguemigration.Candidate, int, error) {
	playerCount := 0
	src := &mflsource.Source{
		Client: client,
		ID:     mflLeague,
		Season: year,
		Now:    now,
		FetchPlayers: func(ctx context.Context, season int) (map[string]string, error) {
			seasonHint.Store(int64(season))
			raw, fetchErr := players.Fetch(ctx, client, strconv.Itoa(season), mflLeague)
			if fetchErr != nil {
				return nil, fmt.Errorf("fetch player directory: %w", fetchErr)
			}
			playerCount = len(raw)
			return Names(raw), nil
		},
	}
	candidate, err := src.Read(ctx, now())
	if err != nil {
		return leaguemigration.Candidate{}, 0, fmt.Errorf("fetch MFL league %s: %w", mflLeague, err)
	}
	if len(candidate.Problems) > 0 {
		return leaguemigration.Candidate{}, 0, fmt.Errorf(
			"prepare MFL league %s: %w", mflLeague, &leaguemigration.Problems{List: candidate.Problems},
		)
	}
	return candidate, playerCount, nil
}

func checkMirrorFetchRequest(req MirrorFetchRequest) error {
	if req.Store == nil {
		return fmt.Errorf("mirror fetch needs a history store")
	}
	if strings.TrimSpace(req.MFLLeague) == "" {
		return fmt.Errorf("mirror fetch needs an MFL league number")
	}
	if req.Year != 0 && (req.Year < 1920 || req.Year > 9999) {
		return fmt.Errorf("mirror fetch year %d must be a 4-digit season year", req.Year)
	}
	return nil
}

func newMirrorMFLClient(
	req MirrorFetchRequest, mflLeague string, rps float64, seasonHint *atomic.Int64,
) (*mfl.Client, error) {
	fetches := &archive.Transport{
		Base:  req.Transport,
		Sink:  req.Store,
		Judge: ingestion.MFLRefusal,
	}
	keySource := req.KeySource
	if keySource == nil {
		keySource = defaultMFLKeySource
	}
	client, err := mfl.New(
		"api",
		rps,
		mfl.WithTransport(fetches),
		mfl.WithKeySource(func(ctx context.Context) (mflkey.Key, error) {
			return keySource(ctx, mflLeague, int(seasonHint.Load()))
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("open the MFL reader: %w", err)
	}
	return client, nil
}

func defaultMFLKeySource(ctx context.Context, league string, season int) (mflkey.Key, error) {
	key, err := mflkey.New(league, strconv.Itoa(season)).Get(ctx)
	if err != nil {
		return "", fmt.Errorf("read MFL key from keyring: %w", err)
	}
	return key, nil
}

func fetchMirrorNFLWeeks(
	ctx context.Context, client *mfl.Client, season int, plan leaguemigration.Plan,
) (int, error) {
	weeks := collectNFLScheduleWeeks(plan)
	year := strconv.Itoa(season)
	for _, week := range weeks {
		raw, err := nflschedule.Fetch(ctx, client, year, week)
		if err != nil {
			return 0, fmt.Errorf("fetch NFL schedule week %d: %w", week, err)
		}
		if raw.Week != strconv.Itoa(week) || len(raw.Matchups) == 0 {
			return 0, fmt.Errorf("NFL schedule week %d returned unexpected week %q", week, raw.Week)
		}
	}
	return len(weeks), nil
}

func collectNFLScheduleWeeks(plan leaguemigration.Plan) []int {
	seen := map[int]bool{}
	addWeek := func(w int) {
		if w >= 1 && w <= 22 {
			seen[w] = true
		}
	}
	for _, w := range plan.Schedule {
		addWeek(w.Week)
	}
	if plan.Recorded != nil {
		for _, r := range plan.Recorded.Teams {
			addWeek(r.Week)
		}
		addWeek(plan.Recorded.CutoverWeek)
	}
	for _, l := range plan.Lineups {
		addWeek(l.Week)
	}
	if len(seen) == 0 {
		return []int{1}
	}
	weeks := make([]int, 0, len(seen))
	for w := range seen {
		weeks = append(weeks, w)
	}
	slices.Sort(weeks)
	return weeks
}

func countArchivedTransactions(
	ctx context.Context, mirror Mirror, season int, mflLeague string,
) (int, error) {
	key := fmt.Sprintf("/%d/export?JSON=1&L=%s&TYPE=transactions", season, mflLeague)
	body, found, err := mirror.Lookup(ctx, key)
	if err != nil {
		return 0, fmt.Errorf("read archived transactions: %w", err)
	}
	if !found {
		return 0, nil
	}
	raw, err := transactions.Parse(body)
	if err != nil {
		return 0, fmt.Errorf("parse archived transactions: %w", err)
	}
	return len(raw.Transactions), nil
}

func buildMirrorFetchResult(
	mflLeague string,
	candidate leaguemigration.Candidate,
	playerCount, txns, nflWeeks int,
	fetchedAt time.Time,
) MirrorFetchResult {
	name := strings.TrimSpace(candidate.Name)
	if name == "" {
		name = mflLeague
	}
	return MirrorFetchResult{
		League:       leaguemigration.AppLeagueID(mflLeague, candidate.Season),
		LeagueName:   name,
		MFLLeague:    mflLeague,
		Season:       candidate.Season,
		Franchises:   len(candidate.Plan.Seed.Franchises),
		Players:      playerCount,
		Contracts:    countPlanContracts(candidate.Plan),
		Transactions: txns,
		ScoreWeeks:   countPlanScoreWeeks(candidate.Plan),
		NFLWeeks:     nflWeeks,
		FetchedAt:    fetchedAt,
	}
}

func countPlanContracts(plan leaguemigration.Plan) int {
	n := 0
	for _, p := range plan.Seed.Players {
		if !p.ContractMissing {
			n++
		}
	}
	return n
}

func countPlanScoreWeeks(plan leaguemigration.Plan) int {
	if plan.Recorded == nil {
		return 0
	}
	weeks := map[int]bool{}
	for _, t := range plan.Recorded.Teams {
		weeks[t.Week] = true
	}
	return len(weeks)
}

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
	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
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
// req.Store through archive.Transport, reusing mflsource.Source and mflsource.Read (the desktop
// Prepare fetch path). It verifies the resulting mirror can build football evidence offline.
func FetchMirror(ctx context.Context, req MirrorFetchRequest) (MirrorFetchResult, error) {
	if err := checkMirrorFetchRequest(req); err != nil {
		return MirrorFetchResult{}, err
	}
	now := req.Now
	if now == nil {
		now = time.Now
	}
	rps := req.RPS
	if rps == 0 {
		rps = defaultMirrorFetchRPS
	}
	mflLeague := strings.TrimSpace(req.MFLLeague)
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
	playerCount := 0
	src := &mflsource.Source{
		Client: client,
		ID:     mflLeague,
		Season: req.Year,
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
	export, err := mflsource.Read(ctx, src, now())
	if err != nil {
		return MirrorFetchResult{}, fmt.Errorf("fetch MFL league %s: %w", mflLeague, err)
	}
	seasonHint.Store(int64(export.Season))
	nflWeeks, err := fetchMirrorNFLWeeks(ctx, client, export)
	if err != nil {
		return MirrorFetchResult{}, err
	}
	fetchedAt := now().UTC()
	if _, _, err := (Mirror{Store: req.Store}).Football(ctx, export.Season, fetchedAt); err != nil {
		return MirrorFetchResult{}, fmt.Errorf("verify mirror football evidence: %w", err)
	}
	return buildMirrorFetchResult(mflLeague, export, playerCount, nflWeeks, fetchedAt), nil
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

func fetchMirrorNFLWeeks(ctx context.Context, client *mfl.Client, export mflsource.Export) (int, error) {
	weeks := collectNFLScheduleWeeks(export)
	season := strconv.Itoa(export.Season)
	for _, week := range weeks {
		raw, err := nflschedule.Fetch(ctx, client, season, week)
		if err != nil {
			return 0, fmt.Errorf("fetch NFL schedule week %d: %w", week, err)
		}
		if raw.Week != strconv.Itoa(week) || len(raw.Matchups) == 0 {
			return 0, fmt.Errorf("NFL schedule week %d returned unexpected week %q", week, raw.Week)
		}
	}
	return len(weeks), nil
}

func collectNFLScheduleWeeks(export mflsource.Export) []int {
	seen := map[int]bool{}
	addWeek := func(w int) {
		if w >= 1 && w <= 22 {
			seen[w] = true
		}
	}
	for _, w := range export.Schedule {
		if n, err := strconv.Atoi(strings.TrimSpace(w.Week)); err == nil {
			addWeek(n)
		}
	}
	for _, r := range export.Results {
		addWeek(r.Week)
	}
	if export.LiveLineups != nil {
		addWeek(export.LiveLineups.Week)
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

func buildMirrorFetchResult(
	mflLeague string, export mflsource.Export, playerCount, nflWeeks int, fetchedAt time.Time,
) MirrorFetchResult {
	name := strings.TrimSpace(export.Name)
	if name == "" {
		name = mflLeague
	}
	txns := 0
	if export.Transactions != nil {
		txns = len(export.Transactions.Transactions)
	}
	return MirrorFetchResult{
		League:       leaguemigration.AppLeagueID(mflLeague, export.Season),
		LeagueName:   name,
		MFLLeague:    mflLeague,
		Season:       export.Season,
		Franchises:   len(export.Config.Franchises),
		Players:      playerCount,
		Contracts:    countRosterContracts(export.Rosters),
		Transactions: txns,
		ScoreWeeks:   countScoredWeeks(export.Results),
		NFLWeeks:     nflWeeks,
		FetchedAt:    fetchedAt,
	}
}

func countRosterContracts(rosters []mflsource.Rostered) int {
	n := 0
	for _, r := range rosters {
		if strings.TrimSpace(r.Salary) != "" ||
			strings.TrimSpace(r.ContractYear) != "" ||
			strings.TrimSpace(r.ContractStatus) != "" {
			n++
		}
	}
	return n
}

func countScoredWeeks(weeks []leaguefeed.WeekResult) int {
	n := 0
	for _, w := range weeks {
		if weekHasScores(w) {
			n++
		}
	}
	return n
}

func weekHasScores(w leaguefeed.WeekResult) bool {
	for _, g := range w.Games {
		if g.Home.Scored || g.Away.Scored {
			return true
		}
	}
	for _, side := range w.Others {
		if side.Scored {
			return true
		}
	}
	return false
}

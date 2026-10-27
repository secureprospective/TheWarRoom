package core

import (
	"context"
	"slices"
	"strconv"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/report"
	"github.com/secureprospective/TheWarRoom/internal/facts"
	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
	"github.com/secureprospective/TheWarRoom/internal/mflkey"
	"github.com/secureprospective/TheWarRoom/internal/normalize"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

func (s *State) Ready() error { <-s.Started; return s.StartupErr }

func (s *State) CommsContext() context.Context {
	if s.Ctx == nil {
		return context.Background()
	}
	return s.Ctx
}

func (s *State) CommsFeeds() (report.Held[leaguefeed.Transaction], [2]report.Held[ingestion.RawLeaguePost], *time.Location) {
	s.SeasonMu.Lock()
	defer s.SeasonMu.Unlock()
	tx := report.Held[leaguefeed.Transaction]{Value: cloneTransactions(s.SeasonTransactions.Value), Feed: s.SeasonTransactions.FeedState()}
	var boards [2]report.Held[ingestion.RawLeaguePost]
	for i, feed := range s.CommsBoards {
		boards[i] = report.Held[ingestion.RawLeaguePost]{Value: slices.Clone(feed.Value), Feed: feed.FeedState()}
	}
	zone := s.LocalZone
	if zone == nil {
		zone = time.Local
	}
	return tx, boards, zone
}

func (s *State) CommsLeague() (normalize.Lookup, map[string]string, state.MirrorSnapshot) {
	s.LookupMu.Lock()
	lookup := s.Lookup // Lookup is immutable and exposes no mutable storage.
	s.LookupMu.Unlock()
	names := s.Rulebook.FranchiseNames()
	mirror, _, _ := s.League.ContractsView()
	return lookup, names, mirror
}

func (s *State) CommsArchive(ctx context.Context) (facts.RosterWeekColumns, facts.NflGameColumns, error) {
	s.FactsMu.Lock()
	defer s.FactsMu.Unlock()
	if s.History == nil || s.ReadCommsArchive == nil {
		return facts.RosterWeekColumns{}, facts.NflGameColumns{}, nil
	}
	rosters, games, err := s.ReadCommsArchive(ctx)
	return cloneRosterWeeks(rosters), cloneNflGames(games), err
}

func (s *State) LogFallback(line string) { s.FallbackLog(line) }

func (s *State) CommsDestination() (string, int) {
	host := ""
	if s.MFLClient != nil {
		host = s.MFLClient.LeagueHost()
	}
	return host, s.Season
}

func (s *State) OpenBrowser(target string) { s.OpenURL(s.Ctx, target) }

func cloneTransactions(rows []leaguefeed.Transaction) []leaguefeed.Transaction {
	out := slices.Clone(rows)
	for i := range out {
		tx := &out[i]
		tx.Trade = clonePtr(tx.Trade)
		if tx.Trade != nil {
			tx.Trade.Gave = cloneAssets(tx.Trade.Gave)
			tx.Trade.CounterpartyGave = cloneAssets(tx.Trade.CounterpartyGave)
		}
		for _, change := range []**leaguefeed.RosterChange{&tx.AddsDrops, &tx.IR, &tx.Taxi} {
			*change = clonePtr(*change)
			if *change != nil {
				(*change).In, (*change).Out = cloneAssets((*change).In), cloneAssets((*change).Out)
			}
		}
	}
	return out
}

func cloneAssets(rows []leaguefeed.Asset) []leaguefeed.Asset {
	out := slices.Clone(rows)
	for i := range out {
		a := &out[i]
		a.Player, a.CurrentPick, a.FuturePick, a.BlindBidCents = clonePtr(a.Player), clonePtr(a.CurrentPick), clonePtr(a.FuturePick), clonePtr(a.BlindBidCents)
	}
	return out
}

func clonePtr[T any](p *T) *T {
	if p == nil {
		return nil
	}
	value := *p
	return &value
}

func clonePointers[T any](rows []*T) []*T {
	out := slices.Clone(rows)
	for i, p := range rows {
		out[i] = clonePtr(p)
	}
	return out
}

func cloneRosterWeeks(in facts.RosterWeekColumns) facts.RosterWeekColumns {
	return facts.RosterWeekColumns{
		Year:           slices.Clone(in.Year),
		Week:           slices.Clone(in.Week),
		Fid:            slices.Clone(in.Fid),
		Pid:            slices.Clone(in.Pid),
		Status:         clonePointers(in.Status),
		Salary:         clonePointers(in.Salary),
		ContractYear:   clonePointers(in.ContractYear),
		ContractStatus: clonePointers(in.ContractStatus),
		ContractInfo:   clonePointers(in.ContractInfo),
		Src:            slices.Clone(in.Src),
	}
}

func cloneNflGames(in facts.NflGameColumns) facts.NflGameColumns {
	return facts.NflGameColumns{
		Year:    slices.Clone(in.Year),
		Week:    slices.Clone(in.Week),
		Team:    slices.Clone(in.Team),
		Opp:     clonePointers(in.Opp),
		IsHome:  clonePointers(in.IsHome),
		Score:   clonePointers(in.Score),
		Spread:  clonePointers(in.Spread),
		Kickoff: clonePointers(in.Kickoff),
		Src:     slices.Clone(in.Src),
	}
}

func (s *State) ReadCommsBoard(ctx context.Context, i int, export string) {
	rows, fresh, err := LiveOrArchive(ctx, s.CommsContext(), s.Season, ArchivedFeed[ingestion.RawLeaguePost]{
		Export: export,
		Fetch: func(ctx context.Context) ([]ingestion.RawLeaguePost, error) {
			return ingestion.FetchLeagueBoard(ctx, s.MFLClient, strconv.Itoa(s.Season), ingestion.LeagueID, export)
		},
		Parse:  func(body []byte) ([]ingestion.RawLeaguePost, error) { return ingestion.ParseLeagueBoard(body, export) },
		Bodies: s.History.ArchivedBodies,
	}, s.FallbackLog)
	s.SeasonMu.Lock()
	HoldFeed(&s.CommsBoards[i], rows, fresh, err)
	s.SeasonMu.Unlock()
}

func (s *State) WithCommsKey(ctx context.Context, read func(context.Context, *mflkey.Store) error) error {
	s.KeyMu.Lock()
	defer s.KeyMu.Unlock()
	return read(ctx, s.KeyStore)
}
func (s *State) CommsRefreshState() (report.Feed, [2]report.Feed) {
	s.SeasonMu.Lock()
	defer s.SeasonMu.Unlock()
	return s.SeasonTransactions.FeedState(), [2]report.Feed{s.CommsBoards[0].FeedState(), s.CommsBoards[1].FeedState()}
}

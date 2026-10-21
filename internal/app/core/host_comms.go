package core

import (
	"context"
	"errors"
	"fmt"
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

// Missing credentials clear member data; a locked keyring retains the last good copy.
var errCommsNeedsKey = errors.New("MFL serves the message board and polls only to league members: connect your MFL key in Control Room › App")

func (s *State) commsKey(ctx context.Context) error {
	s.KeyMu.Lock()
	defer s.KeyMu.Unlock()
	if s.KeyStore == nil {
		return errors.New("MFL keyring is locked or unavailable")
	}
	_, err := s.KeyStore.Get(ctx)
	switch {
	case errors.Is(err, mflkey.ErrNoKey):
		return errCommsNeedsKey
	case err != nil:
		return errors.New("MFL keyring is locked or unavailable")
	}
	return nil
}

func (s *State) RefreshCommsBoards(ctx context.Context) {
	if err := s.commsKey(ctx); err != nil {
		s.SeasonMu.Lock()
		for i := range s.CommsBoards {
			if errors.Is(err, errCommsNeedsKey) {
				s.CommsBoards[i] = HeldFeed[[]ingestion.RawLeaguePost]{}
			}
			s.CommsBoards[i].RefreshError = err.Error()
		}
		s.SeasonMu.Unlock()
		return
	}
	for i, export := range []string{"messageBoard", "polls"} {
		pause := s.SeasonPause
		if pause == nil {
			pause = Pause
		}
		if err := pause(ctx, time.Second); err != nil {
			return
		}
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
}

func (s *State) RefreshComms(force bool) error {
	if err := s.Ready(); err != nil {
		return fmt.Errorf("comms refresh: %w", err)
	}
	s.SeasonMu.Lock()
	tx, boards := s.SeasonTransactions, s.CommsBoards
	s.SeasonMu.Unlock()
	ctx, cancel := context.WithTimeout(s.Ctx, 2*time.Minute)
	defer cancel()
	s.RefreshMu.Lock()
	defer s.RefreshMu.Unlock()
	if force || tx.RefreshError != "" || time.Since(tx.FetchedAt) >= 5*time.Minute {
		s.RefreshTransactions(ctx)
	}
	fresh := true
	for _, b := range boards {
		fresh = fresh && b.RefreshError == "" && time.Since(b.FetchedAt) < 5*time.Minute
	}
	if force || !fresh {
		s.RefreshCommsBoards(ctx)
	}
	return nil
}

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

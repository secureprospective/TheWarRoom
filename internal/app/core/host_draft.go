package core

import (
	"context"
	"slices"
	"strconv"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/report"
	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/draftreports"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/players"
	"github.com/secureprospective/TheWarRoom/internal/lab"
	"github.com/secureprospective/TheWarRoom/internal/normalize"
)

func (s *State) DraftFeeds() (report.Held[draftreports.Pick], report.Held[draftreports.FuturePick]) {
	s.SeasonMu.Lock()
	defer s.SeasonMu.Unlock()
	results, future := s.SeasonDraftResults, s.SeasonFuturePicks
	picks := slices.Clone(results.Value)
	for i := range picks {
		picks[i].Player = clonePtr(picks[i].Player)
	}
	return report.Held[draftreports.Pick]{Value: picks, Feed: results.FeedState()},
		report.Held[draftreports.FuturePick]{Value: slices.Clone(future.Value), Feed: future.FeedState()}
}

func (s *State) DraftInputs() (report.Held[draftreports.Pick], report.Held[draftreports.FuturePick], normalize.Lookup) {
	results, future := s.DraftFeeds()
	s.LookupMu.Lock()
	lookup := s.Lookup // Immutable, with no mutable storage exposed.
	s.LookupMu.Unlock()
	return results, future, lookup
}

func (s *State) DraftLeague() (int, map[string]string, *time.Location) {
	zone := s.LocalZone
	if zone == nil {
		zone = time.Local
	}
	return s.League.Season(), s.Rulebook.FranchiseNames(), zone
}

func (s *State) DraftReady() (context.Context, int, *time.Location, error) {
	if err := s.Ready(); err != nil {
		return nil, 0, nil, err
	}
	zone := s.LocalZone
	if zone == nil {
		zone = time.Local
	}
	return s.Ctx, s.Season, zone, nil
}

// DraftFacts detaches the immutable Lab inputs consumed by Draft and its buy comparison.
// LabData is fixed before publication and has no shared data mutex.
func (s *State) DraftFacts() (*lab.Data, string, error) {
	d, err := s.Lab()
	if err != nil {
		return nil, "", err
	}
	out := &lab.Data{Built: d.Built, ArchiveThrough: d.ArchiveThrough, LastFinished: d.LastFinished,
		Draft: slices.Clone(d.Draft), Trades: slices.Clone(d.Trades), Delivered: slices.Clone(d.Delivered)}
	for i := range out.Draft {
		out.Draft[i] = cloneDraftPick(out.Draft[i])
	}
	for i := range out.Trades {
		out.Trades[i] = cloneCounterpartyTrade(out.Trades[i])
	}
	return out, s.BuildStamp, nil
}

func cloneDraftPick(p lab.DraftPick) lab.DraftPick {
	p.Pos = clonePtr(p.Pos)
	p.Group = clonePtr(p.Group)
	p.Best = clonePtr(p.Best)
	p.NflRound = clonePtr(p.NflRound)
	p.NflPick = clonePtr(p.NflPick)
	p.Age = clonePtr(p.Age)
	p.KeptYear2 = clonePtr(p.KeptYear2)
	p.Wins = clonePointers(p.Wins)
	p.SlotWins = slices.Clone(p.SlotWins)
	p.Player = cloneCounterpartyPlayer(p.Player)
	return p
}

func (s *State) ReadDraftPlayers(ctx context.Context, parse func([]byte) ([]players.RawPlayer, error)) report.Held[players.RawPlayer] {
	held := ArchivedOnly(ctx, s, "players", false, parse)
	return report.Held[players.RawPlayer]{Value: slices.Clone(held.Value), Feed: held.FeedState()}
}

func (s *State) RefreshDraftResults(ctx context.Context) {
	results, rf, re := LiveOrArchive(ctx, s.CommsContext(), s.Season, ArchivedFeed[draftreports.Pick]{
		Export: draftreports.ResultsExport,
		Fetch: func(ctx context.Context) ([]draftreports.Pick, error) {
			return draftreports.FetchResults(ctx, s.MFLClient, strconv.Itoa(s.Season), ingestion.LeagueID)
		}, Parse: draftreports.ParseResults, Bodies: s.History.ArchivedBodies,
	}, s.FallbackLog)
	s.SeasonMu.Lock()
	HoldFeed(&s.SeasonDraftResults, results, rf, re)
	s.SeasonMu.Unlock()
}
func (s *State) RefreshFuturePicks(ctx context.Context) {
	future, ff, fe := LiveOrArchive(ctx, s.CommsContext(), s.Season, ArchivedFeed[draftreports.FuturePick]{
		Export: draftreports.FutureExport,
		Fetch: func(ctx context.Context) ([]draftreports.FuturePick, error) {
			return draftreports.FetchFuture(ctx, s.MFLClient, strconv.Itoa(s.Season), ingestion.LeagueID)
		}, Parse: draftreports.ParseFuture, Bodies: s.History.ArchivedBodies,
	}, s.FallbackLog)
	s.SeasonMu.Lock()
	defer s.SeasonMu.Unlock()
	HoldFeed(&s.SeasonFuturePicks, future, ff, fe)
}

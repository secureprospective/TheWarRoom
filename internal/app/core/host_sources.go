package core

import (
	"context"
	"net/http"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/report"
	"github.com/secureprospective/TheWarRoom/internal/headshot"
	"github.com/secureprospective/TheWarRoom/internal/lab"
	"github.com/secureprospective/TheWarRoom/internal/normalize"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

func (s *State) SourcesContext() context.Context { return s.Ctx }

func (s *State) SourcesInputs() (*history.Store, int, http.RoundTripper, func() (*lab.Data, error)) {
	return s.History, s.Season, s.Fetches, s.LabData
}

func (s *State) SourcesSettings() (string, bool, error) { return s.ReadSourcesSettings() }
func (s *State) SaveSourcesSettings(dir string, off bool) error {
	return s.WriteSourcesSettings(dir, off)
}
func (s *State) SourcesDatabaseChecks() *report.Report { return s.ReadSourcesDatabaseChecks() }
func (s *State) SourcesDirectory(ctx context.Context) (normalize.Lookup, error) {
	return s.ReadSourcesDirectory(ctx)
}

func (s *State) SourcesRostered() map[string]string {
	out := map[string]string{}
	r := s.League.Reader()
	for _, fid := range r.Franchises() {
		roster, _ := r.Roster(fid)
		for _, p := range roster {
			out[p.MFLID] = fid
		}
	}
	return out
}

func (s *State) SourcesPositions() map[string]string {
	s.LookupMu.Lock()
	lk, ok := s.Lookup, s.HasLookup
	s.LookupMu.Unlock()
	out := map[string]string{}
	if !ok {
		return out
	}
	for id := range s.SourcesRostered() {
		if f, ok := lk.Facts(id); ok {
			out[id] = string(f.Position)
		}
	}
	return out
}

func (s *State) SourcesNames() map[string]string { return s.Rulebook.FranchiseNames() }

func (s *State) SourcesPhotos() (headshot.Stats, int, int64, bool, bool) {
	if photos := s.Headshots.Load(); photos != nil {
		st, held, size := photos.Stats()
		return st, held, size, s.PhotosOff.Load(), true
	}
	return headshot.Stats{}, 0, 0, false, false
}

func (s *State) SourcesZone() *time.Location {
	if s.LocalZone != nil {
		return s.LocalZone
	}
	return time.Local
}

func (s *State) RefreshSourcesLeague(ctx context.Context) (int, bool, bool, error) {
	return s.RunSourcesLeagueRefresh(ctx)
}

func (s *State) SerializeSourcesLoad(ctx context.Context, load func()) {
	defer func() {
		if s.SeasonChanged != nil {
			s.SeasonChanged(ctx)
		}
	}()
	s.LoadingSignals.Lock()
	defer s.LoadingSignals.Unlock()
	load()
}

func (s *State) SerializeSourcesCompile(compile func() (string, error)) (string, error) {
	if !s.FactsMu.TryLock() {
		return "The checked facts are already compiling.", nil
	}
	defer s.FactsMu.Unlock()
	return compile()
}

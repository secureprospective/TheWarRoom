package history

import (
	"slices"

	"github.com/secureprospective/TheWarRoom/internal/fantasyscore/source"
)

type liveRevisionOrigin struct {
	sequence          int
	digest, authority string
}

type liveRevisionState struct {
	versions []source.LineRevision
	archived map[liveRevisionOrigin]bool
}

// Preserve the chronology once live evidence joins it: a later final or poll appends,
// rather than reusing a sequence a commissioner may already have pinned.
func (s *Store) mergeLiveVersions(incoming []source.LineRevision, live source.HeldWeek,
	scope source.RawNativeScope) []source.LineRevision {
	s.liveMu.Lock()
	defer s.liveMu.Unlock()
	key := liveStatKey{scope.Player, scope.Season, scope.Week}
	if s.liveRevisions == nil {
		s.liveRevisions = map[liveStatKey]*liveRevisionState{}
	}
	state := s.liveRevisions[key]
	if state == nil {
		state = &liveRevisionState{archived: map[liveRevisionOrigin]bool{}}
		s.liveRevisions[key] = state
	}
	for _, version := range incoming {
		if version.Merged || version.Held.Authority != source.NFLVerse {
			continue
		}
		origin := liveRevisionOrigin{version.Seq, version.Digest, version.Held.Authority}
		if state.archived[origin] {
			continue
		}
		state.archived[origin] = true
		version.Seq = len(state.versions)
		state.versions = append(state.versions, version)
	}
	digest := source.LiveRevisionDigest(*live.Week)
	if !sameLiveRevision(state.versions, digest, live.Authority) {
		state.versions = append(state.versions, source.LineRevision{Seq: len(state.versions), Digest: digest, Held: live})
	}
	state.versions = source.ConsolidateFinal(state.versions)
	return slices.Clone(state.versions)
}

func sameLiveRevision(versions []source.LineRevision, digest, authority string) bool {
	for i := len(versions) - 1; i >= 0; i-- {
		if versions[i].Merged || versions[i].Held.Authority == source.NFLVerse {
			continue
		}
		return versions[i].Digest == digest && versions[i].Held.Authority == authority
	}
	return false
}

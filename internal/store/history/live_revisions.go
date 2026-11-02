package history

import (
	"context"
	"maps"
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
func (s *Store) mergeLiveVersions(ctx context.Context, incoming []source.LineRevision, live source.HeldWeek,
	scope source.RawNativeScope) ([]source.LineRevision, error) {
	s.liveMu.Lock()
	defer s.liveMu.Unlock()
	key := liveStatKey{scope.Player, scope.Season, scope.Week}
	if s.liveRevisions == nil {
		s.liveRevisions = map[liveStatKey]*liveRevisionState{}
	}
	state := s.liveRevisions[key]
	if state == nil {
		state = &liveRevisionState{archived: map[liveRevisionOrigin]bool{}}
	}
	state = &liveRevisionState{versions: slices.Clone(state.versions), archived: maps.Clone(state.archived)}
	start := len(state.versions)
	origins := map[int]int{}
	for _, version := range incoming {
		if version.Merged || version.Held.Authority != source.NFLVerse {
			continue
		}
		origin := liveRevisionOrigin{version.Seq, version.Digest, version.Held.Authority}
		if state.archived[origin] {
			continue
		}
		state.archived[origin] = true
		origins[len(state.versions)] = version.Seq
		version = deepClone(version)
		version.Seq = len(state.versions)
		state.versions = append(state.versions, version)
	}
	digest := source.LiveRevisionDigest(*live.Week)
	if !sameLiveRevision(state.versions, digest, live.Authority) {
		state.versions = append(state.versions, source.LineRevision{Seq: len(state.versions), Digest: digest, Held: deepClone(live)})
	}
	state.versions = source.ConsolidateFinal(state.versions)
	if err := s.storeLiveVersions(ctx, key, state.versions[start:], origins); err != nil {
		return nil, err
	}
	s.liveRevisions[key] = state
	return deepClone(state.versions), nil
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

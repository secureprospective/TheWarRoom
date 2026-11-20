package leagueload

import (
	"cmp"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"reflect"
	"slices"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/leagueweek"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

type preparedFootball struct {
	body      []byte
	oldDigest string
	digest    string
	changed   bool
	env       footballEnv
}

func prepareMirrorFootball(ctx context.Context, r RefreshRequest, season int) (preparedFootball, error) {
	body, digest, err := r.Football(ctx, season)
	if err != nil {
		return preparedFootball{}, fmt.Errorf("build football evidence from mirror: %w", err)
	}
	var raw history.RawFootballEvidence
	if err := json.Unmarshal(body, &raw); err != nil {
		return preparedFootball{}, fmt.Errorf("decode mirror football evidence: %w", err)
	}
	if err := raw.Validate(); err != nil {
		return preparedFootball{}, fmt.Errorf("validate mirror football evidence: %w", err)
	}
	currentPin, err := resolveCurrentFootballPin(ctx, r.Server, r.CurrentPin, season)
	if err != nil {
		return preparedFootball{}, err
	}
	have, hasHave, err := loadExistingFootball(ctx, r.Server, currentPin, season)
	if err != nil {
		return preparedFootball{}, err
	}
	if !hasHave {
		return preparedFootball{
			body: body, oldDigest: currentPin, digest: digest, changed: true, env: footballEnv{raw: raw},
		}, nil
	}
	merged := mergeFootballEvidence(have, raw)
	if sameFootballContent(have, merged) {
		return preparedFootball{
			oldDigest: currentPin, digest: currentPin, changed: false, env: footballEnv{raw: have},
		}, nil
	}
	mergedBody, err := json.Marshal(merged)
	if err != nil {
		return preparedFootball{}, fmt.Errorf("encode merged football evidence: %w", err)
	}
	if err := r.Server.ValidateFootball(ctx, mergedBody); err != nil {
		return preparedFootball{}, fmt.Errorf("validate merged football evidence: %w", err)
	}
	sum := sha256.Sum256(mergedBody)
	return preparedFootball{
		body:      mergedBody,
		oldDigest: currentPin,
		digest:    hex.EncodeToString(sum[:]),
		changed:   true,
		env:       footballEnv{raw: merged},
	}, nil
}

func resolveCurrentFootballPin(
	ctx context.Context, store *history.Store, currentPin string, season int,
) (string, error) {
	if currentPin != "" {
		return currentPin, nil
	}
	var resolved string
	_, err := store.ArchivedBodies(ctx, "urn:twr:football:", func(url string, body []byte, _ time.Time) bool {
		digest, ok := strings.CutPrefix(url, "urn:twr:football:")
		if !ok || len(digest) != 64 {
			return false
		}
		var have history.RawFootballEvidence
		if err := json.Unmarshal(body, &have); err != nil || have.Season != season {
			return false
		}
		if _, err := store.LoadFootball(ctx, digest); err != nil {
			return false
		}
		resolved = digest
		return true
	})
	if err != nil {
		return "", fmt.Errorf("read held football evidence: %w", err)
	}
	return resolved, nil
}

func loadExistingFootball(
	ctx context.Context, store *history.Store, pin string, season int,
) (history.RawFootballEvidence, bool, error) {
	if pin == "" {
		return history.RawFootballEvidence{}, false, nil
	}
	body, err := store.Body(ctx, pin)
	if errors.Is(err, history.ErrNoBody) {
		return history.RawFootballEvidence{}, false, nil
	}
	if err != nil {
		return history.RawFootballEvidence{}, false, fmt.Errorf("read pinned football evidence: %w", err)
	}
	var have history.RawFootballEvidence
	if err := json.Unmarshal(body, &have); err != nil {
		return history.RawFootballEvidence{}, false, fmt.Errorf("decode pinned football evidence: %w", err)
	}
	if have.Season != season {
		return history.RawFootballEvidence{}, false, nil
	}
	return have, true, nil
}

func mergeFootballEvidence(
	have, mirror history.RawFootballEvidence,
) history.RawFootballEvidence {
	merged := history.RawFootballEvidence{
		Version: have.Version,
		Season:  have.Season,
		Source:  have.Source,
		AsOf:    mirror.AsOf.UTC(),
		Players: mergeFootballPlayers(have.Players, mirror.Players),
		Weeks:   mergeFootballWeeks(have.Weeks, mirror.Weeks),
		Stats:   slices.Clone(have.Stats),
	}
	if merged.Version == 0 {
		merged.Version = mirror.Version
	}
	if strings.TrimSpace(merged.Source) == "" {
		merged.Source = mirror.Source
	}
	if have.AsOf.After(merged.AsOf) {
		merged.AsOf = have.AsOf.UTC()
	}
	return merged
}

func mergeFootballPlayers(
	have, mirror map[string]history.RawFootballPlayer,
) map[string]history.RawFootballPlayer {
	out := make(map[string]history.RawFootballPlayer, len(have)+len(mirror))
	for id, hp := range have {
		out[id] = cloneFootballPlayer(hp)
	}
	for id, mp := range mirror {
		hp, ok := out[id]
		if !ok {
			out[id] = cloneFootballPlayer(mp)
			continue
		}
		out[id] = mergeOnePlayer(hp, mp)
	}
	return out
}

func cloneFootballPlayer(p history.RawFootballPlayer) history.RawFootballPlayer {
	if p.NFLRosterSeasons != nil {
		copied := slices.Clone(*p.NFLRosterSeasons)
		p.NFLRosterSeasons = &copied
	}
	return p
}

func mergeOnePlayer(have, mirror history.RawFootballPlayer) history.RawFootballPlayer {
	merged := cloneFootballPlayer(have)
	hasNonMFL := have.Identity != "" && !strings.HasPrefix(have.Identity, history.MFLIdentityPrefix)
	sleeperOnly := have.Identity == "" && have.Sleeper != ""
	if !hasNonMFL && !sleeperOnly && mirror.Identity != "" {
		merged.Identity = mirror.Identity
	}
	if merged.PFR == "" {
		merged.PFR = mirror.PFR
	}
	if merged.Sleeper == "" {
		merged.Sleeper = mirror.Sleeper
	}
	if merged.NFLRosterSeasons == nil && mirror.NFLRosterSeasons != nil {
		copied := slices.Clone(*mirror.NFLRosterSeasons)
		merged.NFLRosterSeasons = &copied
	}
	merged.Facts = mergePlayerFacts(have.Facts, mirror.Facts)
	return merged
}

func mergePlayerFacts(have, mirror nativecheck.PlayerFacts) nativecheck.PlayerFacts {
	out := have
	if mirror.Name != "" {
		out.Name = mirror.Name
	}
	if mirror.Position != "" {
		out.Position = mirror.Position
	}
	if mirror.Team != "" {
		out.Team = mirror.Team
	}
	if mirror.DraftYearHeld {
		out.DraftYear, out.DraftYearHeld = mirror.DraftYear, true
	}
	if mirror.DesignationHeld {
		out.Designation, out.DesignationHeld = mirror.Designation, true
	}
	if mirror.CareerGamesHeld {
		out.CareerGames, out.CareerGamesHeld = mirror.CareerGames, true
	}
	return out
}

func mergeFootballWeeks(have, mirror []leagueweek.Week) []leagueweek.Week {
	byNum := make(map[int]leagueweek.Week, len(have)+len(mirror))
	for _, w := range have {
		byNum[w.Number] = leagueweek.Week{Number: w.Number, Games: slices.Clone(w.Games)}
	}
	for _, mw := range mirror {
		hw, ok := byNum[mw.Number]
		if !ok {
			byNum[mw.Number] = leagueweek.Week{Number: mw.Number, Games: slices.Clone(mw.Games)}
			continue
		}
		byNum[mw.Number] = mergeOneWeek(hw, mw)
	}
	out := make([]leagueweek.Week, 0, len(byNum))
	for _, num := range slices.Sorted(maps.Keys(byNum)) {
		out = append(out, byNum[num])
	}
	return out
}

func mergeOneWeek(have, mirror leagueweek.Week) leagueweek.Week {
	if weekAllFinal(have) {
		return have
	}
	finalByPair := map[string]leagueweek.Game{}
	for _, g := range have.Games {
		if g.Final {
			finalByPair[gamePairKey(g.Teams)] = g
		}
	}
	games := slices.Clone(mirror.Games)
	matchedFinal := 0
	for i, g := range games {
		if hg, ok := finalByPair[gamePairKey(g.Teams)]; ok {
			matchedFinal++
			if !g.Final {
				games[i] = hg
			}
		}
	}
	if matchedFinal < len(finalByPair) {
		return have
	}
	return leagueweek.Week{Number: mirror.Number, Games: games}
}

func weekAllFinal(w leagueweek.Week) bool {
	if len(w.Games) == 0 {
		return false
	}
	for _, g := range w.Games {
		if !g.Final {
			return false
		}
	}
	return true
}

func gamePairKey(teams [2]string) string {
	if cmp.Compare(teams[0], teams[1]) <= 0 {
		return teams[0] + "/" + teams[1]
	}
	return teams[1] + "/" + teams[0]
}

func sameFootballContent(have, merged history.RawFootballEvidence) bool {
	if have.Season != merged.Season ||
		!reflect.DeepEqual(have.Players, merged.Players) ||
		!reflect.DeepEqual(have.Stats, merged.Stats) ||
		len(have.Weeks) != len(merged.Weeks) {
		return false
	}
	haveWeeks := make(map[int]leagueweek.Week, len(have.Weeks))
	for _, w := range have.Weeks {
		haveWeeks[w.Number] = w
	}
	for _, w := range merged.Weeks {
		hw, ok := haveWeeks[w.Number]
		if !ok || !reflect.DeepEqual(hw, w) {
			return false
		}
	}
	return true
}

func ensureFootballStored(
	ctx context.Context, server *history.Store, body []byte, digest string,
) (*history.PinnedFootball, error) {
	evidence, err := server.LoadFootball(ctx, digest)
	if err == nil {
		return evidence, nil
	}
	if !errors.Is(err, history.ErrNoBody) {
		return nil, fmt.Errorf("check held football evidence: %w", err)
	}
	if err := server.ImportFootball(ctx, body, digest); err != nil {
		return nil, fmt.Errorf("import football evidence: %w", err)
	}
	evidence, err = server.LoadFootball(ctx, digest)
	if err != nil {
		return nil, fmt.Errorf("verify imported football evidence: %w", err)
	}
	return evidence, nil
}

type footballEnv struct {
	raw history.RawFootballEvidence
}

func (e footballEnv) PlayerFacts() nativecheck.Facts {
	facts := make(nativecheck.Facts, len(e.raw.Players))
	for id, player := range e.raw.Players {
		f := player.Facts
		if player.NFLRosterSeasons != nil {
			f.NFLSeasons, f.NFLSeasonsHeld = len(*player.NFLRosterSeasons), true
		}
		facts[id] = f
	}
	nativecheck.AddLineCards(facts)
	return facts
}

func (e footballEnv) LineupFacts(ctx context.Context, season, week int) (commissioner.LineupFacts, error) {
	if err := ctx.Err(); err != nil {
		return commissioner.LineupFacts{}, fmt.Errorf("read mirror football schedule: %w", err)
	}
	if season != e.raw.Season {
		return commissioner.LineupFacts{}, fmt.Errorf(
			"mirror football data is for season %d, not %d", e.raw.Season, season)
	}
	for _, held := range e.raw.Weeks {
		if held.Number == week {
			return commissioner.LineupFacts{
				Facts: e.PlayerFacts(), Locks: leagueweek.Locks(held),
				ScheduleHeld: true, Zone: time.UTC,
			}, nil
		}
	}
	return commissioner.LineupFacts{}, fmt.Errorf("mirror football data does not hold NFL week %d", week)
}

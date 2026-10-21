package race

import (
	"maps"
	"slices"
	"sync"

	"github.com/secureprospective/TheWarRoom/internal/app/season"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/leaguestandings"
	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
)

// Held owns Race's private published data. Composition retains the outer shared lock
// when a publication or capture also participates in a shared season snapshot.
type Held struct {
	mu    sync.Mutex
	value season.Held
}

func (h *Held) PublishWeeks(feed season.Feed[[]leaguefeed.WeekResult]) {
	h.mu.Lock()
	defer h.mu.Unlock()
	feed.Value = cloneWeeks(feed.Value)
	h.value.Weeks = feed
}

func (h *Held) PublishBrackets(feed season.Feed[[]leaguefeed.Bracket]) {
	h.mu.Lock()
	defer h.mu.Unlock()
	feed.Value = cloneBrackets(feed.Value)
	h.value.Brackets = feed
}

func (h *Held) PublishStandings(feed season.Feed[[]leaguestandings.RawStanding], power map[string]season.PowerRankingsResult, replacePower bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	feed.Value = slices.Clone(feed.Value)
	h.value.Standings = feed
	if replacePower {
		h.value.Power = clonePower(power)
	}
}

func (h *Held) Snapshot() season.Held {
	h.mu.Lock()
	defer h.mu.Unlock()
	out := h.value
	out.Weeks.Value = cloneWeeks(out.Weeks.Value)
	out.Brackets.Value = cloneBrackets(out.Brackets.Value)
	out.Standings.Value = slices.Clone(out.Standings.Value)
	out.Power = clonePower(out.Power)
	return out
}

func cloneWeeks(in []leaguefeed.WeekResult) []leaguefeed.WeekResult {
	out := slices.Clone(in)
	for i := range out {
		out[i].Games = slices.Clone(out[i].Games)
		for j := range out[i].Games {
			g := &out[i].Games[j]
			g.Home.Players = slices.Clone(g.Home.Players)
			g.Away.Players = slices.Clone(g.Away.Players)
		}
		out[i].Others = slices.Clone(out[i].Others)
		for j := range out[i].Others {
			out[i].Others[j].Players = slices.Clone(out[i].Others[j].Players)
		}
	}
	return out
}

func cloneBrackets(in []leaguefeed.Bracket) []leaguefeed.Bracket {
	out := slices.Clone(in)
	for i := range out {
		out[i].Games = slices.Clone(out[i].Games)
	}
	return out
}

func clonePower(in map[string]season.PowerRankingsResult) map[string]season.PowerRankingsResult {
	out := maps.Clone(in)
	for key, value := range out {
		value.Rows = slices.Clone(value.Rows)
		out[key] = value
	}
	return out
}

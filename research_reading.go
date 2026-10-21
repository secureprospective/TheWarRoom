package main

import (
	"time"

	researchapp "github.com/secureprospective/TheWarRoom/internal/app/research"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/league"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/nflschedule"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/researchfeeds"
	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
	"github.com/secureprospective/TheWarRoom/internal/normalize"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

type RawResearchScope struct {
	Mine     string `json:"mine"`
	Search   string `json:"search"`
	Position string `json:"position"`
	Team     string `json:"team"`
	Group    string `json:"group"`
	Trend    string `json:"trend"`
	Depth    string `json:"depth"`
	Injuries string `json:"injuries"`
}

func (r RawResearchScope) Validate() error {
	return researchapp.Scope(r).Validate() //nolint:wrapcheck // preserve the binding validation error text
}

type ResearchPlayer struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Position    string `json:"position"`
	Team        string `json:"team"`
	FranchiseID string `json:"franchiseId"`
	Owner       string `json:"owner"`
}
type ResearchArticle struct {
	ID       string `json:"id"`
	Article  string `json:"article"`
	Name     string `json:"name"`
	Headline string `json:"headline"`
}
type ResearchBoard struct {
	Board     ReportBoard       `json:"board"`
	News      Report            `json:"news"`
	Articles  []ResearchArticle `json:"articles"`
	Players   []ResearchPlayer  `json:"players"`
	Teams     []string          `json:"teams"`
	Positions []string          `json:"positions"`
	Scope     RawResearchScope  `json:"scope"`
}
type researchInput struct {
	held      researchHeld
	lookup    normalize.Lookup
	mirror    state.MirrorSnapshot
	injuries  heldSeasonFeed[[]leaguefeed.Injury]
	schedules map[int]heldSeasonFeed[[]nflschedule.RawWeek]
	config    league.RawConfig
	week      int
	zone      *time.Location
	now       time.Time
}
type researchView struct{ researchInput }

func (v researchView) feed(kind, week string) heldSeasonFeed[[]researchfeeds.Feed] {
	return v.held.feeds[researchKey(kind, week)]
}
func researchRows(feed heldSeasonFeed[[]researchfeeds.Feed]) []researchfeeds.Entry {
	if len(feed.Value) == 0 {
		return nil
	}
	return feed.Value[0].Players
}
func (v researchView) player(id string) ResearchPlayer {
	return ResearchPlayer(researchapp.Player(v.lookup, v.mirror, v.config, id))
}

type ResearchPlayerReading struct {
	ResearchPlayer
	Report   Report            `json:"report"`
	Articles []ResearchArticle `json:"articles"`
	HowLabel string            `json:"howLabel"`
}

// snapshot detaches mutable indexes. Feed payloads, directory lookups, roster views and
// rulebook configs are published values: refresh replaces them rather than mutating them.
func (in researchInput) snapshot() researchapp.Snapshot {
	if in.now.IsZero() {
		in.now = time.Now()
	}
	v := researchView{researchInput: in}
	rows, _, latest := v.finishedScores()
	scores := make(map[string]*researchapp.Score, len(rows))
	for id, r := range rows {
		scores[id] = &researchapp.Score{Position: r.position, Sum: r.sum, Games: r.games}
	}
	charges := map[string]domain.Money{}
	for _, p := range in.mirror.Players {
		if charge, ok := heldCutCharge(p, in.mirror.Season); ok {
			charges[p.MFLID] = charge
		}
	}
	return researchapp.Snapshot{Held: researchapp.Held{Feeds: researchFeeds(in.held.feeds), Depth: researchFeed(in.held.depth), Links: in.held.links, LinkError: in.held.linkError, News: researchFeeds(in.held.news)},
		Lookup: in.lookup, Mirror: in.mirror, Injuries: researchFeed(in.injuries), Schedules: researchFeeds(in.schedules), Config: in.config, Week: in.week, Zone: in.zone, Now: in.now,
		Finished: scores, Latest: latest, CutCharges: charges}
}
func researchFeed[T any](h heldSeasonFeed[T]) researchapp.Feed[T] {
	return researchapp.Feed[T]{Value: h.Value, FetchedAt: h.FetchedAt, RefreshError: h.RefreshError, ReadPending: h.ReadPending}
}
func researchFeeds[K comparable, T any](held map[K]heldSeasonFeed[T]) map[K]researchapp.Feed[T] {
	if held == nil {
		return nil
	}
	out := make(map[K]researchapp.Feed[T], len(held))
	for key, h := range held {
		out[key] = researchFeed(h)
	}
	return out
}
func buildResearch(in researchInput, scope RawResearchScope) ResearchBoard {
	return wireResearchBoard(researchapp.Build(in.snapshot(), researchapp.Scope(scope)))
}

func wireResearchBoard(b researchapp.Board) ResearchBoard {
	players := make([]ResearchPlayer, len(b.Players))
	for i, p := range b.Players {
		players[i] = ResearchPlayer(p)
	}
	return ResearchBoard{Board: wireBoard(b.Board), News: wireReport(b.News), Articles: wireResearchArticles(b.Articles), Players: wireSlice(b.Players, players), Teams: b.Teams, Positions: b.Positions, Scope: RawResearchScope(b.Scope)}
}
func wireResearchArticles(in []researchapp.Article) []ResearchArticle {
	out := make([]ResearchArticle, len(in))
	for i, a := range in {
		out[i] = ResearchArticle(a)
	}
	return wireSlice(in, out)
}
func wireResearchPlayer(p researchapp.PlayerReading) ResearchPlayerReading {
	return ResearchPlayerReading{ResearchPlayer: ResearchPlayer(p.PlayerInfo), Report: wireReport(p.Report), Articles: wireResearchArticles(p.Articles), HowLabel: p.HowLabel}
}

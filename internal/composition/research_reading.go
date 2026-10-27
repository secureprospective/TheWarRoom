package composition

import (
	"time"

	researchapp "github.com/secureprospective/TheWarRoom/internal/app/research"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/league"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/nflschedule"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/researchfeeds"
	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
	"github.com/secureprospective/TheWarRoom/internal/normalize"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

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
	return researchapp.Rows(feed)
}
func (v researchView) player(id string) researchapp.ResearchPlayer {
	return researchapp.ResearchPlayer(researchapp.Player(v.lookup, v.mirror, v.config, id))
}

// snapshot detaches mutable indexes. Feed payloads, directory lookups, roster views and
// rulebook configs are published values: refresh replaces them rather than mutating them.
func (in researchInput) snapshot() researchapp.Snapshot {
	return researchapp.PrepareSnapshot(researchapp.Snapshot{Held: researchapp.Values{Feeds: in.held.feeds, Depth: in.held.depth, Links: in.held.links, LinkError: in.held.linkError, News: in.held.news}, Lookup: in.lookup, Mirror: in.mirror, Injuries: in.injuries, Schedules: in.schedules, Config: in.config, Week: in.week, Zone: in.zone, Now: in.now}, time.Now, heldCutCharge)
}

// researchHost composes private Research state with Market and Now's shared capture.
// No holder lock spans these callbacks; SeasonMu remains the outer joint lock.
type researchHost struct{ app *App }

func (h researchHost) Ready() error                     { return h.app.ready() }
func (h researchHost) ResearchNames() map[string]string { return h.app.core.ResearchNames() }
func (h researchHost) HydrateResearch(ids ...string)    { h.app.hydrateResearch(ids...) }
func (h researchHost) PrepareResearchDepth() {
	researchapp.PrepareDepth(h.app.core.Fetches != nil, h.app.fallbackParent, rasFetchTimeout, h.app.refreshResearchDepth)
}
func (h researchHost) ResearchSnapshot() researchapp.Snapshot {
	return h.app.heldResearchInput().snapshot()
}
func (h researchHost) RefreshResearch(force bool) error      { return h.app.refreshResearch(force) }
func (h researchHost) RefreshResearchNews(mine string) error { return h.app.refreshResearchNews(mine) }
func (h researchHost) ResearchNews(id string) researchapp.Feed[[]researchfeeds.News] {
	h.app.core.SeasonMu.Lock()
	held := h.app.researchHeld.Snapshot().News[id]
	h.app.core.SeasonMu.Unlock()
	return held
}
func (h researchHost) ResearchDestination() (string, int) { return h.app.core.ResearchDestination() }
func (h researchHost) OpenBrowser(target string)          { h.app.core.OpenBrowser(target) }

// snapshotResearch and publishResearch require SeasonMu: the same lock also guards
// injuries and Now's schedules. The detached compatibility value is still read by Market.
func (a *App) snapshotResearch() researchHeld {
	return researchFromValues(a.researchHeld.Snapshot())
}
func (a *App) publishResearch(value researchHeld) {
	a.researchHeld.Publish(value.values())
}

func (v researchView) scoreInput() researchapp.Snapshot {
	return researchapp.Snapshot{Held: researchapp.Values{Feeds: v.held.feeds}, Lookup: v.lookup, Mirror: v.mirror, Config: v.config, Week: v.week}
}

func researchFromValues(value researchapp.Values) researchHeld {
	return researchHeld{feeds: value.Feeds, depth: value.Depth,
		links: value.Links, linkError: value.LinkError, linkVersion: value.LinkVersion,
		finishedAt: value.FinishedAt, status: value.Status, news: value.News, archiveRead: value.ArchiveRead}
}

func (value researchHeld) values() researchapp.Values {
	return researchapp.Values{Feeds: value.feeds, Depth: value.depth,
		Links: value.links, LinkError: value.linkError, LinkVersion: value.linkVersion,
		FinishedAt: value.finishedAt, Status: value.status, News: value.news, ArchiveRead: value.archiveRead}
}

package research

import (
	"fmt"
	"net/url"

	"github.com/secureprospective/TheWarRoom/internal/app/catalog"
	"github.com/secureprospective/TheWarRoom/internal/app/report"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/researchfeeds"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
)

// Host captures shared inputs and coordinates I/O without exposing locks to the feature.
type Host interface {
	Ready() error
	ResearchNames() map[string]string
	HydrateResearch(...string)
	PrepareResearchDepth()
	ResearchSnapshot() Snapshot
	RefreshResearch(bool) error
	RefreshResearchNews(string) error
	ResearchNews(string) Feed[[]researchfeeds.News]
	ResearchDestination() (string, int)
	OpenBrowser(string)
}

type API struct {
	host Host
	held *Held
}

func New(host Host, held *Held) *API { return &API{host: host, held: held} }

func (a *API) TargetResearch(raw RawResearchScope) (ResearchBoard, error) {
	if err := raw.Validate(); err != nil {
		return ResearchBoard{}, err
	}
	if err := a.host.Ready(); err != nil {
		return ResearchBoard{}, fmt.Errorf("research: %w", err)
	}
	if raw.Mine != "" && a.host.ResearchNames()[raw.Mine] == "" {
		return ResearchBoard{}, fmt.Errorf("research: unknown franchise")
	}
	a.host.HydrateResearch()
	return WireBoard(Build(a.host.ResearchSnapshot(), Scope(raw))), nil
}

func (a *API) TargetResearchRefresh(force bool) error {
	return a.host.RefreshResearch(force) //nolint:wrapcheck // Preserve the existing refresh error text and check order.
}

func (a *API) TargetResearchNewsRefresh(mine string) error {
	return a.host.RefreshResearchNews(mine) //nolint:wrapcheck // Preserve the existing refresh error text and check order.
}

func (a *API) TargetResearchPlayer(id string) (ResearchPlayerReading, error) {
	pid, err := playerid.New(id)
	if err != nil {
		return ResearchPlayerReading{}, fmt.Errorf("research player: %w", err)
	}
	if err := a.host.Ready(); err != nil {
		return ResearchPlayerReading{}, fmt.Errorf("research player: %w", err)
	}
	a.host.HydrateResearch(pid.String())
	a.host.PrepareResearchDepth()
	in := a.host.ResearchSnapshot()
	if _, _, ok := in.Lookup.DisplayFacts(pid.String()); !ok {
		return ResearchPlayerReading{}, fmt.Errorf("research: player not in the held directory")
	}
	return WirePlayer(ReadPlayer(in, pid.String(), a.host.ResearchNames(), "How this is counted")), nil
}

func (a *API) TargetResearchStatus() string { return a.held.SnapshotStatus() }

func (a *API) TargetOpenResearchArticle(id, article string) error {
	pid, err := playerid.New(id)
	if err != nil {
		return fmt.Errorf("news hand-off: %w", err)
	}
	if err := a.host.Ready(); err != nil {
		return fmt.Errorf("news hand-off: %w", err)
	}
	news := a.host.ResearchNews(pid.String())
	found := false
	for _, n := range news.Value {
		found = found || n.Article == article
	}
	if !found {
		return fmt.Errorf("news: article is not held for this player")
	}
	host, season := a.host.ResearchDestination()
	if host == "" {
		return fmt.Errorf("news: MFL host not held; refresh from MFL first")
	}
	target := fmt.Sprintf("https://%s/%d/view_news_article?ID=%s", host, season, url.QueryEscape(article))
	a.host.OpenBrowser(target)
	return nil
}

func Commands() []catalog.Command {
	return []catalog.Command{
		{ID: catalog.Unset, Method: "TargetResearch", Version: catalog.Unset, Role: "gm,commish,admin", Gravity: "G0", Availability: "both"},
		{ID: catalog.Unset, Method: "TargetResearchRefresh", Version: catalog.Unset, Role: "gm,commish,admin", Gravity: "G1", Availability: "both"},
		{ID: catalog.Unset, Method: "TargetResearchNewsRefresh", Version: catalog.Unset, Role: "gm,commish,admin", Gravity: "G1", Availability: "both"},
		{ID: catalog.Unset, Method: "TargetResearchPlayer", Version: catalog.Unset, Role: "gm,commish,admin", Gravity: "G0", Availability: "both"},
		{ID: catalog.Unset, Method: "TargetResearchStatus", Version: catalog.Unset, Role: "gm,commish,admin", Gravity: "G0", Availability: "both"},
		{ID: catalog.Unset, Method: "TargetOpenResearchArticle", Version: catalog.Unset, Role: "gm,commish,admin", Gravity: "G0", Availability: "desktop"},
	}
}

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
	return Scope(r).Validate()
}

//nolint:revive // Preserve the existing exported wire name.
type ResearchPlayer struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Position    string `json:"position"`
	Team        string `json:"team"`
	FranchiseID string `json:"franchiseId"`
	Owner       string `json:"owner"`
}

//nolint:revive // Preserve the existing exported wire name.
type ResearchArticle struct {
	ID       string `json:"id"`
	Article  string `json:"article"`
	Name     string `json:"name"`
	Headline string `json:"headline"`
}

//nolint:revive // Preserve the existing exported wire name.
type ResearchBoard struct {
	Board     report.Board      `json:"board"`
	News      report.Report     `json:"news"`
	Articles  []ResearchArticle `json:"articles"`
	Players   []ResearchPlayer  `json:"players"`
	Teams     []string          `json:"teams"`
	Positions []string          `json:"positions"`
	Scope     RawResearchScope  `json:"scope"`
}

//nolint:revive // Preserve the existing exported wire name.
type ResearchPlayerReading struct {
	ResearchPlayer
	Report   report.Report     `json:"report"`
	Articles []ResearchArticle `json:"articles"`
	HowLabel string            `json:"howLabel"`
}

func WireBoard(b Board) ResearchBoard {
	var players []ResearchPlayer
	if b.Players != nil {
		players = make([]ResearchPlayer, len(b.Players))
	}
	for i, p := range b.Players {
		players[i] = ResearchPlayer(p)
	}
	return ResearchBoard{Board: b.Board, News: b.News, Articles: WireArticles(b.Articles), Players: players, Teams: b.Teams, Positions: b.Positions, Scope: RawResearchScope(b.Scope)}
}
func WireArticles(in []Article) []ResearchArticle {
	out := make([]ResearchArticle, len(in))
	for i, a := range in {
		out[i] = ResearchArticle(a)
	}
	if in == nil {
		return nil
	}
	return out
}
func WirePlayer(p PlayerReading) ResearchPlayerReading {
	return ResearchPlayerReading{ResearchPlayer: ResearchPlayer(p.PlayerInfo), Report: p.Report, Articles: WireArticles(p.Articles), HowLabel: p.HowLabel}
}

package leagueservice

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// NFLNewsPlayer is one rostered player the news read asks the public feed about.
type NFLNewsPlayer struct {
	Player, Name, Position string
}

// NFLNewsSource reads the held public news for the named players. It never touches a league.
type NFLNewsSource interface {
	ReadNFLNews(ctx context.Context, players []NFLNewsPlayer) (NFLNewsReading, error)
}

// NFLNewsReading is what the public feed holds for the players asked about.
type NFLNewsReading struct {
	AsOf  time.Time
	Stale bool
	Items []NFLNewsItem
}

// NFLNewsItem is one public news line, labelled with the league's player id.
type NFLNewsItem struct {
	At       time.Time `json:"at"`
	Player   string    `json:"player"`
	Name     string    `json:"name"`
	Position string    `json:"position"`
	NFLTeam  string    `json:"nflTeam"`
	Tag      string    `json:"tag"`
	Text     string    `json:"text"`
	Source   string    `json:"source"`
}

// NFLNews is the member-safe news read: public facts for the league's rostered players only.
type NFLNews struct {
	AsOf  *time.Time    `json:"asOf"`
	Stale bool          `json:"stale"`
	Items []NFLNewsItem `json:"items"`
}

// WithNFLNews supplies the public news the hosted server holds.
func (a *Service) WithNFLNews(source NFLNewsSource) *Service {
	a.news = source
	return a
}

// TargetNFLNews reads public player news for the league's rostered players; players is an optional
// comma list of league player ids that narrows the read.
func (a *Service) TargetNFLNews(league, players string) (NFLNews, error) {
	if strings.TrimSpace(league) == "" {
		return NFLNews{}, fmt.Errorf("NFL news: league is required")
	}
	if err := a.admitRead(league, "nfl.news.read", ""); err != nil {
		return NFLNews{}, err
	}
	if err := a.host.Ready(); err != nil {
		return NFLNews{}, fmt.Errorf("NFL news startup: %w", err)
	}
	reading := NFLNews{Items: []NFLNewsItem{}}
	if a.news == nil {
		reading.Stale = true
		return reading, nil
	}
	want := map[string]bool{}
	for _, id := range strings.Split(players, ",") {
		if id = strings.TrimSpace(id); id != "" {
			want[id] = true
		}
	}
	ctx, cancel := context.WithTimeout(a.viewContext(a.host.CommissionerContext()), refreshTimeout)
	defer cancel()
	assets, err := a.host.NativeAssets(ctx, league)
	if err != nil {
		return NFLNews{}, fmt.Errorf("NFL news roster: %w", err)
	}
	var rostered []NFLNewsPlayer
	for _, id := range assets.Identities {
		if id.Franchise == "" || (len(want) > 0 && !want[id.Player]) {
			continue
		}
		rostered = append(rostered, NFLNewsPlayer{Player: id.Player, Name: id.Name, Position: id.Position})
	}
	if len(rostered) == 0 {
		return reading, nil
	}
	held, err := a.news.ReadNFLNews(ctx, rostered)
	if err != nil {
		return NFLNews{}, fmt.Errorf("read NFL news: %w", err)
	}
	if !held.AsOf.IsZero() {
		asOf := held.AsOf.UTC()
		reading.AsOf = &asOf
	}
	reading.Stale = held.Stale
	reading.Items = append(reading.Items, held.Items...)
	return reading, nil
}

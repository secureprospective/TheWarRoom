package offers

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	personalapp "github.com/secureprospective/TheWarRoom/internal/app/offers/personal"

	"github.com/secureprospective/TheWarRoom/internal/app/report"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
	"github.com/secureprospective/TheWarRoom/internal/store/personal"
)

type PersonalReading struct {
	Entries []personal.Entry `json:"entries"`
	Players []PersonalPlayer `json:"players"`
	Bait    report.Report    `json:"bait"`
}

func (a *API) TargetPersonal(mine string) (PersonalReading, error) {
	if err := a.host.Ready(); err != nil {
		return PersonalReading{}, fmt.Errorf("personal: startup: %w", err)
	}
	snap, err := a.personalSnapshot(mine)
	if err != nil {
		return PersonalReading{}, err
	}
	ctx, cancel := context.WithTimeout(a.host.OffersContext(), readTimeout)
	defer cancel()
	entries, err := personal.New(a.host.OffersPools()).Read(ctx, mine)
	if err != nil {
		return PersonalReading{}, fmt.Errorf("personal: %w", err)
	}
	feed := a.host.OffersTradeBait(ctx)
	board := report.Shell("M-116", "League trade bait",
		"MFL's held tradeBait export, beside your private list. The app never submits either list to MFL.",
		feed.FeedState(), len(feed.Value), a.host.OffersZone(), "Franchise", "Offering", "Looking for")
	board.Source = "Source: MFL trade bait · " + report.FeedTime(feed.FeedState(), a.host.OffersZone())
	names, franchises := snapshotNames(snap)
	board.Rows = append(board.Rows, personalapp.BaitRows(feed.Value, names, franchises)...)

	return PersonalReading{Entries: entries, Players: a.personalPlayers(entries), Bait: board}, nil
}

func (a *API) TargetPersonalChange(raw personal.RawChange) error {
	if err := raw.Validate(); err != nil {
		return fmt.Errorf("personal: input: %w", err)
	}
	if err := a.host.Ready(); err != nil {
		return fmt.Errorf("personal: startup: %w", err)
	}
	snap, err := a.personalSnapshot(raw.Mine)
	if err != nil {
		return err
	}
	names, franchises := snapshotNames(snap)
	if raw.Entity == "player" && names[raw.ID] == "" || raw.Entity == "franchise" && franchises[raw.ID] == "" {
		return fmt.Errorf("personal: unknown entity")
	}
	ctx, cancel := context.WithTimeout(a.host.OffersContext(), readTimeout)
	defer cancel()
	if err := personal.New(a.host.OffersPools()).Change(ctx, raw, time.Now()); err != nil {
		return fmt.Errorf("personal: change: %w", err)
	}
	return nil
}

func (a *API) TargetPersonalHandOff(kind string) error {
	options := map[string]string{"watch": "178", "bait": "133", "note": "234", "draft": "129", "ir": "18"}
	option, ok := options[kind]
	if !ok {
		return fmt.Errorf("personal: unknown hand-off")
	}
	if err := a.host.Ready(); err != nil {
		return fmt.Errorf("personal: startup: %w", err)
	}
	target, notes := a.host.OffersRosterTarget(option)
	if target.URL == "" {
		return fmt.Errorf("personal: %s", strings.Join(notes, "; "))
	}
	a.host.OpenBrowser(target.URL)
	return nil
}

type PersonalPlayer struct {
	ID     string   `json:"id"`
	Status string   `json:"status"`
	Market []string `json:"market"`
}

func (a *API) personalPlayers(entries []personal.Entry) []PersonalPlayer {
	provenance, statuses := a.host.OffersInjuries()
	facts := a.host.OffersMarketFacts()
	players := personalapp.Players(entries, provenance.Freshness.State == domain.FreshLive, statuses, provenance.Freshness.FetchedAt, a.host.OffersZone(), func(id string) []string {
		market, ok := facts[id]
		if !ok {
			market = facts[""]
		}
		return MarketLines(market)
	})
	out := make([]PersonalPlayer, len(players))
	for i, p := range players {
		out[i] = PersonalPlayer(p)
	}
	return out
}

func snapshotNames(snap snapshot.Snapshot) (map[string]string, map[string]string) {
	return personalapp.Names(snap)
}

func (a *API) personalSnapshot(mine string) (snapshot.Snapshot, error) {
	players, franchises, loaded := a.host.OffersSnapshotNames()
	snap := snapshot.Snapshot{Players: snapshot.Sourced[[]snapshot.Player]{Value: players}, Franchises: snapshot.Sourced[[]snapshot.Franchise]{Value: franchises}}
	if !loaded {
		return snap, fmt.Errorf("personal: snapshot not loaded")
	}
	if !slices.ContainsFunc(snap.Franchises.Value, func(f snapshot.Franchise) bool { return f.ID == mine }) {
		return snap, fmt.Errorf("personal: unknown franchise")
	}
	return snap, nil
}

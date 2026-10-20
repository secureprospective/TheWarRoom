package main

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	personalapp "github.com/secureprospective/TheWarRoom/internal/app/personal"

	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
	"github.com/secureprospective/TheWarRoom/internal/store/personal"
)

type PersonalReading struct {
	Entries []personal.Entry `json:"entries"`
	Players []PersonalPlayer `json:"players"`
	Bait    Report           `json:"bait"`
}

func (a *App) personalSnapshot(mine string) (snapshot.Snapshot, error) {
	a.targetSnapshotMu.Lock()
	snap, loaded := a.targetSnapshot, a.hasTargetSnapshot
	a.targetSnapshotMu.Unlock()
	if !loaded {
		return snap, fmt.Errorf("personal: snapshot not loaded")
	}
	if !slices.ContainsFunc(snap.Franchises.Value, func(f snapshot.Franchise) bool { return f.ID == mine }) {
		return snap, fmt.Errorf("personal: unknown franchise")
	}
	return snap, nil
}

func (a *App) TargetPersonal(mine string) (PersonalReading, error) {
	if err := a.ready(); err != nil {
		return PersonalReading{}, fmt.Errorf("personal: startup: %w", err)
	}
	snap, err := a.personalSnapshot(mine)
	if err != nil {
		return PersonalReading{}, err
	}
	ctx, cancel := context.WithTimeout(a.ctx, m4Timeout)
	defer cancel()
	entries, err := personal.New(a.pools).Read(ctx, mine)
	if err != nil {
		return PersonalReading{}, fmt.Errorf("personal: %w", err)
	}
	feed := readShellFeed(ctx, a, &a.seasonTradeBait, "tradeBait", ingestion.ParseTradeBait)
	report := reportShell("M-116", "League trade bait",
		"MFL's held tradeBait export, beside your private list. The app never submits either list to MFL.",
		feed, time.Local, "Franchise", "Offering", "Looking for")
	report.Source = "Source: MFL trade bait · " + shellFeedTime(feed, time.Local)
	names, franchises := snapshotNames(snap)
	report.Rows = append(report.Rows, wireRows(personalapp.BaitRows(feed.value, names, franchises))...)

	return PersonalReading{Entries: entries, Players: a.personalPlayers(entries), Bait: report}, nil
}

func (a *App) TargetPersonalChange(raw personal.RawChange) error {
	if err := raw.Validate(); err != nil {
		return fmt.Errorf("personal: input: %w", err)
	}
	if err := a.ready(); err != nil {
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
	ctx, cancel := context.WithTimeout(a.ctx, m4Timeout)
	defer cancel()
	if err := personal.New(a.pools).Change(ctx, raw, time.Now()); err != nil {
		return fmt.Errorf("personal: change: %w", err)
	}
	return nil
}

func (a *App) TargetPersonalHandOff(kind string) error {
	options := map[string]string{"watch": "178", "bait": "133", "note": "234", "draft": "129", "ir": "18"}
	option, ok := options[kind]
	if !ok {
		return fmt.Errorf("personal: unknown hand-off")
	}
	if err := a.ready(); err != nil {
		return fmt.Errorf("personal: startup: %w", err)
	}
	target, notes := a.rosterMoveTarget(option)
	if target.URL == "" {
		return fmt.Errorf("personal: %s", strings.Join(notes, "; "))
	}
	a.openURL(a.ctx, target.URL)
	return nil
}

// archivedOnly reads already held exports; opening a list or the clock never calls MFL.
func archivedOnly[T any](ctx context.Context, a *App, export string, leagueless bool,
	parse func([]byte) ([]T, error)) heldSeasonFeed[[]T] {
	feed := heldSeasonFeed[[]T]{}
	if a.history == nil {
		feed.refreshError = "No held archive"
		return feed
	}
	found, err := a.history.ReferenceBodies(ctx, "TYPE="+export, func(src string, body []byte, at time.Time) bool {
		if !sameExport(src, export, a.season, leagueless) {
			return false
		}
		rows, err := parse(body)
		if err != nil {
			return false
		}
		feed.value, feed.fetchedAt = rows, at
		return true
	})
	if err != nil {
		feed.refreshError = "Held archive unreadable: " + err.Error()
	} else if !found {
		feed.refreshError = "No usable held " + export + " export"
	}
	return feed
}

type PersonalPlayer struct {
	ID     string   `json:"id"`
	Status string   `json:"status"`
	Market []string `json:"market"`
}

func (a *App) personalPlayers(entries []personal.Entry) []PersonalPlayer {
	injuries := a.heldInjuries()
	facts := marketFacts(a.heldMarket())
	players := personalapp.Players(entries, injuries.Provenance.Freshness.State == FreshLive, injuries.Status, injuries.Provenance.Freshness.FetchedAt, time.Local, func(id string) []string {
		market, ok := facts[id]
		if !ok {
			market = facts[""]
		}
		return poolMarketLines(market)
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

package main

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

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
	feed := archivedOnly(ctx, a, "tradeBait", false, ingestion.ParseTradeBait)
	report := reportShell("M-116", "League trade bait",
		"MFL's held tradeBait export, beside your private list. The app never submits either list to MFL.",
		feed, time.Local, "Franchise", "Offering", "Looking for")
	report.Source = "Source: held MFL tradeBait archive"
	names, franchises := snapshotNames(snap)
	for _, bait := range feed.value {
		offered := []string{}
		for _, id := range strings.Split(bait.Give, ",") {
			if id != "" {
				canonical, _ := ingestion.ValidatePlayerID(id)
				offered = append(offered, knownName(names, canonical.String()))
			}
		}
		report.Rows = append(report.Rows, ReportRow{Cells: []string{knownName(franchises, bait.Franchise), strings.Join(offered, "\n"), bait.Want}})
	}
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

func snapshotNames(snap snapshot.Snapshot) (map[string]string, map[string]string) {
	names, franchises := map[string]string{}, map[string]string{}
	for _, p := range snap.Players.Value {
		names[p.ID.String()] = p.Name
	}
	for _, f := range snap.Franchises.Value {
		franchises[f.ID] = f.Name
	}
	return names, franchises
}
func knownName(names map[string]string, id string) string {
	if name := names[id]; name != "" {
		return name
	}
	return "Name unavailable"
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
	out := []PersonalPlayer{}
	seen := map[string]bool{}
	for _, entry := range entries {
		if entry.Entity != "player" || seen[entry.ID] {
			continue
		}
		seen[entry.ID] = true
		status := "NFL status unavailable"
		if injuries.Provenance.Freshness.State == FreshLive {
			status = injuries.Status[entry.ID]
			if status == "" {
				status = "No injury designation"
			}
			if at, err := time.Parse(time.RFC3339, injuries.Provenance.Freshness.FetchedAt); err == nil {
				status += " · as of " + reportTime(at, time.Local)
			}
		}
		market, ok := facts[entry.ID]
		if !ok {
			market = facts[""]
		}
		out = append(out, PersonalPlayer{ID: entry.ID, Status: status, Market: poolMarketLines(market)})
	}
	return out
}

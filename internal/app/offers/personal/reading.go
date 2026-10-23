// Package personal shapes the owner's personal board: player names, trade bait and watched players.
package personal

import (
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/report"
	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
	"github.com/secureprospective/TheWarRoom/internal/store/personal"
)

func Names(snap snapshot.Snapshot) (map[string]string, map[string]string) {
	names, franchises := map[string]string{}, map[string]string{}
	for _, p := range snap.Players.Value {
		names[p.ID.String()] = p.Name
	}
	for _, f := range snap.Franchises.Value {
		franchises[f.ID] = f.Name
	}
	return names, franchises
}

func KnownName(names map[string]string, id string) string {
	if name := names[id]; name != "" {
		return name
	}
	return "Name unavailable"
}
func BaitRows(rows []ingestion.RawTradeBait, names, franchises map[string]string) []report.Row {
	out := []report.Row{}
	for _, bait := range rows {
		offered := []string{}
		for _, id := range strings.Split(bait.Give, ",") {
			if id != "" {
				canonical, _ := ingestion.ValidatePlayerID(id)
				offered = append(offered, KnownName(names, canonical.String()))
			}
		}
		out = append(out, report.Row{Cells: []string{KnownName(franchises, bait.Franchise), strings.Join(offered, "\n"), bait.Want}})
	}
	return out
}

type Player struct {
	ID     string   `json:"id"`
	Status string   `json:"status"`
	Market []string `json:"market"`
}

func Players(entries []personal.Entry, live bool, statuses map[string]string, fetchedAt string, zone *time.Location, marketLines func(string) []string) []Player {
	out := []Player{}
	seen := map[string]bool{}
	for _, entry := range entries {
		if entry.Entity != "player" || seen[entry.ID] {
			continue
		}
		seen[entry.ID] = true
		status := "NFL status unavailable"
		if live {
			status = statuses[entry.ID]
			if status == "" {
				status = "No injury designation"
			}
			if at, err := time.Parse(time.RFC3339, fetchedAt); err == nil {
				status += " · as of " + report.Time(at, zone)
			}
		}

		out = append(out, Player{ID: entry.ID, Status: status, Market: marketLines(entry.ID)})
	}
	return out
}

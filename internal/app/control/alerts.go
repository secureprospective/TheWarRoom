package control

import (
	"cmp"
	"fmt"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/leagueclock"
	"github.com/secureprospective/TheWarRoom/internal/lineup"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
)

type Alert struct {
	Kind       string              `json:"kind"`
	Urgency    string              `json:"urgency"`
	Title      string              `json:"title"`
	Detail     string              `json:"detail"`
	Node       string              `json:"node"`
	Workspace  string              `json:"workspace"`
	At         time.Time           `json:"at"`
	Provenance snapshot.Provenance `json:"provenance"`
	PlayerID   string              `json:"playerId,omitempty"`
	MFLURL     string              `json:"mflUrl,omitempty"`
}

func CompareAlerts(a, b Alert) int {
	if order := cmp.Compare(b.Urgency, a.Urgency); order != 0 {
		return order
	}
	gravity := func(alert Alert) int {
		if alert.Kind == "ir" {
			return 2
		}
		return 1
	}
	if order := cmp.Compare(gravity(b), gravity(a)); order != 0 {
		return order
	}
	return cmp.Compare(a.Kind+":"+a.PlayerID, b.Kind+":"+b.PlayerID)
}

func OffersTitle(count int) string {
	if count == 1 {
		return "1 offer to you"
	}
	return fmt.Sprintf("%d offers to you", count)
}

func AlertUrgency(now, at time.Time) string {
	reading := leagueclock.Clock(now, leagueclock.Inputs{Events: []leagueclock.Event{{
		Status: "PLANNED", At: at,
	}}})
	return string(reading.Deadlines[0].Urgency)
}

func ByeAlert(short lineup.ByeShortfall, players []lineup.ByePlayer, names []string, byes map[string]int,
	now, lock time.Time, provenance snapshot.Provenance) Alert {
	positions := map[domain.Position]string{domain.PosQB: "quarterback", domain.PosRB: "running back",
		domain.PosWR: "wide receiver", domain.PosTE: "tight end", domain.PosK: "kicker",
		domain.PosDE: "defensive end", domain.PosDT: "defensive tackle", domain.PosLB: "linebacker",
		domain.PosCB: "cornerback", domain.PosS: "safety"}
	position := positions[short.Position]
	title := fmt.Sprintf("Week %d: %s shortfall", short.Week, position)
	if short.Available == 0 {
		title = fmt.Sprintf("Week %d: no %s available", short.Week, position)
	}
	onBye := []string{}
	for i, p := range players {
		if p.Active && p.Position == short.Position && byes[p.Team] == short.Week {
			name := names[i]
			if name == "" {
				name = "Player name unavailable"
			}
			onBye = append(onBye, name)
		}
	}
	detail := fmt.Sprintf("%d available; needs %d. Checks position minimums only.", short.Available, short.Required)
	if len(onBye) > 0 {
		detail += " On bye: " + strings.Join(onBye, ", ") + "."
	}
	return Alert{Kind: "bye", Urgency: AlertUrgency(now, lock), Title: title, Detail: detail,
		Node: "hq", Workspace: "lineup-and-roster", At: lock, Provenance: provenance}
}

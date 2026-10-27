// Package native is the native league's pure logic, moved out of package main: the football facts a
// native check reads, the owner's lineup request, the NFL week a bid is judged in and the starting
// snapshot shaped from held data. Who is calling, the action gate and every ledger write stay at the
// root adapters; nothing here reads a session, takes a lock or opens a database.
package native

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
	"github.com/secureprospective/TheWarRoom/internal/normalize"
)

// mflNoDraftYear is the year MFL fills in where it holds no draft year at all.
const mflNoDraftYear = 1970

// Lookup is the held players directory the facts are read from; the caller holds its lock.
type Lookup interface {
	IDs() []string
	Facts(id string) (normalize.PlayerFacts, bool)
}

// PlayerFacts reads positions and draft years from the held players directory, plus the O-Line cards,
// which no feed lists. A fact that is not held is left unmarked as held, so a check says it cannot tell
// rather than guessing.
func PlayerFacts(lookup Lookup) nativecheck.Facts {
	facts := nativecheck.Facts{}
	for _, id := range lookup.IDs() {
		if f, ok := lookup.Facts(id); ok {
			facts[id] = nativecheck.PlayerFacts{Name: f.Name, Position: Position(f.Position), Team: f.Team,
				DraftYear: f.DraftYear, DraftYearHeld: f.HasDraftYear && f.DraftYear > mflNoDraftYear}
		}
	}
	nativecheck.AddLineCards(facts)
	return facts
}

// Designate marks every player's NFL listing as held, from the held injuries feed's statuses by id.
func Designate(facts nativecheck.Facts, listed map[string]string) {
	for id, f := range facts {
		f.DesignationHeld, f.Designation = true, Designation(listed[id])
		facts[id] = f
	}
}

// Position maps the engine's position onto the league rules' position codes.
func Position(p domain.Position) leaguerules.Position {
	switch p {
	case domain.PosK:
		return leaguerules.PK
	case domain.PosFlag:
		return ""
	case domain.PosQB, domain.PosRB, domain.PosWR, domain.PosTE, domain.PosDE, domain.PosDT, domain.PosLB,
		domain.PosCB, domain.PosS:
	}
	return leaguerules.Position(p)
}

// Designation reads MFL's injury vocabulary as the NFL listings a league can accept for IR. A listing the
// menu does not name is kept as MFL words it, so the check can say what it saw.
func Designation(status string) leaguerules.NFLStatus {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "":
		return ""
	case "ir", "ir-r":
		return leaguerules.InjuredReserve
	case "ir-pup":
		return leaguerules.PUP
	case "ir-nfi":
		return leaguerules.NFI
	case "out":
		return leaguerules.Out
	case "suspended":
		return leaguerules.Suspended
	}
	return leaguerules.NFLStatus(strings.TrimSpace(status))
}

// LineupSet is an owner's request to set the starters of their own franchise for a week.
type LineupSet struct {
	League    string   `json:"league"`
	Franchise string   `json:"franchise"`
	Season    int      `json:"season"`
	Week      int      `json:"week"`
	Starters  []string `json:"starters"`
	Expected  int64    `json:"expected"`
	Request   string   `json:"request"`
}

// DecodeLineupSet reads exactly one lineup request with no unknown fields, for an app league.
func DecodeLineupSet(body []byte) (LineupSet, error) {
	var in LineupSet
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&in); err != nil {
		return in, fmt.Errorf("read the lineup: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return in, fmt.Errorf("send exactly one lineup")
	}
	if !strings.HasPrefix(in.League, "native.") || in.League == "native." || strings.TrimSpace(in.League) != in.League ||
		strings.TrimSpace(in.Request) == "" || in.Expected < 0 {
		return in, fmt.Errorf("choose an app league and a valid request and league version")
	}
	return in, nil
}

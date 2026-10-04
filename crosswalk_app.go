package main

import (
	"context"
	"fmt"
	"log"
	"maps"
	"net/http"
	"slices"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/crosswalk"
	"github.com/secureprospective/TheWarRoom/internal/normalize"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

// dynastyProcessSource is the crosswalk's id in sources.csv; its loads are the directory's.
const dynastyProcessSource = "dynastyprocess"

// matchIDType is the id a match is measured on: nflverse, the production source, keys on gsis.
const matchIDType = "gsis"

// Why a player has no gsis link, beyond the crosswalk's own guards.
const (
	reasonNotInDP = "not in DynastyProcess (commissioner-created, or too new)"
	reasonNoGSIS  = "no NFL id yet"
)

// reportPositions is the order the league's ten positions are reported in.
func reportPositions() []domain.Position {
	return []domain.Position{domain.PosQB, domain.PosRB, domain.PosWR, domain.PosTE, domain.PosK,
		domain.PosDT, domain.PosDE, domain.PosLB, domain.PosCB, domain.PosS}
}

// CrosswalkRate is one position's match rate, rostered and in the free-agent pool.
type CrosswalkRate struct {
	Position          string `json:"position"`
	Rostered          int    `json:"rostered"`
	RosteredMatched   int    `json:"rosteredMatched"`
	FreeAgents        int    `json:"freeAgents"`
	FreeAgentsMatched int    `json:"freeAgentsMatched"`
}

// CrosswalkMiss is one rostered player with no NFL id link, and why.
type CrosswalkMiss struct {
	MFLID     string `json:"mflId"`
	Name      string `json:"name"`
	Position  string `json:"position"`
	Franchise string `json:"franchise"`
	Reason    string `json:"reason"`
}

// CrosswalkReport is the latest crosswalk load: links written and the match rates they give.
type CrosswalkReport struct {
	OK        bool            `json:"ok"`
	Error     string          `json:"error"`
	LoadedAt  string          `json:"loadedAt"`
	Links     int             `json:"links"`
	Promoted  int             `json:"promoted"`
	Rates     []CrosswalkRate `json:"rates"`
	Total     CrosswalkRate   `json:"total"`
	Unmatched []CrosswalkMiss `json:"unmatched"`
}

// LoadCrosswalk fetches DynastyProcess, links it into the player directory and reports the match
// rates. ScoreLeague does the same with the crosswalk it already fetches.
func (a *App) LoadCrosswalk() CrosswalkReport {
	if err := a.ready(); err != nil {
		return CrosswalkReport{Error: err.Error()}
	}
	ctx, cancel := context.WithTimeout(a.ctx, rasFetchTimeout)
	defer cancel()
	lk, err := a.directory(ctx)
	if err != nil {
		return CrosswalkReport{Error: err.Error()}
	}
	cw, err := a.fetchCrosswalk(ctx)
	if err != nil {
		return CrosswalkReport{Error: err.Error()}
	}
	rep, err := a.linkCrosswalk(ctx, cw, lk)
	if err != nil {
		return CrosswalkReport{Error: err.Error()}
	}
	return rep
}

// GetCrosswalkReport returns the latest load's report, held since this launch.
func (a *App) GetCrosswalkReport() CrosswalkReport {
	if err := a.ready(); err != nil {
		return CrosswalkReport{Error: err.Error()}
	}
	a.crosswalkMu.Lock()
	defer a.crosswalkMu.Unlock()
	return a.crosswalk
}

// fetchCrosswalk reads DynastyProcess through the archive. A failure is recorded against the
// source, so its health shows it.
func (a *App) fetchCrosswalk(ctx context.Context) (crosswalk.Map, error) {
	client := &http.Client{Timeout: rasFetchTimeout, Transport: a.fetches}
	cw, err := crosswalk.Fetch(ctx, client, crosswalk.SourceURL)
	if err != nil {
		err = fmt.Errorf("app: fetch crosswalk: %w", err)
		if lerr := a.history.LoadFailed(ctx, dynastyProcessSource, err); lerr != nil {
			log.Printf("the war room: %v", lerr)
		}
		return crosswalk.Map{}, err
	}
	return cw, nil
}

// linkCrosswalk writes the crosswalk's links to the player directory and holds the report.
func (a *App) linkCrosswalk(ctx context.Context, cw crosswalk.Map, lk normalize.Lookup) (CrosswalkReport, error) {
	links, rejected := cw.Links(lk.Name)
	rows := make([]history.PlayerIDLink, len(links))
	for i, l := range links {
		rows[i] = history.PlayerIDLink{IDType: l.IDType, IDValue: l.IDValue, PlayerID: l.MFLID}
	}
	promoted, err := a.history.LinkPlayerIDs(ctx, dynastyProcessSource, rows)
	if err != nil {
		return CrosswalkReport{}, fmt.Errorf("app: link crosswalk: %w", err)
	}
	linked, err := a.history.LinkedPlayers(ctx, matchIDType)
	if err != nil {
		return CrosswalkReport{}, fmt.Errorf("app: link crosswalk: %w", err)
	}
	rep := crosswalkReport(rosteredBy(a.league.Reader()), lk, linked, cw, rejected, a.rulebook.FranchiseNames())
	rep.OK, rep.Links, rep.Promoted = true, len(links), promoted
	rep.LoadedAt = time.Now().Format(time.RFC3339)
	log.Printf("the war room: crosswalk: %d links, %d waiting facts resolved; rostered %d of %d matched, free agents %d of %d",
		rep.Links, rep.Promoted, rep.Total.RosteredMatched, rep.Total.Rostered, rep.Total.FreeAgentsMatched, rep.Total.FreeAgents)
	a.crosswalkMu.Lock()
	a.crosswalk = rep
	a.crosswalkMu.Unlock()
	return rep, nil
}

// rosteredBy maps every rostered MFL id to its franchise.
func rosteredBy(r state.Reader) map[string]string {
	out := map[string]string{}
	for _, fid := range r.Franchises() {
		roster, _ := r.Roster(fid)
		for _, p := range roster {
			out[p.MFLID] = fid
		}
	}
	return out
}

// crosswalkReport measures the match rate per position for the rostered players and for the
// free-agent pool (MFL's list minus the rosters), and lists every unmatched rostered player with
// one reason.
func crosswalkReport(rostered map[string]string, lk normalize.Lookup, linked map[string]bool, cw crosswalk.Map,
	rejected map[string]string, names map[string]string) CrosswalkReport {
	rates := map[domain.Position]*CrosswalkRate{}
	rate := func(pos domain.Position) *CrosswalkRate {
		if rates[pos] == nil {
			rates[pos] = &CrosswalkRate{Position: string(pos)}
		}
		return rates[pos]
	}
	var rep CrosswalkReport
	for _, id := range slices.Sorted(maps.Keys(rostered)) {
		f, _ := lk.Facts(id)
		r := rate(f.Position)
		r.Rostered++
		if linked[id] {
			r.RosteredMatched++
			continue
		}
		rep.Unmatched = append(rep.Unmatched, CrosswalkMiss{MFLID: id, Name: f.Name, Position: string(f.Position),
			Franchise: domain.FranchiseLabel(names, rostered[id]), Reason: missReason(id, cw, rejected)})
	}
	for _, id := range lk.IDs() {
		f, _ := lk.Facts(id)
		if _, ok := rostered[id]; ok || f.Position == domain.PosFlag {
			continue
		}
		r := rate(f.Position)
		r.FreeAgents++
		if linked[id] {
			r.FreeAgentsMatched++
		}
	}
	rep.Total.Position = "ALL"
	order := append(reportPositions(), extraPositions(rates)...)
	for _, pos := range order {
		r, ok := rates[pos]
		if !ok {
			continue
		}
		rep.Rates = append(rep.Rates, *r)
		rep.Total.Rostered += r.Rostered
		rep.Total.RosteredMatched += r.RosteredMatched
		rep.Total.FreeAgents += r.FreeAgents
		rep.Total.FreeAgentsMatched += r.FreeAgentsMatched
	}
	return rep
}

// extraPositions lists any reported position outside the league's ten (a flagged or unknown
// rostered player), sorted, so nothing counted goes unshown.
func extraPositions(rates map[domain.Position]*CrosswalkRate) []domain.Position {
	var out []domain.Position
	for pos := range rates {
		if !slices.Contains(reportPositions(), pos) {
			out = append(out, pos)
		}
	}
	slices.Sort(out)
	return out
}

// missReason says why a rostered player has no gsis link.
func missReason(id string, cw crosswalk.Map, rejected map[string]string) string {
	if why, ok := rejected[id]; ok {
		return why
	}
	if _, ok := cw.Entry(id); ok {
		return reasonNoGSIS
	}
	return reasonNotInDP
}

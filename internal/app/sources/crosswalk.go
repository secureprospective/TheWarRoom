package sources

import (
	"context"
	"fmt"
	"log"
	"maps"
	"net/http"
	"slices"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/ingestion/players"

	"github.com/secureprospective/TheWarRoom/internal/store/history"

	"github.com/secureprospective/TheWarRoom/internal/app/report"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/crosswalk"
	"github.com/secureprospective/TheWarRoom/internal/normalize"
)

const matchIDType = "gsis"
const reasonNotInDP = "not in DynastyProcess (commissioner-created, or too new)"
const reasonNoGSIS = "no NFL id yet"

type CrosswalkRate struct {
	Position          string `json:"position"`
	Rostered          int    `json:"rostered"`
	RosteredMatched   int    `json:"rosteredMatched"`
	FreeAgents        int    `json:"freeAgents"`
	FreeAgentsMatched int    `json:"freeAgentsMatched"`
}

type CrosswalkMiss struct {
	MFLID     string `json:"mflId"`
	Name      string `json:"name"`
	Team      string `json:"team"`
	Position  string `json:"position"`
	Franchise string `json:"franchise"`
	Reason    string `json:"reason"`
}

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

func reportPositions() []domain.Position {
	return []domain.Position{domain.PosQB, domain.PosRB, domain.PosWR, domain.PosTE, domain.PosK,
		domain.PosDT, domain.PosDE, domain.PosLB, domain.PosCB, domain.PosS}
}

func BuildCrosswalk(rostered map[string]string, lk normalize.Lookup, linked map[string]bool, cw crosswalk.Map,
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
		rep.Unmatched = append(rep.Unmatched, CrosswalkMiss{MFLID: id, Name: f.Name, Team: f.Team, Position: string(f.Position),
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

func missReason(id string, cw crosswalk.Map, rejected map[string]string) string {
	if why, ok := rejected[id]; ok {
		return why
	}
	if _, ok := cw.Entry(id); ok {
		return reasonNoGSIS
	}
	return reasonNotInDP
}

func CrosswalkReading(c CrosswalkReport, zone *time.Location) report.Report {
	r := report.Report{ID: "C-08", Title: "Player crosswalk", Status: "ok", Source: "Source: DynastyProcess", AsOf: "As of: this launch",
		Columns: []string{"Position", "Rostered matched", "Free agents matched"}, Rows: []report.Row{},
		How: "DynastyProcess links MFL players to NFL ids. Unmatched rostered players are listed with the reason; their facts wait for a link."}
	if !c.OK {
		r.Status, r.Note = "empty", "No crosswalk loaded since launch. Refresh the Player crosswalk scope to load it."
		if c.Error != "" {
			r.Status, r.Note = "failed", c.Error
		}
		return r
	}
	r.AsOf = "Loaded: " + LabTime(c.LoadedAt, zone)
	r.Note = fmt.Sprintf("%s links; %s waiting facts resolved.", Thousands(c.Links), Thousands(c.Promoted))
	for _, rate := range append(c.Rates, c.Total) {
		r.Rows = append(r.Rows, report.Row{Cells: []string{rate.Position, crosswalkShare(rate.RosteredMatched, rate.Rostered), crosswalkShare(rate.FreeAgentsMatched, rate.FreeAgents)}})
	}
	for _, m := range c.Unmatched {
		r.Rows = append(r.Rows, report.Row{Group: "Unmatched rostered players", Cells: []string{m.Name, m.Position + " · " + nflTeamText(m.Team), m.Reason}})
	}
	return r
}

func crosswalkShare(matched, total int) string {
	if total == 0 {
		return "None"
	}
	return fmt.Sprintf("%s of %s · %.1f%%", Thousands(matched), Thousands(total), 100*float64(matched)/float64(total))
}

func nflTeamText(team string) string {
	if team == "" {
		return "NFL team not held"
	}
	return team
}

const DynastyProcessSource = "dynastyprocess"

func (a Service) FetchCrosswalk(ctx context.Context) (crosswalk.Map, error) {
	client := &http.Client{Timeout: RASFetchTimeout, Transport: a.Transport}
	cw, err := crosswalk.Fetch(ctx, client, crosswalk.SourceURL)
	if err != nil {
		err = fmt.Errorf("app: fetch crosswalk: %w", err)
		if lerr := a.Crosswalk.LoadFailed(ctx, DynastyProcessSource, err); lerr != nil {
			log.Printf("the war room: %v", lerr)
		}
		return crosswalk.Map{}, err
	}
	return cw, nil
}
func (a Service) LinkCrosswalk(ctx context.Context, cw crosswalk.Map, lk normalize.Lookup, rostered func() map[string]string, names func() map[string]string) (CrosswalkReport, error) {
	links, rejected := cw.Links(lk.Name)
	rows := make([]history.PlayerIDLink, len(links))
	for i, l := range links {
		rows[i] = history.PlayerIDLink{IDType: l.IDType, IDValue: l.IDValue, PlayerID: l.MFLID}
	}
	promoted, err := a.Crosswalk.LinkPlayerIDs(ctx, DynastyProcessSource, rows)
	if err != nil {
		return CrosswalkReport{}, fmt.Errorf("app: link crosswalk: %w", err)
	}
	linked, err := a.Crosswalk.LinkedPlayers(ctx, matchIDType)
	if err != nil {
		return CrosswalkReport{}, fmt.Errorf("app: link crosswalk: %w", err)
	}
	rep := BuildCrosswalk(rostered(), lk, linked, cw, rejected, names())
	rep.OK, rep.Links, rep.Promoted = true, len(links), promoted
	rep.LoadedAt = time.Now().Format(time.RFC3339)
	log.Printf("the war room: crosswalk: %d links, %d waiting facts resolved; rostered %d of %d matched, free agents %d of %d",
		rep.Links, rep.Promoted, rep.Total.RosteredMatched, rep.Total.Rostered, rep.Total.FreeAgentsMatched, rep.Total.FreeAgents)
	return rep, nil
}

// Directory publishes only a completely normalized player lookup. Caller owns its lookup lock.
func Directory(ctx context.Context, held normalize.Lookup, loaded bool,
	fetch func(context.Context) ([]players.RawPlayer, error), publish func(normalize.Lookup, time.Time)) (normalize.Lookup, error) {
	if loaded {
		return held, nil
	}
	raw, err := fetch(ctx)
	if err != nil {
		return normalize.Lookup{}, fmt.Errorf("app: fetch players db: %w", err)
	}
	lookup, err := normalize.NewLookup(raw)
	if err != nil {
		return normalize.Lookup{}, fmt.Errorf("app: build players lookup: %w", err)
	}
	publish(lookup, time.Now().UTC())
	return lookup, nil
}

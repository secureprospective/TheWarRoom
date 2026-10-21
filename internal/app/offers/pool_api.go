package offers

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	poolapp "github.com/secureprospective/TheWarRoom/internal/app/offers/pool"
	"github.com/secureprospective/TheWarRoom/internal/app/report"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/normalize"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
)

type PoolRow struct {
	Engine      *float64                `json:"engine"`
	PPG         *float64                `json:"ppg"`
	Lab         *float64                `json:"lab"`
	EngineText  string                  `json:"engineText"`
	PPGText     string                  `json:"ppgText"`
	LabText     string                  `json:"labText"`
	MarketLines []string                `json:"marketLines"`
	Market      []report.MarketFact     `json:"market"`
	ID          string                  `json:"id"`
	Name        string                  `json:"name"`
	Position    string                  `json:"position"`
	Team        string                  `json:"team"`
	Age         string                  `json:"age"`
	Experience  string                  `json:"experience"`
	Minimum     snapshot.ContractAmount `json:"minimum"`
	Plan        string                  `json:"plan"`   // the owner's open bid plan on him, in words; else empty
	PlanID      string                  `json:"planId"` // that plan's correlation id; else empty
}

type PoolSort struct {
	Key   string `json:"key"`
	Label string `json:"label"`
}

func PoolSorts() []PoolSort {
	return []PoolSort{{"name", "Player"}, {"position", "Pos"}, {"team", "NFL team"},
		{"engine", "Engine points"}, {"ppg", "This season points per game"}, {"lab", "Lab dynasty value"}, {"minimum", "Minimum"},
		{"projected", "Projected points"}, {"rank", "MFL rank"}, {"adp", "ADP"}, {"aav", "AAV"}}
}

type Pool struct {
	EngineNote string     `json:"engineNote"`
	PPGNote    string     `json:"ppgNote"`
	Sorts      []PoolSort `json:"sorts"`
	Rows       []PoolRow  `json:"rows"`
	Positions  []string   `json:"positions"`
	Summary    string     `json:"summary"`
}

// RawPoolScope is the pool's filter and sort. FranchiseID only picks whose bid plans show.
type RawPoolScope struct {
	ShowNoTeam  bool   `json:"showNoTeam"`
	FranchiseID string `json:"franchiseId"`
	Position    string `json:"position"`
	Sort        string `json:"sort"`
	Desc        bool   `json:"desc"`
}

func (r RawPoolScope) Validate() error {
	if !slices.ContainsFunc(PoolSorts(), func(s PoolSort) bool { return s.Key == r.Sort }) {
		return fmt.Errorf("pool scope: unknown sort %q", r.Sort)
	}
	if r.FranchiseID != "" {
		if err := ingestion.FeedFranchise(r.FranchiseID); err != nil {
			return fmt.Errorf("pool scope: %w", err)
		}
	}
	if len(r.Position) > 8 {
		return fmt.Errorf("pool scope: position too long")
	}
	return nil
}

func poolReading(value Pool, p snapshot.Provenance, status report.ReadingStatus, reason, how, stamp string) report.Reading[Pool] {
	if p.Kind == "" {
		p = snapshot.Provenance{Source: "MFL freeAgents", Kind: "live", Freshness: domain.Freshness{State: domain.FreshFail}}
	}
	r := report.Reading[Pool]{
		Sourced: snapshot.Sourced[Pool]{Value: value, Provenance: p},
		Status:  status, Count: len(value.Rows), Baseline: "Latest MFL free-agent list",
		Cutoff: p.Freshness.FetchedAt, Method: "MFL freeAgents export; §6 minimum by experience, computed in Go",
		BuildStamp: stamp, HowCounted: how,
	}
	if reason != "" {
		r.AbsenceReason = &reason
	}
	return r
}

const poolHow = "Every player MFL lists as a free agent, joined to the held players directory for name, " +
	"position, NFL team, age and draft year. Experience is this season minus the draft year; a player " +
	"with no real draft year is shown at the rookie minimum and marked. The minimum is the Official " +
	"Rulebook §6 table by experience. Free agents the directory does not know (team units) are left out."

// TargetPool reads the held free-agent list: no network. Open the pool with TargetPoolRefresh
// first; a list never fetched is a failed reading, not an empty one.
func (a *API) TargetPool(raw RawPoolScope) report.Reading[Pool] {
	empty := Pool{Rows: []PoolRow{}, Positions: []string{}, Sorts: PoolSorts()}
	if err := raw.Validate(); err != nil {
		return poolReading(empty, snapshot.Provenance{}, report.ReadingFailed, err.Error(), poolHow, a.host.OffersBuildStamp())
	}
	if err := a.host.Ready(); err != nil {
		return poolReading(empty, snapshot.Provenance{}, report.ReadingFailed, err.Error(), poolHow, a.host.OffersBuildStamp())
	}
	feed := a.host.OffersFreeAgents()
	p := feed.Provenance("MFL freeAgents", "free agents")
	if feed.FetchedAt.IsZero() {
		reason := "Free agents have not been fetched from MFL yet"
		if feed.RefreshError != "" {
			reason = "MFL free agents unavailable: " + feed.RefreshError
		}
		return poolReading(empty, p, report.ReadingFailed, reason, poolHow, a.host.OffersBuildStamp())
	}
	ctx, cancel := context.WithTimeout(a.host.OffersContext(), readTimeout)
	defer cancel()
	lk, err := a.host.OffersDirectory(ctx)
	if err != nil {
		return poolReading(empty, p, report.ReadingFailed, "Player names unavailable: "+err.Error(), poolHow, a.host.OffersBuildStamp())
	}
	value, left := a.poolRows(ctx, feed.Value, lk, raw)
	if len(feed.Value) == 0 {
		return poolReading(value, p, report.ReadingThin, "MFL lists no free agents right now", poolHow, a.host.OffersBuildStamp())
	}
	how := poolHow
	if left > 0 {
		how += fmt.Sprintf(" Left out of this list: %d ids the directory does not know.", left)
	}
	return poolReading(value, p, report.ReadingOK, "", how, a.host.OffersBuildStamp())
}

// poolRows builds, filters and sorts the rows, and counts the ids the directory does not know.
func (a *API) poolRows(ctx context.Context, ids []playerid.PlayerID, lk normalize.Lookup, raw RawPoolScope) (Pool, int) {
	plans := a.openBidPlansByPlayer(ctx, raw.FranchiseID)
	season, now := a.host.OffersSeason(), time.Now()
	all, seen, left := []PoolRow{}, map[string]bool{}, 0
	facts := a.host.OffersMarketFacts()
	values, engineNote := a.host.OffersEngineValues(ctx)
	assets, ppgNote := a.host.OffersAssetValues(ctx, ids, now)
	for _, id := range ids {
		f, ok := lk.Facts(id.String())
		if !ok {
			left++
			continue
		}
		seen[string(f.Position)] = true
		if (!raw.ShowNoTeam && noNFLTeam(f.Team)) || (raw.Position != "" && raw.Position != string(f.Position)) {
			continue
		}
		row := poolRow(id, f, season, now, plans[id.String()])
		valuePoolRow(&row, values, assets)
		row.Market = facts[id.String()]
		if row.Market == nil {
			row.Market = facts[""]
		}
		row.MarketLines = MarketLines(row.Market)
		all = append(all, row)
	}
	SortPool(all, raw)
	positions := poolapp.Positions(seen)
	summary := fmt.Sprintf("%d free agents", len(all))
	if raw.Position != "" {
		summary = fmt.Sprintf("%d free agents at %s of %d on MFL's list", len(all), raw.Position, len(ids)-left)
	}
	return Pool{Rows: all, Positions: positions, Summary: summary, Sorts: PoolSorts(), EngineNote: engineNote, PPGNote: ppgNote}, left
}

// positionOrder is the order the position filter lists in; any other position follows.

func poolMarketValue(row PoolRow, index int) *float64 {
	if index >= len(row.Market) {
		return nil
	}
	return row.Market[index].Value
}

// NoMarket stands in for four dashes when MFL lists no projection, rank, ADP or AAV for a player.
const NoMarket = "No MFL market data"

func MarketLines(facts []report.MarketFact) []string {
	parts := []string{"Proj —", "Rank —", "ADP —", "AAV —"}
	held := false
	for i, fact := range facts {
		if i >= len(parts) || fact.Value == nil {
			continue
		}
		held = true
		switch i {
		case 0:
			parts[i] = fmt.Sprintf("Proj %.1f", *fact.Value)
		case 1:
			parts[i] = "Rank " + strings.Replace(fact.Text, " · ", ", ", 1)
		case 2:
			parts[i] = fmt.Sprintf("ADP %.0f (%.0f–%.0f)", *fact.Value, fact.Range.Lo, fact.Range.Hi)
		case 3:
			parts[i] = fmt.Sprintf("AAV $%.1f on MFL’s $1,000 scale", *fact.Value)
		}
	}
	if !held {
		return []string{NoMarket}
	}
	return []string{parts[0] + " · " + parts[1], parts[2] + " · " + parts[3]}
}

func poolRow(id playerid.PlayerID, f normalize.PlayerFacts, season int, now time.Time, plan bidPlanBrief) PoolRow {
	p := poolapp.Row(id, f, season, now, poolapp.Plan(plan))
	return PoolRow{ID: p.ID, Name: p.Name, Position: p.Position, Team: p.Team, Age: p.Age, Experience: p.Experience, Minimum: p.Minimum, Plan: p.Plan, PlanID: p.PlanID}
}

func noNFLTeam(team string) bool { return team == "" || team == "FA" || team == "—" }

func valuePoolRow(row *PoolRow, engine map[string]float64, assets map[string]AssetValue) {
	row.EngineText, row.PPGText, row.LabText = "Not valued", "No score", "Not valued"
	if v, ok := engine[row.ID]; ok {
		row.Engine = &v
		row.EngineText = fmt.Sprintf("%.2f points/game", v)
	}
	if v, ok := assets[row.ID]; ok {
		row.PPGText, row.LabText = v.NowText, v.DynastyText
		if v.NowHeld {
			row.PPG = &v.NowValue
		}
		if v.DynastyHeld {
			row.Lab = &v.DynastyValue
		} else {
			row.LabText = "Not valued"
		}
	}
}

func poolOrder(row PoolRow) poolapp.Order {
	order := poolapp.Order{ID: row.ID, Name: row.Name, Position: row.Position, Team: row.Team, Minimum: row.Minimum.Cents, Engine: row.Engine, PPG: row.PPG, Lab: row.Lab}
	for i := range order.Market {
		order.Market[i] = poolMarketValue(row, i)
	}
	return order
}
func SortPool(rows []PoolRow, raw RawPoolScope) {
	slices.SortFunc(rows, func(x, y PoolRow) int { return poolapp.Compare(poolOrder(x), poolOrder(y), raw.Sort, raw.Desc) })
}
func PoolNumber(row PoolRow, key string) *float64 { return poolOrder(row).Number(key) }

func (a *API) TargetPoolRefresh(force bool) error {
	return a.host.RefreshOffersPool(force) //nolint:wrapcheck // Preserve refresh errors and serializer span.
}

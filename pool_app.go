package main

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	poolapp "github.com/secureprospective/TheWarRoom/internal/app/pool"
	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/freeagents"
	"github.com/secureprospective/TheWarRoom/internal/normalize"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
)

// poolFreshFor is how long a held free-agent list is trusted before opening the pool fetches
// again; the Refresh act always fetches.
const poolFreshFor = 5 * time.Minute

type PoolRow struct {
	Engine      *float64                `json:"engine"`
	PPG         *float64                `json:"ppg"`
	Lab         *float64                `json:"lab"`
	EngineText  string                  `json:"engineText"`
	PPGText     string                  `json:"ppgText"`
	LabText     string                  `json:"labText"`
	MarketLines []string                `json:"marketLines"`
	Market      []MarketFact            `json:"market"`
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

func poolSorts() []PoolSort {
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
	if !slices.ContainsFunc(poolSorts(), func(s PoolSort) bool { return s.Key == r.Sort }) {
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

// refreshFreeAgents pulls MFL's free-agent list live, with the archive as fallback, and holds
// the last good copy like the other season feeds. The caller holds refreshMu.
func (a *App) refreshFreeAgents(ctx context.Context) {
	a.restoreFreeAgents(ctx)
	rows, fresh, err := liveOrArchive(ctx, a.fallbackParent(), a.season,
		archivedFeed[freeagents.RawFreeAgent]{
			export: freeagents.Export,
			fetch: func(ctx context.Context) ([]freeagents.RawFreeAgent, error) {
				raw, ferr := freeagents.Fetch(ctx, a.mflClient, strconv.Itoa(a.season), ingestion.LeagueID)
				if ferr != nil {
					return nil, fmt.Errorf("free agents refresh: %w", ferr)
				}
				return raw, nil
			},
			parse: func(body []byte) ([]freeagents.RawFreeAgent, error) {
				raw, perr := freeagents.Parse(body)
				if perr != nil {
					return nil, fmt.Errorf("free agents refresh: %w", perr)
				}
				return raw, nil
			},
			bodies: a.history.ArchivedBodies,
		}, a.fallbackLog)
	var value []playerid.PlayerID
	if err == nil {
		value, err = freeagents.ToPlayerIDs(rows)
		sortFreeAgentIDs(value)
	}
	a.seasonMu.Lock()
	defer a.seasonMu.Unlock()
	if err == nil && !slices.Equal(a.seasonFreeAgents.value, value) {
		a.modelDirty.Store(true)
	}
	holdSeasonFeed(&a.seasonFreeAgents, value, fresh, err)
}

// restoreFreeAgents makes the first refresh compare with the last held list across launches.
func (a *App) restoreFreeAgents(ctx context.Context) {
	a.seasonMu.Lock()
	held := !a.seasonFreeAgents.fetchedAt.IsZero()
	a.seasonMu.Unlock()
	if held {
		return
	}
	_, err := a.history.ArchivedBodies(ctx, "TYPE="+freeagents.Export, func(src string, body []byte, at time.Time) bool {
		if !sameExport(src, freeagents.Export, a.season, false) {
			return false
		}
		rows, err := freeagents.Parse(body)
		if err != nil {
			return false
		}
		ids, err := freeagents.ToPlayerIDs(rows)
		if err != nil {
			return false
		}
		sortFreeAgentIDs(ids)
		a.seasonMu.Lock()
		holdSeasonFeed(&a.seasonFreeAgents, ids, liveFreshness(at), nil)
		a.seasonMu.Unlock()
		return true
	})
	if err != nil {
		a.fallbackLog("engine: previous free-agent list unavailable: " + err.Error())
	}
}

// TargetPoolRefresh fetches the free-agent list from MFL. Without force it skips a copy fetched
// in the last few minutes, so opening the pool twice asks MFL once.
func (a *App) TargetPoolRefresh(force bool) error {
	if err := a.ready(); err != nil {
		return fmt.Errorf("target pool refresh: startup: %w", err)
	}
	a.seasonMu.Lock()
	held := a.seasonFreeAgents
	a.seasonMu.Unlock()
	if !force && held.refreshError == "" && time.Since(held.fetchedAt) < poolFreshFor {
		return a.TargetMarketReportsRefresh(false)
	}
	ctx, cancel := context.WithTimeout(a.ctx, refreshTimeout)
	defer cancel()
	a.refreshMu.Lock()
	defer a.refreshMu.Unlock()
	a.refreshFreeAgents(ctx)
	a.refreshMarketFeeds(ctx, force)
	return nil
}

func poolReading(value Pool, p snapshot.Provenance, status ReadingStatus, reason, how string) Reading[Pool] {
	if p.Kind == "" {
		p = snapshot.Provenance{Source: "MFL freeAgents", Kind: "live", Freshness: Freshness{State: FreshFail}}
	}
	r := Reading[Pool]{
		Sourced: snapshot.Sourced[Pool]{Value: value, Provenance: p},
		Status:  status, Count: len(value.Rows), Baseline: "Latest MFL free-agent list",
		Cutoff: p.Freshness.FetchedAt, Method: "MFL freeAgents export; §6 minimum by experience, computed in Go",
		BuildStamp: buildLabel() + " " + buildDate, HowCounted: how,
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
func (a *App) TargetPool(raw RawPoolScope) Reading[Pool] {
	empty := Pool{Rows: []PoolRow{}, Positions: []string{}, Sorts: poolSorts()}
	if err := raw.Validate(); err != nil {
		return poolReading(empty, snapshot.Provenance{}, ReadingFailed, err.Error(), poolHow)
	}
	if err := a.ready(); err != nil {
		return poolReading(empty, snapshot.Provenance{}, ReadingFailed, err.Error(), poolHow)
	}
	a.seasonMu.Lock()
	feed := a.seasonFreeAgents
	feed.value = slices.Clone(feed.value)
	a.seasonMu.Unlock()
	p := feed.provenance("MFL freeAgents", "free agents")
	if feed.fetchedAt.IsZero() {
		reason := "Free agents have not been fetched from MFL yet"
		if feed.refreshError != "" {
			reason = "MFL free agents unavailable: " + feed.refreshError
		}
		return poolReading(empty, p, ReadingFailed, reason, poolHow)
	}
	ctx, cancel := context.WithTimeout(a.ctx, m4Timeout)
	defer cancel()
	lk, err := a.directory(ctx)
	if err != nil {
		return poolReading(empty, p, ReadingFailed, "Player names unavailable: "+err.Error(), poolHow)
	}
	value, left := a.poolRows(ctx, feed.value, lk, raw)
	if len(feed.value) == 0 {
		return poolReading(value, p, ReadingThin, "MFL lists no free agents right now", poolHow)
	}
	how := poolHow
	if left > 0 {
		how += fmt.Sprintf(" Left out of this list: %d ids the directory does not know.", left)
	}
	return poolReading(value, p, ReadingOK, "", how)
}

// poolRows builds, filters and sorts the rows, and counts the ids the directory does not know.
func (a *App) poolRows(ctx context.Context, ids []playerid.PlayerID, lk normalize.Lookup, raw RawPoolScope) (Pool, int) {
	plans := a.openBidPlansByPlayer(ctx, raw.FranchiseID)
	season, now := a.league.Season(), time.Now()
	all, seen, left := []PoolRow{}, map[string]bool{}, 0
	facts := marketFacts(a.heldMarket())
	values, engineNote := a.draftReportValues(ctx)
	assets, ppgNote := a.poolAssetValues(ctx, ids, now)
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
		row.MarketLines = poolMarketLines(row.Market)
		all = append(all, row)
	}
	sortPool(all, raw)
	positions := poolapp.Positions(seen)
	summary := fmt.Sprintf("%d free agents", len(all))
	if raw.Position != "" {
		summary = fmt.Sprintf("%d free agents at %s of %d on MFL's list", len(all), raw.Position, len(ids)-left)
	}
	return Pool{Rows: all, Positions: positions, Summary: summary, Sorts: poolSorts(), EngineNote: engineNote, PPGNote: ppgNote}, left
}

// positionOrder is the order the position filter lists in; any other position follows.

func poolMarketValue(row PoolRow, index int) *float64 {
	if index >= len(row.Market) {
		return nil
	}
	return row.Market[index].value
}

// poolNoMarket stands in for four dashes when MFL lists no projection, rank, ADP or AAV for a player.
const poolNoMarket = "No MFL market data"

func poolMarketLines(facts []MarketFact) []string {
	parts := []string{"Proj —", "Rank —", "ADP —", "AAV —"}
	held := false
	for i, fact := range facts {
		if i >= len(parts) || fact.value == nil {
			continue
		}
		held = true
		switch i {
		case 0:
			parts[i] = fmt.Sprintf("Proj %.1f", *fact.value)
		case 1:
			parts[i] = "Rank " + strings.Replace(fact.Text, " · ", ", ", 1)
		case 2:
			parts[i] = fmt.Sprintf("ADP %.0f (%.0f–%.0f)", *fact.value, fact.Range.Lo, fact.Range.Hi)
		case 3:
			parts[i] = fmt.Sprintf("AAV $%.1f on MFL’s $1,000 scale", *fact.value)
		}
	}
	if !held {
		return []string{poolNoMarket}
	}
	return []string{parts[0] + " · " + parts[1], parts[2] + " · " + parts[3]}
}

func poolRow(id playerid.PlayerID, f normalize.PlayerFacts, season int, now time.Time, plan bidPlanBrief) PoolRow {
	p := poolapp.Row(id, f, season, now, poolapp.Plan(plan))
	return PoolRow{ID: p.ID, Name: p.Name, Position: p.Position, Team: p.Team, Age: p.Age, Experience: p.Experience, Minimum: p.Minimum, Plan: p.Plan, PlanID: p.PlanID}
}
func sortFreeAgentIDs(ids []playerid.PlayerID) { poolapp.SortIDs(ids) }

func noNFLTeam(team string) bool { return team == "" || team == "FA" || team == "—" }

func (a *App) poolAssetValues(ctx context.Context, ids []playerid.PlayerID, at time.Time) (map[string]AssetValue, string) {
	a.hydrateResearchContext(ctx)
	var model assetModel
	if held := a.assetModel.Load(); held != nil {
		model = *held
	}
	also := make([]string, 0, len(ids))
	for _, id := range ids {
		also = append(also, id.String())
	}
	board := buildAssetBoard(researchView{researchInput: a.heldResearchInput()}, model, at, also)
	out := make(map[string]AssetValue, len(board.Values))
	for _, v := range board.Values {
		out[v.ID] = v
	}
	return out, board.NowSource
}

func valuePoolRow(row *PoolRow, engine map[string]float64, assets map[string]AssetValue) {
	row.EngineText, row.PPGText, row.LabText = "Not valued", "No score", "Not valued"
	if v, ok := engine[row.ID]; ok {
		row.Engine = &v
		row.EngineText = fmt.Sprintf("%.2f points/game", v)
	}
	if v, ok := assets[row.ID]; ok {
		row.PPGText, row.LabText = v.Now.Text, v.Dynasty.Text
		if v.Now.Held {
			row.PPG = &v.NowValue
		}
		if v.Dynasty.Held {
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
func sortPool(rows []PoolRow, raw RawPoolScope) {
	slices.SortFunc(rows, func(x, y PoolRow) int { return poolapp.Compare(poolOrder(x), poolOrder(y), raw.Sort, raw.Desc) })
}
func poolNumber(row PoolRow, key string) *float64 { return poolOrder(row).Number(key) }

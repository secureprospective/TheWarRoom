package main

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/engine/bid"
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
	Market     []MarketFact            `json:"market"`
	ID         string                  `json:"id"`
	Name       string                  `json:"name"`
	Position   string                  `json:"position"`
	Team       string                  `json:"team"`
	Age        string                  `json:"age"`
	Experience string                  `json:"experience"`
	Minimum    snapshot.ContractAmount `json:"minimum"`
	Plan       string                  `json:"plan"`   // the owner's open bid plan on him, in words; else empty
	PlanID     string                  `json:"planId"` // that plan's correlation id; else empty
}

type Pool struct {
	Rows      []PoolRow `json:"rows"`
	Positions []string  `json:"positions"`
	Summary   string    `json:"summary"`
}

// RawPoolScope is the pool's filter and sort. FranchiseID only picks whose bid plans show.
type RawPoolScope struct {
	FranchiseID string `json:"franchiseId"`
	Position    string `json:"position"`
	Sort        string `json:"sort"`
	Desc        bool   `json:"desc"`
}

func (r RawPoolScope) Validate() error {
	if !slices.Contains([]string{"name", "position", "minimum"}, r.Sort) {
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
	}
	a.seasonMu.Lock()
	defer a.seasonMu.Unlock()
	holdSeasonFeed(&a.seasonFreeAgents, value, fresh, err)
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
	empty := Pool{Rows: []PoolRow{}, Positions: []string{}}
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
	for _, id := range ids {
		f, ok := lk.Facts(id.String())
		if !ok {
			left++
			continue
		}
		seen[string(f.Position)] = true
		if raw.Position != "" && raw.Position != string(f.Position) {
			continue
		}
		row := poolRow(id, f, season, now, plans[id.String()])
		row.Market = facts[id.String()]
		if row.Market == nil {
			row.Market = facts[""]
		}
		all = append(all, row)
	}
	sortPool(all, raw)
	positions := poolPositions(seen)
	summary := fmt.Sprintf("%d free agents", len(all))
	if raw.Position != "" {
		summary = fmt.Sprintf("%d free agents at %s of %d on MFL's list", len(all), raw.Position, len(ids)-left)
	}
	return Pool{Rows: all, Positions: positions, Summary: summary}, left
}

func poolRow(id playerid.PlayerID, f normalize.PlayerFacts, season int, now time.Time, plan bidPlanBrief) PoolRow {
	exp := bid.ExperienceFrom(season, f.DraftYear, f.HasDraftYear)
	experience := "Unknown (rookie minimum)"
	switch {
	case exp.Known && exp.Years == 0:
		experience = "Rookie"
	case exp.Known && exp.Years == 1:
		experience = "1 year"
	case exp.Known:
		experience = fmt.Sprintf("%d years", exp.Years)
	}
	minimum := bid.MinimumSalary(exp.Years)
	return PoolRow{ID: id.String(), Name: f.Name, Position: string(f.Position), Team: f.Team,
		Age: ageText(f, now), Experience: experience,
		Minimum: snapshot.ContractAmount{Cents: minimum, Text: bid.MoneyText(minimum)},
		Plan:    plan.State, PlanID: plan.ID}
}

func ageText(f normalize.PlayerFacts, now time.Time) string {
	if !f.HasBirthdate {
		return "—"
	}
	born, today := time.Unix(f.Birthdate, 0).UTC(), now.UTC()
	years := today.Year() - born.Year()
	if today.Month() < born.Month() || (today.Month() == born.Month() && today.Day() < born.Day()) {
		years--
	}
	return strconv.Itoa(years)
}

// positionOrder is the order the position filter lists in; any other position follows.
func positionOrder(pos string) int {
	order := []domain.Position{domain.PosQB, domain.PosRB, domain.PosWR, domain.PosTE, domain.PosK,
		domain.PosDE, domain.PosDT, domain.PosLB, domain.PosCB, domain.PosS}
	if i := slices.Index(order, domain.Position(pos)); i >= 0 {
		return i
	}
	return len(order)
}

func poolPositions(seen map[string]bool) []string {
	out := []string{}
	for pos := range seen {
		out = append(out, pos)
	}
	slices.SortFunc(out, func(x, y string) int {
		return cmp.Or(cmp.Compare(positionOrder(x), positionOrder(y)), strings.Compare(x, y))
	})
	return out
}

func sortPool(rows []PoolRow, raw RawPoolScope) {
	byName := func(x, y PoolRow) int {
		return cmp.Or(strings.Compare(x.Name, y.Name), strings.Compare(x.ID, y.ID))
	}
	slices.SortFunc(rows, func(x, y PoolRow) int {
		c := 0
		switch raw.Sort {
		case "position":
			c = cmp.Compare(positionOrder(x.Position), positionOrder(y.Position))
		case "minimum":
			c = cmp.Compare(x.Minimum.Cents, y.Minimum.Cents)
		}
		if c == 0 {
			c = byName(x, y)
		}
		if raw.Desc {
			return -c
		}
		return c
	})
}

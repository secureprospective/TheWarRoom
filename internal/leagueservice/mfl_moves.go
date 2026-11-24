package leagueservice

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/appvalue"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/engine/bid"
	"github.com/secureprospective/TheWarRoom/internal/mflmirror"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
)

const (
	MflMovesRefresh  = "mflmoves.refresh"
	MflMovesPoolRead = "mflmoves.pool.read"

	// mflNoRequestFranchise is the safeguard text the private native readings use.
	mflNoRequestFranchise = "authenticated seated franchise; no request franchise; uncached private result"
)

// RawMflMovesScope selects the app league. The seat names the franchise; a request may not.
type RawMflMovesScope struct {
	League    string `json:"league"`
	Franchise string `json:"franchise,omitempty"`
}

func (r RawMflMovesScope) Validate() error {
	if !validPersonalLeague(r.League) {
		return fmt.Errorf("choose an app league")
	}
	if r.Franchise != "" {
		return fmt.Errorf("%s", mflNoRequestFranchise)
	}
	return nil
}

// RawMflPoolScope reads the held free-agent pool, optionally narrowed and sorted.
type RawMflPoolScope struct {
	League    string `json:"league"`
	Franchise string `json:"franchise,omitempty"`
	Position  string `json:"position,omitempty"`
	Sort      string `json:"sort,omitempty"`
}

func mflPoolSorts() []string { return []string{"name", "position", "team", "ppg", "lab", "minimum"} }

func (r RawMflPoolScope) Validate() error {
	if err := (RawMflMovesScope{League: r.League, Franchise: r.Franchise}).Validate(); err != nil {
		return err
	}
	if len(r.Position) > 8 || strings.TrimSpace(r.Position) != r.Position {
		return fmt.Errorf("choose a position from the pool list")
	}
	if r.Sort != "" && !slices.Contains(mflPoolSorts(), r.Sort) {
		return fmt.Errorf("sort by one of: %s", strings.Join(mflPoolSorts(), ", "))
	}
	return nil
}

// MflRefresh says what a refresh did: the copy held, how old it is, and whether MFL was asked.
type MflRefresh struct {
	League     string    `json:"league"`
	MFLLeague  string    `json:"mflLeague"`
	Season     int       `json:"season"`
	FetchedAt  time.Time `json:"fetchedAt"`
	AgeSeconds int       `json:"ageSeconds"`
	AskedMFL   bool      `json:"askedMfl"`
	FreeAgents int       `json:"freeAgents"`
	Words      string    `json:"words"`
}

// MflPoolRow is one free agent with this league's values beside him.
type MflPoolRow struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	Position   string   `json:"position"`
	Team       string   `json:"team"`
	Age        string   `json:"age"`
	Experience string   `json:"experience"`
	Minimum    string   `json:"minimum"`
	NowPPG     *float64 `json:"nowPPG,omitempty"`
	Dynasty    *float64 `json:"dynastyPPG,omitempty"`
	Value      string   `json:"value"`
}

type MflPool struct {
	League     string       `json:"league"`
	MFLLeague  string       `json:"mflLeague"`
	Season     int          `json:"season"`
	FetchedAt  time.Time    `json:"fetchedAt"`
	AgeSeconds int          `json:"ageSeconds"`
	Rows       []MflPoolRow `json:"rows"`
	Positions  []string     `json:"positions"`
	Sorts      []string     `json:"sorts"`
	Words      string       `json:"words"`
}

type mflMirrorHost interface {
	MFLMirror() *mflmirror.Store
}

// mflFetcherHost lets a host choose the transport the mirror reads MFL through (tests use a fake).
type mflFetcherHost interface {
	MFLFetcher() mflmirror.Source
}

func (a *Service) mflAccess(ctx context.Context, league, operation, mode string) (
	context.Context, *mflmirror.Store, mflmirror.Registration, context.CancelFunc, error) {
	ctx, _, _, cancel, err := a.franchiseOwnerAccess(ctx, league, operation, mode)
	if err != nil {
		return nil, nil, mflmirror.Registration{}, nil, err
	}
	host, ok := a.host.(mflMirrorHost)
	if !ok || host.MFLMirror() == nil {
		cancel()
		return nil, nil, mflmirror.Registration{}, nil, fmt.Errorf("MFL moves storage is not available on this host")
	}
	store := host.MFLMirror()
	reg, ok, err := store.Registration(ctx)
	if err != nil {
		cancel()
		return nil, nil, mflmirror.Registration{}, nil, fmt.Errorf("read MFL league setting: %w", err)
	}
	if !ok {
		cancel()
		return nil, nil, mflmirror.Registration{}, nil, fmt.Errorf("this league has no MFL league number yet; set it first")
	}
	return ctx, store, reg, cancel, nil
}

func (a *Service) mflSource() mflmirror.Source {
	if h, ok := a.host.(mflFetcherHost); ok {
		return h.MFLFetcher()
	}
	return nil
}

func (a *Service) MflMovesRefresh(raw RawMflMovesScope) (MflRefresh, error) {
	return a.MflMovesRefreshContext(a.host.CommissionerContext(), raw)
}

// MflMovesRefreshContext reads MFL's public data for the seated owner's league, at most once
// every five minutes; inside that window it returns the held copy and its age.
func (a *Service) MflMovesRefreshContext(ctx context.Context, raw RawMflMovesScope) (MflRefresh, error) {
	if err := raw.Validate(); err != nil {
		return MflRefresh{}, fmt.Errorf("MFL moves input: %w", err)
	}
	ctx, store, reg, cancel, err := a.mflAccess(ctx, raw.League, MflMovesRefresh, "execute")
	if err != nil {
		return MflRefresh{}, err
	}
	defer cancel()
	res, err := store.Refresh(ctx, a.mflSource(), reg.MFLLeague)
	if err != nil {
		return MflRefresh{}, fmt.Errorf("refresh MFL data: %w", err)
	}
	words := fmt.Sprintf("Held MFL copy is %s old; MFL was not asked again (it is asked at most once every 5 minutes).",
		res.Age.Round(time.Second))
	if res.Fetched {
		words = "Fresh from MFL just now."
	}
	return MflRefresh{League: raw.League, MFLLeague: reg.MFLLeague, Season: res.Held.Season,
		FetchedAt: res.Held.FetchedAt, AgeSeconds: int(res.Age.Seconds()), AskedMFL: res.Fetched,
		FreeAgents: len(res.Held.FreeAgents), Words: words}, nil
}

func (a *Service) MflMovesPoolRead(raw RawMflPoolScope) (MflPool, error) {
	return a.MflMovesPoolReadContext(a.host.CommissionerContext(), raw)
}

// MflMovesPoolReadContext reads the held free-agent pool with this league's values; it never calls MFL.
func (a *Service) MflMovesPoolReadContext(ctx context.Context, raw RawMflPoolScope) (MflPool, error) {
	if err := raw.Validate(); err != nil {
		return MflPool{}, fmt.Errorf("MFL pool input: %w", err)
	}
	ctx, store, reg, cancel, err := a.mflAccess(ctx, raw.League, MflMovesPoolRead, "read")
	if err != nil {
		return MflPool{}, err
	}
	defer cancel()
	held, ok, err := store.Read(ctx, reg.MFLLeague)
	if err != nil {
		return MflPool{}, fmt.Errorf("read held MFL pool: %w", err)
	}
	if !ok {
		return MflPool{}, fmt.Errorf("no MFL free agents are held yet; refresh first")
	}
	source := a.mflSource()
	if source == nil {
		return MflPool{}, fmt.Errorf("MFL data source is not available on this host")
	}
	lookup, err := source.Directory(held.PlayersJSON)
	if err != nil {
		return MflPool{}, fmt.Errorf("held MFL player directory: %w", err)
	}
	now := a.host.CommissionerNow()
	values, valueNote := a.mflPoolValues(ctx, raw.League, held.Season)
	rows, positions := mflPoolRows(held.FreeAgents, lookup, held.Season, now, values, raw.Position)
	sortMflPool(rows, raw.Sort)
	words := fmt.Sprintf("%d free agents on MFL, held %s ago.", len(rows), now.Sub(held.FetchedAt).Round(time.Second))
	if valueNote != "" {
		words += " " + valueNote
	}
	return MflPool{League: raw.League, MFLLeague: reg.MFLLeague, Season: held.Season, FetchedAt: held.FetchedAt,
		AgeSeconds: int(now.Sub(held.FetchedAt).Seconds()), Rows: rows, Positions: positions,
		Sorts: mflPoolSorts(), Words: words}, nil
}

// mflPoolValues joins the league's own valuation where it holds the player; a free agent the
// valuation has never seen is shown as not valued, in words.
func (a *Service) mflPoolValues(ctx context.Context, league string, season int) (map[string]appvalue.Player, string) {
	host, ok := a.host.(valueHost)
	if !ok {
		return nil, "Values are not available on this host."
	}
	reading, err := host.NativeValues(ctx, league, "", season)
	if err != nil {
		return nil, "Values are not available right now."
	}
	out := make(map[string]appvalue.Player, len(reading.Players))
	for _, p := range reading.Players {
		out[p.Player] = p
	}
	return out, ""
}

func mflPoolRows(ids []string, lookup mflmirror.Directory, season int, now time.Time,
	values map[string]appvalue.Player, position string) ([]MflPoolRow, []string) {
	rows := []MflPoolRow{}
	seen := map[string]bool{}
	for _, raw := range ids {
		id, err := playerid.New(raw)
		if err != nil {
			continue
		}
		facts, ok := lookup.Facts(id.String())
		if !ok {
			continue
		}
		pos := string(facts.Position)
		seen[pos] = true
		if position != "" && pos != position {
			continue
		}
		years, experience := mflExperience(season, facts)
		row := MflPoolRow{ID: id.String(), Name: facts.Name, Position: pos, Team: facts.Team, Age: mflAge(facts, now),
			Experience: experience, Minimum: bid.MoneyText(bid.MinimumSalary(years)), Value: "Not valued"}
		if v, ok := values[row.ID]; ok && v.Available {
			now, dyn := v.NowPPG, v.DynastyPP
			row.NowPPG, row.Dynasty = &now, &dyn
			row.Value = fmt.Sprintf("%.2f points/game now, %.2f dynasty", now, dyn)
		}
		rows = append(rows, row)
	}
	return rows, mflPositions(seen)
}

func sortMflPool(rows []MflPoolRow, key string) {
	num := func(p *float64) float64 {
		if p == nil {
			return -1
		}
		return *p
	}
	slices.SortStableFunc(rows, func(x, y MflPoolRow) int {
		switch key {
		case "position":
			return cmpFirst(mflPositionOrder(x.Position)-mflPositionOrder(y.Position), strings.Compare(x.Name, y.Name))
		case "team":
			return cmpFirst(strings.Compare(x.Team, y.Team), strings.Compare(x.Name, y.Name))
		case "ppg":
			return cmpFirst(cmpFloat(num(y.NowPPG), num(x.NowPPG)), strings.Compare(x.Name, y.Name))
		case "lab":
			return cmpFirst(cmpFloat(num(y.Dynasty), num(x.Dynasty)), strings.Compare(x.Name, y.Name))
		case "minimum":
			return cmpFirst(strings.Compare(x.Minimum, y.Minimum), strings.Compare(x.Name, y.Name))
		default:
			return strings.Compare(x.Name, y.Name)
		}
	})
}

func cmpFloat(a, b float64) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	}
	return 0
}

func cmpFirst(first, second int) int {
	if first != 0 {
		return first
	}
	return second
}

func mflExperience(season int, f domain.PlayerFacts) (int, string) {
	exp := bid.ExperienceFrom(season, f.DraftYear, f.HasDraftYear)
	switch {
	case !exp.Known:
		return 0, "Unknown (rookie minimum)"
	case exp.Years == 0:
		return 0, "Rookie"
	case exp.Years == 1:
		return 1, "1 year"
	}
	return exp.Years, fmt.Sprintf("%d years", exp.Years)
}

func mflAge(f domain.PlayerFacts, now time.Time) string {
	if !f.HasBirthdate {
		return "—"
	}
	born, today := time.Unix(f.Birthdate, 0).UTC(), now.UTC()
	years := today.Year() - born.Year()
	if today.Month() < born.Month() || (today.Month() == born.Month() && today.Day() < born.Day()) {
		years--
	}
	return fmt.Sprint(years)
}

func mflPositionOrder(pos string) int {
	order := []domain.Position{domain.PosQB, domain.PosRB, domain.PosWR, domain.PosTE, domain.PosK,
		domain.PosDE, domain.PosDT, domain.PosLB, domain.PosCB, domain.PosS}
	if i := slices.Index(order, domain.Position(pos)); i >= 0 {
		return i
	}
	return len(order)
}

func mflPositions(seen map[string]bool) []string {
	out := make([]string, 0, len(seen))
	for pos := range seen {
		out = append(out, pos)
	}
	slices.SortFunc(out, func(x, y string) int {
		return cmpFirst(mflPositionOrder(x)-mflPositionOrder(y), strings.Compare(x, y))
	})
	return out
}

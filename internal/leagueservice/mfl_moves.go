package leagueservice

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/engine/bid"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/mflmirror"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
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
	ID           string       `json:"id"`
	Name         string       `json:"name"`
	Position     string       `json:"position"`
	Team         string       `json:"team"`
	Age          string       `json:"age"`
	Experience   string       `json:"experience"`
	Minimum      string       `json:"minimum"`
	minimumCents domain.Money // numeric cents for sort:"minimum"; omitted from JSON
	NowPPG       *float64     `json:"nowPPG,omitempty"`
	Dynasty      *float64     `json:"dynastyPPG,omitempty"`
	Value        string       `json:"value"`
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

// mflSeat is what a seated owner's MFL request runs on: the league's held-copy store, the MFL league
// number from the league's rules, and the franchise from the seat.
type mflSeat struct {
	store     *mflmirror.Store
	mflLeague string
	franchise string
	records   []commissioner.Record
	cancel    context.CancelFunc
}

func mflMirrorFor(host Host) *mflmirror.Store {
	if provider, ok := host.(mflMirrorHost); ok {
		return provider.MFLMirror()
	}
	return nil
}

func mflFetcherFor(host Host) mflmirror.Source {
	if provider, ok := host.(mflFetcherHost); ok {
		return provider.MFLFetcher()
	}
	return nil
}

func latestModelScoresFor(ctx context.Context, host Host) ([]history.ModelScore, bool, error) {
	if provider, ok := host.(modelScoreHost); ok {
		return provider.LatestModelScores(ctx) //nolint:wrapcheck // forwards the wrapped host's model-score reading unchanged
	}
	return nil, false, nil
}

// mflRefusal promotes safe owner-facing mflmirror refusals to commissioner.Refusal for hosted replies.
func mflRefusal(err error) error {
	var r mflmirror.Refusal
	if errors.As(err, &r) {
		return fmt.Errorf("MFL moves: %w", commissioner.Refused(err))
	}
	return err
}

func mflRefused(msg string) error {
	return fmt.Errorf("MFL moves: %w", commissioner.Refused(errors.New(msg)))
}

func (a *Service) mflAccess(ctx context.Context, league, operation, mode string) (context.Context, mflSeat, error) {
	return a.mflSeatAccess(ctx, league, operation, mode, false)
}

func (a *Service) mflSeatAccess(ctx context.Context, league, operation, mode string, allowUnset bool) (context.Context, mflSeat, error) {
	ctx, franchise, records, cancel, err := a.franchiseOwnerAccess(ctx, league, operation, mode)
	if err != nil {
		return nil, mflSeat{}, err
	}
	store := mflMirrorFor(a.host)
	if store == nil {
		cancel()
		return nil, mflSeat{}, fmt.Errorf("MFL moves storage is not available on this host")
	}
	rules, err := commissioner.RulesNow(records, a.host.CommissionerNow())
	if err != nil {
		cancel()
		return nil, mflSeat{}, fmt.Errorf("read MFL league setting: %w", err)
	}
	number := strings.TrimSpace(rules.MFL.League.Value)
	if rules.MFL.League.State != leaguerules.IsKnown || number == "" {
		if allowUnset {
			return ctx, mflSeat{store: store, franchise: franchise, records: records, cancel: cancel}, nil
		}
		cancel()
		return nil, mflSeat{}, mflRefused("this league follows no MFL league yet; the commissioner sets it in league settings")
	}
	return ctx, mflSeat{store: store, mflLeague: number, franchise: franchise, records: records, cancel: cancel}, nil
}

func (a *Service) mflSource() mflmirror.Source {
	return mflFetcherFor(a.host)
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
	ctx, seat, err := a.mflAccess(ctx, raw.League, MflMovesRefresh, "execute")
	if err != nil {
		return MflRefresh{}, err
	}
	defer seat.cancel()
	res, err := seat.store.Refresh(ctx, a.mflSource(), seat.mflLeague)
	if err != nil {
		return MflRefresh{}, fmt.Errorf("refresh MFL data: %w", err)
	}
	words := fmt.Sprintf("Held MFL copy is %s old; MFL was not asked again (it is asked at most once every 5 minutes).",
		res.Age.Round(time.Second))
	if res.Fetched {
		words = "Fresh from MFL just now."
	}
	return MflRefresh{League: raw.League, MFLLeague: seat.mflLeague, Season: res.Held.Season,
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
	ctx, seat, err := a.mflAccess(ctx, raw.League, MflMovesPoolRead, "read")
	if err != nil {
		return MflPool{}, err
	}
	defer seat.cancel()
	held, ok, err := seat.store.Read(ctx, seat.mflLeague)
	if err != nil {
		return MflPool{}, fmt.Errorf("read held MFL pool: %w", err)
	}
	if !ok {
		return MflPool{}, mflRefused("no MFL free agents are held yet; refresh first")
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
	return MflPool{League: raw.League, MFLLeague: seat.mflLeague, Season: held.Season, FetchedAt: held.FetchedAt,
		AgeSeconds: int(now.Sub(held.FetchedAt).Seconds()), Rows: rows, Positions: positions,
		Sorts: mflPoolSorts(), Words: words}, nil
}

// mflValue is one pool player's value and where it came from, or why he has none.
type mflValue struct {
	now, dynasty float64
	ok           bool
	league       bool // from this league's own valuation, not the held model run
	why          string
}

type modelScoreHost interface {
	LatestModelScores(context.Context) ([]history.ModelScore, bool, error)
}

// mflPoolValues values every player it can. This league's own valuation (NativeValues) wins where the
// host has one; otherwise the held model run values every player it scored, rostered or not.
func (a *Service) mflPoolValues(ctx context.Context, league string, season int) (map[string]mflValue, string) {
	out := map[string]mflValue{}
	note := "Values are not available on this host."
	if host, ok := a.host.(modelScoreHost); ok {
		scores, held, err := host.LatestModelScores(ctx)
		switch {
		case err != nil:
			note = "Values are not available right now."
		case !held:
			note = "No model run is held yet, so no player can be valued."
		default:
			note = "Values come from the held model run (not this league's own scoring)."
			for _, sc := range scores {
				out[sc.MFLID] = mflValue{now: sc.NowPPG, dynasty: sc.DynastyPPG, ok: !sc.Unavailable, why: sc.Reason}
			}
		}
	}
	if host, ok := a.host.(valueHost); ok {
		if reading, err := host.NativeValues(ctx, league, "", season); err == nil {
			note = ""
			for _, p := range reading.Players {
				if p.Available {
					out[p.Player] = mflValue{now: p.NowPPG, dynasty: p.DynastyPP, ok: true, league: true}
				} else if _, has := out[p.Player]; !has {
					out[p.Player] = mflValue{why: p.Words}
				}
			}
		}
	}
	return out, note
}

func mflPoolRows(ids []string, lookup mflmirror.Directory, season int, now time.Time,
	values map[string]mflValue, position string) ([]MflPoolRow, []string) {
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
		minSalary := bid.MinimumSalary(years)
		row := MflPoolRow{ID: id.String(), Name: facts.Name, Position: pos, Team: facts.Team, Age: mflAge(facts, now),
			Experience: experience, Minimum: bid.MoneyText(minSalary), minimumCents: minSalary, Value: "Not valued"}
		if v, ok := values[row.ID]; ok && v.ok {
			now, dyn := v.now, v.dynasty
			row.NowPPG, row.Dynasty = &now, &dyn
			row.Value = fmt.Sprintf("%.2f points/game now, %.2f dynasty", now, dyn)
			if !v.league {
				row.Value += " (model run)"
			}
		} else if ok && v.why != "" {
			row.Value = "Not valued: " + v.why
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
			return cmpFirst(cmp.Compare(x.minimumCents, y.minimumCents), strings.Compare(x.Name, y.Name))
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

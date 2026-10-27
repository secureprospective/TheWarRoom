package commissioner

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/engine/competition"

	"github.com/secureprospective/TheWarRoom/internal/engine/rookiedraft"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

// The two draft tie rules the app can carry out. They order only the franchises the standings leave level,
// by where the league lists them.
const (
	TieByFranchiseOrder        = "League franchise order"
	TieByReverseFranchiseOrder = "Reverse league franchise order"
)

// DraftStandings supplies the final standings a draft's first round reads. It is a seam because the standings
// need the season's result sources, which only the store holds: the reading takes it as a parameter.
type DraftStandings interface {
	// Final is the standings of a finished season as they stood through an instant.
	Final(records []Record, season int, through time.Time) rookiedraft.FinalStandings
}

// NoDraftStandings is for a reading made where no result sources are held.
type NoDraftStandings struct{}

func (NoDraftStandings) Final([]Record, int, time.Time) rookiedraft.FinalStandings {
	return rookiedraft.MissingFinalStandings{}
}

// LazyStandings reads the standings only when a draft actually needs them, once for each season and instant.
type LazyStandings struct {
	Read  func(records []Record, season int, through time.Time) (NativeStandings, error)
	cache map[string]finalOrder
}

// plainWords is a sentence written for a GM that travels as an error: the draft shows it as it stands.
type plainWords string

func (w plainWords) Error() string { return string(w) }

type lazyOrder func(tie leaguerules.Value[leaguerules.TieRule]) ([]string, error)

func (f lazyOrder) FirstRound(tie leaguerules.Value[leaguerules.TieRule]) ([]string, error) {
	return f(tie)
}

func (l *LazyStandings) Final(records []Record, season int, through time.Time) rookiedraft.FinalStandings {
	return lazyOrder(func(tie leaguerules.Value[leaguerules.TieRule]) ([]string, error) {
		key := fmt.Sprintf("%d/%d/%d", season, through.UnixNano(), len(records))
		order, ok := l.cache[key]
		if !ok {
			standings, err := l.Read(records, season, through)
			if err != nil {
				return nil, fmt.Errorf("read the final standings: %w", err)
			}
			order = finalOrderOf(standings, records[0].Command.Args.Franchises)
			if l.cache == nil {
				l.cache = map[string]finalOrder{}
			}
			l.cache[key] = order
		}
		return order.firstRound(tie)
	})
}

// finalOrder is a season's standings, worst team first, with the groups the standings left level.
type finalOrder struct {
	waiting string
	worst   []string
	names   map[string]string
	level   [][]string
	league  []string
}

func finalOrderOf(s NativeStandings, league []string) finalOrder {
	out := finalOrder{names: map[string]string{}, league: league, waiting: standingsWaiting(s)}
	if out.waiting != "" {
		return out
	}
	rows, level := s.inForce()
	for i := len(rows) - 1; i >= 0; i-- {
		out.worst = append(out.worst, rows[i].Franchise)
		out.names[rows[i].Franchise] = rows[i].Name
	}
	out.level = level
	return out
}

// inForce is the ranking that stands: the commissioner's where there is one, else the computed one.
func (s NativeStandings) inForce() ([]StandingRow, [][]string) {
	if s.Override == nil {
		return s.Computed, s.level
	}
	if s.Override.Order != nil {
		return s.Override.Rows, nil
	}
	return s.Override.Rows, s.overrideLevel
}

// standingsWaiting says, in words, why a season's standings are not final yet; empty once they are.
func standingsWaiting(s NativeStandings) string {
	switch {
	case s.Status != "Ranked":
		return fmt.Sprintf("The draft order needs the final standings of season %d. %s.", s.Season, s.Words)
	case s.Pending > 0:
		return fmt.Sprintf("The draft order needs the final standings of season %d, but %d of its games "+
			"still have no result.", s.Season, s.Pending)
	case s.regularWeeks > 0 && s.Through < s.regularWeeks:
		return fmt.Sprintf("The draft order needs the final standings of season %d, but its schedule covers "+
			"only %d of %d regular-season weeks.", s.Season, s.Through, s.regularWeeks)
	}
	return ""
}

// firstRound is the worst-first order, with each level group put in order by the draft's tie rule. The rule
// is asked for only where a tie exists.
func (f finalOrder) firstRound(tie leaguerules.Value[leaguerules.TieRule]) ([]string, error) {
	if f.waiting != "" {
		return nil, plainWords(f.waiting)
	}
	out := slices.Clone(f.worst)
	for _, group := range f.level {
		ordered, err := f.tied(group, tie)
		if err != nil {
			return nil, err
		}
		first := len(out)
		for _, id := range group {
			first = min(first, slices.Index(out, id))
		}
		copy(out[first:], ordered)
	}
	return out, nil
}

func (f finalOrder) tied(group []string, tie leaguerules.Value[leaguerules.TieRule]) ([]string, error) {
	names := make([]string, len(group))
	for i, id := range group {
		names[i] = f.names[id]
	}
	together := competition.JoinAnd(names)
	if tie.State != leaguerules.IsKnown {
		return nil, plainWords(fmt.Sprintf("Needs a ruling: how a tie in the draft order is broken. %s finished "+
			"level in the standings.", competition.Capital(together)))
	}
	ordered := slices.SortedFunc(slices.Values(group), func(a, b string) int {
		return slices.Index(f.league, a) - slices.Index(f.league, b)
	})
	switch rule := strings.TrimSpace(string(tie.Value)); {
	case strings.EqualFold(rule, TieByFranchiseOrder):
	case strings.EqualFold(rule, TieByReverseFranchiseOrder):
		slices.Reverse(ordered)
	default:
		return nil, plainWords(fmt.Sprintf("The app cannot carry out the draft tie rule %q for %s, who finished "+
			"level in the standings. Choose %q or %q, or set round one's order by hand.", rule, together,
			TieByFranchiseOrder, TieByReverseFranchiseOrder))
	}
	return ordered, nil
}

// FinalStandingsReading reads one finished season's native standings, counting only what was recorded
// through an instant. The draft takes the standings as they stood when it began, so a result corrected
// afterwards cannot move a first round that has begun.
func FinalStandingsReading(ctx context.Context, league string, records []Record, src ResultSources, season int,
	through time.Time) (NativeStandings, error) {
	records = RecordsThrough(records, through)
	return NativeStandingsReading(ctx, league, records, src, season, through)
}

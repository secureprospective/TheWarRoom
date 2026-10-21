package now

import (
	"fmt"
	"regexp"

	"github.com/secureprospective/TheWarRoom/internal/app/catalog"
)

// Host supplies Now's detached held readings and runs the root-owned refresh and live-game poll
// under their original locks. No lock enters the API.
type Host interface {
	Ready() error
	KnownFranchise(id string) bool
	// NowInput captures the league's weeks, the held schedules, the live lineups and Now's held
	// byes and first scores in one season-lock section.
	NowInput(week, reportWeek int, mine string) Snapshot
	// NowRefresh refreshes the race, then, under the refresh lock, the first scores and the
	// chosen week's NFL schedule; it checks the week is listed.
	NowRefresh(week int, force bool) error
	// LiveGame polls under the live-game lock after the API's checks.
	LiveGame(open bool, follow, mine string) (LiveGameView, error)
}

// API is League Pulse › Now's binding. held is the App's one Now holder; the root host writes
// through the same instance.
type API struct {
	host Host
	held *Held
}

func New(host Host, held *Held) *API { return &API{host: host, held: held} }

var followID = regexp.MustCompile(`^(?:[0-9]{1,20}|[A-Z]{2,3}-[A-Z]{2,3}|nfl\.g\.[0-9]+)?$`)

// TargetNowRefresh refreshes the league's weekly results, then the chosen week's NFL schedule
// (0: the current week), unless fresh; force always refreshes. Go checks the week is listed.
func (a *API) TargetNowRefresh(week int, force bool) error {
	return a.host.NowRefresh(week, force) //nolint:wrapcheck // The host preserves the refresh refusals exactly.
}

// TargetNow reads held data only; refresh and scheduling remain owned by the host.
func (a *API) TargetNow(week, reportWeek int, mine string) (Board, error) {
	if err := a.host.Ready(); err != nil {
		return Board{}, fmt.Errorf("now: startup: %w", err)
	}
	in := a.host.NowInput(week, reportWeek, mine)
	board, err := Build(in)
	if err == nil && in.Race.Held.Weeks.FetchedAt.IsZero() && in.Race.Held.Weeks.RefreshError == "" {
		board.Waiting = "Reading the league's weeks from MFL…"
	}
	return board, err
}

// TargetLiveGame is a demand-driven poll: only the mounted Now view asks to refresh.
func (a *API) TargetLiveGame(open bool, follow, mine string) (LiveGameView, error) {
	if err := a.host.Ready(); err != nil {
		return LiveGameView{}, fmt.Errorf("live game: startup: %w", err)
	}
	if !followID.MatchString(follow) {
		return LiveGameView{}, fmt.Errorf("game.follow id: invalid")
	}
	if mine != "" && !a.host.KnownFranchise(mine) {
		return LiveGameView{}, fmt.Errorf("live game: unknown franchise")
	}
	return a.host.LiveGame(open, follow, mine) //nolint:wrapcheck // The host's errors keep their original text.
}

// Commands records existing bindings without inventing publication metadata. Only following a
// game has a registry verb (game.follow, X-01).
func Commands() []catalog.Command {
	return []catalog.Command{
		{ID: "game.follow", Method: "TargetLiveGame", Version: catalog.Unset, Role: catalog.Unset, Gravity: catalog.Unset, Availability: "both"},
		{ID: catalog.Unset, Method: "TargetNow", Version: catalog.Unset, Role: catalog.Unset, Gravity: catalog.Unset, Availability: "both"},
		{ID: catalog.Unset, Method: "TargetNowRefresh", Version: catalog.Unset, Role: catalog.Unset, Gravity: catalog.Unset, Availability: "both"},
	}
}

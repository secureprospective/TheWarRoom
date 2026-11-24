package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/commissioner/market"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/leagueservice"
	"github.com/secureprospective/TheWarRoom/internal/serverclient"
	"github.com/secureprospective/TheWarRoom/internal/serverhttp"
)

const standingsTolerance = leaguerules.Decimal(50)

type journey struct {
	id, name, command string
	reason            string
	excluded          bool
}

type canaryConfig struct {
	target, league, franchise, seedLabel, confirmation string
	season, week                                       int
	allowlisted                                        bool
}

func (c canaryConfig) canExecute() bool {
	return c.target == "lab" && c.allowlisted && strings.HasPrefix(c.league, "native.")
}

func alwaysJourneys() []string { return []string{"1", "2", "3", "4", "8", "13", "20"} }

func journeys() []journey {
	return []journey{
		{id: "1", name: "connect", command: "native.directory.read"},
		{id: "2", name: "week and lineup", command: "week.read,lineup.read"},
		{id: "3", name: "set lineup", command: "lineup.read,commish.lineup.set"},
		{id: "4", name: "live score", command: "nfl.games.read"},
		{id: "5", name: "trade propose", command: "trade.offer", reason: "no safe fixture-independent trade payload"},
		{id: "6", name: "trade accept", command: "trade.accept", reason: "requires second owner token", excluded: true},
		{id: "7", name: "trade review", command: "trade.review", reason: "commissioner confirmation is human-only; preview arguments unavailable"},
		{id: "8", name: "waiver claim", command: "waivers.read,waivers.claim"},
		{id: "9", name: "waiver processing", command: "waivers.process", reason: "timed commissioner operation"},
		{id: "10", name: "free-agent bid", command: "bid.place", reason: "no safe fixture-independent bid payload"},
		{id: "11", name: "roster change", command: "roster.cut", reason: "no safe fixture-independent roster payload"},
		{id: "12", name: "week final", command: "results.read"},
		{id: "13", name: "standings", command: "standings.read"},
		{id: "14", name: "playoff bracket", command: "bracket.read"},
		{id: "15", name: "weekly recap", command: "", reason: "no command exists", excluded: true},
		{id: "16", name: "league feed", command: "league.activity.read"},
		{id: "17", name: "chat", command: "", reason: "no command exists", excluded: true},
		{id: "18", name: "notification delivery", command: "", reason: "no command exists", excluded: true},
		{id: "19", name: "commissioner act", command: "commish.notice.publish", reason: "preview payload unavailable; never execute commissioner changes"},
		{id: "20", name: "rule read-back", command: "commish.rules.read"},
		{id: "21", name: "stat correction", command: "commish.stat.correct", reason: "preview payload unavailable"},
		{id: "22", name: "invite and join", command: "", reason: "human-only token boundary", excluded: true},
		{id: "23", name: "draft pick trade", command: "trade.draft.read", reason: "no safe fixture-independent pick trade payload"},
		{id: "24", name: "rookie draft", command: "draft.read"},
	}
}

func runOneJourney(ctx context.Context, agent *serverclient.Agent, cfg canaryConfig, j journey) (string, string) {
	if j.reason != "" {
		return "SKIPPED", j.reason
	}
	switch j.id {
	case "1":
		return runConnect(ctx, agent, cfg)
	case "2":
		return runWeekAndLineup(ctx, agent, cfg)
	case "3":
		return runSetLineup(ctx, agent, cfg)
	case "4":
		return runLiveScore(ctx, agent, cfg)
	case "8":
		return runWaiverClaimPreview(ctx, agent, cfg)
	case "12":
		return runWeekResults(ctx, agent, cfg)
	case "13":
		return runStandings(ctx, agent, cfg)
	case "14":
		return runBracket(ctx, agent, cfg)
	case "16":
		return runLeagueFeed(ctx, agent, cfg)
	case "20":
		return runRuleReadBack(ctx, agent, cfg)
	case "24":
		return runDraftRead(ctx, agent, cfg)
	default:
		return "SKIPPED", "no automated check"
	}
}

func readJSON[T any](ctx context.Context, agent *serverclient.Agent, command string, scope serverhttp.RawWorkflowScope) (T, error) {
	var out T
	data, err := agent.Read(ctx, command, scope)
	if err != nil {
		return out, fmt.Errorf("%s: %w", command, err)
	}
	if len(data) == 0 || string(data) == "null" {
		return out, fmt.Errorf("%s: empty response", command)
	}
	if err := json.Unmarshal(data, &out); err != nil {
		return out, fmt.Errorf("%s: unreadable response: %w", command, err)
	}
	return out, nil
}

func runConnect(ctx context.Context, agent *serverclient.Agent, cfg canaryConfig) (string, string) {
	dir, err := readJSON[commissioner.NativeDirectory](ctx, agent, "native.directory.read", serverhttp.RawWorkflowScope{})
	if err != nil {
		return "FAIL", err.Error()
	}
	if dir.League != cfg.league || len(dir.Franchises) == 0 {
		return "FAIL", fmt.Sprintf("unexpected directory league=%q franchises=%d", dir.League, len(dir.Franchises))
	}
	return "PASS", fmt.Sprintf("connected to %s (%d franchises)", dir.League, len(dir.Franchises))
}

func runWeekAndLineup(ctx context.Context, agent *serverclient.Agent, cfg canaryConfig) (string, string) {
	wk, err := readJSON[commissioner.NativeWeek](ctx, agent, "week.read", serverhttp.RawWorkflowScope{
		Franchise: cfg.franchise, Season: cfg.season, LocalZone: "UTC",
	})
	if err != nil {
		return "FAIL", err.Error()
	}
	if wk.Schedule.League != cfg.league || wk.Schedule.Season != cfg.season {
		return "FAIL", fmt.Sprintf("unexpected week league=%q season=%d", wk.Schedule.League, wk.Schedule.Season)
	}
	if cfg.franchise == "" {
		return "PASS", fmt.Sprintf("week read for season %d (next week %d)", wk.Schedule.Season, wk.NextWeek)
	}
	lu, err := readJSON[commissioner.NativeLineup](ctx, agent, "lineup.read", serverhttp.RawWorkflowScope{
		Franchise: cfg.franchise, Week: cfg.week,
	})
	if err != nil {
		return "FAIL", err.Error()
	}
	if lu.League != cfg.league || lu.Franchise != cfg.franchise || lu.Week != cfg.week {
		return "FAIL", fmt.Sprintf("unexpected lineup league=%q franchise=%q week=%d", lu.League, lu.Franchise, lu.Week)
	}
	return "PASS", fmt.Sprintf("week and lineup read (%d starters)", len(lu.Starters))
}

func buildLineupCommand(ctx context.Context, agent *serverclient.Agent, cfg canaryConfig) (commissioner.RawCommand, int, error) {
	lu, err := readJSON[commissioner.NativeLineup](ctx, agent, "lineup.read", serverhttp.RawWorkflowScope{
		Franchise: cfg.franchise, Week: cfg.week,
	})
	if err != nil {
		return commissioner.RawCommand{}, 0, err
	}
	starters := make([]string, 0, len(lu.Starters))
	for _, p := range lu.Starters {
		starters = append(starters, p.Player)
	}
	if len(starters) == 0 {
		return commissioner.RawCommand{}, 0, fmt.Errorf("lineup.read returned no starters to resubmit")
	}
	ledger, err := agent.Ledger(ctx)
	if err != nil {
		return commissioner.RawCommand{}, 0, fmt.Errorf("read ledger: %w", err)
	}
	return commissioner.RawCommand{
		League:     cfg.league,
		Command:    commissioner.LineupSet,
		Request:    fmt.Sprintf("canary.%s.lineup", cfg.seedLabel),
		Effective:  "now",
		Visibility: "private",
		Expected:   ledger.Revision,
		Args: commissioner.RawArgs{Lineup: &commissioner.RawLineupChange{
			Season: cfg.season, Franchise: cfg.franchise, Week: cfg.week, Starters: starters,
		}},
	}, len(starters), nil
}

func runSetLineup(ctx context.Context, agent *serverclient.Agent, cfg canaryConfig) (string, string) {
	if cfg.franchise == "" {
		return "SKIPPED", "-franchise not supplied"
	}
	cmd, count, err := buildLineupCommand(ctx, agent, cfg)
	if err != nil {
		return "FAIL", err.Error()
	}
	previewRaw, err := agent.Dispatch(ctx, cmd, "preview", "")
	if err != nil {
		return "FAIL", "preview failed: " + err.Error()
	}
	var preview commissioner.Preview
	if err := json.Unmarshal(previewRaw, &preview); err != nil {
		return "FAIL", "malformed lineup preview: " + err.Error()
	}
	if len(preview.Blockers) > 0 {
		return "FAIL", "preview blocked: " + strings.Join(preview.Blockers, "; ")
	}
	def, err := cmd.EffectiveDefinition()
	if err != nil {
		return "FAIL", "lineup gravity check failed: " + err.Error()
	}
	if (def.Gravity == "G2" || def.Gravity == "G3") && cfg.confirmation == "" {
		return "PASS", fmt.Sprintf("preview ok (%d starters, rev %d); execute gated on %s human confirmation", count, cmd.Expected, def.Gravity)
	}
	if !cfg.canExecute() {
		return "FAIL", "refusing execute: lab/native/allowlist gate not satisfied"
	}
	execRaw, err := agent.Dispatch(ctx, cmd, "execute", cfg.confirmation)
	if err != nil {
		return "FAIL", "execute failed: " + err.Error()
	}
	var receipt commissioner.Receipt
	if err := json.Unmarshal(execRaw, &receipt); err != nil || receipt.Entry == "" {
		return "FAIL", "malformed lineup receipt"
	}
	return "PASS", fmt.Sprintf("previewed and recorded lineup (%d starters, rev %d)", count, receipt.Revision)
}

func runLiveScore(ctx context.Context, agent *serverclient.Agent, cfg canaryConfig) (string, string) {
	games, err := readJSON[leagueservice.NFLGames](ctx, agent, "nfl.games.read", serverhttp.RawWorkflowScope{})
	if err != nil {
		return "FAIL", err.Error()
	}
	if games.League != cfg.league {
		return "FAIL", fmt.Sprintf("unexpected games league=%q", games.League)
	}
	return "PASS", fmt.Sprintf("nfl scoreboard read (%d games)", len(games.Board.Games))
}

func waiverPreviewTarget(ctx context.Context, agent *serverclient.Agent, franchise string) (string, error) {
	waivers, err := readJSON[market.NativeWaivers](ctx, agent, "waivers.read", serverhttp.RawWorkflowScope{
		Franchise: franchise, At: time.Now().UTC().Format(time.RFC3339Nano),
	})
	if err != nil {
		return "", err
	}
	for _, p := range waivers.Players {
		if p.Window != "" && p.Status != "processed" {
			return p.Window, nil
		}
	}
	if len(waivers.Players) > 0 && waivers.Players[0].Window != "" {
		return waivers.Players[0].Window, nil
	}
	assets, err := readJSON[commissioner.NativeAssets](ctx, agent, "native.assets.read", serverhttp.RawWorkflowScope{})
	if err != nil {
		return "", err
	}
	if len(assets.Pool) == 0 || assets.Pool[0].Player == "" {
		return "", fmt.Errorf("no waiver windows or free agents in pool")
	}
	return assets.Pool[0].Player, nil
}

func runWaiverClaimPreview(ctx context.Context, agent *serverclient.Agent, cfg canaryConfig) (string, string) {
	if cfg.franchise == "" {
		return "SKIPPED", "-franchise not supplied"
	}
	target, err := waiverPreviewTarget(ctx, agent, cfg.franchise)
	if err != nil {
		return "FAIL", err.Error()
	}
	ledger, err := agent.Ledger(ctx)
	if err != nil {
		return "FAIL", "read ledger: " + err.Error()
	}
	cmd := commissioner.RawCommand{
		League:     cfg.league,
		Command:    commissioner.WaiverClaim,
		Request:    fmt.Sprintf("canary.%s.waiver", cfg.seedLabel),
		Effective:  "now",
		Visibility: "private",
		Expected:   ledger.Revision,
		Args: commissioner.RawArgs{Waiver: &commissioner.RawWaiverArgs{
			Window: target, Franchise: cfg.franchise,
		}},
	}
	data, err := agent.Dispatch(ctx, cmd, "preview", "")
	if err != nil {
		if errors.Is(err, serverclient.ErrPreviewRefused) {
			return "PASS", "preview refused by rule: " + err.Error()
		}
		return "FAIL", "waiver preview failed: " + err.Error()
	}
	var preview commissioner.Preview
	if err := json.Unmarshal(data, &preview); err != nil {
		return "FAIL", "malformed waiver preview: " + err.Error()
	}
	if len(preview.Blockers) > 0 {
		return "PASS", "preview refused by rule: " + strings.Join(preview.Blockers, "; ")
	}
	if preview.Target.League == "" && len(preview.Fields) == 0 && preview.Question == "" {
		return "FAIL", "malformed waiver preview: empty preview body"
	}
	return "PASS", "waiver claim preview accepted for " + target
}

func runWeekResults(ctx context.Context, agent *serverclient.Agent, cfg canaryConfig) (string, string) {
	res, err := readJSON[commissioner.NativeResults](ctx, agent, "results.read", serverhttp.RawWorkflowScope{
		Season: cfg.season, Week: cfg.week,
	})
	if err != nil {
		return "FAIL", err.Error()
	}
	if res.League != cfg.league || res.Season != cfg.season || res.Week != cfg.week {
		return "FAIL", fmt.Sprintf("unexpected results league=%q season=%d week=%d", res.League, res.Season, res.Week)
	}
	return "PASS", fmt.Sprintf("results read (%d games)", len(res.Games))
}

type standingsTally struct {
	w, l, t int
	pf      leaguerules.Decimal
}

func absDec(d leaguerules.Decimal) leaguerules.Decimal {
	if d < 0 {
		return -d
	}
	return d
}

func tallyDecidedGames(ctx context.Context, agent *serverclient.Agent, season, through int) (map[string]*standingsTally, error) {
	got := map[string]*standingsTally{}
	entry := func(id string) *standingsTally {
		if got[id] == nil {
			got[id] = &standingsTally{}
		}
		return got[id]
	}
	for w := 1; w <= through; w++ {
		res, err := readJSON[commissioner.NativeResults](ctx, agent, "results.read", serverhttp.RawWorkflowScope{Season: season, Week: w})
		if err != nil {
			return nil, err
		}
		for _, g := range res.Games {
			if (g.Status != commissioner.GameFinal && g.Status != commissioner.GameTie) || g.HomePoints == nil || g.AwayPoints == nil {
				continue
			}
			h, a := entry(g.Home), entry(g.Away)
			h.pf += *g.HomePoints
			a.pf += *g.AwayPoints
			switch {
			case g.Status == commissioner.GameTie:
				h.t++
				a.t++
			case g.Winner == g.Home:
				h.w++
				a.l++
			default:
				a.w++
				h.l++
			}
		}
	}
	return got, nil
}

func verifyStandings(ctx context.Context, agent *serverclient.Agent, cfg canaryConfig, st commissioner.NativeStandings) error {
	if st.League != cfg.league || st.Season != cfg.season {
		return fmt.Errorf("standings mismatch: league=%q season=%d", st.League, st.Season)
	}
	if len(st.Computed) == 0 {
		return fmt.Errorf("standings returned no rows")
	}
	if st.Through < 0 {
		return fmt.Errorf("standings reported negative through week %d", st.Through)
	}
	tallies, err := tallyDecidedGames(ctx, agent, cfg.season, st.Through)
	if err != nil {
		return err
	}
	for _, row := range st.Computed {
		want := tallies[row.Franchise]
		if want == nil {
			want = &standingsTally{}
		}
		if row.Wins != want.w || row.Losses != want.l || row.Ties != want.t {
			return fmt.Errorf("%s record is %d-%d-%d, expected %d-%d-%d",
				row.Franchise, row.Wins, row.Losses, row.Ties, want.w, want.l, want.t)
		}
		if absDec(row.PointsFor-want.pf) > standingsTolerance {
			return fmt.Errorf("%s pointsFor is %d, expected %d", row.Franchise, row.PointsFor, want.pf)
		}
	}
	return nil
}

func runStandings(ctx context.Context, agent *serverclient.Agent, cfg canaryConfig) (string, string) {
	st, err := readJSON[commissioner.NativeStandings](ctx, agent, "standings.read", serverhttp.RawWorkflowScope{Season: cfg.season})
	if err != nil {
		return "FAIL", err.Error()
	}
	if err := verifyStandings(ctx, agent, cfg, st); err != nil {
		return "FAIL", err.Error()
	}
	return "PASS", fmt.Sprintf("standings verified (%d teams through week %d)", len(st.Computed), st.Through)
}

func runBracket(ctx context.Context, agent *serverclient.Agent, cfg canaryConfig) (string, string) {
	br, err := readJSON[commissioner.NativeBracket](ctx, agent, "bracket.read", serverhttp.RawWorkflowScope{Season: cfg.season})
	if err != nil {
		return "FAIL", err.Error()
	}
	if br.League != cfg.league || br.Season != cfg.season {
		return "FAIL", fmt.Sprintf("unexpected bracket league=%q season=%d", br.League, br.Season)
	}
	return "PASS", fmt.Sprintf("bracket read (%d rounds)", len(br.Rounds))
}

func runLeagueFeed(ctx context.Context, agent *serverclient.Agent, cfg canaryConfig) (string, string) {
	data, err := agent.Workflow(ctx, "league.activity.read", "read", json.RawMessage("{}"))
	if err != nil {
		return "FAIL", "league.activity.read: " + err.Error()
	}
	var feed commissioner.LeagueActivity
	if err := json.Unmarshal(data, &feed); err != nil || feed.League != cfg.league {
		return "FAIL", "unreadable league activity feed"
	}
	return "PASS", fmt.Sprintf("league feed read (%d events)", len(feed.Events))
}

func runRuleReadBack(ctx context.Context, agent *serverclient.Agent, cfg canaryConfig) (string, string) {
	rules, err := readJSON[commissioner.Rules](ctx, agent, "commish.rules.read", serverhttp.RawWorkflowScope{})
	if err != nil {
		return "FAIL", err.Error()
	}
	if rules.League != cfg.league || len(rules.Sections) == 0 {
		return "FAIL", fmt.Sprintf("unexpected rules league=%q sections=%d", rules.League, len(rules.Sections))
	}
	return "PASS", fmt.Sprintf("rules read (%d sections)", len(rules.Sections))
}

func runDraftRead(ctx context.Context, agent *serverclient.Agent, cfg canaryConfig) (string, string) {
	if cfg.franchise == "" {
		return "SKIPPED", "-franchise not supplied"
	}
	draft, err := readJSON[commissioner.NativeDraft](ctx, agent, "draft.read", serverhttp.RawWorkflowScope{
		Franchise: cfg.franchise, Season: cfg.season,
	})
	if err != nil {
		return "FAIL", err.Error()
	}
	if draft.League != cfg.league {
		return "FAIL", fmt.Sprintf("unexpected draft league=%q", draft.League)
	}
	return "PASS", fmt.Sprintf("draft read (%d rounds)", len(draft.Order))
}

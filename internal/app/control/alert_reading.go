package control

import (
	"fmt"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/report"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/engine/eligibility"
	"github.com/secureprospective/TheWarRoom/internal/envelope"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/league"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/nflschedule"
	"github.com/secureprospective/TheWarRoom/internal/leagueweek"
	"github.com/secureprospective/TheWarRoom/internal/lineup"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
)

type LineupAlertInput struct {
	Week        int
	Check       lineup.Result
	Provenance  snapshot.Provenance
	RulesSource snapshot.Provenance
}

type TradeAlertOffer struct {
	Direction string
	Expires   time.Time
}

type TradeAlertInput struct {
	Offers     []TradeAlertOffer
	Provenance snapshot.Provenance
}

func LineupAlert(r *AlertReading, saved LineupAlertInput, week leagueweek.Week, failed string, now time.Time) {
	if failed != "" || len(week.Games) == 0 {
		note := report.JoinNote("lineup lock unknown", failed)
		r.Unavailable = append(r.Unavailable, Unavailable{Kind: "lineup", Note: note})
		return
	}
	if !leagueweek.LastLock(week).After(now) {
		return
	}
	reason := envelope.TradeFeedReason(saved.Provenance)
	if reason == "" && saved.Week != week.Number {
		reason = fmt.Sprintf("saved lineup is week %d, current week is %d", saved.Week, week.Number)
	}
	if reason == "" && saved.RulesSource.Freshness.State == domain.FreshFail {
		reason = saved.RulesSource.Freshness.Note
	}
	if reason != "" {
		r.Unavailable = append(r.Unavailable, Unavailable{Kind: "lineup", Note: reason})
		return
	}
	if saved.Check.Legal && saved.Check.Full {
		return
	}
	lock, _ := leagueweek.NextLock(now, week)
	problems := make([]string, 0, len(saved.Check.Problems))
	for _, problem := range saved.Check.Problems {
		problems = append(problems, strings.TrimRight(problem.Message, ".; "))
	}
	r.Alerts = append(r.Alerts, Alert{
		Kind: "lineup", Urgency: AlertUrgency(now, lock), Title: "Check your lineup",
		Detail: strings.Join(problems, "; "), Node: "hq", Workspace: "lineup-and-roster",
		At: lock, Provenance: saved.Provenance,
	})
}

func TradeAlert(r *AlertReading, trades TradeAlertInput, now time.Time) {
	if reason := envelope.TradeFeedReason(trades.Provenance); reason != "" {
		r.Unavailable = append(r.Unavailable, Unavailable{Kind: "trades", Note: reason})
		return
	}
	count := 0
	var soonest time.Time
	for _, offer := range trades.Offers {
		if offer.Direction != "to_you" || (!offer.Expires.IsZero() && !offer.Expires.After(now)) {
			continue
		}
		count++
		if !offer.Expires.IsZero() && (soonest.IsZero() || offer.Expires.Before(soonest)) {
			soonest = offer.Expires
		}
	}
	if count == 0 {
		return
	}
	r.Alerts = append(r.Alerts, Alert{
		Kind: "trades", Urgency: AlertUrgency(now, soonest), Title: OffersTitle(count),
		Detail: "Review pending trade offers", Node: "trade", Workspace: "trade-desk",
		At: soonest, Provenance: trades.Provenance,
	})
}

func IRAlerts(r *AlertReading, franchiseID string, snap snapshot.Snapshot, loaded bool, injuries snapshot.Injuries, targetURL func() string) {
	if !loaded || snap.Rosters.Provenance.Freshness.State == domain.FreshFail ||
		snap.Rosters.Provenance.Freshness.State == domain.FreshStale || injuries.Provenance.Freshness.State != domain.FreshLive {
		r.Unavailable = append(r.Unavailable, Unavailable{Kind: "ir", Note: "IR eligibility not checked; injuries or roster unavailable or stale"})
		return
	}
	names := map[string]string{}
	for _, p := range snap.Players.Value {
		names[p.ID.String()] = p.Name
	}
	seen := map[string]bool{}
	for _, roster := range snap.Rosters.Value {
		if roster.FranchiseID != franchiseID {
			continue
		}
		for _, p := range roster.Players {
			id := p.ID.String()
			if p.RosterStatus != domain.RosterIR || seen[id] {
				continue
			}
			seen[id] = true
			decision := eligibility.IRClassification(eligibility.Injuries{Status: injuries.Status,
				Fresh: injuries.Provenance.Freshness}, id)
			if decision.Verdict != eligibility.Ineligible {
				continue
			}
			name := names[id]
			if name == "" {
				name = "Player name unavailable"
			}
			target := targetURL()
			r.Alerts = append(r.Alerts, Alert{Kind: "ir", Urgency: "U0", Title: "Possible IR violation: " + name,
				Detail: decision.Reason, Node: "hq", Workspace: "lineup-and-roster", PlayerID: id,
				MFLURL: target, Provenance: injuries.Provenance})
		}
	}
}

func ByeInputsNote(snap snapshot.Snapshot, mine string, byes report.HeldFeed[[]nflschedule.RawBye],
	week leagueweek.Week) string {
	if byes.FetchedAt.IsZero() || len(byes.Value) == 0 || byes.RefreshError != "" {
		return "Bye-week check unavailable: NFL bye weeks not held."
	}
	if week.Number == 0 {
		return "Bye-week check unavailable: current NFL week not held."
	}
	for _, roster := range snap.Rosters.Value {
		if roster.FranchiseID == mine && snap.Rosters.Provenance.Freshness.State != domain.FreshFail &&
			snap.Rosters.Provenance.Freshness.State != domain.FreshStale {
			return ""
		}
	}
	return "Bye-week check unavailable: your roster not held or out of date."
}

func ByeRoster(snap snapshot.Snapshot, mine string, byes []nflschedule.RawBye) ([]lineup.ByePlayer, []string, map[string]int) {
	identities := map[string]snapshot.Player{}
	for _, p := range snap.Players.Value {
		identities[p.ID.String()] = p
	}
	byeMap := map[string]int{}
	for _, b := range byes {
		byeMap[b.Team] = b.ByeWeek()
	}
	players, names := []lineup.ByePlayer{}, []string{}
	for _, roster := range snap.Rosters.Value {
		if roster.FranchiseID != mine {
			continue
		}
		for _, p := range roster.Players {
			identity := identities[p.ID.String()]
			players = append(players, lineup.ByePlayer{Position: identity.Position, Team: identity.Team,
				Active: p.RosterStatus == domain.RosterActive})
			names = append(names, identity.Name)
		}
	}
	return players, names, byeMap
}

type ByeAlertInput struct {
	Snapshot       snapshot.Snapshot
	Franchise      string
	Byes           report.HeldFeed[[]nflschedule.RawBye]
	Week           leagueweek.Week
	WeekFailure    string
	Starters       league.Starters
	RulesFreshness func() domain.Freshness
	Now            time.Time
}

func ByeAlerts(r *AlertReading, in ByeAlertInput, weekLock func(int) time.Time) {
	rules, err := lineup.ParseRules(in.Starters)
	note := ByeInputsNote(in.Snapshot, in.Franchise, in.Byes, in.Week)
	if err != nil || in.RulesFreshness().State == domain.FreshFail {
		note = "Bye-week check unavailable: lineup rules not held."
	}
	if in.WeekFailure != "" {
		note = "Bye-week check unavailable: current NFL week could not be checked."
	}
	if note != "" {
		r.Unavailable = append(r.Unavailable, Unavailable{Kind: "bye", Note: note})
		return
	}
	players, names, byeMap := ByeRoster(in.Snapshot, in.Franchise, in.Byes.Value)
	for _, p := range players {
		_, known := byeMap[p.Team]
		if p.Active && (!known || p.Position == "" || p.Position == domain.PosFlag) {
			r.Unavailable = append(r.Unavailable, Unavailable{Kind: "bye",
				Note: "Bye-week check unavailable: an active player's position, NFL team or bye week is missing."})
			return
		}
	}
	for _, short := range lineup.ByeShortfalls(rules, players, byeMap) {
		if short.Week < in.Week.Number || short.Week > in.Week.Number+1 {
			continue
		}
		lock := weekLock(short.Week)
		if lock.IsZero() {
			r.Unavailable = append(r.Unavailable, Unavailable{Kind: "bye",
				Note: fmt.Sprintf("Week %d lineup lock unavailable; check Franchise HQ for bye shortfalls.", short.Week)})
		}
		r.Alerts = append(r.Alerts, ByeAlert(short, players, names, byeMap, in.Now, lock,
			in.Byes.Provenance("MFL nflByeWeeks", "Bye weeks")))
	}
}

func ByeWeekLock(current leagueweek.Week, week int, now time.Time, read func(int) ([]nflschedule.RawWeek, bool, error)) time.Time {
	if week != current.Number {
		rows, found, err := read(week)
		if err != nil || !found || len(rows) == 0 {
			return time.Time{}
		}
		current, err = nflschedule.ToWeek(rows[0], now)
		if err != nil {
			return time.Time{}
		}
	}
	lock, _ := leagueweek.NextLock(now, current)
	return lock
}

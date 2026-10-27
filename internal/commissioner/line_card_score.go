package commissioner

import (
	"fmt"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/fantasyscore"
	"github.com/secureprospective/TheWarRoom/internal/fantasyscore/source"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

// lineCardWeek finds a team's line game in a week; nflverse game ids read SSSS_WW_AWAY_HOME and the held
// games are already one season's. A week the held games reach in which the team did not play is a bye,
// scored 0; a week they do not reach is not held.
func lineCardWeek(team string, week int, games map[source.LineGame]fantasyscore.UnitLine) (fantasyscore.UnitLine, bool) {
	weekHeld := false
	for game, line := range games {
		parts := strings.Split(game.GameID, "_")
		if len(parts) != 4 || parts[1] != fmt.Sprintf("%02d", week) {
			continue
		}
		if game.Team == team {
			return line, true
		}
		weekHeld = true
	}
	return fantasyscore.UnitLine{}, weekHeld
}

// countLineCard scores a started O-Line card as its team's line unit (DEFAULT-32 spec §3, Set B). The
// commissioner's stat corrections replace held components, and score changes add to the total, as they
// do for a player.
func (out NativePlayerScore) countLineCard(rules leaguerules.ScoringRules, team string,
	evidence source.NativeEvidence, corrections map[leaguerules.StatID]StatCorrection,
	delta leaguerules.Decimal) (NativePlayerScore, error) {
	out.Position, out.PositionWhy = leaguerules.OL, team+" O-Line"
	if rules.LineUnit.State == leaguerules.IsKnown && !rules.LineUnit.Value.Used {
		out.Words = "This league does not score an O-Line"
		return out, nil
	}
	var line fantasyscore.UnitLine
	held := false
	if evidence.OLine != nil {
		out.SHA256, out.FetchedAt = evidence.OLine.SHA256, &evidence.OLine.FetchedAt
		line, held = lineCardWeek(team, out.Week, evidence.OLine.Games)
	}
	switch {
	case len(corrections) != 0:
		line = correctedLine(line, held, corrections)
	case evidence.OLine == nil:
		out.Words = "No O-Line statistics are held for this season"
		return out, nil
	case !held:
		out.Words = "No O-Line statistics are held for this week"
		return out, nil
	}
	points, items, err := fantasyscore.ScoreUnit(rules, line)
	if err != nil {
		return out, fmt.Errorf("count held O-Line statistics: %w", err)
	}
	for _, item := range items {
		out.Items = append(out.Items, nativeScoreItem(item))
	}
	switch {
	case rules.LineUnit.State != leaguerules.IsKnown:
		out.Status, out.Words = "Needs a ruling", "Needs a ruling: "+points.Citation
		return out, nil
	case points.State != leaguerules.IsKnown:
		out.Status, out.Words = "Incomplete", "Not recorded yet: "+points.Citation
		out.Missing = append(out.Missing, points.Citation)
		return out, nil
	}
	total, err := addScoreDelta(points.Value, delta)
	if err != nil {
		return out, err
	}
	out.Status, out.Points = "Complete", &total
	out.Words = scorePointsWords(total) + " points"
	if !line.Played {
		out.Words += " (bye)"
	}
	return out, nil
}

// correctedLine is the line game with the commissioner's corrections in place of the held components. A
// corrected week the held games do not reach counts as played, from the corrections alone.
func correctedLine(line fantasyscore.UnitLine, held bool, corrections map[leaguerules.StatID]StatCorrection) fantasyscore.UnitLine {
	out := fantasyscore.UnitLine{Played: line.Played || !held,
		Quantities: make(map[leaguerules.StatID]leaguerules.Decimal, len(line.Quantities)+len(corrections))}
	for id, quantity := range line.Quantities {
		out.Quantities[id] = quantity
	}
	for id, correction := range corrections {
		out.Quantities[id] = correction.Value
	}
	return out
}

package fit

// keepGames is the games a season needs before it says where a player stands.
const keepGames = 6

// fitKeep is the share of a player's scoring edge at the position that lasts into his next season:
// the slope of next season's points per game on this one's, over every pair of seasons he played
// keepGames in, held to 0..1. A position whose order reshuffles each year keeps little.
func fitKeep(sm samples, last int) float64 {
	var x, y []float64
	for _, p := range sm.pairs(last) {
		if p.next != nil && p.from.games >= keepGames && p.next.games >= keepGames {
			x, y = append(x, p.from.s.PPG()), append(y, p.next.s.PPG())
		}
	}
	if len(x) < 10 {
		return 1
	}
	mx, my := avg(x), avg(y)
	var sxy, sxx float64
	for i := range x {
		sxy += (x[i] - mx) * (y[i] - my)
		sxx += (x[i] - mx) * (x[i] - mx)
	}
	if sxx == 0 {
		return 1
	}
	return clamp(sxy/sxx, 0, 1)
}

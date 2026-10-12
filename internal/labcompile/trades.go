package labcompile

import "time"

func assetCounts(gave []Asset) (int64, int64) {
	players, picks := int64(0), int64(0)
	for _, a := range gave {
		switch a.Kind {
		case "player":
			players++
		case "pick":
			picks++
		}
	}
	return players, picks
}
func daysTo(d *time.Time, ts int64) *int64 {
	if d == nil {
		return nil
	}
	return ptr(int64(d.Sub(dayOf(ts)) / (24 * time.Hour)))
}
func (l *Lab) tradeRow(t Trade, deal AnalysisDeal) tradeRow {
	a, b := l.situation(t.Year, t.Ts, t.A), l.situation(t.Year, t.Ts, t.B)
	ap, ak := assetCounts(t.AGave)
	bp, bk := assetCounts(t.BGave)
	mk := l.Marks[t.Year]
	return tradeRow{Ts: t.Ts, Year: t.Year, A: t.A, B: t.B, ARank: a.Rank, BRank: b.Rank, RankBasis: a.Basis, ARecord: a.Record, BRecord: b.Record, APrev: l.yearBefore(t.Year, a.Basis, t.A), BPrev: l.yearBefore(t.Year, b.Basis, t.B), APlayers: ap, APicks: ak, BPlayers: bp, BPicks: bk, AOther: int64(len(t.AGave)) - ap - ak, BOther: int64(len(t.BGave)) - bp - bk, OneSided: t.OneSided, Phase: l.phase(t.Year, t.Ts), Week: l.weekAt(t.Year, t.Ts), ToDeadline: daysTo(mk.Deadline, t.Ts), ToDraft: daysTo(mk.DraftStart, t.Ts), ACap: l.capRoom(t.Year, t.Ts, t.A), BCap: l.capRoom(t.Year, t.Ts, t.B), AName: l.Names[t.A][t.Year], BName: l.Names[t.B][t.Year], ANet: deal.ANet, ARealNet: deal.ARealNet, RealSeasons: deal.RealSeasons, Window: deal.Window, Season: deal.Season}
}
func (l *Lab) legRow(t Trade, tid int64, side int, a Asset, lv AnalysisLeg, known, estimated map[teamYear]int64, bySlot map[pickKey]Pick) legRow {
	giver, receiver := t.A, t.B
	if side == 1 {
		giver, receiver = receiver, giver
	}
	g, r := l.situation(t.Year, t.Ts, giver), l.situation(t.Year, t.Ts, receiver)
	out := legRow{Trade: tid, From: giver, To: receiver, FromRank: g.Rank, ToRank: r.Rank, FromPrev: l.yearBefore(t.Year, g.Basis, giver), ToPrev: l.yearBefore(t.Year, r.Basis, receiver), Kind: a.Kind, XValue: lv.Value, XClass: lv.Class, XPrice: lv.Price, XReal: lv.Real, XRealSeasons: lv.RealSeasons}
	switch a.Kind {
	case "player":
		l.playerLeg(&out, a, t, giver, receiver)
	case "pick":
		l.pickLeg(&out, a, t, known, estimated, bySlot)
	}
	return out
}
func (l *Lab) playerLeg(out *legRow, a Asset, t Trade, giver, receiver string) {
	pid := value(a.ID)
	c, seen := l.contract(pid, t.Year, t.Ts, giver), "before"
	if c == nil {
		c, seen = l.contractAfter(pid, t.Year, t.Ts, receiver), "after"
	}
	use := l.afterUse(pid, t.Year, t.Ts, receiver)
	out.Pid = a.ID
	out.Pos = l.pos(pid, t.Year)
	out.Age = l.ageAt(pid, t.Ts)
	out.Tier = l.knownTier(pid, t.Year, t.Ts)
	out.AfterStarts = use.Starts
	out.AfterPts = use.Points
	out.AfterWeeks = ptr(use.Weeks)
	if c != nil {
		out.ContractFrom = ptr(seen)
		out.Salary = c.Salary
		out.Contract = c.Status
		if value(c.Until) != 0 {
			out.YearsLeft = ptr(max(int64(0), *c.Until-max(t.Year, int64(dayOf(t.Ts).Year()))+1))
		}
	}
}
func (l *Lab) pickLeg(out *legRow, a Asset, t Trade, known, estimated map[teamYear]int64, bySlot map[pickKey]Pick) {
	slot := a.Slot
	if value(slot) == 0 {
		if n, ok := known[teamYear{value(a.Year), value(a.Orig)}]; ok {
			slot = ptr(n)
		} else {
			slot = nil
		}
	}
	out.PickYear = a.Year
	out.Round = a.Round
	out.YearsOut = ptr(l.yearsOut(value(a.Year), t.Ts))
	out.Slot = slot
	out.Orig = a.Orig
	if slot != nil {
		if made, ok := bySlot[pickKey{value(a.Year), value(a.Round), *slot}]; ok {
			out.PickPid = ptr(made.Pid)
		}
	}
	guess := slot
	if guess == nil {
		if n, ok := estimated[teamYear{value(a.Year), value(a.Orig)}]; ok {
			guess = ptr(n)
		}
	}
	if guess != nil {
		third := "late"
		if *guess <= 11 {
			third = "early"
		} else if *guess <= 22 {
			third = "middle"
		}
		out.Third = &third
	}
	if value(a.Orig) != "" {
		out.OrigRank = l.situation(t.Year, t.Ts, *a.Orig).Rank
	}
}
func (l *Lab) yearsOut(pickYear, ts int64) int64 {
	d := dayOf(ts)
	y := int64(d.Year())
	end := l.Marks[y].DraftEnd
	if end == nil {
		end = ptrDate(time.Date(d.Year(), time.June, 1, 0, 0, 0, 0, time.UTC))
	}
	next := y
	if d.After(*end) {
		next++
	}
	return pickYear - next
}

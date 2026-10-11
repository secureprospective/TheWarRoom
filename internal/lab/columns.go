package lab

import "fmt"

func (r RawTradeColumns) Validate() error {
	n := len(r.Ts)
	columns := []struct {
		name   string
		length int
	}{
		{"trades.year", len(r.Year)},
		{"trades.a", len(r.A)},
		{"trades.b", len(r.B)},
		{"trades.aRank", len(r.ARank)},
		{"trades.bRank", len(r.BRank)},
		{"trades.rankBasis", len(r.RankBasis)},
		{"trades.aRecord", len(r.ARecord)},
		{"trades.bRecord", len(r.BRecord)},
		{"trades.aPrev", len(r.APrev)},
		{"trades.bPrev", len(r.BPrev)},
		{"trades.aPlayers", len(r.APlayers)},
		{"trades.aPicks", len(r.APicks)},
		{"trades.bPlayers", len(r.BPlayers)},
		{"trades.bPicks", len(r.BPicks)},
		{"trades.aOther", len(r.AOther)},
		{"trades.bOther", len(r.BOther)},
		{"trades.oneSided", len(r.OneSided)},
		{"trades.phase", len(r.Phase)},
		{"trades.week", len(r.Week)},
		{"trades.toDeadline", len(r.ToDeadline)},
		{"trades.toDraft", len(r.ToDraft)},
		{"trades.aCap", len(r.ACap)},
		{"trades.bCap", len(r.BCap)},
		{"trades.aName", len(r.AName)},
		{"trades.bName", len(r.BName)},
		{"trades.aNet", len(r.ANet)},
		{"trades.aRealNet", len(r.ARealNet)},
		{"trades.realSeasons", len(r.RealSeasons)},
		{"trades.window", len(r.Window)},
		{"trades.season", len(r.Season)},
	}
	for _, c := range columns {
		if c.length != n {
			return fmt.Errorf("%s: column length differs", c.name)
		}
	}
	return nil
}
func (r RawTradeColumns) rows() []Trade {
	out := make([]Trade, len(r.Ts))
	for i := range out {
		out[i] = Trade{
			Ts:          r.Ts[i],
			Year:        r.Year[i],
			A:           r.A[i],
			B:           r.B[i],
			ARank:       r.ARank[i],
			BRank:       r.BRank[i],
			RankBasis:   r.RankBasis[i],
			ARecord:     r.ARecord[i],
			BRecord:     r.BRecord[i],
			APrev:       r.APrev[i],
			BPrev:       r.BPrev[i],
			APlayers:    r.APlayers[i],
			APicks:      r.APicks[i],
			BPlayers:    r.BPlayers[i],
			BPicks:      r.BPicks[i],
			AOther:      r.AOther[i],
			BOther:      r.BOther[i],
			OneSided:    r.OneSided[i],
			Phase:       r.Phase[i],
			Week:        r.Week[i],
			ToDeadline:  r.ToDeadline[i],
			ToDraft:     r.ToDraft[i],
			ACap:        r.ACap[i],
			BCap:        r.BCap[i],
			AName:       r.AName[i],
			BName:       r.BName[i],
			ANet:        r.ANet[i],
			ARealNet:    r.ARealNet[i],
			RealSeasons: r.RealSeasons[i],
			Window:      r.Window[i],
			Season:      r.Season[i],
		}
	}
	return out
}
func (r RawLegColumns) Validate() error {
	n := len(r.XValue)
	columns := []struct {
		name   string
		length int
	}{
		{"legs.xClass", len(r.XClass)},
		{"legs.xPrice", len(r.XPrice)},
		{"legs.xReal", len(r.XReal)},
		{"legs.xRealSeasons", len(r.XRealSeasons)},
		{"legs.trade", len(r.Trade)},
		{"legs.from", len(r.From)},
		{"legs.to", len(r.To)},
		{"legs.fromRank", len(r.FromRank)},
		{"legs.toRank", len(r.ToRank)},
		{"legs.fromPrev", len(r.FromPrev)},
		{"legs.toPrev", len(r.ToPrev)},
		{"legs.kind", len(r.Kind)},
		{"legs.contractFrom", len(r.ContractFrom)},
		{"legs.pid", len(r.Pid)},
		{"legs.pos", len(r.Pos)},
		{"legs.age", len(r.Age)},
		{"legs.salary", len(r.Salary)},
		{"legs.yearsLeft", len(r.YearsLeft)},
		{"legs.contract", len(r.Contract)},
		{"legs.tier", len(r.Tier)},
		{"legs.afterStarts", len(r.AfterStarts)},
		{"legs.afterPts", len(r.AfterPts)},
		{"legs.afterWeeks", len(r.AfterWeeks)},
		{"legs.pickYear", len(r.PickYear)},
		{"legs.round", len(r.Round)},
		{"legs.yearsOut", len(r.YearsOut)},
		{"legs.slot", len(r.Slot)},
		{"legs.third", len(r.Third)},
		{"legs.orig", len(r.Orig)},
		{"legs.origRank", len(r.OrigRank)},
		{"legs.pickPid", len(r.PickPid)},
	}
	for _, c := range columns {
		if c.length != n {
			return fmt.Errorf("%s: column length differs", c.name)
		}
	}
	return nil
}
func (r RawLegColumns) rows() []Leg {
	out := make([]Leg, len(r.XValue))
	for i := range out {
		out[i] = Leg{
			XValue:       r.XValue[i],
			XClass:       r.XClass[i],
			XPrice:       r.XPrice[i],
			XReal:        r.XReal[i],
			XRealSeasons: r.XRealSeasons[i],
			Trade:        r.Trade[i],
			From:         r.From[i],
			To:           r.To[i],
			FromRank:     r.FromRank[i],
			ToRank:       r.ToRank[i],
			FromPrev:     r.FromPrev[i],
			ToPrev:       r.ToPrev[i],
			Kind:         r.Kind[i],
			ContractFrom: r.ContractFrom[i],
			Pid:          r.Pid[i],
			Pos:          r.Pos[i],
			Age:          r.Age[i],
			Salary:       r.Salary[i],
			YearsLeft:    r.YearsLeft[i],
			Contract:     r.Contract[i],
			Tier:         r.Tier[i],
			AfterStarts:  r.AfterStarts[i],
			AfterPts:     r.AfterPts[i],
			AfterWeeks:   r.AfterWeeks[i],
			PickYear:     r.PickYear[i],
			Round:        r.Round[i],
			YearsOut:     r.YearsOut[i],
			Slot:         r.Slot[i],
			Third:        r.Third[i],
			Orig:         r.Orig[i],
			OrigRank:     r.OrigRank[i],
			PickPid:      r.PickPid[i],
		}
	}
	return out
}
func (r RawDeliveryColumns) Validate() error {
	n := len(r.Trade)
	columns := []struct {
		name   string
		length int
	}{
		{"delivered.class", len(r.Class)},
		{"delivered.expected", len(r.Expected)},
		{"delivered.delivered", len(r.Delivered)},
		{"delivered.seasons", len(r.Seasons)},
	}
	for _, c := range columns {
		if c.length != n {
			return fmt.Errorf("%s: column length differs", c.name)
		}
	}
	return nil
}
func (r RawDeliveryColumns) rows() []Delivery {
	out := make([]Delivery, len(r.Trade))
	for i := range out {
		out[i] = Delivery{
			Trade:     r.Trade[i],
			Class:     r.Class[i],
			Expected:  r.Expected[i],
			Delivered: r.Delivered[i],
			Seasons:   r.Seasons[i],
		}
	}
	return out
}
func (r RawPlayerColumns) Validate() error {
	n := len(r.ID)
	columns := []struct {
		name   string
		length int
	}{
		{"players.name", len(r.Name)},
		{"players.pos", len(r.Pos)},
		{"players.birth", len(r.Birth)},
		{"players.nflYear", len(r.NflYear)},
		{"players.nflRound", len(r.NflRound)},
	}
	for _, c := range columns {
		if c.length != n {
			return fmt.Errorf("%s: column length differs", c.name)
		}
	}
	return nil
}
func (r RawPlayerColumns) rows() []Player {
	out := make([]Player, len(r.ID))
	for i := range out {
		out[i] = Player{
			ID:       r.ID[i],
			Name:     r.Name[i],
			Pos:      r.Pos[i],
			Birth:    r.Birth[i],
			NflYear:  r.NflYear[i],
			NflRound: r.NflRound[i],
		}
	}
	return out
}
func (r RawDraftPickColumns) Validate() error {
	n := len(r.Year)
	columns := []struct {
		name   string
		length int
	}{
		{"draft.round", len(r.Round)},
		{"draft.slot", len(r.Slot)},
		{"draft.overall", len(r.Overall)},
		{"draft.fid", len(r.Fid)},
		{"draft.pid", len(r.Pid)},
		{"draft.pos", len(r.Pos)},
		{"draft.ts", len(r.Ts)},
		{"draft.group", len(r.Group)},
		{"draft.starterSeasons", len(r.StarterSeasons)},
		{"draft.best", len(r.Best)},
		{"draft.seasonsSeen", len(r.SeasonsSeen)},
		{"draft.pts3", len(r.Pts3)},
		{"draft.nflRound", len(r.NflRound)},
		{"draft.nflPick", len(r.NflPick)},
		{"draft.age", len(r.Age)},
		{"draft.rank", len(r.Rank)},
		{"draft.prev", len(r.Prev)},
		{"draft.keptYear2", len(r.KeptYear2)},
		{"draft.wins", len(r.Wins)},
		{"draft.slotWins", len(r.SlotWins)},
	}
	for _, c := range columns {
		if c.length != n {
			return fmt.Errorf("%s: column length differs", c.name)
		}
	}
	return nil
}
func (r RawDraftPickColumns) rows() []DraftPick {
	out := make([]DraftPick, len(r.Year))
	for i := range out {
		out[i] = DraftPick{
			Year:           r.Year[i],
			Round:          r.Round[i],
			Slot:           r.Slot[i],
			Overall:        r.Overall[i],
			Fid:            r.Fid[i],
			Pid:            r.Pid[i],
			Pos:            r.Pos[i],
			Ts:             r.Ts[i],
			Group:          r.Group[i],
			StarterSeasons: r.StarterSeasons[i],
			Best:           r.Best[i],
			SeasonsSeen:    r.SeasonsSeen[i],
			Pts3:           r.Pts3[i],
			NflRound:       r.NflRound[i],
			NflPick:        r.NflPick[i],
			Age:            r.Age[i],
			Rank:           r.Rank[i],
			Prev:           r.Prev[i],
			KeptYear2:      r.KeptYear2[i],
			Wins:           r.Wins[i],
			SlotWins:       r.SlotWins[i],
		}
	}
	return out
}
func (r RawTeamSeasonColumns) Validate() error {
	n := len(r.Year)
	columns := []struct {
		name   string
		length int
	}{
		{"teamSeasons.fid", len(r.Fid)},
		{"teamSeasons.rank", len(r.Rank)},
		{"teamSeasons.liveRank", len(r.LiveRank)},
		{"teamSeasons.w", len(r.W)},
		{"teamSeasons.l", len(r.L)},
		{"teamSeasons.t", len(r.T)},
		{"teamSeasons.pf", len(r.Pf)},
		{"teamSeasons.playoffs", len(r.Playoffs)},
		{"teamSeasons.champion", len(r.Champion)},
		{"teamSeasons.finalist", len(r.Finalist)},
		{"teamSeasons.capUsed", len(r.CapUsed)},
		{"teamSeasons.extended", len(r.Extended)},
		{"teamSeasons.tagged", len(r.Tagged)},
		{"teamSeasons.starterAge", len(r.StarterAge)},
		{"teamSeasons.name", len(r.Name)},
		{"teamSeasons.allPlay", len(r.AllPlay)},
		{"teamSeasons.expectedWins", len(r.ExpectedWins)},
		{"teamSeasons.luck", len(r.Luck)},
	}
	for _, c := range columns {
		if c.length != n {
			return fmt.Errorf("%s: column length differs", c.name)
		}
	}
	return nil
}
func (r RawTeamSeasonColumns) rows() []TeamSeason {
	out := make([]TeamSeason, len(r.Year))
	for i := range out {
		out[i] = TeamSeason{
			Year:         r.Year[i],
			Fid:          r.Fid[i],
			Rank:         r.Rank[i],
			LiveRank:     r.LiveRank[i],
			W:            r.W[i],
			L:            r.L[i],
			T:            r.T[i],
			Pf:           r.Pf[i],
			Playoffs:     r.Playoffs[i],
			Champion:     r.Champion[i],
			Finalist:     r.Finalist[i],
			CapUsed:      r.CapUsed[i],
			Extended:     r.Extended[i],
			Tagged:       r.Tagged[i],
			StarterAge:   r.StarterAge[i],
			Name:         r.Name[i],
			AllPlay:      r.AllPlay[i],
			ExpectedWins: r.ExpectedWins[i],
			Luck:         r.Luck[i],
		}
	}
	return out
}
func (r RawOfferColumns) Validate() error {
	n := len(r.Ts)
	columns := []struct {
		name   string
		length int
	}{
		{"offers.year", len(r.Year)},
		{"offers.season", len(r.Season)},
		{"offers.phase", len(r.Phase)},
		{"offers.by", len(r.By)},
		{"offers.to", len(r.To)},
		{"offers.outcome", len(r.Outcome)},
		{"offers.decided", len(r.Decided)},
	}
	for _, c := range columns {
		if c.length != n {
			return fmt.Errorf("%s: column length differs", c.name)
		}
	}
	return validateOfferValues(r)
}
func (r RawOfferColumns) rows() []Offer {
	out := make([]Offer, len(r.Ts))
	for i := range out {
		out[i] = Offer{
			Ts:      r.Ts[i],
			Year:    r.Year[i],
			Season:  r.Season[i],
			Phase:   r.Phase[i],
			By:      r.By[i],
			To:      r.To[i],
			Outcome: r.Outcome[i],
			Decided: r.Decided[i],
		}
	}
	return out
}

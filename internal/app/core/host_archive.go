package core

import (
	"context"
	"slices"

	"github.com/secureprospective/TheWarRoom/internal/app/report"
	"github.com/secureprospective/TheWarRoom/internal/facts"
)

func (s *State) ArchiveContext() context.Context { return s.Ctx }

func (s *State) ArchiveOpponent(team string) string {
	s.WeekMu.Lock()
	week := s.Week.Number
	s.WeekMu.Unlock()
	s.SeasonMu.Lock()
	defer s.SeasonMu.Unlock()
	for _, m := range s.SeasonLineups.Value.Matchups {
		if s.SeasonLineups.Value.Week != week {
			break
		}
		if m.Home == team {
			return m.Away
		}
		if m.Away == team {
			return m.Home
		}
	}
	if s.ArchiveOpponentFallback != nil {
		return s.ArchiveOpponentFallback(week, team)
	}
	return ""
}

func (s *State) ArchiveFacts(ctx context.Context) (report.ArchiveSnapshot, bool, error) {
	s.FactsMu.Lock()
	defer s.FactsMu.Unlock()
	cache, stale, err := s.ReadArchive(ctx)
	cache.Facts = cloneArchiveFacts(cache.Facts)
	return cache, stale, err
}

func cloneArchiveFacts(in facts.Facts) facts.Facts {
	return facts.Facts{
		Season:          cloneArchiveSeason(in.Season),
		FranchiseSeason: cloneArchiveFranchiseSeason(in.FranchiseSeason),
		Standings:       cloneArchiveStandings(in.Standings),
		ScoringRule:     cloneArchiveScoringRule(in.ScoringRule),
		RuleCode:        cloneArchiveRuleCode(in.RuleCode),
		CalendarEvent:   cloneArchiveCalendarEvent(in.CalendarEvent),
		Player:          cloneArchivePlayer(in.Player),
		PlayerConflict:  cloneArchivePlayerConflict(in.PlayerConflict),
		PlayerSeason:    cloneArchivePlayerSeason(in.PlayerSeason),
		SourceFile:      cloneArchiveSourceFile(in.SourceFile),
		LoadNote:        cloneArchiveLoadNote(in.LoadNote),
		NoteReading:     cloneArchiveNoteReading(in.NoteReading),
		Game:            cloneArchiveGame(in.Game),
		Lineup:          cloneArchiveLineup(in.Lineup),
		PlayerWeek:      cloneArchivePlayerWeek(in.PlayerWeek),
		InjuryWeek:      cloneArchiveInjuryWeek(in.InjuryWeek),
		RosterWeek:      cloneRosterWeeks(in.RosterWeek),
		NflGame:         cloneNflGames(in.NflGame),
		PointsAllowed:   cloneArchivePointsAllowed(in.PointsAllowed),
		PlayoffBracket:  cloneArchivePlayoffBracket(in.PlayoffBracket),
		PlayoffGame:     cloneArchivePlayoffGame(in.PlayoffGame),
		Contract:        cloneArchiveContract(in.Contract),
		CapCharge:       cloneArchiveCapCharge(in.CapCharge),
		DraftPick:       cloneArchiveDraftPick(in.DraftPick),
		PickOwned:       cloneArchivePickOwned(in.PickOwned),
		Txn:             cloneArchiveTxn(in.Txn),
		TxnAsset:        cloneArchiveTxnAsset(in.TxnAsset),
		TradeBait:       cloneArchiveTradeBait(in.TradeBait),
		MflAdp:          cloneArchiveMflAdp(in.MflAdp),
		MflAav:          cloneArchiveMflAav(in.MflAav),
	}
}

func cloneArchiveSeason(in facts.SeasonColumns) facts.SeasonColumns {
	return facts.SeasonColumns{
		Year:            slices.Clone(in.Year),
		LeagueID:        slices.Clone(in.LeagueID),
		Name:            clonePointers(in.Name),
		StartWeek:       clonePointers(in.StartWeek),
		EndWeek:         clonePointers(in.EndWeek),
		LastRegularWeek: clonePointers(in.LastRegularWeek),
		SalaryCap:       clonePointers(in.SalaryCap),
		RosterSize:      clonePointers(in.RosterSize),
		TaxiSquad:       clonePointers(in.TaxiSquad),
		InjuredReserve:  clonePointers(in.InjuredReserve),
		Starters:        clonePointers(in.Starters),
		Raw:             slices.Clone(in.Raw),
		Src:             slices.Clone(in.Src),
	}
}

func cloneArchiveFranchiseSeason(in facts.FranchiseSeasonColumns) facts.FranchiseSeasonColumns {
	return facts.FranchiseSeasonColumns{
		Year:           slices.Clone(in.Year),
		Fid:            slices.Clone(in.Fid),
		Name:           clonePointers(in.Name),
		Abbrev:         clonePointers(in.Abbrev),
		Division:       clonePointers(in.Division),
		DivisionName:   clonePointers(in.DivisionName),
		Conference:     clonePointers(in.Conference),
		ConferenceName: clonePointers(in.ConferenceName),
		Src:            slices.Clone(in.Src),
	}
}

func cloneArchiveStandings(in facts.StandingsColumns) facts.StandingsColumns {
	return facts.StandingsColumns{
		Year:     slices.Clone(in.Year),
		Fid:      slices.Clone(in.Fid),
		H2hW:     clonePointers(in.H2hW),
		H2hL:     clonePointers(in.H2hL),
		H2hT:     clonePointers(in.H2hT),
		Pf:       clonePointers(in.Pf),
		Pa:       clonePointers(in.Pa),
		Pp:       clonePointers(in.Pp),
		AllPlayW: clonePointers(in.AllPlayW),
		AllPlayL: clonePointers(in.AllPlayL),
		AllPlayT: clonePointers(in.AllPlayT),
		Eff:      clonePointers(in.Eff),
		Raw:      slices.Clone(in.Raw),
		Src:      slices.Clone(in.Src),
	}
}

func cloneArchiveScoringRule(in facts.ScoringRuleColumns) facts.ScoringRuleColumns {
	return facts.ScoringRuleColumns{
		Year:      slices.Clone(in.Year),
		Positions: slices.Clone(in.Positions),
		Event:     slices.Clone(in.Event),
		Range:     clonePointers(in.Range),
		Points:    clonePointers(in.Points),
		Src:       slices.Clone(in.Src),
	}
}

func cloneArchiveRuleCode(in facts.RuleCodeColumns) facts.RuleCodeColumns {
	return facts.RuleCodeColumns{
		Code:        slices.Clone(in.Code),
		Short:       clonePointers(in.Short),
		Description: clonePointers(in.Description),
		Src:         slices.Clone(in.Src),
	}
}

func cloneArchiveCalendarEvent(in facts.CalendarEventColumns) facts.CalendarEventColumns {
	return facts.CalendarEventColumns{
		Year:    slices.Clone(in.Year),
		EventID: slices.Clone(in.EventID),
		Type:    clonePointers(in.Type),
		Title:   clonePointers(in.Title),
		StartTs: clonePointers(in.StartTs),
		EndTs:   clonePointers(in.EndTs),
		Src:     slices.Clone(in.Src),
	}
}

func cloneArchivePlayer(in facts.PlayerColumns) facts.PlayerColumns {
	return facts.PlayerColumns{
		Pid:         slices.Clone(in.Pid),
		Name:        slices.Clone(in.Name),
		Position:    clonePointers(in.Position),
		Birthdate:   clonePointers(in.Birthdate),
		BirthdateTs: clonePointers(in.BirthdateTs),
		DraftYear:   clonePointers(in.DraftYear),
		DraftTeam:   clonePointers(in.DraftTeam),
		DraftRound:  clonePointers(in.DraftRound),
		DraftPick:   clonePointers(in.DraftPick),
		College:     clonePointers(in.College),
		Height:      clonePointers(in.Height),
		Weight:      clonePointers(in.Weight),
		FirstYear:   clonePointers(in.FirstYear),
		LastYear:    clonePointers(in.LastYear),
		Src:         slices.Clone(in.Src),
	}
}

func cloneArchivePlayerConflict(in facts.PlayerConflictColumns) facts.PlayerConflictColumns {
	return facts.PlayerConflictColumns{
		Pid:   slices.Clone(in.Pid),
		Field: slices.Clone(in.Field),
		Year:  slices.Clone(in.Year),
		Value: clonePointers(in.Value),
		Kept:  clonePointers(in.Kept),
	}
}

func cloneArchivePlayerSeason(in facts.PlayerSeasonColumns) facts.PlayerSeasonColumns {
	return facts.PlayerSeasonColumns{
		Year:     slices.Clone(in.Year),
		Pid:      slices.Clone(in.Pid),
		Name:     clonePointers(in.Name),
		Position: clonePointers(in.Position),
		NflTeam:  clonePointers(in.NflTeam),
		Src:      slices.Clone(in.Src),
	}
}

func cloneArchiveSourceFile(in facts.SourceFileColumns) facts.SourceFileColumns {
	return facts.SourceFileColumns{
		Src:    slices.Clone(in.Src),
		Bytes:  slices.Clone(in.Bytes),
		Sha256: slices.Clone(in.Sha256),
	}
}

func cloneArchiveLoadNote(in facts.LoadNoteColumns) facts.LoadNoteColumns {
	return facts.LoadNoteColumns{
		Step: slices.Clone(in.Step),
		Src:  clonePointers(in.Src),
		Note: slices.Clone(in.Note),
		N:    slices.Clone(in.N),
	}
}

func cloneArchiveNoteReading(in facts.NoteReadingColumns) facts.NoteReadingColumns {
	return facts.NoteReadingColumns{
		TxnID:      slices.Clone(in.TxnID),
		FromFid:    clonePointers(in.FromFid),
		ToFid:      clonePointers(in.ToFid),
		Kind:       slices.Clone(in.Kind),
		PickYear:   clonePointers(in.PickYear),
		PickRound:  clonePointers(in.PickRound),
		PickSlot:   clonePointers(in.PickSlot),
		PickOrig:   clonePointers(in.PickOrig),
		CapM:       clonePointers(in.CapM),
		CapSeasons: clonePointers(in.CapSeasons),
		Player:     clonePointers(in.Player),
		Clarity:    slices.Clone(in.Clarity),
		Reading:    slices.Clone(in.Reading),
		DraftCheck: clonePointers(in.DraftCheck),
		Confirmed:  slices.Clone(in.Confirmed),
	}
}

func cloneArchiveGame(in facts.GameColumns) facts.GameColumns {
	return facts.GameColumns{
		Year:   slices.Clone(in.Year),
		Week:   slices.Clone(in.Week),
		Fid:    slices.Clone(in.Fid),
		Opp:    clonePointers(in.Opp),
		IsHome: clonePointers(in.IsHome),
		Score:  clonePointers(in.Score),
		Result: clonePointers(in.Result),
		OptPts: clonePointers(in.OptPts),
		Src:    slices.Clone(in.Src),
	}
}

func cloneArchiveLineup(in facts.LineupColumns) facts.LineupColumns {
	return facts.LineupColumns{
		Year:        slices.Clone(in.Year),
		Week:        slices.Clone(in.Week),
		Fid:         slices.Clone(in.Fid),
		Pid:         slices.Clone(in.Pid),
		Started:     slices.Clone(in.Started),
		ShouldStart: clonePointers(in.ShouldStart),
		Score:       clonePointers(in.Score),
		Src:         slices.Clone(in.Src),
	}
}

func cloneArchivePlayerWeek(in facts.PlayerWeekColumns) facts.PlayerWeekColumns {
	return facts.PlayerWeekColumns{
		Year:      slices.Clone(in.Year),
		Week:      slices.Clone(in.Week),
		Pid:       slices.Clone(in.Pid),
		Score:     clonePointers(in.Score),
		Projected: clonePointers(in.Projected),
	}
}

func cloneArchiveInjuryWeek(in facts.InjuryWeekColumns) facts.InjuryWeekColumns {
	return facts.InjuryWeekColumns{
		Year:      slices.Clone(in.Year),
		Week:      slices.Clone(in.Week),
		Pid:       slices.Clone(in.Pid),
		Status:    clonePointers(in.Status),
		Details:   clonePointers(in.Details),
		ExpReturn: clonePointers(in.ExpReturn),
		Src:       slices.Clone(in.Src),
	}
}

func cloneArchivePointsAllowed(in facts.PointsAllowedColumns) facts.PointsAllowedColumns {
	return facts.PointsAllowedColumns{
		Year:     slices.Clone(in.Year),
		NflTeam:  slices.Clone(in.NflTeam),
		Position: slices.Clone(in.Position),
		Points:   clonePointers(in.Points),
		Src:      slices.Clone(in.Src),
	}
}

func cloneArchivePlayoffBracket(in facts.PlayoffBracketColumns) facts.PlayoffBracketColumns {
	return facts.PlayoffBracketColumns{
		Year:        slices.Clone(in.Year),
		BracketID:   slices.Clone(in.BracketID),
		Name:        clonePointers(in.Name),
		WinnerTitle: clonePointers(in.WinnerTitle),
		StartWeek:   clonePointers(in.StartWeek),
		Teams:       clonePointers(in.Teams),
		Src:         slices.Clone(in.Src),
	}
}

func cloneArchivePlayoffGame(in facts.PlayoffGameColumns) facts.PlayoffGameColumns {
	return facts.PlayoffGameColumns{
		Year:      slices.Clone(in.Year),
		BracketID: slices.Clone(in.BracketID),
		Week:      clonePointers(in.Week),
		GameID:    clonePointers(in.GameID),
		Fid:       clonePointers(in.Fid),
		Side:      clonePointers(in.Side),
		Points:    clonePointers(in.Points),
		Seed:      clonePointers(in.Seed),
		Src:       slices.Clone(in.Src),
	}
}

func cloneArchiveContract(in facts.ContractColumns) facts.ContractColumns {
	return facts.ContractColumns{
		Year:           slices.Clone(in.Year),
		Pid:            slices.Clone(in.Pid),
		Salary:         clonePointers(in.Salary),
		ContractYear:   clonePointers(in.ContractYear),
		ContractStatus: clonePointers(in.ContractStatus),
		ContractInfo:   clonePointers(in.ContractInfo),
		Src:            slices.Clone(in.Src),
	}
}

func cloneArchiveCapCharge(in facts.CapChargeColumns) facts.CapChargeColumns {
	return facts.CapChargeColumns{
		Year:        slices.Clone(in.Year),
		AdjID:       slices.Clone(in.AdjID),
		Fid:         clonePointers(in.Fid),
		Amount:      clonePointers(in.Amount),
		Description: clonePointers(in.Description),
		Ts:          clonePointers(in.Ts),
		Src:         slices.Clone(in.Src),
	}
}

func cloneArchiveDraftPick(in facts.DraftPickColumns) facts.DraftPickColumns {
	return facts.DraftPickColumns{
		Year:     slices.Clone(in.Year),
		Round:    slices.Clone(in.Round),
		Pick:     slices.Clone(in.Pick),
		Fid:      clonePointers(in.Fid),
		Pid:      clonePointers(in.Pid),
		Ts:       clonePointers(in.Ts),
		Day:      clonePointers(in.Day),
		Comments: clonePointers(in.Comments),
		Src:      slices.Clone(in.Src),
	}
}

func cloneArchivePickOwned(in facts.PickOwnedColumns) facts.PickOwnedColumns {
	return facts.PickOwnedColumns{
		SnapshotYear: slices.Clone(in.SnapshotYear),
		Source:       slices.Clone(in.Source),
		Fid:          slices.Clone(in.Fid),
		PickYear:     clonePointers(in.PickYear),
		Round:        clonePointers(in.Round),
		Slot:         clonePointers(in.Slot),
		OrigFid:      clonePointers(in.OrigFid),
		Code:         clonePointers(in.Code),
		Description:  clonePointers(in.Description),
		Src:          slices.Clone(in.Src),
	}
}

func cloneArchiveTxn(in facts.TxnColumns) facts.TxnColumns {
	return facts.TxnColumns{
		TxnID:     slices.Clone(in.TxnID),
		Year:      slices.Clone(in.Year),
		Ts:        clonePointers(in.Ts),
		Day:       clonePointers(in.Day),
		Type:      slices.Clone(in.Type),
		Fid:       clonePointers(in.Fid),
		Fid2:      clonePointers(in.Fid2),
		ByCommish: slices.Clone(in.ByCommish),
		Expires:   clonePointers(in.Expires),
		Comments:  clonePointers(in.Comments),
		Raw:       slices.Clone(in.Raw),
		Src:       slices.Clone(in.Src),
		SrcIdx:    slices.Clone(in.SrcIdx),
	}
}

func cloneArchiveTxnAsset(in facts.TxnAssetColumns) facts.TxnAssetColumns {
	return facts.TxnAssetColumns{
		TxnID:     slices.Clone(in.TxnID),
		Role:      slices.Clone(in.Role),
		FromFid:   clonePointers(in.FromFid),
		ToFid:     clonePointers(in.ToFid),
		Kind:      slices.Clone(in.Kind),
		Pid:       clonePointers(in.Pid),
		PickYear:  clonePointers(in.PickYear),
		PickRound: clonePointers(in.PickRound),
		PickSlot:  clonePointers(in.PickSlot),
		PickOrig:  clonePointers(in.PickOrig),
		Code:      slices.Clone(in.Code),
	}
}

func cloneArchiveTradeBait(in facts.TradeBaitColumns) facts.TradeBaitColumns {
	return facts.TradeBaitColumns{
		Year:          slices.Clone(in.Year),
		Fid:           slices.Clone(in.Fid),
		Ts:            clonePointers(in.Ts),
		WillGiveUp:    clonePointers(in.WillGiveUp),
		InExchangeFor: clonePointers(in.InExchangeFor),
		Src:           slices.Clone(in.Src),
	}
}

func cloneArchiveMflAdp(in facts.MflAdpColumns) facts.MflAdpColumns {
	return facts.MflAdpColumns{
		Year:    slices.Clone(in.Year),
		Pid:     slices.Clone(in.Pid),
		Rank:    clonePointers(in.Rank),
		AvgPick: clonePointers(in.AvgPick),
		MinPick: clonePointers(in.MinPick),
		MaxPick: clonePointers(in.MaxPick),
		Drafts:  clonePointers(in.Drafts),
		Src:     slices.Clone(in.Src),
	}
}

func cloneArchiveMflAav(in facts.MflAavColumns) facts.MflAavColumns {
	return facts.MflAavColumns{
		Year:     slices.Clone(in.Year),
		Pid:      slices.Clone(in.Pid),
		Rank:     clonePointers(in.Rank),
		AvgValue: clonePointers(in.AvgValue),
		MinValue: clonePointers(in.MinValue),
		MaxValue: clonePointers(in.MaxValue),
		Src:      slices.Clone(in.Src),
	}
}

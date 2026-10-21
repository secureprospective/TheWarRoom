package core

import (
	"fmt"
	"maps"
	"slices"

	"github.com/secureprospective/TheWarRoom/internal/lab"
)

func (s *State) CounterpartiesBuildStamp() string { return s.BuildStamp }

// LabData is initialized once, before bindings are published, and never refreshed.
// No shared mutex protects these immutable facts. Detach the subset this reading uses;
// neither the service nor its returned values may expose the shared decode's storage.
func (s *State) CounterpartiesFacts() (*lab.Data, error) {
	d, err := s.Lab()
	if err != nil {
		return nil, err
	}
	out := &lab.Data{Built: d.Built, ArchiveThrough: d.ArchiveThrough,
		Trades: slices.Clone(d.Trades), TeamSeasons: slices.Clone(d.TeamSeasons), Franchises: maps.Clone(d.Franchises)}
	for i := range out.Trades {
		out.Trades[i] = cloneCounterpartyTrade(out.Trades[i])
	}
	for i := range out.TeamSeasons {
		v := &out.TeamSeasons[i]
		v.Rank, v.Playoffs = clonePtr(v.Rank), clonePtr(v.Playoffs)
		v.AllPlay, v.ExpectedWins, v.Luck = clonePtr(v.AllPlay), clonePtr(v.ExpectedWins), clonePtr(v.Luck)
	}
	for id, f := range out.Franchises {
		f.Division, f.FormerNames = slices.Clone(f.Division), slices.Clone(f.FormerNames)
		out.Franchises[id] = f
	}
	return out, nil
}

func cloneCounterpartyTrade(v lab.Trade) lab.Trade {
	v.ARecord, v.BRecord = clonePtr(v.ARecord), clonePtr(v.BRecord)
	v.APrev, v.BPrev, v.ToDraft = clonePtr(v.APrev), clonePtr(v.BPrev), clonePtr(v.ToDraft)
	v.ACap, v.BCap, v.ANet, v.ARealNet = clonePtr(v.ACap), clonePtr(v.BCap), clonePtr(v.ANet), clonePtr(v.ARealNet)
	v.RealSeasons, v.Window, v.Season = clonePtr(v.RealSeasons), clonePtr(v.Window), clonePtr(v.Season)
	v.Legs = slices.Clone(v.Legs)
	for i := range v.Legs {
		v.Legs[i] = cloneCounterpartyLeg(v.Legs[i])
	}
	return v
}

func cloneCounterpartyLeg(v lab.Leg) lab.Leg {
	v.XValue, v.XClass, v.XPrice = clonePtr(v.XValue), clonePtr(v.XClass), clonePtr(v.XPrice)
	v.XReal, v.XRealSeasons = clonePtr(v.XReal), clonePtr(v.XRealSeasons)
	v.FromPrev, v.ToPrev = clonePtr(v.FromPrev), clonePtr(v.ToPrev)
	v.ContractFrom, v.Pid, v.Pos = clonePtr(v.ContractFrom), clonePtr(v.Pid), clonePtr(v.Pos)
	v.Age, v.Salary, v.YearsLeft = clonePtr(v.Age), clonePtr(v.Salary), clonePtr(v.YearsLeft)
	v.Contract, v.Tier = clonePtr(v.Contract), clonePtr(v.Tier)
	v.AfterStarts, v.AfterPts, v.AfterWeeks = clonePtr(v.AfterStarts), clonePtr(v.AfterPts), clonePtr(v.AfterWeeks)
	v.PickYear, v.Round, v.YearsOut, v.Slot = clonePtr(v.PickYear), clonePtr(v.Round), clonePtr(v.YearsOut), clonePtr(v.Slot)
	v.Third, v.Orig, v.OrigRank, v.PickPid = clonePtr(v.Third), clonePtr(v.Orig), clonePtr(v.OrigRank), clonePtr(v.PickPid)
	v.Player, v.PickPlayer = cloneCounterpartyPlayer(v.Player), cloneCounterpartyPlayer(v.PickPlayer)
	return v
}

func cloneCounterpartyPlayer(v *lab.Player) *lab.Player {
	out := clonePtr(v)
	if out != nil {
		out.Pos, out.Birth = clonePtr(out.Pos), clonePtr(out.Birth)
		out.NflYear, out.NflRound = clonePtr(out.NflYear), clonePtr(out.NflRound)
	}
	return out
}

// Lab returns the shared decode, or a fresh one for state built without NewApp.
// Its storage is immutable; feature hosts must detach what they expose.
func (s *State) Lab() (*lab.Data, error) {
	if s.LabData == nil {
		d, err := lab.Load()
		if err != nil {
			return nil, fmt.Errorf("load lab facts: %w", err)
		}
		return d, nil
	}
	return s.LabData()
}

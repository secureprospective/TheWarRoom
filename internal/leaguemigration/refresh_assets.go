package leaguemigration

import (
	"cmp"
	"fmt"
	"maps"
	"slices"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/domain"
)

func (p *refreshPlanner) replayTransactions(records []commissioner.Record) {
	if p.plan.Transactions == nil {
		return
	}
	origSeed, ok := initialSeedSnapshot(records)
	if !ok {
		return
	}
	groups, err := p.plan.Transactions(p.league, p.reason, origSeed)
	if err != nil {
		return
	}
	seen := requestSet(records)
	for _, group := range groups {
		if !p.canReplayGroup(group, seen, origSeed.RosterAt) {
			continue
		}
		nextTrial, ok := tryReplayGroup(p.trial, group, p.principal, p.now, p.opts.LineupFacts)
		if !ok {
			continue
		}
		p.trial = nextTrial
		p.raws = append(p.raws, group...)
		for _, cmd := range group {
			seen[cmd.Request] = true
			p.countReplayedCommand(cmd)
		}
		p.summary.Transactions++
	}
}

func (p *refreshPlanner) countReplayedCommand(cmd commissioner.RawCommand) {
	if cmd.Args.Correction == nil {
		return
	}
	switch cmd.Command {
	case commissioner.CorrectRoster:
		switch cmd.Args.Correction.Action {
		case "add":
			p.summary.RosterAdds++
		case "remove":
			p.summary.RosterRemoves++
		case "status":
			p.summary.RosterStatusChanges++
		}
	case commissioner.CorrectPick:
		p.summary.Picks++
	}
}

func (p *refreshPlanner) canReplayGroup(
	group []commissioner.RawCommand, seen map[string]bool, minTime time.Time,
) bool {
	if len(group) == 0 {
		return false
	}
	for _, cmd := range group {
		if cmd.League != p.league || seen[cmd.Request] ||
			!strings.HasPrefix(cmd.Request, "transactions.sync.") ||
			strings.HasPrefix(cmd.Request, "transactions.sync.baseline.") {
			return false
		}
		if cmd.Command != commissioner.CorrectRoster && cmd.Command != commissioner.CorrectPick {
			return false
		}
		at, err := cmd.EffectiveTime(p.now)
		if err != nil || at.Before(minTime) || at.After(p.now) {
			return false
		}
	}
	return true
}

func tryReplayGroup(
	trial []commissioner.Record,
	group []commissioner.RawCommand,
	principal string,
	now time.Time,
	lineupFacts func(season, week int) (commissioner.LineupFacts, error),
) ([]commissioner.Record, bool) {
	candidate := trial
	for _, cmd := range group {
		next, err := appendTrialRecord(candidate, cmd, principal, now, lineupFacts)
		if err != nil {
			return nil, false
		}
		candidate = next
	}
	return candidate, true
}

func (p *refreshPlanner) reconcileAssets(serverCharges map[string]bool) error {
	assets, err := commissioner.NativeAssetsReading(p.league, p.trial, p.now)
	if err != nil {
		return fmt.Errorf("read trial assets: %w", err)
	}
	if err := p.reconcileRosters(assets.Snapshot); err != nil {
		return err
	}
	if err := p.reconcileAdjustments(assets.Snapshot, serverCharges); err != nil {
		return err
	}
	return p.reconcilePicks(assets.Snapshot)
}

type playerCopyKey struct {
	side string
	id   string
}

func comparePlayerKey(a, b playerCopyKey) int {
	if c := strings.Compare(a.side, b.side); c != 0 {
		return c
	}
	return strings.Compare(a.id, b.id)
}

func (p *refreshPlanner) reconcileRosters(cur *commissioner.RawNativeSnapshot) error {
	curMap := map[playerCopyKey]commissioner.NativePlayer{}
	for _, pl := range cur.Players {
		curMap[playerCopyKey{side: cur.SideOf(pl.Franchise), id: pl.ID}] = pl
	}
	wantMap := map[playerCopyKey]commissioner.NativePlayer{}
	for _, pl := range p.plan.Seed.Players {
		wantMap[playerCopyKey{side: cur.SideOf(pl.Franchise), id: pl.ID}] = pl
	}
	if err := p.emitRosterRemovals(curMap, wantMap); err != nil {
		return err
	}
	return p.emitRosterAddsAndUpdates(curMap, wantMap)
}

func (p *refreshPlanner) emitRosterRemovals(
	curMap, wantMap map[playerCopyKey]commissioner.NativePlayer,
) error {
	keys := slices.SortedFunc(maps.Keys(curMap), comparePlayerKey)
	for _, k := range keys {
		curP := curMap[k]
		wantP, ok := wantMap[k]
		if ok && wantP.Franchise == curP.Franchise {
			continue
		}
		raw := p.correctionCmd(commissioner.CorrectRoster, "roster-remove",
			fmt.Sprintf("remove player %s from %s", curP.ID, curP.Franchise),
			commissioner.RawCorrection{Player: curP.ID, Franchise: curP.Franchise, Action: "remove"})
		if err := p.emit(raw); err != nil {
			return err
		}
		p.summary.RosterRemoves++
	}
	return nil
}

func (p *refreshPlanner) emitRosterAddsAndUpdates(
	curMap, wantMap map[playerCopyKey]commissioner.NativePlayer,
) error {
	keys := slices.SortedFunc(maps.Keys(wantMap), comparePlayerKey)
	for _, k := range keys {
		wantP := wantMap[k]
		curP, ok := curMap[k]
		if !ok || curP.Franchise != wantP.Franchise {
			if err := p.emitRosterAdd(wantP); err != nil {
				return err
			}
			continue
		}
		if curP.Status != wantP.Status {
			raw := p.correctionCmd(commissioner.CorrectRoster, "roster-status",
				fmt.Sprintf("set player %s on %s to %s", wantP.ID, wantP.Franchise, wantP.Status),
				commissioner.RawCorrection{
					Player: wantP.ID, Franchise: wantP.Franchise, Action: "status", Status: wantP.Status,
				})
			if err := p.emit(raw); err != nil {
				return err
			}
			p.summary.RosterStatusChanges++
		}
		if !sameContract(curP, wantP) {
			contract := contractFromPlayer(wantP)
			raw := p.correctionCmd(commissioner.CorrectRoster, "contract",
				fmt.Sprintf("update contract for player %s on %s", wantP.ID, wantP.Franchise),
				commissioner.RawCorrection{
					Player: wantP.ID, Franchise: wantP.Franchise, Action: "contract", Contract: &contract,
				})
			if err := p.emit(raw); err != nil {
				return err
			}
			p.summary.Contracts++
		}
	}
	return nil
}

func (p *refreshPlanner) emitRosterAdd(wantP commissioner.NativePlayer) error {
	contract := contractFromPlayer(wantP)
	addRaw := p.correctionCmd(commissioner.CorrectRoster, "roster-add",
		fmt.Sprintf("add player %s to %s", wantP.ID, wantP.Franchise),
		commissioner.RawCorrection{
			Player: wantP.ID, Franchise: wantP.Franchise, Action: "add", Contract: &contract,
		})
	if err := p.emit(addRaw); err != nil {
		return err
	}
	p.summary.RosterAdds++
	if wantP.Status != domain.RosterActive {
		statusRaw := p.correctionCmd(commissioner.CorrectRoster, "roster-status",
			fmt.Sprintf("set player %s on %s to %s", wantP.ID, wantP.Franchise, wantP.Status),
			commissioner.RawCorrection{
				Player: wantP.ID, Franchise: wantP.Franchise, Action: "status", Status: wantP.Status,
			})
		if err := p.emit(statusRaw); err != nil {
			return err
		}
		p.summary.RosterStatusChanges++
	}
	return nil
}

func (p *refreshPlanner) reconcileAdjustments(
	cur *commissioner.RawNativeSnapshot, serverCharges map[string]bool,
) error {
	curMap := map[string]commissioner.NativeCharge{}
	for _, a := range cur.Adjustments {
		curMap[a.ID] = a
	}
	wantMap := map[string]commissioner.NativeCharge{}
	for _, a := range p.plan.Seed.Adjustments {
		wantMap[a.ID] = a
	}
	for _, id := range slices.Sorted(maps.Keys(wantMap)) {
		w := wantMap[id]
		if c, ok := curMap[id]; ok && c == w {
			continue
		}
		charge := w
		raw := p.correctionCmd(commissioner.CorrectRoster, "adjustment",
			fmt.Sprintf("record salary adjustment %s on %s", w.ID, w.Franchise),
			commissioner.RawCorrection{Franchise: w.Franchise, Action: "adjustment", Charge: &charge})
		if err := p.emit(raw); err != nil {
			return err
		}
		p.summary.Adjustments++
	}
	for _, id := range slices.Sorted(maps.Keys(curMap)) {
		if _, inMFL := wantMap[id]; inMFL {
			continue
		}
		if serverCharges[id] && !p.opts.DropAppAdjustments {
			continue
		}
		c := curMap[id]
		charge := commissioner.NativeCharge{ID: c.ID, Franchise: c.Franchise}
		raw := p.correctionCmd(commissioner.CorrectRoster, "adjustment-remove",
			fmt.Sprintf("remove salary adjustment %s from %s", c.ID, c.Franchise),
			commissioner.RawCorrection{Franchise: c.Franchise, Action: "remove_adjustment", Charge: &charge})
		if err := p.emit(raw); err != nil {
			return err
		}
		p.summary.Adjustments++
	}
	return nil
}

type pickIdentity struct {
	season   int
	round    int
	slot     int
	original string
}

func comparePickIdentity(a, b pickIdentity) int {
	if c := cmp.Compare(a.season, b.season); c != 0 {
		return c
	}
	if c := cmp.Compare(a.round, b.round); c != 0 {
		return c
	}
	if c := cmp.Compare(a.slot, b.slot); c != 0 {
		return c
	}
	return strings.Compare(a.original, b.original)
}

func (p *refreshPlanner) reconcilePicks(cur *commissioner.RawNativeSnapshot) error {
	curMap := map[pickIdentity]commissioner.NativePick{}
	for _, pk := range cur.Picks {
		curMap[pickIdentity{pk.Season, pk.Round, pk.Slot, pk.Original}] = pk
	}
	wantMap := map[pickIdentity]commissioner.NativePick{}
	for _, pk := range p.plan.Seed.Picks {
		wantMap[pickIdentity{pk.Season, pk.Round, pk.Slot, pk.Original}] = pk
	}
	for _, k := range slices.SortedFunc(maps.Keys(wantMap), comparePickIdentity) {
		w := wantMap[k]
		c, exists := curMap[k]
		if exists && c.Owner == w.Owner && c.Player == w.Player && c.Skipped == w.Skipped {
			continue
		}
		if err := p.emitPickUpdate(w, c, exists); err != nil {
			return err
		}
	}
	for _, k := range slices.SortedFunc(maps.Keys(curMap), comparePickIdentity) {
		if _, inMFL := wantMap[k]; inMFL {
			continue
		}
		c := curMap[k]
		pick := commissioner.CorrectionPick{
			Season: c.Season, Round: c.Round, Slot: c.Slot, Original: c.Original, Remove: true,
		}
		raw := p.correctionCmd(commissioner.CorrectPick, "pick-remove", "remove "+pickKey(c),
			commissioner.RawCorrection{Owner: c.Owner, Pick: &pick})
		if err := p.emit(raw); err != nil {
			return err
		}
		p.summary.Picks++
	}
	return nil
}

func (p *refreshPlanner) emitPickUpdate(w, c commissioner.NativePick, exists bool) error {
	pick := commissioner.CorrectionPick{Season: w.Season, Round: w.Round, Slot: w.Slot, Original: w.Original}
	switch {
	case !exists:
		pick.Add, pick.Player, pick.Skipped = true, w.Player, w.Skipped
	case w.Player != "" || w.Skipped:
		pick.Player, pick.Skipped = w.Player, w.Skipped
	case c.Player != "" || c.Skipped:
		pick.Clear = true
	}
	raw := p.correctionCmd(commissioner.CorrectPick, "pick", "update "+pickKey(w),
		commissioner.RawCorrection{Owner: w.Owner, Pick: &pick})
	if err := p.emit(raw); err != nil {
		return err
	}
	p.summary.Picks++
	return nil
}

func contractFromPlayer(p commissioner.NativePlayer) commissioner.CorrectionContract {
	return commissioner.CorrectionContract{
		ContractMissing:    p.ContractMissing,
		Salary:             p.Salary,
		ExpirationYear:     p.ExpirationYear,
		ContractStatus:     p.ContractStatus,
		ContractInfo:       p.ContractInfo,
		HeldContractStatus: p.HeldContractStatus,
		ContractSchedule:   slices.Clone(p.ContractSchedule),
	}
}

func sameContract(a, b commissioner.NativePlayer) bool {
	return a.ContractMissing == b.ContractMissing &&
		a.Salary == b.Salary &&
		a.ExpirationYear == b.ExpirationYear &&
		a.ContractStatus == b.ContractStatus &&
		a.ContractInfo == b.ContractInfo &&
		a.HeldContractStatus == b.HeldContractStatus &&
		slices.Equal(a.ContractSchedule, b.ContractSchedule)
}

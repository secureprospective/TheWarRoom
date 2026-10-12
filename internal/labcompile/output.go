package labcompile

func (l *Lab) outputTrades(doc *document, trades []Trade, draft []Pick, known, estimated map[teamYear]int64, a analysisIndex) (map[string]bool, map[int]int64, error) {
	seen := map[string]bool{}
	tradeIndex := map[int]int64{}
	bySlot := map[pickKey]Pick{}
	for _, d := range draft {
		bySlot[pickKey{d.Year, d.Round, d.Slot}] = d
	}
	for tid, i := range tradeOrder(trades) {
		tradeIndex[i] = int64(tid)
		t := trades[i]
		if err := appendRow(doc.Trades, l.tradeRow(t, a.Deals[i])); err != nil {
			return nil, nil, err
		}
		for gi, gave := range [][]Asset{t.AGave, t.BGave} {
			for ai, asset := range gave {
				row := l.legRow(t, int64(tid), gi, asset, a.Legs[legKey{i, gi, ai}], known, estimated, bySlot)
				if value(row.Pid) != "" {
					seen[*row.Pid] = true
				}
				if value(row.PickPid) != "" {
					seen[*row.PickPid] = true
				}
				if err := appendRow(doc.Legs, row); err != nil {
					return nil, nil, err
				}
			}
		}
	}
	return seen, tradeIndex, nil
}
func (l *Lab) outputDraft(doc *document, draft []Pick, a analysisIndex, seen map[string]bool) error {
	order := map[int64]int64{}
	for _, d := range draft {
		order[d.Year]++
		seen[d.Pid] = true
		if err := appendRow(doc.Draft, l.draftRow(d, order[d.Year], a)); err != nil {
			return err
		}
	}
	return nil
}
func (l *Lab) outputDirectories(doc *document, a analysisIndex, seen map[string]bool) error {
	for y := int64(firstYear); y <= lastYear; y++ {
		for _, fid := range sortedKeys(l.Names) {
			if _, ok := l.Names[fid][y]; !ok {
				continue
			}
			if err := appendRow(doc.TeamSeasons, l.teamRow(y, fid, a.AllPlay[teamYear{y, fid}])); err != nil {
				return err
			}
		}
	}
	born := 0
	pids := make([]string, 0, len(seen))
	for pid := range seen {
		pids = append(pids, pid)
	}
	sortStrings(pids)
	for _, pid := range pids {
		if l.Birth[pid] != 0 {
			born++
		}
		if err := appendRow(doc.Players, l.playerRow(pid)); err != nil {
			return err
		}
	}
	doc.Checks.BirthdateCoverage = pythonRound(float64(born)/float64(max(1, len(pids))), 3)
	return nil
}

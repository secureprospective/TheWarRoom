package facts

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

func (c *compiler) noteDependencies(ctx context.Context) (noteDependencies, error) {
	dep := noteDependencies{users: map[pickKey]string{}, moves: map[pickKey][]transfer{}, trades: map[int64]int64{}, transactions: map[int64]bool{}}
	slots, err := c.draftUsers(&dep)
	if err != nil {
		return dep, err
	}
	var tid int64
	for _, year := range c.years {
		if err := ctx.Err(); err != nil {
			return dep, fmt.Errorf("compile note dependencies: %w", err)
		}
		src := seasonFile(year, "transactions")
		d, err := c.read(src)
		if err != nil {
			return dep, err
		}
		for _, x := range d.get("transactions", "transaction").rows() {
			tid++
			dep.transactions[tid] = true
			if value(x.get("type").str()) != "TRADE" {
				continue
			}
			ts := x.get("timestamp").whole()
			if ts == nil {
				return dep, fmt.Errorf("%s: trade %d lacks timestamp", src, tid)
			}
			dep.trades[tid] = *ts
			addTradeMoves(&dep, slots, x, year, *ts)
		}
	}
	return dep, nil
}

func (c *compiler) draftUsers(dep *noteDependencies) (map[pickSlot]string, error) {
	names := map[string]map[string]bool{}
	for _, f := range c.facts.FranchiseSeason {
		name := value(f.Name)
		if names[name] == nil {
			names[name] = map[string]bool{}
		}
		names[name][f.Fid] = true
	}
	slots := map[pickSlot]string{}
	chainPattern := regexp.MustCompile(`Pick traded from ([^.\n]+)\.`)
	for _, year := range c.years {
		src := seasonFile(year, "draftResults")
		d, err := c.read(src)
		if err != nil {
			return nil, err
		}
		for _, unit := range d.get("draftResults", "draftUnit").rows() {
			for _, p := range unit.get("draftPick").rows() {
				used := value(p.get("franchise").str())
				orig := used
				chain := chainPattern.FindAllStringSubmatch(value(p.get("comments").str()), -1)
				if len(chain) > 0 {
					orig = uniqueName(names[strings.TrimSpace(chain[0][1])])
				}
				round, slot := p.get("round").whole(), p.get("pick").whole()
				if round == nil || slot == nil {
					return nil, fmt.Errorf("%s: draft pick lacks round or slot", src)
				}
				slots[pickSlot{year, *round, *slot}] = orig
				if orig != "" {
					dep.users[pickKey{year, *round, orig}] = used
				}
			}
		}
	}
	return slots, nil
}

func uniqueName(ids map[string]bool) string {
	if len(ids) != 1 {
		return ""
	}
	for id := range ids {
		return id
	}
	return ""
}

func addTradeMoves(dep *noteDependencies, slots map[pickSlot]string, x RawValue, year, ts int64) {
	f1, f2 := value(x.get("franchise").str()), value(x.get("franchise2").str())
	for _, side := range []struct{ field, from, to string }{{"franchise1_gave_up", f1, f2}, {"franchise2_gave_up", f2, f1}} {
		for _, code := range strings.Split(value(x.get(side.field).str()), ",") {
			key, slot, ok := pickCode(strings.TrimSpace(code), year)
			if !ok {
				continue
			}
			if key.orig == "" {
				key.orig = slots[pickSlot{key.year, key.round, slot}]
			}
			if key.orig != "" {
				dep.moves[key] = append(dep.moves[key], transfer{ts, side.from, side.to, -1})
			}
		}
	}
}

func pickCode(code string, year int64) (pickKey, int64, bool) {
	parts := strings.Split(code, "_")
	if len(parts) == 4 && parts[0] == "FP" && len(parts[1]) == 4 && len(parts[2]) == 4 {
		y, errY := strconv.ParseInt(parts[2], 10, 64)
		r, errR := strconv.ParseInt(parts[3], 10, 64)
		_, errO := strconv.ParseInt(parts[1], 10, 64)
		return pickKey{y, r, parts[1]}, 0, errY == nil && errR == nil && errO == nil
	}
	if len(parts) == 3 && parts[0] == "DP" {
		r, errR := strconv.ParseInt(parts[1], 10, 64)
		s, errS := strconv.ParseInt(parts[2], 10, 64)
		return pickKey{year, r + 1, ""}, s + 1, errR == nil && errS == nil
	}
	return pickKey{}, 0, false
}

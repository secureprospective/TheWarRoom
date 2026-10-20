package commissioner

import "time"

// A reversal toggles its original change even when it reverses another reversal.
// Draft restores and transaction corrections cannot target rollover rules.
type rolloverSetting struct {
	entry string
	order int
	date  time.Time
}

type rolloverIndex struct {
	roots    map[string]string
	active   map[string]bool
	settings map[string]rolloverSetting
	heap     []rolloverSetting
	next     int
}

func newRolloverIndex() *rolloverIndex {
	return &rolloverIndex{roots: map[string]string{}, active: map[string]bool{}, settings: map[string]rolloverSetting{}}
}

func (f *rolloverIndex) add(r Record) (time.Time, error) {
	id := r.Receipt.Entry
	f.next++
	if r.Command.Command == Revert {
		f.roots[id] = f.roots[r.Command.Args.Entry]
		root := f.roots[id]
		if root != "" {
			f.active[root] = !f.active[root]
			if f.active[root] {
				f.push(f.settings[root])
			}
		}
	} else if r.Command.Command == RulesSet && r.Command.Args.Field == rolloverField && r.Command.Effective == "now" {
		date, err := rolloverDate(r)
		if err != nil {
			return time.Time{}, err
		}
		f.roots[id], f.active[id] = id, true
		setting := rolloverSetting{entry: id, order: f.next, date: date}
		f.settings[id] = setting
		f.push(setting)
	}
	for len(f.heap) > 0 && !f.active[f.heap[0].entry] {
		f.pop()
	}
	if len(f.heap) == 0 {
		return time.Time{}, nil
	}
	return f.heap[0].date, nil
}

func (f *rolloverIndex) push(r rolloverSetting) {
	f.heap = append(f.heap, r)
	for i := len(f.heap) - 1; i > 0; {
		parent := (i - 1) / 2
		if f.heap[parent].order >= f.heap[i].order {
			break
		}
		f.heap[parent], f.heap[i] = f.heap[i], f.heap[parent]
		i = parent
	}
}

func (f *rolloverIndex) pop() {
	last := len(f.heap) - 1
	f.heap[0] = f.heap[last]
	f.heap = f.heap[:last]
	for i := 0; 2*i+1 < len(f.heap); {
		child := 2*i + 1
		if child+1 < len(f.heap) && f.heap[child+1].order > f.heap[child].order {
			child++
		}
		if f.heap[i].order >= f.heap[child].order {
			break
		}
		f.heap[i], f.heap[child] = f.heap[child], f.heap[i]
		i = child
	}
}

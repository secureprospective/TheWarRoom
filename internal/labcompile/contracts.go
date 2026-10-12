package labcompile

func (l *Lab) contractCoverage(y int64) float64 {
	total, salaried := 0, 0
	for key, r := range l.Rosters {
		if key.Year != y || key.Week != 5 {
			continue
		}
		for _, c := range r.Players {
			total++
			if value(c.Salary) != 0 {
				salaried++
			}
		}
	}
	if total == 0 {
		return 0
	}
	return pythonRound(float64(salaried)/float64(total), 3)
}
func (l *Lab) capRoom(y, ts int64, fid string) *float64 {
	w := l.weekAt(y, ts)
	capLimit := l.Cap[y]
	if w < 1 || capLimit == nil || l.Coverage[y] < 0.85 {
		return nil
	}
	r := l.Rosters[teamWeek{y, w, fid}]
	if len(r.Players) == 0 {
		return nil
	}
	salaries := []float64{}
	for _, pid := range r.Order {
		salaries = append(salaries, value(r.Players[pid].Salary))
	}
	dead := []float64{}
	for _, c := range l.Dead[teamYear{y, fid}] {
		if c.Ts <= ts {
			dead = append(dead, c.Amount)
		}
	}
	return ptr(pythonRound(*capLimit-(pythonSum(salaries)+pythonSum(dead)), 2))
}
func (l *Lab) contract(pid string, y, ts int64, fid string) *Contract {
	for w := l.weekAt(y, ts); w > 0; w-- {
		if c, ok := l.Rosters[teamWeek{y, w, fid}].Players[pid]; ok {
			return &c
		}
	}
	for w := int64(17); w > 0; w-- {
		if c, ok := l.Rosters[teamWeek{y - 1, w, fid}].Players[pid]; ok {
			return &c
		}
	}
	return nil
}
func (l *Lab) contractAfter(pid string, y, ts int64, fid string) *Contract {
	for w := l.weekAt(y, ts) + 1; w < 18; w++ {
		if c, ok := l.Rosters[teamWeek{y, w, fid}].Players[pid]; ok {
			return &c
		}
	}
	for w := int64(1); w < 4; w++ {
		if c, ok := l.Rosters[teamWeek{y + 1, w, fid}].Players[pid]; ok {
			return &c
		}
	}
	return nil
}

package db

import (
	"context"
	"time"
)

// RoutineCheck is the last attempted check during this process, including execution failures.
type RoutineCheck struct {
	At       time.Time
	Findings CheckFindings
	Error    string
}

// CheckDue checks on first use and weekly thereafter. Restarting checks again rather than
// trusting an earlier process's result. Readers retain the prior result while a check runs.
func (p *Pools) CheckDue(ctx context.Context, now time.Time) {
	p.checkRunMu.Lock()
	defer p.checkRunMu.Unlock()
	last := p.LastCheck()
	if !last.At.IsZero() && now.Before(last.At.Add(7*24*time.Hour)) {
		return
	}
	findings, err := Check(ctx, p.read)
	result := RoutineCheck{At: now, Findings: findings}
	if err != nil {
		result.Error = err.Error()
	}
	p.checkMu.Lock()
	p.lastCheck = result
	p.checkMu.Unlock()
}

// LastCheck returns owned findings without starting database work.
func (p *Pools) LastCheck() RoutineCheck {
	p.checkMu.Lock()
	defer p.checkMu.Unlock()
	result := p.lastCheck
	result.Findings = cloneCheckFindings(result.Findings)
	return result
}

func cloneCheckFindings(result CheckFindings) CheckFindings {
	result.Integrity = append([]string(nil), result.Integrity...)
	result.ForeignKeys = append([]ForeignKeyFinding(nil), result.ForeignKeys...)
	return result
}

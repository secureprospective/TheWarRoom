import { describe, expect, it } from 'vitest';
import { pendingChanges, sectionCounts } from './CtrlRulesV3';
import type { RuleRow, Rules } from '../commissioner/rules';

const row = (label: string, state: RuleRow['state'], pending: RuleRow['pending'] = []): RuleRow =>
  ({ field: label, label, value: '1', state, citation: '', changedAt: '', competitive: false,
    nowOnly: false, pending, current: 1, unit: '', governed: null,
    schema: { kind: 'count', choices: [], fields: [] },
    safeguards: { nextRollover: 'confirm', now: 'confirm' } });
const read = (...sections: Rules['sections']): Rules =>
  ({ league: 'L', name: 'L', revision: 1, rollover: 'Mon', sections });

describe('sectionCounts', () => {
  it('counts rules and rulings needed, skipping empty sections', () => {
    const rules = read({ label: 'Roster', rows: [row('a', 'Set'), row('b', 'Needs a ruling')] },
      { label: 'Empty', rows: [] });
    expect(sectionCounts(rules)).toEqual([{ label: 'Roster', rules: 2, needsRuling: 1 }]);
  });
  it('is empty without a read', () => expect(sectionCounts(undefined)).toEqual([]));
});

describe('pendingChanges', () => {
  it('lists waiting changes across sections', () => {
    const p = { entry: 'e1', value: '5', when: '2026-12-01T00:00:00Z' };
    const rules = read({ label: 'A', rows: [row('x', 'Set', [p])] }, { label: 'B', rows: [row('y', 'Set')] });
    expect(pendingChanges(rules)).toEqual([{ label: 'x', value: '5', when: p.when, entry: 'e1' }]);
  });
  it('is empty without a read', () => expect(pendingChanges(undefined)).toEqual([]));
});

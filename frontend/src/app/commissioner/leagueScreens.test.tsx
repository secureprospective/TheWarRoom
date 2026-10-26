import { afterEach, beforeEach, describe, expect, it } from 'vitest';
import { all, mount, present, render, settle, unmount, type Element } from './commissionerDom';
import { RuleChangeForm } from './RuleChangeForm';
import { parseRules } from './rules';
import { LedgerEntries } from './LeagueLedger';
import { parseLedger } from './proxy';
import ledgerWire from './wire.testdata.json';
import { stub } from './commissionerStub';
import LeaguePeople from './LeaguePeople';
import LeagueRules from './LeagueRules';
import { leagueState } from './proxy';
import peopleWire from './people.wire.testdata.json';
import legacy from './rules.legacy.wire.testdata.json';
import rulesWire from './rules.wire.testdata.json';
import replacedWire from './replaced.wire.testdata.json';

beforeEach(() => leagueState.forget());
afterEach(unmount);

const show = async (node: ReturnType<typeof render>, readings: object) => {
  const { app } = stub({}, readings);
  const host = await mount(node, app);
  await leagueState.choose('native.rules');
  await settle();
  return host;
};
const text = (host: Element) => host.textContent;
const used = (changes: Record<string, boolean>, current: unknown) => Object.fromEntries(
  Object.entries(current as Record<string, object>).map(([k, v]) => [k, { ...v, Used: changes[k] ?? true }]));
const withStats = (changes: Record<string, boolean>) => ({ ...legacy, sections: legacy.sections.map(s => ({
  ...s, rows: s.rows.map(r => r.field !== 'scoring.catalogue' ? r
    : { ...r, current: used(changes, r.current) }) })) });

describe('League rules screen', () => {
  it('lists each section with its settings in words, their state, source and what is waiting', async () => {
    const host = await show(render(LeagueRules), { rules: rulesWire });
    const view = text(host);
    for (const part of ['League shape', 'Roster', 'Active roster size', '20 to 70',
      'Needs a ruling',
      `Changed in the league ledger on ${new Date('2026-10-14T04:17:03Z').toLocaleString()}`])
      expect(view).toContain(part);
    expect(view).toContain('$125,000,000: Waits for the next rollover');
    expect(view).toContain('Next rollover: The next rollover has no date yet; the commissioner sets it.');
    expect(view).not.toMatch(/roster\.limits|CM-\d|G[123]\b/);
  });
  it('keeps the full Needs-a-ruling sentence in the source cell without requiring a colon', async () => {
    const sentence = 'Needs a ruling; the commissioner sets the exact trade deadline.';
    const reading = { ...rulesWire, sections: rulesWire.sections.map(s => ({ ...s,
      rows: s.rows.map(r => r.field === 'calendar.tradeDeadline' ? { ...r, citation: sentence } : r),
    })) };
    const host = await show(render(LeagueRules), { rules: reading });
    const row = all(host, 'tr').find(tr => tr.textContent.startsWith('Trade deadline'));
    const cell = row && all(row, 'td')[2];
    expect(cell?.getAttribute('data-label')).toBe('Where it comes from');
    expect(cell?.textContent).toBe(sentence);
  });
  it('shows a waiting change beside the value still in force, never as in force', async () => {
    const waiting = { ...rulesWire, sections: rulesWire.sections.map(s => ({ ...s, rows: s.rows.map(r =>
      r.field === 'roster.limits' ? { ...r, pending: [{ entry: 'act.x', value: '25 to 60',
        when: 'Waits for the next rollover' }] } : r) })) };
    const host = await show(render(LeagueRules), { rules: waiting });
    const row = all(host, 'tr').find(tr => tr.textContent.startsWith('Active roster size'));
    expect(row?.textContent).toContain('20 to 70');
    expect(row?.textContent).toContain('25 to 60: Waits for the next rollover');
    const cells = row?.children.map(c => c.textContent) ?? [];
    expect(cells[1]).toContain('20 to 70');
    expect(cells[1]).not.toContain('25 to 60');
  });
  it('draws scoring as grouped tables from its schema, with unused stats folded away', async () => {
    const host = await show(render(LeagueRules), { rules: withStats({ assist: false, assist_line: false }) });
    const used = all(host, 'section').find(s => s.getAttribute('aria-label') === 'Scoring: stats in use');
    expect(used).toBeDefined();
    expect(text(used as Element)).toContain('Passing');
    expect(text(used as Element)).not.toContain('Assisted tackle');
    const folded = all(host, 'details').find(d => d.textContent.startsWith('Stats not used (2)'));
    expect(folded?.textContent).toContain('Assisted tackle');
    expect(folded?.textContent).toContain('Defensive line assisted tackle');
    expect(all(host, 'tr').some(tr => tr.textContent.startsWith('Scoring values'))).toBe(false);
    expect(text(host)).not.toContain('Passing yards: 0.04; Passing');
  });
});

describe('League people screen', () => {
  it('shows seats, roles, the trade review count with its ruling state, votes and succession', async () => {
    const host = await show(render(LeaguePeople), { people: peopleWire.commissioner });
    const view = text(host);
    for (const part of ['Franchise 1 (F1)', 'owner-one', 'Vacant', 'Co-commissioner', 'private-deputy',
      'Needs a ruling: DOT has 0 of 4 seats', 'Trade review seats (DOT): 0',
      'Commissioner removal voting is open', '1 of 4 franchises have voted', 'Voted: Franchise 2 (F2)',
      'The commissioner is serving; no succession is pending', 'You are: Commissioner'])
      expect(view).toContain(part);
  });
  it('gives an owner no role or invitation rows and no list of who voted', async () => {
    const host = await show(render(LeaguePeople), { people: peopleWire.owner });
    const view = text(host);
    expect(view).toContain('You are: Owner');
    expect(view).toContain('1 of 4 franchises have voted');
    for (const hidden of ['private-deputy', 'Roles', 'Owner invitations', 'Voted:'])
      expect(view).not.toContain(hidden);
    expect(present(host, 'Franchise 3 (F3)')).toBe(true);
  });
});

describe('Plain rule provenance', () => {
  it('never shows census or fact labels in the Legacy rules screen or any rule form', async () => {
    const reading = parseRules(legacy);
    const host = await show(render('div', null, render(LeagueRules),
      ...reading.sections.flatMap(s => s.rows).map(r => render(RuleChangeForm,
        { key: r.field, rules: reading, initialField: r.field }))), { rules: legacy });
    expect(host.textContent).not.toMatch(/Census|CM-0|fact R/);
    expect(host.textContent).not.toContain('(Set)');
    expect(all(host, 'th').map(t => t.textContent)).not.toContain('State');
  });
  it('shows the scoring sources and counting settings in plain words', async () => {
    const host = await show(render(LeagueRules), { rules: legacy });
    const source = 'Legacy NFL rules (the league’s MFL scoring)';
    // Seven scoring settings carry the source: values, rounding and the five rule kinds.
    expect(host.textContent.split(source)).toHaveLength(8);
    expect(host.textContent).not.toMatch(/Rulebook §|Math §/);
    expect(host.textContent).toContain('Once per player-week');
    expect(host.textContent).toContain('Round each stat’s points to the cent');
  });
  it('agrees with the ledger for a change made before midnight Central but after midnight UTC', async () => {
    const when = '2026-10-14T04:17:03Z';
    const changed = { ...rulesWire, sections: rulesWire.sections.map(s => ({ ...s,
      rows: s.rows.map(r => r.field === 'roster.limits' ? { ...r, changedAt: when } : r) })) };
    const ledger = parseLedger({ ...ledgerWire,
      entries: ledgerWire.entries.map(e => ({ ...e, when })) });
    const host = await show(render('div', null, render(LeagueRules),
      render(RuleChangeForm, { rules: parseRules(changed), initialField: 'roster.limits' }),
      render(LedgerEntries, { reading: ledger })), { rules: changed });
    const local = new Date(when).toLocaleString();
    expect(host.textContent.split(`Changed in the league ledger on ${local}`)).toHaveLength(3);
    expect(all(host, 'time')[0].textContent).toBe(local);
  });
});

describe('Replaced waiting changes', () => {
  it('shows Go’s full smaller-league ruling sentences in each source cell', async () => {
    const host = await show(render(LeagueRules), { rules: replacedWire.rules });
    const sources = [
      ['Schedule pattern', 'Choose how often teams meet in this smaller league.'],
      ['First round bye seed', 'Choose first-round byes for this smaller playoff field.'],
      ['Final playoff week', 'Choose the final week after setting the smaller bracket.'],
      ['Playoff bracket', 'Choose a bracket for this smaller playoff field.'],
    ];
    for (const [label, sentence] of sources) {
      const row = all(host, 'tr').find(tr => tr.textContent.startsWith(label));
      const cell = row && all(row, 'td')[2];
      expect(cell?.textContent).toBe(`Needs a ruling: ${sentence}`);
    }
  });
  it('shows Go’s replacement date in viewer-local time in the rules and ledger', async () => {
    const replacedAt = '2026-10-14T04:19:03Z';
    const changed = parseRules(replacedWire.rules);
    const ledger = parseLedger(replacedWire.ledger);
    const host = await show(render('div', null, render(LeagueRules),
      render(LedgerEntries, { reading: ledger })), { rules: changed });
    const words = `Replaced by a later change on ${new Date(replacedAt).toLocaleString()}`;
    expect(host.textContent.split(words)).toHaveLength(5);
    const row = all(host, 'tr').find(tr => tr.textContent.startsWith('Active roster size'));
    expect(row?.textContent).toContain('24 to 78: Waits for the next rollover');
    expect(row?.textContent).not.toContain('20 to 70: Waits for the next rollover');
    expect(row?.textContent).not.toContain('22 to 75: Waits for the next rollover');
  });
});

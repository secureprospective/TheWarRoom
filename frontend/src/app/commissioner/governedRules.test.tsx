import { afterEach, beforeEach, describe, expect, it } from 'vitest';
import { all, button, click, mount, render, settle, unmount, type Element } from './commissionerDom';
import { ControlRow } from './ControlRow';
import { entriesOf, parseControls } from './controls';
import LeagueRules from './LeagueRules';
import { leagueState } from './proxy';
import { parseRules } from './rules';
import { stub } from './commissionerStub';
import controlsWire from './controls.wire.testdata.json';
import rulesWire from './rules.wire.testdata.json';

beforeEach(() => leagueState.forget());
afterEach(unmount);

const glyphs = (host: Element) => all(host, 'button')
  .filter(b => b.getAttribute('aria-label')?.startsWith('Who set '));
const rules = parseRules(rulesWire);

describe('The governed glyph on commissioner screens', () => {
  it('marks exactly the rules the commissioner changed in the ledger, and no preset value', async () => {
    const { app } = stub();
    const host = await mount(render(LeagueRules), app);
    await leagueState.choose('native.rules');
    await settle();
    const changed = rules.sections.flatMap(s => s.rows).filter(r => r.governed);
    expect(changed.length).toBeGreaterThan(0);
    expect(glyphs(host).map(g => g.getAttribute('aria-label')))
      .toEqual(changed.map(r => `Who set ${r.label}, when, and with which change`));
    const row = all(host, 'tr').find(tr => tr.textContent.startsWith('Active roster size'))!;
    expect(glyphs(row)).toHaveLength(1);
    await click(glyphs(row)[0]);
    expect(row.textContent).toContain('Set bycreator');
    expect(row.textContent).toContain(changed[0].governed!.change);
    const preset = all(host, 'tr').find(tr => tr.textContent.startsWith('Teams'))!;
    expect(glyphs(preset)).toHaveLength(0);
  });
  it('marks a changed value beside its control too, with the same glyph', async () => {
    const entry = entriesOf(parseControls(controlsWire)).find(e => e.id === 'CM-003')!;
    const host = await mount(render(ControlRow, { entry, rules: new Map(rules.sections.flatMap(s => s.rows)
      .map((r): [string, typeof r] => [r.field, r])), mayChange: true, refer() {} }));
    expect(glyphs(host).map(g => g.getAttribute('aria-label')))
      .toEqual(['Who set Active roster size, when, and with which change']);
    expect(button(host, 'Who set Active roster size, when, and with which change')).toBeDefined();
  });
  it('refuses a rules reading that leaves out who set a value', () => {
    const row = { ...rulesWire.sections[0].rows[0] } as Record<string, unknown>;
    delete row.governed;
    expect(() => parseRules({ ...rulesWire, sections: [{ ...rulesWire.sections[0], rows: [row] }] }))
      .toThrow();
  });
});

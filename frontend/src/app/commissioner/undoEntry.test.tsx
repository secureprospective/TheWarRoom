import { afterEach, beforeEach, describe, expect, it } from 'vitest';
import { readFileSync } from 'node:fs';
import { act } from 'react';
import { all, button, control, disabled, mount, present, render, settle, type, unmount }
  from './commissionerDom';
import { stub } from './commissionerStub';
import { LedgerEntries } from './LeagueLedger';
import { leagueState, parseLedger } from './proxy';
import wire from './wire.testdata.json';

beforeEach(() => leagueState.forget());
afterEach(unmount);

const as = (undo: string) => parseLedger({ ...wire, entries: wire.entries.map(e => ({ ...e, undo })) });
const undo = 'Undo this change';

describe('Undo this change', () => {
  it('is not offered to an owner, whatever the entry', async () => {
    const host = await mount(render(LedgerEntries, { reading: as('') }), stub().app);
    expect(all(host, 'summary')).toHaveLength(0);
    expect(present(host, undo)).toBe(false);
  });
  it('sends a revert of that entry with the typed reason, once Go has offered it', async () => {
    const { app, calls } = stub();
    const host = await mount(render(LedgerEntries, { reading: as('league') }), app);
    expect(all(host, 'summary').map(s => s.textContent)).toEqual([undo]);
    expect(disabled(button(host, undo))).toBe(true);
    await type(control(host, 'Reason'), 'We entered the wrong size');
    expect(disabled(button(host, undo))).toBe(false);
    await act(async () => { (button(host, undo).props().onClick as () => void)(); });
    await settle();
    expect(calls.sent).toHaveLength(1);
    expect(calls.sent[0]).toMatchObject({ command: 'commish.correction.revert', league: wire.league,
      args: { entry: wire.entries[0].id }, reason: 'We entered the wrong size', expected: 2,
      effective: 'now', visibility: 'league' });
  });
  it('corrects a private entry privately', async () => {
    const { app, calls } = stub();
    const host = await mount(render(LedgerEntries, { reading: as('private') }), app);
    expect(present(host, 'its correction is private too')).toBe(true);
    await type(control(host, 'Reason'), 'Wrong person');
    await act(async () => { (button(host, undo).props().onClick as () => void)(); });
    await settle();
    expect(calls.sent[0].visibility).toBe('private');
  });
});

it('lays out the undo reason above a full-width textarea, independently of the league form', async () => {
  const host = await mount(render(LedgerEntries, { reading: as('league') }), stub().app);
  const reason = control(host, 'Reason');
  expect(reason.parentNode?.getAttribute('class')).toBe('change-reason');
  const css = readFileSync(new URL('./commissioner.css', import.meta.url), 'utf8');
  const label = css.slice(css.indexOf('.twr-app .change-reason {')).split('}')[0];
  const textarea = css.slice(css.indexOf('.twr-app .change-reason textarea {')).split('}')[0];
  expect(label).toContain('display: grid');
  expect(label).toContain('width: 100%');
  expect(textarea).toContain('width: 100%');
});

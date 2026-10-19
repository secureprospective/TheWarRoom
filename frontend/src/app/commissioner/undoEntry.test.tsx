import { afterEach, beforeEach, describe, expect, it } from 'vitest';
import { readFileSync } from 'node:fs';
import { act } from 'react';
import { all, button, control, disabled, mount, present, render, settle, type, unmount }
  from './commissionerDom';
import { stub } from './commissionerStub';
import { LedgerEntries } from './LeagueLedger';
import { leagueState, parseLedger } from './proxy';
import { UndoForm } from './UndoForm';
import wire from './wire.testdata.json';

beforeEach(() => leagueState.forget());
afterEach(unmount);

const as = (undo: string) => parseLedger({ ...wire, entries: wire.entries.map(e => ({ ...e, undo })) });
const undo = 'Undo this change';
const form = (undoAt: string) => render(UndoForm, { reading: as(undoAt), entry: as(undoAt).entries[0] });
const offered = (host: Parameters<typeof all>[0]) =>
  all(host, 'button').filter(b => b.textContent === undo);

describe('Undo this change', () => {
  it('is not offered to an owner, whatever the entry', async () => {
    const host = await mount(render(LedgerEntries, { reading: as('') }), stub().app);
    expect(all(host, 'summary')).toHaveLength(0);
    expect(offered(host)).toHaveLength(0);
  });
  it('is only a way into the tray beside each entry Go offered it for', async () => {
    const host = await mount(render(LedgerEntries, { reading: as('league') }), stub().app);
    const launchers = offered(host);
    expect(launchers).toHaveLength(wire.entries.length);
    expect(launchers[0].getAttribute('aria-label')).toBe(`Undo this change: ${wire.entries[0].what}`);
    expect(all(host, 'textarea')).toHaveLength(0);
    expect(all(host, 'section').filter(s => s.getAttribute('aria-label') === undo)).toHaveLength(0);
  });
  it('sends a revert of that entry with the typed reason, once Go has offered it', async () => {
    const { app, calls } = stub();
    const host = await mount(form('league'), app);
    await settle();
    expect(disabled(button(host, undo))).toBe(true);
    await type(control(host, 'Reason'), 'We entered the wrong size');
    await settle();
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
    const host = await mount(form('private'), app);
    expect(present(host, 'its correction is private too')).toBe(true);
    await type(control(host, 'Reason'), 'Wrong person');
    await settle();
    await act(async () => { (button(host, undo).props().onClick as () => void)(); });
    await settle();
    expect(calls.sent[0].visibility).toBe('private');
  });
  it('says so, rather than offering a form, when Go no longer offers the correction', async () => {
    const host = await mount(form(''), stub().app);
    expect(present(host, 'can no longer be undone from here')).toBe(true);
    expect(all(host, 'textarea')).toHaveLength(0);
  });
});

it('lays out the undo reason above a full-width textarea, independently of the league form', async () => {
  const host = await mount(form('league'), stub().app);
  const reason = control(host, 'Reason');
  expect(reason.parentNode?.getAttribute('class')).toBe('change-reason');
  const css = readFileSync(new URL('./commissioner.css', import.meta.url), 'utf8');
  const label = css.slice(css.indexOf('.twr-app .change-reason {')).split('}')[0];
  const textarea = css.slice(css.indexOf('.twr-app .change-reason textarea {')).split('}')[0];
  expect(label).toContain('display: grid');
  expect(label).toContain('width: 100%');
  expect(textarea).toContain('width: 100%');
});

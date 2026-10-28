import type { ParamsResult } from '../control/api';
import { createElement } from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import { describe, expect, it } from 'vitest';
import { parseSources, parseOutcome, type DataProvider } from './provider';
import { createDataState } from './state';
import { report } from '../reports/contract';
import { ReportView } from '../reports/ReportView';
import { endpointById, placementRoute } from '../registry';
import { commands } from '../commands/registry';
import wire from './wire.testdata.json';
const data = () => parseSources(wire.data);
const provider = (): DataProvider => ({
  read: async () => data(), crosswalk: async () => report(wire.crosswalk, 'Crosswalk'),
  params: async () => wire.params as ParamsResult,
  refresh: async () => parseOutcome(wire.result), folder: async () => parseOutcome(wire.result),
  photos: async () => parseOutcome(wire.result), parameter: async () => {},
});
const html = renderToStaticMarkup;
describe('Control Room data and sources', () => {
  it('renders Go health, the manifest standing and each check with its differences in words', () => {
    const d = data();
    for (const r of [d.health, d.facts, d.checks, d.databaseChecks]) {
      const markup = html(createElement(ReportView, { report: r, howLabel: 'How this is counted' }));
      expect(markup).toContain(r.title);
      expect(markup).toContain(`data-endpoint="${r.id}"`);
    }
    expect(d.facts.status).toBe('empty');
    expect(d.facts.note).toContain('No manifest yet');
    const checks = html(createElement(ReportView, { report: d.checks, howLabel: 'How this is counted' }));
    expect(checks).toContain('Passed: 448 compared, 446 agree, 1 explained, 1 for review');
    expect(checks).toContain('2014 Houston Texans: MFL standings 9-4-0, weekly games 9-3-0.');
    expect(checks).toContain('For review');
    expect(checks).not.toContain('2014/0009');
    expect(d.checks.rows.some(r => r.cells[2] === 'Not run yet')).toBe(false);
  });
  it('routes all five built rows to the lazy workspace', () => {
    for (const id of ['C-07', 'C-08', 'C-09', 'L-34', 'S-09'] as const) {
      const e = endpointById(id);
      expect(e.status).not.toBe('spec');
      expect(e.disposition).toBe('retired');
      expect(placementRoute(e.placement)).toBeUndefined();
    }
    expect(commands.registry['refresh.run'].args).toEqual(['scope']);
  });
  it('refuses malformed wire and unknown or duplicate scopes', () => {
    expect(() => parseSources({ ...wire.data, extra: 1 })).toThrow(/unexpected/);
    expect(() => parseSources({ ...wire.data, scopes: [] })).toThrow(/scopes/);
    expect(() => parseSources({ ...wire.data,
      scopes: Array(4).fill(wire.data.scopes[0]) })).toThrow(/repeated/);
    expect(() => parseSources({ ...wire.data, health: { ...wire.data.health,
      columns: [] } })).toThrow(/column count/);
    expect(() => parseOutcome({ ok: false, error: '', summary: '' })).toThrow(/reason/);
  });
});
describe('scoped refresh state', () => {
  it('holds the last good reading and reports failures', async () => {
    const p = provider();
    const state = createDataState(p);
    await state.run({ kind: 'read' });
    const good = state.read().data;
    p.read = async () => { throw new Error('local read failed'); };
    await state.run({ kind: 'read' });
    expect(state.read().data).toBe(good);
    expect(state.read().error).toContain('local read failed');
    expect(state.read().busy).toBe('');
    p.read = async () => data();
    p.refresh = async () => ({ ok: false, error: 'Bad archive: 2014/rules.json', summary: '' });
    await state.run({ kind: 'refresh', scope: 'archive' });
    expect(state.read().error).toContain('2014/rules.json');
  });
  it('serializes Acts and shows the result without a blind second refresh', async () => {
    const p = provider();
    let finish!: (v: ReturnType<typeof parseOutcome>) => void;
    let calls = 0;
    p.refresh = async () => { calls++; return new Promise(resolve => { finish = resolve; }); };
    const state = createDataState(p);
    const pending = state.run({ kind: 'refresh', scope: 'archive' });
    expect(state.read().busy).toBe('archive');
    await state.run({ kind: 'refresh', scope: 'league' });
    expect(calls).toBe(1);
    finish(parseOutcome(wire.result));
    await pending;
    expect(state.read().busy).toBe('');
    expect(state.read().result).toBe(wire.result.summary);
  });
});

it('places and retains progress, success and failure beside every scope button', async () => {
  for (const scope of data().scopes) {
    const p = provider();
    let finish!: (v: ReturnType<typeof parseOutcome>) => void;
    p.refresh = async () => new Promise(resolve => { finish = resolve; });
    const state = createDataState(p);
    const pending = state.run({ kind: 'refresh', scope: scope.id });
    finish({ ok: true, error: '', summary: `Finished ${scope.label}` });
    await pending;
    expect(state.read().outcomes[scope.id]).toBe(`Finished ${scope.label}`);
    await state.run({ kind: 'read' });
    expect(state.read().outcomes[scope.id]).toBe(`Finished ${scope.label}`);
    expect(state.read().outcomes.read).toBe(data().words.readResult);
    p.refresh = async () => ({ ok: false, error: `Failed ${scope.label}`, summary: '' });
    await state.run({ kind: 'refresh', scope: scope.id });
    expect(state.read().outcomes[scope.id]).toBe(`Failed ${scope.label}`);
    await state.run({ kind: 'read' });
    expect(state.read().outcomes[scope.id]).toBe(`Failed ${scope.label}`);
    p.refresh = async () => { throw new Error('transport unavailable'); };
    await state.run({ kind: 'refresh', scope: scope.id });
    expect(state.read().outcomes[scope.id]).toContain('transport unavailable');
  }
});

it('keeps held-read progress and its Go result beside the button, including failures', async () => {
  const p = provider();
  const state = createDataState(p);
  let finish!: (v: ReturnType<typeof data>) => void;
  p.read = async () => new Promise(resolve => { finish = resolve; });
  const pending = state.run({ kind: 'read' });
  expect(state.read().busy).toBe('read');
  finish(data());
  await pending;
  p.read = async () => { throw new Error('held reader unavailable'); };
  await state.run({ kind: 'read' });
  expect(state.read().outcomes.read).toContain('held reader unavailable');
});

it('turns photos off when they are on, and keeps Go\'s result line', async () => {
  const asked: boolean[] = [];
  const p = { ...provider(), photos: async (on: boolean) => {
    asked.push(on); return { ok: true, error: '', summary: 'Player photos are off.' }; } };
  const state = createDataState(p);
  await state.run({ kind: 'read' });
  expect(state.read().data!.photosOn).toBe(true);
  await state.run({ kind: 'photos', on: false });
  expect(asked).toEqual([false]);
  expect(state.read().outcomes.photos).toBe('Player photos are off.');

});

it('publishes sources before parallel details, retaining both old details if either fails', async () => {
  const p = provider();
  const state = createDataState(p);
  await state.run({ kind: 'read' });
  const old = state.read();
  let finish!: (v: ReturnType<typeof data>) => void;
  let fail!: (cause: Error) => void;
  const calls: string[] = [];
  p.read = async () => { calls.push('sources'); return new Promise(resolve => { finish = resolve; }); };
  p.crosswalk = async () => {
    calls.push('crosswalk');
    return new Promise((_, reject) => { fail = reject; });
  };
  p.params = async () => { calls.push('params'); return wire.params as ParamsResult; };
  const pending = state.run({ kind: 'read' });
  expect(calls).toEqual(['sources']);
  const next = { ...data(), archiveDir: 'new held folder' };
  finish(next);
  await new Promise(resolve => setTimeout(resolve, 0));
  expect(calls).toEqual(['sources', 'crosswalk', 'params']);
  expect(state.read().data).toBe(next);
  expect(state.read().crosswalk).toBe(old.crosswalk);
  expect(state.read().params).toBe(old.params);
  fail(new Error('crosswalk unavailable'));
  await pending;
  expect(state.read().data).toBe(next);
  expect(state.read().crosswalk).toBe(old.crosswalk);
  expect(state.read().params).toBe(old.params);
  expect(state.read().outcomes.read).toBe('Error: crosswalk unavailable');
  expect(state.read().busy).toBe('');
});

it('does not start detail reads after sources fails, and retries without replaying an action', async () => {
  const p = provider();
  let details = 0;
  let actions = 0;
  p.crosswalk = async () => { details++; return report(wire.crosswalk, 'Crosswalk'); };
  p.read = async () => { throw new Error('sources unavailable'); };
  p.photos = async () => { actions++; return parseOutcome(wire.result); };
  const state = createDataState(p);
  await state.run({ kind: 'photos', on: false });
  expect(details).toBe(0);
  expect(state.read().error).toBe('Error: sources unavailable');
  p.read = async () => data();
  await state.run({ kind: 'read' });
  expect(details).toBe(1);
  expect(actions).toBe(1);
  expect(state.read().error).toBe('');
  expect(state.read().outcomes.photos).toBe('Error: sources unavailable');
});

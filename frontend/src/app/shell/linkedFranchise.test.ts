import { afterEach, expect, it, vi } from 'vitest';
import { createElement } from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import fixture from '../data/fixtures/snapshot.json';
import { parseSnapshot } from '../data/parse';
import { LiveProvider } from '../data/provider';
import { createCommands } from '../commands/registry';
import * as registry from '../commands/registry';
import { FRANCHISE_KEY } from './franchise';
import { OwnerFooter } from './TargetApp';
import TradeBuilder from './TradeBuilder';
import { parseBuilder, useBuilder } from './tradeBuilderState';
import * as builderState from './tradeBuilderState';
import tradeWire from './trade-wire.testdata.json';

const snapshot = parseSnapshot(fixture);
const arizona = snapshot.franchises.value.find(f => f.name === 'Arizona Cardinals')!;
const buffalo = snapshot.franchises.value.find(f => f.name === 'Buffalo Bills')!;
const connected = { state: 'connected' as const, league: '14432', season: 2026,
  myFranchise: arizona.id };
function storage() {
  const values = new Map([[FRANCHISE_KEY, buffalo.id]]);
  return { getItem: (key: string) => values.get(key) ?? null,
    setItem: (key: string, value: string) => { values.set(key, value); } };
}
afterEach(() => vi.restoreAllMocks());

it('replaces the remembered team when identity arrives and renders the owner and 31 partners', async () => {
  const saved = storage();
  const provider = new LiveProvider();
  let finish!: (status: typeof connected) => void;
  vi.spyOn(provider, 'mflKey').mockImplementation(() => new Promise(resolve => { finish = resolve; }));
  const c = createCommands(() => saved, provider);
  c.loadSnapshot(snapshot);
  expect(c.read().franchiseId).toBe(buffalo.id);
  finish(connected);
  await vi.waitFor(() => expect(c.read().franchiseId).toBe(arizona.id));
  expect(saved.getItem(FRANCHISE_KEY)).toBe(arizona.id);
  vi.spyOn(registry.commands, 'use').mockImplementation(c.read);
  vi.spyOn(registry.commands, 'useMFLKey').mockImplementation(c.readMFLKey);
  expect(renderToStaticMarkup(createElement(OwnerFooter, { snapshot }))).toContain('Arizona Cardinals');
  const reading = parseBuilder(tradeWire);
  reading.value!.teams = snapshot.franchises.value.map(f => ({ id: f.id, name: f.name ?? f.id, assets: [] }));
  useBuilder.setState({ reading, other: '', give: [], get: [], busy: false, error: '' });
  vi.spyOn(builderState, 'useBuilder').mockImplementation(useBuilder.getState);
  const html = renderToStaticMarkup(createElement(TradeBuilder, { mine: c.read().franchiseId! }));
  expect((html.match(/<option /g) ?? [])).toHaveLength(32);
  expect(html).toContain('Buffalo Bills');
  expect(html).not.toContain(`<option value="${arizona.id}">`);
  c.dispatch('franchise.set', { franchiseId: buffalo.id });
  expect(renderToStaticMarkup(createElement(OwnerFooter, { snapshot }))).toContain('Arizona Cardinals');
});

it('uses identity at snapshot startup and retains the remembered team without identity', async () => {
  const saved = storage();
  const provider = new LiveProvider();
  vi.spyOn(provider, 'mflKey').mockResolvedValue(connected);
  const c = createCommands(() => saved, provider);
  await vi.waitFor(() => expect(c.readMFLKey().myFranchise).toBe(arizona.id));
  c.loadSnapshot(snapshot);
  expect(c.read().franchiseId).toBe(arizona.id);
  expect(saved.getItem(FRANCHISE_KEY)).toBe(arizona.id);
  const other = storage();
  const fallback = createCommands(() => other);
  fallback.loadSnapshot(snapshot);
  expect(fallback.read().franchiseId).toBe(buffalo.id);
});

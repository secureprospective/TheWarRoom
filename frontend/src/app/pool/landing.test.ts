import { createElement } from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import { expect, it, vi } from 'vitest';
import { PoolFrame } from './Pool';
import { parsePool } from './provider';
import { createPoolState } from './state';
import wire from './wire.testdata.json';
import { MarketWorkspace } from '../market/Market';
import { renderState } from '../shell/state';
import { Act } from '../commands/Act';
import { commands } from '../commands/registry';
import { ReportsView } from '../draftreports/Reports';
import { parseBoard } from '../draftreports/provider';
import draftWire from '../draftreports/wire.testdata.json';
import { BuilderAsset } from '../shell/TradeBuilder';
import { parseAssets } from '../shell/tradeBuilder';

it('keeps engine, season PPG and Lab in separate sortable headed columns', () => {
  const reading = parsePool(wire.pool);
  reading.value.engineNote = 'Engine run: October 5, 2026.';
  Object.assign(reading.value.rows[0], {
    engineText: '12.34 points/game', ppgText: '9.87 PPG', labText: '0.56 wins',
  });
  const markup = renderToStaticMarkup(createElement(PoolFrame, { reading, refreshing: false }));
  for (const label of ['Player', 'Pos', 'NFL team', 'Engine points',
    'This season points per game', 'Lab dynasty value']) {
    expect(markup).toContain(`Sort by ${label}`);
  }
  for (const text of ['12.34 points/game', '9.87 PPG', '0.56 wins', 'Not valued',
    'Engine run: October 5, 2026.']) expect(markup).toContain(text);
  expect(markup.indexOf('Sort by Engine points')).toBeLessThan(markup.indexOf('Sort by Lab dynasty value'));
});

it('starts with descending engine points and sends the no-team toggle to the pool', async () => {
  const provider = {
    ...await import('./provider').then(m => m.poolProvider),
    pool: vi.fn(async () => parsePool(wire.pool)),
  };
  const state = createPoolState({ provider, franchise: () => '0001', delay: 0 });
  expect(state.peek().scope).toMatchObject({ sort: 'engine', desc: true, showNoTeam: false });
  await state.change({ field: 'pool-no-team', value: '' });
  expect(provider.pool).toHaveBeenLastCalledWith('0001', expect.objectContaining({ showNoTeam: true }));
  await state.change({ field: 'pool-sort', value: 'engine' });
  expect(state.peek().scope.desc).toBe(false);
});

it('uses endpoint.open L-15 from Market and offers Draft lab alongside draft reports', () => {
  const spy = vi.spyOn(commands, 'use').mockReturnValue({ ...renderState(commands.read()), endpointIds: [] });
  try {
    const tree = MarketWorkspace();
    const controls = tree.props.children[0];
    const door = controls.props.children[1];
    expect(door.props.verb).toBe('endpoint.open');
    expect(door.props.args).toEqual({ id: 'L-15' });
    Act(door.props).props.onClick();
    expect(commands.read().endpointIds).toEqual(['L-15']);
    const reportTree = ReportsView({ board: parseBoard(draftWire), selected: [] });
    const draftDoor = reportTree.props.children[1].props.children[0];
    expect(draftDoor.props.verb).toBe('endpoint.open');
    expect(draftDoor.props.args).toEqual({ id: 'L-15' });
    expect(draftDoor.props.label).toBe('Draft lab');
  } finally { spy.mockRestore(); }
});

it('parses and renders trade player details and shared value cells, and the Lab pick value', () => {
  const cell = { text: '12.34 PPG', sub: 'WR 1/50', detail: 'league scores', held: true };
  const assets = parseAssets([
    { token: '13593', name: 'Player', position: 'WR', team: 'DAL', contract: '$530,000 · 2 years',
      now: cell, dynasty: { ...cell, text: '0.56 wins' } },
    { token: 'FP_0001_2027_1', name: '2027 round 1', dynasty: { ...cell, text: '0.78 wins' } },
  ], 'Assets');
  const player = renderToStaticMarkup(createElement(BuilderAsset, { asset: assets[0] }));
  for (const text of ['WR', 'DAL', '$530,000 · 2 years', 'On-field-now', '12.34 PPG',
    'Dynasty value', '0.56 wins', 'asset-value']) expect(player).toContain(text);
  const pick = renderToStaticMarkup(createElement(BuilderAsset, { asset: assets[1] }));
  expect(pick).toContain('Lab pick value');
  expect(pick).toContain('0.78 wins');
});

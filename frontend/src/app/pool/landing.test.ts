import { createElement } from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import { expect, it, vi } from 'vitest';
import { PoolFrame } from './Pool';
import { parsePool } from './provider';
import { createPoolState } from './state';
import wire from './wire.testdata.json';
import { renderState } from '../shell/state';
import { Act } from '../commands/Act';
import { commands } from '../commands/registry';
import { ReportsView } from '../draftreports/Reports';
import { parseBoard } from '../draftreports/provider';
import draftWire from '../draftreports/wire.testdata.json';

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

import { afterEach, expect, it, vi } from 'vitest';
import { readFileSync } from 'node:fs';
import { builderAction, parseBuilder, useBuilder } from './tradeBuilderState';
import { commands } from '../commands/registry';

vi.mock('../commands/registry', () => ({ commands: { loadMoves: vi.fn(async () => {}) } }));

const originalWindow = Object.getOwnPropertyDescriptor(globalThis, 'window');
afterEach(() => {
  if (originalWindow) Object.defineProperty(globalThis, 'window', originalWindow);
  else Reflect.deleteProperty(globalThis, 'window');
});
const builder = JSON.parse(readFileSync('../testdata/wire/trades/TargetTradeBuilder-preview.json', 'utf8'));

it('reads through the held read command, previews and saves through their own commands', async () => {
  const API = {
    TargetTradeBuilderRead: vi.fn(async () => builder),
    TargetTradeBuilderPreview: vi.fn(async () => builder),
    TargetTradeBuilder: vi.fn(async () => builder),
  };
  vi.stubGlobal('window', { go: { trades: { API } } });
  await builderAction({ action: 'read', mine: '0001' });
  expect(API.TargetTradeBuilderRead).toHaveBeenCalledWith('0001');
  expect(useBuilder.getState()).toMatchObject({ mine: '0001', busy: false, pending: false, error: '' });
  expect(useBuilder.getState().reading).toEqual(parseBuilder(builder));

  await builderAction({ action: 'other', value: '0025' });
  await builderAction({ action: 'give', value: '11675' });
  await builderAction({ action: 'get', value: '15761' });
  await builderAction({ action: 'preview', mine: '0001' });
  const deal = { mine: '0001', other: '0025', give: ['11675'], get: ['15761'], rationale: '', id: '' };
  expect(API.TargetTradeBuilderPreview).toHaveBeenCalledWith({ ...deal, action: 'preview' });
  expect(commands.loadMoves).not.toHaveBeenCalled();

  await builderAction({ action: 'save', mine: '0001' });
  expect(API.TargetTradeBuilder).toHaveBeenCalledWith({ ...deal, action: 'save' });
  expect(API.TargetTradeBuilder).toHaveBeenCalledTimes(1);
  expect(commands.loadMoves).toHaveBeenCalledWith('0001');

  // A failed read keeps the last good view and reports the error as before.
  API.TargetTradeBuilderRead.mockRejectedValueOnce(new Error('offline'));
  await builderAction({ action: 'read', mine: '0001' });
  expect(useBuilder.getState()).toMatchObject({ busy: false, error: 'Error: offline' });
  expect(useBuilder.getState().reading).toEqual(parseBuilder(builder));
  expect(useBuilder.getState().give).toEqual(['11675']);
});

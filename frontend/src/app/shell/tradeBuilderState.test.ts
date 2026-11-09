import { afterEach, expect, it, vi } from 'vitest';
import { readFileSync } from 'node:fs';
import { builderAction, useBuilder } from './tradeBuilderState';
import { commands } from '../commands/registry';

vi.mock('../commands/registry', () => ({ commands: { loadMoves: vi.fn(async () => {}) } }));

const originalWindow = Object.getOwnPropertyDescriptor(globalThis, 'window');
afterEach(() => {
  if (originalWindow) Object.defineProperty(globalThis, 'window', originalWindow);
  else Reflect.deleteProperty(globalThis, 'window');
});
const builder = JSON.parse(readFileSync('../testdata/wire/trades/TargetTradeBuilder-preview.json', 'utf8'));

it.each(['reject', 'revoke', 'remove'] as const)('keeps the inspector/receipt %s action', async action => {
  const API = { TargetTradeBuilder: vi.fn(async () => builder) };
  vi.stubGlobal('window', { go: { trades: { API } } });
  await builderAction({ action, mine: '0001', id: '901' });
  expect(API.TargetTradeBuilder).toHaveBeenCalledWith({ mine: '0001', id: '901', action,
    other: '', give: [], get: [], rationale: '' });
  expect(useBuilder.getState()).toEqual({ busy: false, error: '' });
  expect(commands.loadMoves).toHaveBeenCalledWith('0001');
  API.TargetTradeBuilder.mockRejectedValueOnce(new Error('offline'));
  await builderAction({ action, mine: '0001', id: '901' });
  expect(useBuilder.getState()).toEqual({ busy: false, error: 'Error: offline' });
});

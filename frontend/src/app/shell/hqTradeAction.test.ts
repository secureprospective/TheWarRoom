import { afterEach, expect, it, vi } from 'vitest';
import { readFileSync } from 'node:fs';
import { addHQTrade } from './hqReads';
import { parseBuilder, useBuilder } from './tradeBuilderState';
import { commands } from '../commands/registry';

const wire = JSON.parse(readFileSync('../testdata/wire/trades/TargetTradeBuilder-preview.json', 'utf8'));
const original = Object.getOwnPropertyDescriptor(globalThis, 'window');
afterEach(() => {
  vi.restoreAllMocks();
  if (original) Object.defineProperty(globalThis, 'window', original);
  else Reflect.deleteProperty(globalThis, 'window');
});
it('adds a held player to the existing trade draft once, without proposing or submitting a deal',
  async () => {
  const mine = '0001';
  const player = parseBuilder(wire).value!.teams.find(t => t.id === mine)!.assets[0].token;
  const read = vi.fn(async () => wire), submit = vi.fn();
  vi.stubGlobal('window', { go: { trades: { API: { TargetTradeBuilderRead: read,
    TargetTradeBuilder: submit } } } });
  const navigate = vi.spyOn(commands, 'dispatch').mockImplementation(() => undefined);
  await addHQTrade(mine, player); await addHQTrade(mine, player);
  expect(useBuilder.getState().give).toEqual([player]);
  expect(submit).not.toHaveBeenCalled();
  expect(navigate).toHaveBeenCalledWith('nav.open', { node: 'trade', workspace: 'trade-desk' });
});
it('does not change the existing trade draft when the roster read fails', async () => {
  const before = [...useBuilder.getState().give];
  vi.stubGlobal('window', { go: { trades: { API: { TargetTradeBuilderRead: async () => {
    throw new Error('offline'); } } } } });
  await expect(addHQTrade('0001', '0042')).rejects.toThrow('offline');
  expect(useBuilder.getState().give).toEqual(before);
});

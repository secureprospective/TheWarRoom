import { afterEach, expect, it, vi } from 'vitest';
import { readFileSync } from 'node:fs';
import { targetDraftTradeAccept, targetTradeBuilder, targetTradeBuilderPreview, targetTradeBuilderRead, targetTrades } from './api';
import { draftTradeAccept, trades } from '../data/tradeProvider';
import { parseTrades } from '../data/parseTrades';
import { parseReceipt } from '../data/parseEnvelope';
import { parseBuilder } from '../shell/tradeBuilderState';

const originalWindow = Object.getOwnPropertyDescriptor(globalThis, 'window');
afterEach(() => {
  if (originalWindow) Object.defineProperty(globalThis, 'window', originalWindow);
  else Reflect.deleteProperty(globalThis, 'window');
});
const golden = (name: string) =>
  JSON.parse(readFileSync(`../testdata/wire/trades/${name}.json`, 'utf8'));
const request = {
  mine: '0001', other: '0025', action: 'preview', id: '', give: ['0099'], get: [], rationale: 'why',
};

it('routes every call through trades.API and parses their wire output', async () => {
  const reading = golden('TargetTrades');
  const receipt = golden('TargetDraftTradeAccept');
  const builder = golden('TargetTradeBuilder-preview');
  const API = {
    TargetTrades: vi.fn(async () => reading),
    TargetDraftTradeAccept: vi.fn(async () => receipt),
    TargetTradeBuilder: vi.fn(async () => builder),
    TargetTradeBuilderRead: vi.fn(async () => builder),
    TargetTradeBuilderPreview: vi.fn(async () => builder),
  };
  vi.stubGlobal('window', { go: { trades: { API } } });
  expect(await trades('0025')).toEqual(parseTrades(reading));
  expect(API.TargetTrades).toHaveBeenCalledWith('0025');
  expect(await draftTradeAccept('0025', '77')).toEqual(parseReceipt(receipt));
  expect(API.TargetDraftTradeAccept).toHaveBeenCalledWith('0025', '77');
  expect(parseBuilder(await targetTradeBuilder(request))).toEqual(parseBuilder(builder));
  expect(API.TargetTradeBuilder).toHaveBeenCalledWith(request);
  expect(parseBuilder(await targetTradeBuilderRead('0001'))).toEqual(parseBuilder(builder));
  expect(API.TargetTradeBuilderRead).toHaveBeenCalledWith('0001');
  expect(parseBuilder(await targetTradeBuilderPreview(request))).toEqual(parseBuilder(builder));
  expect(API.TargetTradeBuilderPreview).toHaveBeenCalledWith(request);
  API.TargetTrades.mockRejectedValueOnce(new Error('target trades: offline'));
  await expect(trades('0025')).rejects.toThrow('target trades: offline');
});

it('preserves the plain-browser refusal without substituting fixtures', async () => {
  for (const window of [undefined, {}]) {
    vi.stubGlobal('window', window);
    await expect(targetTrades('0025')).rejects.toThrow('Trades require the desktop app');
    await expect(targetDraftTradeAccept('0025', '77')).rejects.toThrow('Trades require the desktop app');
    await expect(targetTradeBuilder(request)).rejects.toThrow('Trades require the desktop app');
    await expect(targetTradeBuilderRead('0001')).rejects.toThrow('Trades require the desktop app');
    await expect(targetTradeBuilderPreview(request)).rejects.toThrow('Trades require the desktop app');
  }
});

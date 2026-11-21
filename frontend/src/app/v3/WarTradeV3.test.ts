import { describe, expect, it } from 'vitest';
import { shelfPasses } from './WarTradeV3';
import type { WarRoomAsset } from '../shell/warroomTradeReads';

const a = (token: string, position?: string) => ({ token, name: token, position }) as WarRoomAsset;

describe('shelfPasses', () => {
  it('splits the shelf into offence, defence and picks', () => {
    expect(shelfPasses(a('1', 'WR'), 'OFF')).toBe(true);
    expect(shelfPasses(a('2', 'LB'), 'DEF')).toBe(true);
    expect(shelfPasses(a('3', 'WR'), 'DEF')).toBe(false);
    expect(shelfPasses(a('pick:2027:1'), 'PICKS')).toBe(true);
    expect(shelfPasses(a('pick:2027:1'), 'OFF')).toBe(false);
  });
});

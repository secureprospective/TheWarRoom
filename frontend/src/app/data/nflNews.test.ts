import { describe, expect, it, vi } from 'vitest';
import { parseNFLNews, readNFLNews } from './nflNews';

describe('nfl.news.read', () => {
  it('asks the server for the league and the named players, and parses the reply', async () => {
    const port = vi.fn(async () => ({ data: { asOf: '2026-11-20T15:00:00Z', stale: false, items: [{
      at: '2026-11-20T14:00:00Z', player: '0099', name: 'A Player', position: 'WR', nflTeam: 'DAL',
      tag: 'INJ', text: 'Questionable (Ankle)', source: 'sleeper' }] } }));
    const news = await readNFLNews('native.test', ['0099', '0100'], port as never);
    expect(port).toHaveBeenCalledWith({ command: 'nfl.news.read', mode: 'read',
      arguments: { league: 'native.test', players: '0099,0100' } });
    await readNFLNews('native.test', [], port as never);
    expect(port).toHaveBeenLastCalledWith({ command: 'nfl.news.read', mode: 'read',
      arguments: { league: 'native.test', players: '' } });
    expect(news.stale).toBe(false);
    expect(news.items[0]).toMatchObject({ player: '0099', tag: 'INJ', source: 'sleeper' });
  });

  it('refuses a reply with a missing flag or a malformed item', () => {
    expect(() => parseNFLNews({ items: [] })).toThrow('stale');
    expect(() => parseNFLNews({ stale: false, items: [{ player: '1' }] })).toThrow();
    expect(parseNFLNews({ stale: true, items: [] })).toEqual({ asOf: null, stale: true, items: [] });
  });
});

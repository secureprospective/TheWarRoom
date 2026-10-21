import { afterEach, beforeEach, describe, expect, it } from 'vitest';
import { all, button, click, mount, render, settle, unmount }
  from '../commissioner/commissionerDom';
import { commands } from '../commands/registry';
import { CacheTools } from './CacheTools';
import { cacheState, parseCacheScopes } from './cacheState';

beforeEach(() => cacheState.forget());
afterEach(unmount);

const scopes = [
  { id: 'pool', label: 'Free agents and market reports', note: 'The free-agent list.',
    action: 'Read free agents again', progress: 'Reading…' },
  { id: 'race', label: 'League race', note: 'The standings.', action: 'Read the league race again',
    progress: 'Reading…' },
];
type Clear = (scope: string) => Promise<unknown>;
// The Go bindings, as the desktop app hands them over; the click travels the real verb and state.
function goes(clear: Clear, list: () => Promise<unknown> = async () => scopes) {
  return { sources: { API: { TargetCacheScopes: list, TargetCacheClear: clear } } };
}
const ok = (summary: string) => ({ ok: true, error: '', summary });

describe('Read data again (Control Room › Data and sources)', () => {
  it('lists the held copies Go can read afresh and asks for exactly the one chosen', async () => {
    const asked: string[] = [];
    const host = await mount(render(CacheTools), {}, goes(async scope => {
      asked.push(scope);
      return ok(`Asked for ${scope}.`);
    }));
    await settle();
    expect(all(host, 'section')[0].getAttribute('data-endpoint')).toBe('M-137');
    expect(all(host, 'dt').map(d => d.textContent)).toEqual(scopes.map(s => s.label));
    await click(button(host, 'Read the league race again'));
    await settle();
    expect(asked).toEqual(['race']);
    expect(host.textContent).toContain('Asked for race.');
    expect(host.textContent).not.toContain('Asked for pool.');
  });
  it('says what is happening and blocks a second request while one runs', async () => {
    let finish: () => void = () => undefined;
    const asked: string[] = [];
    const host = await mount(render(CacheTools), {}, goes(scope => new Promise(resolve => {
      asked.push(scope);
      finish = () => resolve(ok('Done.'));
    })));
    await settle();
    await click(button(host, 'Read free agents again'));
    await settle();
    expect(host.textContent).toContain('Reading…');
    expect(button(host, 'Read the league race again').attributes.has('disabled')).toBe(true);
    await click(button(host, 'Read the league race again'));
    finish();
    await settle();
    expect(asked).toEqual(['pool']);
    expect(host.textContent).toContain('Done.');
    expect(button(host, 'Read the league race again').attributes.has('disabled')).toBe(false);
  });
  it('shows Go’s own words when a source did not answer', async () => {
    const host = await mount(render(CacheTools), {}, goes(async () => ({ ok: false, summary: '',
      error: 'League race could not be read again: no answer. The copy held stays.' })));
    await settle();
    await click(button(host, 'Read the league race again'));
    await settle();
    expect(host.textContent).toContain('The copy held stays.');
  });
  it('says so when the list cannot be read, and invents no copies of its own', async () => {
    const host = await mount(render(CacheTools), {}, goes(async () => ok(''), async () => {
      throw new Error('startup failed');
    }));
    await settle();
    expect(host.textContent).toContain('startup failed');
    expect(all(host, 'dt')).toHaveLength(0);
  });
  it('reads the list strictly and uses no internal words', () => {
    expect(parseCacheScopes(scopes)).toEqual(scopes);
    expect(() => parseCacheScopes([{ ...scopes[0], hidden: true }])).toThrow('unexpected field');
    expect(() => parseCacheScopes([{ id: 'pool' }])).toThrow();
    expect(JSON.stringify(scopes)).not.toMatch(/cache|force|endpoint|MFL/i);
  });
  it('is a registered verb that needs one confirm', () => {
    expect(commands.registry['cache.run'].gravity).toBe('G1');
    expect(commands.registry['cache.run'].label).toBe('Read data again');
  });
});

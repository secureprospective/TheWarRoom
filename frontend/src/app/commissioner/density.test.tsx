import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { all, button, click, mount, render, settle, unmount } from './commissionerDom';
import { stub } from './commissionerStub';
import { leagueState } from './proxy';
import LeagueDesk from './LeagueDesk';
import LeagueRules from './LeagueRules';
import LeaguePeople from './LeaguePeople';
import LeagueLedger from './LeagueLedger';
import { DensityScreen } from './DensityScreen';

const storageDescriptor = Object.getOwnPropertyDescriptor(globalThis, 'localStorage');
let stored: Map<string, string>;
beforeEach(() => {
  stored = new Map();
  vi.stubGlobal('localStorage', {
    getItem: (key: string) => stored.get(key) ?? null,
    setItem: (key: string, value: string) => stored.set(key, value),
  });
  leagueState.forget();
});
afterEach(async () => {
  await unmount();
  if (storageDescriptor) Object.defineProperty(globalThis, 'localStorage', storageDescriptor);
  else Reflect.deleteProperty(globalThis, 'localStorage');
});

const screens = [
  { node: LeagueDesk, key: 'desk' }, { node: LeagueRules, key: 'rules' },
  { node: LeaguePeople, key: 'people' }, { node: LeagueLedger, key: 'ledger' },
];

describe('Commissioner density', () => {
  for (const screen of screens) {
    it(`${screen.key} opens Tactical and keeps all three views on return`, async () => {
      const app = stub().app;
      let host = await mount(render(screen.node), app);
      await leagueState.choose('native.rules');
      await settle();
      const section = () => all(host, 'section').find(s => s.attributes.has('data-density'))!;
      expect(section().getAttribute('data-density')).toBe('tactical');
      if (screen.key === 'rules') {
        expect(all(host, 'td').some(cell => cell.getAttribute('data-label') === 'Setting')).toBe(true);
      }
      expect(button(host, 'Tactical').getAttribute('aria-pressed')).toBe('true');
      for (const [label, value] of [['Narrative', 'narrative'], ['Matrix', 'matrix']]) {
        await click(button(host, label));
        expect(section().getAttribute('data-density')).toBe(value);
        expect(button(host, label).getAttribute('aria-pressed')).toBe('true');
        expect(stored.get(`twr.density.commissioner.${screen.key}`)).toBe(value);
      }
      await unmount();
      host = await mount(render(screen.node), app);
      await settle();
      expect(section().getAttribute('data-density')).toBe('matrix');
    });
  }
  it('changes the view through CSS without rerendering the commissioner content', async () => {
    let renders = 0;
    const Content = () => { renders++; return render('p', null, 'League facts'); };
    const host = await mount(render(DensityScreen,
      { screen: 'ledger', label: 'Ledger', children: render(Content) }));
    for (const view of ['Narrative', 'Matrix', 'Tactical']) await click(button(host, view));
    expect(renders).toBe(1);
  });
  it('does not inherit another screen or the owner shell density', async () => {
    stored.set('twr.density', 'matrix');
    stored.set('twr.density.commissioner.people', 'narrative');
    const host = await mount(render(DensityScreen, { screen: 'ledger', label: 'Ledger', children: 'Facts' }));
    expect(button(host, 'Tactical').getAttribute('aria-pressed')).toBe('true');
    await click(button(host, 'Matrix'));
    expect(stored.get('twr.density.commissioner.people')).toBe('narrative');
    expect(stored.get('twr.density')).toBe('matrix');
  });
  it('opens Tactical with corrupt or inaccessible storage and still switches views', async () => {
    stored.set('twr.density.commissioner.ledger', 'bad');
    const props = { screen: 'ledger', label: 'Ledger', children: 'Facts' };
    let host = await mount(render(DensityScreen, props));
    expect(button(host, 'Tactical').getAttribute('aria-pressed')).toBe('true');
    await unmount();
    vi.stubGlobal('localStorage', {
      getItem: () => { throw new Error('Unavailable'); },
      setItem: () => { throw new Error('Unavailable'); },
    });
    host = await mount(render(DensityScreen, props));
    await click(button(host, 'Narrative'));
    expect(button(host, 'Narrative').getAttribute('aria-pressed')).toBe('true');
  });
});

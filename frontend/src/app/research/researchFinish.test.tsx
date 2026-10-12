import { readFileSync } from 'node:fs';
import { act, createElement } from 'react';
import { createRoot, type Root } from 'react-dom/client';
import { renderToStaticMarkup } from 'react-dom/server';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { KeyDocument, descendants } from '../shell/mflKeyTestDom';
import { commands } from '../commands/registry';
import { ResearchView } from './Research';
import { PlayerResearchView } from './PlayerInspector';
import { parseResearch, type PlayerReading } from './contract';
import { createResearchState, type ResearchReader } from './state';
import wire from './wire.testdata.json';

const fixture = parseResearch(wire);
const globals = ['window', 'document', 'IS_REACT_ACT_ENVIRONMENT'].map(key => ({
  key, descriptor: Object.getOwnPropertyDescriptor(globalThis, key),
}));
let root: Root | undefined;
afterEach(async () => {
  if (root) await act(async () => root?.unmount());
  root = undefined;
  vi.restoreAllMocks(); vi.useRealTimers();
  for (const { key, descriptor } of globals) {
    if (descriptor) Object.defineProperty(globalThis, key, descriptor);
    else Reflect.deleteProperty(globalThis, key);
  }
});
describe('Research live-look fixes', () => {
  it('submits the search through the native Enter form path and its submit button', async () => {
    const document = new KeyDocument();
    const element = document.createElement('div');
    vi.stubGlobal('document', document);
    vi.stubGlobal('window', { document, HTMLIFrameElement: class {} });
    vi.stubGlobal('IS_REACT_ACT_ENVIRONMENT', true);
    const dispatch = vi.spyOn(commands, 'dispatch').mockImplementation(() => undefined);
    root = createRoot(element as unknown as HTMLElement);
    await act(async () => root?.render(createElement(ResearchView, {
      research: fixture, selected: ['M-037'],
    })));
    const form = descendants(element).find(e => e.tagName === 'FORM')!;
    const button = descendants(element).find(e =>
      e.tagName === 'BUTTON' && e.textContent === 'Search players')!;
    expect(button.getAttribute('type')).toBe('submit');
    expect(button.props().onClick).toBeUndefined();
    const preventDefault = vi.fn();
    await act(async () => {
      (form.props().onSubmit as (event: { preventDefault(): void }) => void)({ preventDefault });
    });
    expect(preventDefault).toHaveBeenCalledOnce();
    expect(dispatch).toHaveBeenCalledWith('research.read', { action: 'search', value: 'Gibbs' });
    expect(element.textContent).toContain('Gibbs, Jahmyr · RB · DET ·');
    expect(descendants(element).some(e => e.tagName === 'SUMMARY' && e.textContent.includes('matches')))
      .toBe(false);
  });
  it('prompts on an empty query and uses one shared dark-theme search rule', () => {
    const html = renderToStaticMarkup(createElement(ResearchView, {
      research: { ...fixture, scope: { ...fixture.scope, search: '' } }, selected: ['M-037'],
    }));
    expect(html).toContain('Enter a name, position or NFL team');
    expect(html).not.toContain('matches');
    const read = (p: string) => readFileSync(new URL(`../${p}`, import.meta.url), 'utf8');
    expect(read('look/look.css').match(/\.twr-app \.search-input \{/g)).toHaveLength(1);
    expect(read('control/control.css')).not.toContain("input[type='search']");
    expect(read('control/LeagueRoom.tsx')).toContain('className="search-input"');
    expect(read('research/Research.tsx')).toContain('className="search-input"');
    expect(read('research/research.css')).toContain('scrollbar-gutter: stable');
  });
  it('renders held reports and missing feed times under both Inspector shapes', () => {
    const player: PlayerReading = { ...fixture.players[0], report: {
      ...fixture.news, columns: ['Reading', 'Held fact', 'Feed time'], rows: [
        { cells: ['NFL depth', 'not held', 'not held · feed time not held'], range: null },
      ], summary: [], status: 'ok',
    }, articles: fixture.articles, howLabel: fixture.board.howLabel };
    for (const identity of [true, false]) {
      const html = renderToStaticMarkup(createElement(PlayerResearchView, { player, identity }));
      expect(html).toContain('not held · feed time not held');
      expect(html).toContain('Open article on MFL');
      expect(html.includes('<h4>Gibbs, Jahmyr</h4>')).toBe(identity);
      expect(html).not.toContain('Open this player from your roster');
    }
    const shell = readFileSync(new URL('../shell/PlayerInspector.tsx', import.meta.url), 'utf8');
    expect(shell).toContain('<ResearchInspector id={subject.id} />');
    expect(shell).toContain('<ResearchInspector id={subject.id} identity={false} />');
    expect(shell).toContain('<PlayerLists id={subject.id} />');
    expect(shell).toContain('<ContractPanel id={subject.id} />');
  });
  it('opens Inspector-held articles without a Research board and rejects forged articles', async () => {
    const player: PlayerReading = { ...fixture.players[0], report: fixture.news,
      articles: fixture.articles, howLabel: fixture.board.howLabel };
    const open = vi.fn(async () => undefined);
    const reader: ResearchReader = { read: async () => fixture, refresh: async () => undefined,
      news: async () => undefined, player: async () => player, open };
    const state = createResearchState(reader, () => '');
    await state.choose('open', '16162:rotowire640841');
    expect(open).toHaveBeenCalledWith('16162', 'rotowire640841');
    await expect(state.choose('open', '16162:forged')).rejects.toThrow('not held');
    expect(open).toHaveBeenCalledOnce();
  });
  it('shows the waiting and week-reading stages without losing the archived board', async () => {
    vi.useFakeTimers();
    let finish: (() => void) | undefined;
    let status = 'Waiting for the MFL refresh to finish';
    const reader: ResearchReader = { read: async () => fixture,
      refresh: () => new Promise(resolve => { finish = resolve; }),
      status: async () => status, news: async () => undefined, open: async () => undefined };
    const state = createResearchState(reader, () => '0001');
    await state.load();
    const pending = state.load(true);
    expect(state.peek().research).toBe(fixture);
    expect(state.peek().status).toBe(status);
    status = 'Reading week 3 of 5';
    vi.advanceTimersByTime(250);
    await Promise.resolve();
    expect(state.peek().status).toBe(status);
    finish?.(); await pending;
    expect(state.peek().pending).toBe(false);
    expect(vi.getTimerCount()).toBe(0);
  });
});

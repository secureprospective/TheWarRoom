// @vitest-environment jsdom
import { act } from 'react';
import { createRoot, type Root } from 'react-dom/client';
import { afterEach, describe, expect, it } from 'vitest';
import type { Game } from '../now/liveContract';
import { LivePregame, type PregameGuy } from './LivePregame';

const basePre: Game = {
  id: '401873100',
  kickoff: '2026-11-22T18:00:00Z',
  away: { id: '15', abbreviation: 'MIA', score: 0, record: '4-6' },
  home: { id: '2', abbreviation: 'BUF', score: 0, record: '8-2' },
  status: 'pre',
  clock: '',
  period: 0,
  state: null,
  last: null,
  box: [],
  plays: [],
  predictor: { away: 40.6, home: 59.1 },
  venue: { name: 'Highmark Stadium', city: 'Orchard Park', state: 'NY', country: 'USA', indoor: false },
  weather: { text: 'Mostly Cloudy', tempF: 66, wind: '7 mph S', precipPct: 0, source: 'NWS' },
};

const guys: PregameGuy[] = [
  { id: 'p1', name: 'Khalil Shakir', position: 'WR', team: 'BUF', side: 'YOURS', mine: true,
    projection: 11.4 },
  { id: 'p2', name: 'Keon Coleman', position: 'WR', team: 'BUF', side: 'CAR', mine: false,
    projection: 8.2 },
];

let root: Root | undefined;
let host: HTMLDivElement | undefined;
afterEach(async () => {
  if (root) await act(async () => root!.unmount());
  host?.remove(); root = undefined; host = undefined;
});

async function mount(node: React.ReactElement) {
  Object.assign(globalThis, { IS_REACT_ACT_ENVIRONMENT: true });
  host = document.createElement('div');
  document.body.append(host);
  root = createRoot(host);
  await act(async () => root!.render(node));
}

describe('LivePregame', () => {
  it('renders the Broadcast split stage, CT clock + countdown, win-chance bar, stadium and crawl',
    async () => {
    const now = Date.parse('2026-11-22T15:45:00Z');
    await mount(<LivePregame game={basePre} guys={guys} now={now} reduced={false} />);

    const wrap = host!.querySelector<HTMLElement>('[data-testid="live-pregame"]')!;
    expect(wrap.style.getPropertyValue('--bc-lp')).toBe('#008E97');
    expect(wrap.style.getPropertyValue('--bc-rp')).toBe('#00338D');
    expect(wrap.querySelector('.bc-l')?.textContent).toContain('MIA');
    expect(wrap.querySelector('.bc-l')?.textContent).toContain('DOLPHINS');
    expect(wrap.querySelector('.bc-l')?.textContent).toContain('4-6');
    expect(wrap.querySelector('.bc-r')?.textContent).toContain('BUF');
    expect(wrap.querySelector('.bc-r')?.textContent).toContain('BILLS');
    expect(wrap.querySelector('.bc-r')?.textContent).toContain('8-2');

    const clock = host!.querySelector('[data-testid="pregame-clock"]')!.textContent!;
    expect(clock).toContain('SUN');
    expect(clock).toContain('12:00');
    expect(clock).toContain('PM CT');
    expect(clock).toContain('IN 2H 15M');

    const bar = host!.querySelector('[data-testid="pregame-winbar"]')!;
    expect(bar.getAttribute('aria-label')).toBe('MIA 41% · BUF 59% win chance');
    expect(bar.textContent).toContain('MIA 41%');
    expect(bar.textContent).toContain('BUF 59%');

    const stadium = host!.querySelector('[data-testid="pregame-stadium"]')!;
    expect(stadium.textContent).toContain('HIGHMARK STADIUM · ORCHARD PARK, NY');
    expect(host!.querySelector('[data-testid="pregame-weather"]')?.textContent)
      .toBe('66°F · Mostly Cloudy · wind 7 mph S · rain 0%');

    const crawl = host!.querySelector('[data-testid="pregame-crawl"]')!;
    expect(crawl.classList.contains('is-static')).toBe(false);
    expect(crawl.textContent).toContain('Khalil Shakir WR · BUF · YOURS');
    expect(crawl.textContent).toContain('Keon Coleman WR · BUF · CAR');
    // Marquee duplicates items once for seamless loop when motion is allowed.
    expect(crawl.querySelectorAll('.cr')).toHaveLength(4);
  });

  it('hides the win-chance bar when predictor is missing', async () => {
    const noPred: Game = { ...basePre, predictor: undefined };
    await mount(<LivePregame game={noPred} guys={guys} now={Date.parse('2026-11-22T17:00:00Z')}
      reduced={false} />);
    expect(host!.querySelector('[data-testid="pregame-winbar"]')).toBeNull();
  });

  it('hides the weather line when weather is missing', async () => {
    const noWeather: Game = { ...basePre, weather: undefined };
    await mount(<LivePregame game={noWeather} guys={guys} now={Date.parse('2026-11-22T17:00:00Z')}
      reduced={false} />);
    expect(host!.querySelector('[data-testid="pregame-stadium"]')?.textContent)
      .toContain('HIGHMARK STADIUM');
    expect(host!.querySelector('[data-testid="pregame-weather"]')).toBeNull();
  });

  it('shows exact Indoors with no temperature for indoor venues', async () => {
    const indoor: Game = {
      ...basePre,
      venue: { name: 'Ford Field', city: 'Detroit', state: 'MI', indoor: true },
      weather: { text: 'Clear', tempF: 72 },
    };
    await mount(<LivePregame game={indoor} guys={guys} now={Date.parse('2026-11-22T17:00:00Z')}
      reduced={false} />);
    const wx = host!.querySelector('[data-testid="pregame-weather"]')!.textContent!;
    expect(wx).toBe('Indoors');
    expect(wx).not.toContain('72');
    expect(wx).not.toContain('°');
  });

  it('hides the crawl completely when no selected-matchup players play in this game', async () => {
    await mount(<LivePregame game={basePre} guys={[]} now={Date.parse('2026-11-22T17:00:00Z')}
      reduced={false} />);
    expect(host!.querySelector('[data-testid="pregame-crawl"]')).toBeNull();
  });

  it('turns the crawl into a static list with no duplicate items under reduced motion', async () => {
    await mount(<LivePregame game={basePre} guys={guys} now={Date.parse('2026-11-22T17:00:00Z')}
      reduced={true} />);
    const crawl = host!.querySelector('[data-testid="pregame-crawl"]')!;
    expect(crawl.classList.contains('is-static')).toBe(true);
    expect(crawl.querySelectorAll('.cr')).toHaveLength(2);
    expect(crawl.querySelectorAll('.bc-dup')).toHaveLength(0);
  });
});

import { describe, expect, it } from 'vitest';
import { renderToStaticMarkup } from 'react-dom/server';
import { HQBoard, boardSlots, FormChart } from './HQBoard';
import { heldLineup, heldSnapshot } from '../data/lineupTestData';
import { projections, totalProjection, formPoints, byeWeeks } from './hqPlayerData';
import { unavailable } from './hqModel';
import type { Board, Report } from '../reports/contract';
import type { PlayerReading } from '../research/contract';

function report(id: string): Report { return { id, title: 'Projected points · NFL week 5', status: 'ok',
  source: '', asOf: '', how: '', note: '', columns: ['Player', 'Points'], summary: [], link: null, rows: []
    }; }
const board = (r: Report): Board => ({ title: '', loading: '', refresh: '', howLabel: '', mflLabel: '',
  mflUrl: '', reports: [r] });
describe('HQ starter and bench board', () => {
  it('maps projections only to a unique player and the requested week, never missing to zero', () => {
    const snap = heldSnapshot(); const p = snap.players.value[0]; const r = report('M-046');
    r.rows = [{ cells: [p.name!, '12.25'], range: null }];
    expect(projections(snap, board(r), 5).get(p.id)).toBe(12.25);
    expect(projections(snap, board(r), 6).size).toBe(0);
    snap.players.value.push({ ...p, id: '99999' });
    expect(projections(snap, board(r), 5).size).toBe(0);
    expect(totalProjection(['missing'], new Map())).toBe(unavailable);
    expect(totalProjection(['one', 'two'], new Map([['one', 1], ['two', 2.5]]))).toBe('3.5');
  });
  it('renders saved starter and bench names, actual projections, and honest missing form/opponents', () => {
    const snapshot = heldSnapshot(); const reading = heldLineup(); const id = reading.starters[0].id;
    const html = renderToStaticMarkup(<HQBoard snapshot={snapshot} mine="0025" reading={reading}
      settings={{ positions: [], starters: reading.starterCount }} values={new Map([[id, 17.25]])}
        select={() => {}} />);
    expect(html).toContain(snapshot.players.value.find(p => p.id === id)!.name!);
    expect(html).toContain('17.3');
    expect(html).toContain('STARTING 21');
    expect(html).toContain('BENCH'); expect(html).toContain(unavailable);
    expect(html.indexOf('STARTING')).toBeLessThan(html.indexOf('BENCH'));
  });
  it('changes open rows when the starter count changes; never uses the mock count', () => {
    const reading = heldLineup(); reading.starters = [];
    expect(boardSlots({ ...reading, starterCount: 3 }, { positions: [] })).toHaveLength(3);
    expect(boardSlots({ ...reading, starterCount: 7 }, { positions: [] })).toHaveLength(7);
    expect(boardSlots(undefined, { positions: [] })).toEqual([]);
  });
  it('charts only held finished scores, retaining real zero and negative weeks', () => {
    const research = { report: { rows: [{ cells: ['Form', 'W2: 0.00, W3: -2.25, W4: 15.50 · 4.42 PPG'] }] }
      } as PlayerReading;
    expect(formPoints(research)).toEqual([{ week: 2, points: 0 }, { week: 3, points: -2.25 }, { week: 4,
      points: 15.5 }]);
    const html = renderToStaticMarkup(<FormChart id="1" projection={10} facts={new Map([['1', { research,
      error: '' }]])} />);
    expect(html).toContain('Wk 3 · -2.3'); expect(html).toContain('3 held finished weeks');
    expect(renderToStaticMarkup(<FormChart id="none" />)).toContain(unavailable);
  });
  it('maps bye week from the held roster report, not a team-name assumption', () => {
    const snapshot = heldSnapshot(); const player = snapshot.players.value[0]; const r = report('M-154');
    r.rows = [{ cells: [player.name!, '', '', '5'], range: null }];
    expect(byeWeeks(snapshot, [r]).get(player.id)).toBe(5);
    expect(byeWeeks(snapshot).size).toBe(0);
  });
});

it('renders real opponent projection only when the existing preview reports complete inputs', async () => {
  const { HQMatchup } = await import('./HQBoard');
  const matchup = report('X-03'); matchup.columns = ['Slot', 'Mine', 'Points', 'Opponent', 'Points'];
  matchup.summary = [{ cells: ['Opponent', 'Other team'], range: null }];
  matchup.rows = [{ cells: ['QB', 'My QB', '', 'Other QB', ''], range: null }];
  matchup.entities = [{ kind: 'player', id: '0042', text: 'My QB' },
    { kind: 'player', id: '0043', text: 'Other QB' }];
  const preview = report('M-018'); preview.rows = [{ cells: ['Mine', '171.2', 'Other team', '164.9', 'Mine'],
    mine: true, range: null }];
  preview.summary = [{ cells: ['Starters MFL does not project', '0'], range: null }];
  const now = { matchup, board: board(preview), week: 5, weeks: [], reportWeek: 5, reportWeeks: [],
    waiting: '' };
  const render = () => renderToStaticMarkup(<HQMatchup now={now} ownProjection="171.2"
    values={new Map([['0042', 22.1], ['0043', 20.8]])} />);
  expect(render()).toContain('164.9'); expect(render()).toContain('+1.3');
  preview.summary[0].cells[1] = '1';
  expect(render()).not.toContain('164.9'); expect(render()).toContain(unavailable);
});

it('renders inline taxi and IR from roster reads and reacts to capacity changes', async () => {
  const { InlineReserves } = await import('./HQBoard');
  const snapshot = heldSnapshot();
  const roster = snapshot.rosters.value.find(r => r.franchiseId === '0025')!;
  roster.players = roster.players.slice(0, 2);
  roster.players[0].rosterStatus = 'TAXI_SQUAD';
  roster.players[1].rosterStatus = 'IR';
  snapshot.players.value.find(p => p.id === roster.players[0].id)!.isRookie = true;
  const render = (taxi: number) => renderToStaticMarkup(<InlineReserves snapshot={snapshot} mine="0025"
    settings={{ positions: [], taxi, ir: 12 }} select={() => {}} injuries={[{
      id: roster.players[1].id, status: 'IR', details: 'Ankle', returnSeason: snapshot.league.value.season,
      returnWeek: 11, returnReason: 'Reported return',
    }]} />);
  expect(render(8)).toContain('TAXI SQUAD'); expect(render(8)).toContain('1 of 8');
  expect(render(8)).toContain('yr 1'); expect(render(8)).toContain('Wk 11');
  expect(render(9)).toContain('1 of 9'); expect(render(9)).not.toContain('1 of 8');
});

it('renders five weekly bars, hatched bye, projection and average from the extended held read', () => {
  const research = { report: { rows: Array.from({ length: 5 }, (_, i) => ({
    cells: [`Form · Wk ${i + 4}`, i === 2 ? 'not held' : String(i * 2)],
  })) } } as PlayerReading;
  const html = renderToStaticMarkup(<FormChart id="0042" week={9} bye={6} projection={9}
    facts={new Map([['0042', { research, error: '' }]])} />);
  expect((html.match(/<i /g) ?? []).length).toBe(5);
  expect(html).toContain('class="bye"'); expect(html).toContain('Wk 6 · bye');
  expect(html).toContain('avg 4.0'); expect(html).toContain('class="pj"');
});

import { describe, expect, it } from 'vitest';
import { renderToStaticMarkup } from 'react-dom/server';
import { HQHeader } from './HQHeader';
import { capFromReads, lineupCheck, settingsFromReport, unavailable } from './hqModel';
import { heldLineup, heldSnapshot } from '../data/lineupTestData';
import type { Report } from '../reports/contract';

const settings = (active: number): Report => ({ id: 'M-091', title: '', note: '', status: 'ok',
  source: '', asOf: '', how: '', columns: ['Setting', 'Value'], summary: [], link: null,
  rows: [{ cells: ['Roster size', `${active} players`], range: null }] });
describe('HQ franchise plate', () => {
  it('renders read values, preserving cents and unknown committed/dead split', () => {
    const snapshot = heldSnapshot();
    const cap = capFromReads(snapshot, '0025');
    const html = renderToStaticMarkup(<HQHeader name="My franchise" record="4–2" standing="2nd East"
      active={38} settings={settingsFromReport(settings(44))} cap={cap}
      check={lineupCheck(heldLineup())} lock={{ id: 'lineup-w5', text: '2d 5h' }} />);
    expect(html).toContain('My franchise');
    expect(html).toContain('38 / 44');
    expect(html).toContain('Lineup is full');
    expect(html).toContain('2d 5h');
    expect(html).toContain(`Committed ${unavailable}`);
    expect(html).toContain(`Dead ${unavailable}`);
  });
  it('changes roster capacity and cap when the league reads change', () => {
    const snapshot = heldSnapshot();
    const screen = (limit: number) => renderToStaticMarkup(<HQHeader name="Team" active={38}
      settings={settingsFromReport(settings(limit))} cap={capFromReads(snapshot, '0025')}
      check={unavailable} />);
    const before = screen(44);
    snapshot.league.value.salaryCap = 12_345_678_900;
    const after = screen(52);
    expect(before).toContain('38 / 44');
    expect(after).toContain('38 / 52');
    expect(after).toContain('$123.46M');
    expect(before).not.toContain('$123.46M');
  });
  it('uses the honest empty state for unavailable lineup rules and settings', () => {
    expect(settingsFromReport()).toEqual({ positions: [] });
    expect(lineupCheck(undefined)).toBe(unavailable);
    expect(lineupCheck(heldLineup(), 'Read failed')).toBe(unavailable);
  });
});

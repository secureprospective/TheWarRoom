import { expect, it } from 'vitest';
import { renderToStaticMarkup } from 'react-dom/server';
import { readFileSync } from 'node:fs';
import { WarRoomDates, WarRoomTrayMessage } from './WarRoomHeader';
import { heldSnapshot } from '../data/lineupTestData';
import { HQInspector } from './HQInspector';
import { WarRoomInspector } from './WarRoomInspector';
import { pickPlayer, unavailable } from './hqModel';

it('asks for a player instead of reporting failures when the HQ inspector has none selected', () => {
  const html = renderToStaticMarkup(<HQInspector native snapshot={heldSnapshot()} mine="franchise.01" />);
  expect(html).toContain(pickPlayer);
  expect(html).not.toContain(unavailable);
});

it('asks for a player instead of reporting failures when the War Room inspector has none selected', () => {
  const html = renderToStaticMarkup(<WarRoomInspector context={{ snapshot: heldSnapshot(),
    mine: 'franchise.01', week: 0 }} />);
  expect(html).toContain(pickPlayer);
  expect(html).not.toContain(unavailable);
  expect(html).not.toContain('pts / game');
});

it('says a quiet War Room strip is quiet, and never breaks a word to fit a narrow cell', () => {
  const held = renderToStaticMarkup(<WarRoomDates events={[{ id: 'w', label: 'Lineup',
    at: '2026-11-13T12:00:00Z' }]} />);
  expect(held).toContain('Lineup');
  const none = renderToStaticMarkup(<WarRoomDates events={[]} />);
  expect(none).toContain('No dates yet');
  expect(none).not.toContain(unavailable);
  expect(renderToStaticMarkup(<WarRoomDates events={[]} failed reason="port down" />))
    .toContain(unavailable);
  expect(renderToStaticMarkup(<WarRoomTrayMessage problems={['', '']} />)).not.toContain(unavailable);
  expect(renderToStaticMarkup(<WarRoomTrayMessage problems={['header read failed']} />))
    .toContain(unavailable);
  const css = readFileSync(new URL('./warroom.css', import.meta.url), 'utf8');
  expect(css).toContain('.wr-v6.embedded .empty{overflow-wrap:break-word}');
  expect(css).not.toContain('overflow-wrap:anywhere');
});

// The dates strip is eight equal columns. Its message is one child, so without a rule of its own it
// is squeezed into an eighth of the strip and wraps between words: "No dates / yet", "Not /
// available / yet". The message takes the whole row and stays on one line.
it('gives the War Room dates strip message the whole row, on one line', () => {
  const css = readFileSync(new URL('./warroom.css', import.meta.url), 'utf8');
  const rule = /\.wr-v6 \.cal>\.empty\{([^}]*)\}/.exec(css);
  expect(rule).not.toBeNull();
  expect(rule![1]).toContain('grid-column:1/-1');
  expect(rule![1]).toContain('white-space:nowrap');
  for (const empty of [<WarRoomDates key="none" events={[]} />,
    <WarRoomDates key="failed" events={[]} failed reason="port down" />]) {
    const html = renderToStaticMarkup(empty);
    expect(html.match(/class="empty"/g)).toHaveLength(1);
  }
});

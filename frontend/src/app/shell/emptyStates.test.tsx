import { expect, it } from 'vitest';
import { renderToStaticMarkup } from 'react-dom/server';
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

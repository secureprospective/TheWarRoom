import { renderToStaticMarkup } from 'react-dom/server';
import { describe, expect, it, vi } from 'vitest';

vi.mock('./warroomReads', () => ({ useWarRoomRead: () => ({ reading: {
  offers: [{ id: 'offer-1', partner: 'River City', date: 'Nov 17', status: 'accepted' }],
  partners: [{ name: 'River City', talks: 2, deals: 1 }], saved: 3,
} }) }));

import { WarRoomPlans } from './WarRoomPlans';

describe('War Room plans', () => {
  it('shows one honest plans empty state and keeps looking-back history', () => {
    const html = renderToStaticMarkup(<WarRoomPlans context={{} as never} openPocket={() => {}} />);
    expect(html.match(/Plans are not available yet/g)).toHaveLength(1);
    expect(html).toContain('LOOKING BACK');
    expect(html).toContain('River City · Nov 17');
    expect(html).toContain('ACCEPTED');
    expect(html).toContain('2 talks · 1 deal');
    expect(html).toContain('3 built, not sent');
    expect(html).not.toContain('MY NEEDS');
    expect(html).not.toContain('WHERE THINGS STAND');
  });
});
